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

type BleveIndexManager struct {
	index bleve.Index
	mu    sync.RWMutex
}

func NewBleveIndexManager(indexPath string) (*BleveIndexManager, error) {
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

	return &BleveIndexManager{
		index: index,
	}, nil
}

func (m *BleveIndexManager) IndexLog(ctx context.Context, log *LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := positionToId(&log.position)
	if err := m.index.Index(id, log); err != nil {
		return fmt.Errorf("failed to index log %s: %v", id, err)
	}
	return nil
}

func (m *BleveIndexManager) IndexBatch(ctx context.Context, batch []LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Println("indexing a batch...")
	println(m.index.DocCount())

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

func (m *BleveIndexManager) DeleteFromIndex(ctx context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range ids {
		if err := m.index.Delete(id); err != nil {
			return fmt.Errorf("failed to delete log %s: %v", id, err)
		}
	}
	return nil
}

func (m *BleveIndexManager) DeleteSingleLogFromIndex(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.index.Delete(id); err != nil {
		return fmt.Errorf("failed to delete log %s: %v", id, err)
	}
	return nil
}

func (m *BleveIndexManager) Search(ctx context.Context, q Query) (*SearchResult, error) {
	searchQuery := bleve.NewConjunctionQuery()

	println(m.index.DocCount())

	if q.Text != "" {
		textQuery := bleve.NewMatchQuery(q.Text)
		textQuery.SetField("line")
		searchQuery.AddQuery(textQuery)
	}

	startAsFloat := float64(q.TimeRange.Start)
	endAsFloat := float64(q.TimeRange.End)

	if q.TimeRange.Start != 0 || q.TimeRange.End != 0 {
		timeQuery := bleve.NewNumericRangeQuery(&startAsFloat, &endAsFloat)
		timeQuery.SetField("timestamp")
		searchQuery.AddQuery(timeQuery)
	}

	for key, value := range q.Filters {
		termQuery := bleve.NewTermQuery(value)
		termQuery.SetField(fmt.Sprintf("kv.%s", key))
		searchQuery.AddQuery(termQuery)
	}

	searchRequest := bleve.NewSearchRequest(searchQuery)
	searchRequest.Size = q.MaxResults

	startTime := time.Now()
	bleveResult, err := m.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	result := &SearchResult{
		Total:      int(bleveResult.Total),
		SearchTime: time.Since(startTime).Milliseconds(),
	}

	for _, hit := range bleveResult.Hits {
		logPosition := idToPosition(hit.ID)
		result.Matches[logPosition.BatchPath] = append(result.Matches[logPosition.BatchPath], logPosition)
	}

	return result, nil
}

func (m *BleveIndexManager) Close(ctx context.Context) error {
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
