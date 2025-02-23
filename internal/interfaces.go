package internal

import (
	"context"
	"time"
)

type LogLens interface {
	// Ingestion
	Ingest(ctx context.Context, log LogEntry) error
	IngestBatch(ctx context.Context, logs []LogEntry) error

	// Query
	Search(ctx context.Context, query Query) (*SearchResult, error)

	// Maintenance
	ForceFlush(ctx context.Context) error
	Cleanup(ctx context.Context, retentionDays int) (CleanupReport, error)
	Stats(ctx context.Context) (SystemStats, error)

	// Lifecycle
	Shutdown(ctx context.Context) error
}

type IndexManager interface {
	IndexLog(ctx context.Context, log *LogEntry) error
	IndexBatch(ctx context.Context, batch []LogEntry) error
	DeleteFromIndex(ctx context.Context, ids []string) error
	DeleteSingleLogFromIndex(ctx context.Context, id string) error
	Search(ctx context.Context, query Query) (*SearchResult, error)
	Close(ctx context.Context) error
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
	DeleteBatch(ctx context.Context, batchPath string) error
}

type BatchManager interface {
	CreateBatch(ctx context.Context, logs []LogEntry) (string, error)
	ParseBatch(ctx context.Context, batchPath string, positions []LogPosition) ([]LogEntry, error)
	RetrieveLogs(ctx context.Context, positions LogPositions) ([]LogEntry, error)
	DeleteBatch(ctx context.Context, batchPath string) error
}

type LogBuffer interface {
	Add(ctx context.Context, entry *LogEntry)
	Flush(ctx context.Context) []LogEntry
	LatestEntryTimestamp(ctx context.Context) int64
	Size(ctx context.Context) int
}

type RetentionManager interface {
	Run(ctx context.Context, retentionDays int) (CleanupReport, error)
	Stats(ctx context.Context) RetentionStats
}
