package internal

import (
	"context"
	"sync"
)

type MemoryLogBuffer struct {
	logs                 []LogEntry
	latestEntryTimestamp int64

	mu sync.RWMutex
}

func NewLogBuffer() LogBuffer {
	return &MemoryLogBuffer{
		logs: []LogEntry{},
	}
}

func (lb *MemoryLogBuffer) Add(ctx context.Context, entry *LogEntry) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// track the latest entry timestamp
	if entry.Timestamp > lb.latestEntryTimestamp {
		lb.latestEntryTimestamp = entry.Timestamp
	}

	entry.position.Offset = len(lb.logs) + 1
	entry.position.Size = -1
	lb.logs = append(lb.logs, *entry)
}

func (lb *MemoryLogBuffer) Flush(ctx context.Context) []LogEntry {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	logs := make([]LogEntry, len(lb.logs))
	copy(logs, lb.logs)

	// clear the buffer
	lb.logs = []LogEntry{}

	return logs
}

func (lb *MemoryLogBuffer) LatestEntryTimestamp(ctx context.Context) int64 {
	lb.mu.RLock()
	lb.mu.RUnlock()

	return lb.latestEntryTimestamp
}

func (lb *MemoryLogBuffer) Size(ctx context.Context) int {
	lb.mu.RLock()
	lb.mu.RUnlock()

	return len(lb.logs)
}
