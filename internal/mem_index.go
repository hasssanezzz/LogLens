package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
)

type MemoryIndexManagerImpl struct {
	index bleve.Index
	mu    sync.Mutex
}

func NewMemoryIndexManager() (InMemoryIndexManager, error) {
	m := &MemoryIndexManagerImpl{}
	if err := m.newInMemIndex(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *MemoryIndexManagerImpl) newInMemIndex() error {
	indexMapping := createIndexMapping()
	index, err := bleve.NewMemOnly(indexMapping)
	if err != nil {
		return fmt.Errorf("failed to create in-memory Bleve index: %v", err)
	}

	m.index = index
	return nil
}

func (m *MemoryIndexManagerImpl) Index(ctx context.Context, entry *LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := positionToId(&entry.position)
	if err := m.index.Index(id, entry); err != nil {
		return fmt.Errorf("failed to index log %s: %v", id, err)
	}
	return nil
}

func (m *MemoryIndexManagerImpl) IndexBatch(ctx context.Context, batch []*LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b := m.index.NewBatch()
	for _, entry := range batch {
		if err := b.Index(positionToId(&entry.position), entry); err != nil {
			return fmt.Errorf("failed to index log: %v", err)
		}
	}

	if err := m.index.Batch(b); err != nil {
		return fmt.Errorf("failed to index batch: %v", err)
	}

	return nil
}

func (m *MemoryIndexManagerImpl) Clear(ctx context.Context) error {
	return m.newInMemIndex()
}

func (m *MemoryIndexManagerImpl) Search(ctx context.Context, q Query) (*SearchResult, error) {
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

func (m *MemoryIndexManagerImpl) Size(ctx context.Context) uint64 {
	size, err := m.index.DocCount()
	if err != nil {
		panic(err)
	}

	return size
}

func (m *MemoryIndexManagerImpl) Close(ctx context.Context) error {
	return m.index.Close()
}
