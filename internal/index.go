package internal

import (
	"context"
	"fmt"
	"log"
	glog "log"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

const ChanSize = 1000

type DiskIndexManager struct {
	consumeLogChan   chan *LogEntry
	consumeBatchChan chan []*LogEntry
	index            bleve.Index
	mu               sync.RWMutex
}

func NewBleveIndexManager(indexPath string) (IndexManager, error) {
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		mapping := createIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create Bleve index: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to open Bleve index: %v", err)
	}

	m := &DiskIndexManager{
		index:            index,
		consumeLogChan:   make(chan *LogEntry, ChanSize),
		consumeBatchChan: make(chan []*LogEntry, ChanSize),
	}

	go m.Listen(context.Background())

	return m, nil
}

func (m *DiskIndexManager) Listen(ctx context.Context) {
	for {
		select {
		case batch := <-m.consumeBatchChan:
			if err := m.IndexBatch(ctx, batch); err != nil {
				glog.Printf("failed to index a batch: %v\n", err)
			}
		}
	}
}

func (m *DiskIndexManager) Consume(ctx context.Context, batch []*LogEntry) {
	println("[batch c]", len(m.consumeBatchChan))
	m.consumeBatchChan <- batch
}

func (m *DiskIndexManager) IndexLog(ctx context.Context, log *LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := positionToId(&log.position)
	if err := m.index.Index(id, log); err != nil {
		errString := fmt.Sprintf("failed to index log %s: %v", id, err)
		glog.Println(errString)
		return err
	}
	return nil
}

func (m *DiskIndexManager) IndexBatch(ctx context.Context, batch []*LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Println("indexing a batch...")

	start := time.Now()

	b := m.index.NewBatch()
	for _, log := range batch {
		id := positionToId(&log.position)
		if err := b.Index(id, log); err != nil {
			glog.Printf("failed to index log %s: %v\n", id, err)
			// TODO: collect per-log errors and allow partial success
			return fmt.Errorf("failed to index log %s: %v", id, err)
		}
	}

	if err := m.index.Batch(b); err != nil {
		return fmt.Errorf("failed to index batch: %v", err)
	}

	log.Printf("batch indexing took %f seconds", time.Since(start).Seconds())
	return nil
}

func (m *DiskIndexManager) DeleteFromIndex(ctx context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	start := time.Now()
	errors := []error{}
	for _, id := range ids {
		if err := m.index.Delete(id); err != nil {
			log.Printf("failed to delete log %s: %v\n", id, err)
			errors = append(errors, err) // TODO: return errors
		}
	}

	log.Printf("deleted %d entries from the index in %dms\n", len(ids), time.Since(start).Milliseconds())

	return nil
}

func (m *DiskIndexManager) DeleteSingleLogFromIndex(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.index.Delete(id); err != nil {
		return fmt.Errorf("failed to delete log %s: %v", id, err)
	}
	return nil
}

func (m *DiskIndexManager) Search(ctx context.Context, q Query) (*SearchResult, error) {
	count, _ := m.index.DocCount()
	println("index count (search):", count)

	searchRequest := createSearchRequest(&q)

	startTime := time.Now()
	bleveResult, err := m.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	result := &SearchResult{
		Total:      int(bleveResult.Total),
		SearchTime: time.Since(startTime).Milliseconds(),
		Matches:    LogPositions{},
	}

	for _, hit := range bleveResult.Hits {
		logPosition := idToPosition(hit.ID)
		result.Matches[logPosition.BatchPath] = append(result.Matches[logPosition.BatchPath], logPosition)
	}

	result.RetrievalTime = time.Since(startTime).Milliseconds()

	return result, nil
}

func (m *DiskIndexManager) Size(ctx context.Context) uint64 {
	size, err := m.index.DocCount()
	if err != nil {
		panic(err)
	}

	return size
}

func (m *DiskIndexManager) Close(ctx context.Context) error {
	return m.index.Close()
}

func createIndexMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	// Disable default mapping to enforce strict schema
	indexMapping.DefaultMapping.Enabled = false

	// LogEntry document mapping
	logEntryMapping := bleve.NewDocumentMapping()

	// Timestamp (numeric)
	timestampMapping := bleve.NewNumericFieldMapping()
	timestampMapping.Store = false
	logEntryMapping.AddFieldMappingsAt("timestamp", timestampMapping)

	// Line (full-text search)
	lineMapping := bleve.NewTextFieldMapping()
	lineMapping.Analyzer = "standard" // Use standard analyzer
	lineMapping.Store = false
	logEntryMapping.AddFieldMappingsAt("line", lineMapping)

	// KV pairs (dynamic fields)
	kvMapping := bleve.NewDocumentMapping()
	kvMapping.Enabled = true
	kvMapping.Dynamic = true
	logEntryMapping.AddSubDocumentMapping("kv", kvMapping)

	indexMapping.AddDocumentMapping("log_entry", logEntryMapping)
	indexMapping.DefaultMapping = logEntryMapping

	return indexMapping
}
