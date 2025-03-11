package internal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type MemoryLogBuffer struct {
	logs                 []*LogEntry
	latestEntryTimestamp int64
	index                InMemoryIndexManager

	mu sync.RWMutex
}

func NewLogBuffer() (LogBuffer, error) {
	index, err := NewMemoryIndexManager()
	if err != nil {
		return nil, err
	}

	return &MemoryLogBuffer{
		logs:  []*LogEntry{},
		index: index,
	}, nil
}

func (lb *MemoryLogBuffer) Add(ctx context.Context, entry *LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// track the latest entry timestamp
	if entry.Timestamp > lb.latestEntryTimestamp {
		lb.latestEntryTimestamp = entry.Timestamp
	}

	// add the position
	entry.position.Offset = len(lb.logs)

	// append to the buffer
	lb.logs = append(lb.logs, entry)

	// index the log
	currLen := len(lb.logs)
	if currLen >= TemporalIndexBatchSize && currLen%TemporalIndexBatchSize == 0 {
		start := time.Now()
		// index only the last #TemporalIndexBatchSize elements
		logsToIndex := lb.logs[currLen-TemporalIndexBatchSize:]
		err := lb.index.IndexBatch(ctx, logsToIndex)
		if err != nil {
			panic(err) // FOR NOW
		}
		log.Printf("[periodic buffer index flush] %d in %dms", currLen, time.Since(start).Milliseconds())
	}
}

func (lb *MemoryLogBuffer) Index(ctx context.Context) {
	currLen := len(lb.logs)
	logsToIndex := lb.logs[min(0, currLen-TemporalIndexBatchSize):]
	err := lb.index.IndexBatch(ctx, logsToIndex)
	if err != nil {
		panic(err) // FOR NOW
	}
}

func (lb *MemoryLogBuffer) AddBuffer(ctx context.Context, batch []*LogEntry) {
	panic("unimplemented")
}

func (lb *MemoryLogBuffer) Search(ctx context.Context, query Query) (*SearchResult, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	results, err := lb.index.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	retrievalTimeStart := time.Now()
	logsFromBuffer := []LogEntry{}
	for _, match := range results.Matches {
		for _, position := range match {
			entry, err := lb.getByIndex(ctx, position.Offset)
			if err != nil {
				panic(err)
			}
			logsFromBuffer = append(logsFromBuffer, *entry)
		}
	}
	results.Logs = logsFromBuffer
	results.RetrievalTime += time.Since(retrievalTimeStart).Milliseconds()

	return results, nil
}

func (lb *MemoryLogBuffer) Flush(ctx context.Context) ([]*LogEntry, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	logs := make([]*LogEntry, len(lb.logs))
	copy(logs, lb.logs)

	// clear the buffer
	lb.logs = []*LogEntry{}
	lb.latestEntryTimestamp = 0

	// clear the index
	if err := lb.index.Clear(ctx); err != nil {
		return nil, err
	}

	return logs, nil
}

func (lb *MemoryLogBuffer) LatestEntryTimestamp(ctx context.Context) int64 {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return lb.latestEntryTimestamp
}

func (lb *MemoryLogBuffer) Size(ctx context.Context) int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return len(lb.logs)
}

func (lb *MemoryLogBuffer) getByIndex(ctx context.Context, index int) (*LogEntry, error) {
	if index >= len(lb.logs) {
		return nil, fmt.Errorf("memory buffer trying to access index out of range (len: %d i: %d)", lb.Size(ctx), index)
	}

	return lb.logs[index], nil
}
