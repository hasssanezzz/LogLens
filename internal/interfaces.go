package internal

import (
	"context"
	"time"
)

type LogLens interface {
	// Ingestion
	Ingest(ctx context.Context, log LogEntry)
	IngestBatch(ctx context.Context, logs []*LogEntry) error

	// Query
	Search(ctx context.Context, query Query) (*SearchResult, error)

	// Maintenance
	Cleanup(ctx context.Context, retentionDays int) (CleanupReport, error)
	Stats(ctx context.Context) (SystemStats, error)

	// Lifecycle
	Shutdown(ctx context.Context) error
}

type IndexManager interface {
	Listen(context.Context)
	Consume(ctx context.Context, batch []*LogEntry)
	Search(ctx context.Context, query Query) (*SearchResult, error)
	DeleteFromIndex(ctx context.Context, ids []string) error
	Size(ctx context.Context) uint64
	Close(ctx context.Context) error
}

type InMemoryIndexManager interface {
	Index(context.Context, *LogEntry) error
	IndexBatch(context.Context, []*LogEntry) error
	Search(context.Context, Query) (*SearchResult, error)
	Size(context.Context) uint64
	Clear(context.Context) error
	Close(context.Context) error
}

type WAL interface {
	Append(ctx context.Context, log LogEntry) error
	Read(ctx context.Context, maxEntries int) ([]LogEntry, error)
	Clear(ctx context.Context) error
	Close(ctx context.Context) error
}

type StorageManager interface {
	WriteBatch(ctx context.Context, path string, data []byte) (string, error)
	GenerateBatchPath(ctx context.Context, start time.Time, end time.Time) string
	ReadBatch(ctx context.Context, path string) ([]byte, error)
	ReadFirstNBytesFromBatch(ctx context.Context, path string, n uint32) ([]byte, error)
	DeleteBatch(ctx context.Context, batchPath string) (int64, error)
}

type BatchManager interface {
	CreateBatch(ctx context.Context, logs []*LogEntry) (string, error)
	RetrieveLogs(ctx context.Context, positions LogPositions) ([]LogEntry, error)
	ReadBatchEntries(ctx context.Context, batchPath string, positions []LogPosition) ([]LogEntry, error)
	DeleteBatch(ctx context.Context, batchPath string) (int64, error)
}

type LogBuffer interface {
	Add(ctx context.Context, entry *LogEntry)
	AddBuffer(ctx context.Context, batch []*LogEntry)
	Search(ctx context.Context, q Query) (*SearchResult, error)
	Flush(ctx context.Context) ([]*LogEntry, error)
	LatestEntryTimestamp(ctx context.Context) int64
	Size(ctx context.Context) int
}

type RetentionManager interface {
	Run(ctx context.Context)
	Scan(ctx context.Context) (CleanupReport, error)
	Stats(ctx context.Context) RetentionStats
}
