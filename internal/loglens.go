package internal

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
)

const (
	BufferThreshold = 200
)

type LogLensImpl struct {
	batchManager     BatchManager
	buffer           LogBuffer
	wal              WAL
	retentionManager RetentionManager
	indexManager     IndexManager
}

func NewLogLens(homepath string) (LogLens, error) {
	wal, err := NewDiskWAL(filepath.Join("wal"))
	if err != nil {
		return nil, err
	}

	indexManager, err := NewBleveIndexManager(filepath.Join(homepath, "indexes"))
	if err != nil {
		return nil, err
	}

	storageManager := NewDiskStorageManager(homepath)
	batchManager, err := NewBatchManager(storageManager)
	if err != nil {
		return nil, err
	}

	lens := &LogLensImpl{
		wal:          wal,
		buffer:       NewLogBuffer(),
		indexManager: indexManager,
		batchManager: batchManager,
	}

	return lens, nil
}

func (lens *LogLensImpl) FlushBuffer(ctx context.Context) error {
	// ignore if buffer is empty
	if lens.buffer.Size(ctx) <= 0 {
		return nil
	}
	log.Println("triggering a buffer flush")

	logs := lens.buffer.Flush(ctx)

	// Create error channel to collect errors from goroutines
	errChan := make(chan error, 1)

	// Create a context that can be canceled
	flushCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start the batch ingestion in a separate goroutine
	go func() {
		err := lens.IngestBatch(flushCtx, logs)
		errChan <- err // Send nil or the error to channel
	}()

	// Wait for the result or context cancellation
	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("failed to index batch when flushing: %v", err)
		}
		return nil
	case <-ctx.Done():
		// The parent context was canceled
		return ctx.Err()
	}
}

func (lens *LogLensImpl) Ingest(ctx context.Context, entry LogEntry) error {
	log.Println("ingesting a log with message:", entry.Line)

	if lens.buffer.Size(ctx) >= BufferThreshold ||
		!isSameCalendarDay(lens.buffer.LatestEntryTimestamp(ctx), entry.Timestamp) {
		if err := lens.FlushBuffer(ctx); err != nil {
			return err
		}
	}

	lens.buffer.Add(ctx, &entry)

	// index the log
	err := lens.indexManager.IndexLog(ctx, &entry)
	if err != nil {
		return fmt.Errorf("failed to index a log: %v", err)
	}

	return nil
}

func (lens *LogLensImpl) IngestBatch(ctx context.Context, logs []LogEntry) error {
	var err error

	// Step 1: Delete the past indexes
	if err = lens.indexManager.DeleteFromIndex(ctx, logsToIds(logs)); err != nil {
		return fmt.Errorf("failed to delete indexes when ingesting a batch: %v", err)
	}

	// Defer a function to undo the delete operation if any subsequent step fails
	defer func() {
		if err != nil {
			// Re-index the deleted logs
			if undoErr := lens.indexManager.IndexBatch(ctx, logs); undoErr != nil {
				log.Printf("failed to undo delete operation: %v", undoErr)
			}
		}
	}()

	// Step 2: Create batch
	batchPath, err := lens.batchManager.CreateBatch(ctx, logs)
	if err != nil {
		return fmt.Errorf("failed to create batch: %v", err)
	}

	// Defer a function to delete the batch if any subsequent step fails
	defer func() {
		if err != nil {
			if undoErr := lens.batchManager.DeleteBatch(ctx, batchPath); undoErr != nil {
				log.Printf("failed to undo create batch operation: %v", undoErr)
			}
		}
	}()

	// Step 3: Update the index
	if err = lens.indexManager.IndexBatch(ctx, logs); err != nil {
		return fmt.Errorf("failed to index when ingesting a batch: %v", err)
	}

	return nil
}

func (lens *LogLensImpl) Search(ctx context.Context, query Query) (*SearchResult, error) {
	results, err := lens.indexManager.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	logs, err := lens.batchManager.RetrieveLogs(ctx, results.Matches)
	if err != nil {
		return nil, err
	}

	results.Logs = logs
	return results, nil
}

// Maintenance
func (lens *LogLensImpl) ForceFlush(ctx context.Context) error {
	return nil
}

func (lens *LogLensImpl) Cleanup(ctx context.Context, retentionDays int) (CleanupReport, error) {
	return CleanupReport{}, nil
}

func (lens *LogLensImpl) Stats(ctx context.Context) (SystemStats, error) {
	return SystemStats{}, nil
}

// Lifecycle
func (lens *LogLensImpl) Shutdown(ctx context.Context) error {
	return nil
}
