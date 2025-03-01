package internal

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
)

const (
	BufferThreshold          = 2000
	RetentionThresholdInDays = 100
)

type LogLensImpl struct {
	batchManager     BatchManager
	buffer           LogBuffer
	wal              WAL
	retentionManager RetentionManager
	indexManager     IndexManager
}

func NewLogLens(homepath string) (LogLens, error) {
	wal, err := NewDiskWAL(filepath.Join(homepath, "wal"))
	if err != nil {
		return nil, err
	}

	indexManager, err := NewBleveIndexManager(filepath.Join(homepath, "indexes"))
	if err != nil {
		return nil, err
	}

	batchManager, err := NewBatchManager(NewDiskStorageManager(homepath))
	if err != nil {
		return nil, err
	}

	walEntries, err := wal.Read(context.Background(), 1e5)
	if err != nil {
		return nil, err
	}

	buffer, err := NewLogBuffer()
	if err != nil {
		return nil, err
	}

	for _, entry := range walEntries {
		buffer.Add(context.Background(), &entry)
	}

	ctx := context.Background()
	bufSize, indexSize := buffer.Size(ctx), indexManager.Size(ctx)

	log.Println("log lens initiated, current buffer size:", bufSize)
	log.Println("                    current index size: ", indexSize)
	log.Println("                    all logs count:     ", uint64(bufSize)+indexSize)

	lens := &LogLensImpl{
		wal:          wal,
		buffer:       buffer,
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
	logs, err := lens.buffer.Flush(ctx)
	if err != nil {
		return err
	}

	if err := lens.IngestBatch(ctx, logs); err != nil {
		return err
	}

	if err := lens.wal.Clear(ctx); err != nil {
		log.Println("failed to clear the WAL:", err)
	}

	return nil
}

func (lens *LogLensImpl) Ingest(ctx context.Context, entry LogEntry) error {
	// start := time.Now()
	// defer func() {
	// 	s := time.Since(start).Milliseconds()
	// 	if s >= 1000 {
	// 		log.Printf("[TIME] LogLens.Ingest took: %d\n", s)
	// 	}
	// }()

	if lens.buffer.Size(ctx) >= BufferThreshold ||
		!isSameCalendarDay(lens.buffer.LatestEntryTimestamp(ctx), entry.Timestamp) {
		if err := lens.FlushBuffer(ctx); err != nil {
			return err
		}
	}

	// add to the buffer
	lens.buffer.Add(ctx, &entry)
	if err := lens.wal.Append(ctx, entry); err != nil {
		return fmt.Errorf("failed to append entry to WAL: %v", err)
	}

	return nil
}

func (lens *LogLensImpl) IngestBatch(ctx context.Context, logs []*LogEntry) error {
	// start := time.Now()
	// defer func() {
	// 	log.Printf("[TIME] LogLens.IngestBatch took: %d\n", time.Since(start).Milliseconds())
	// }()

	// step 2: create a batch
	_, err := lens.batchManager.CreateBatch(ctx, logs)
	if err != nil {
		return fmt.Errorf("failed to create batch: %v", err)
	}

	// step 3: index the new logs
	lens.indexManager.Consume(ctx, logs)

	return nil
}

func (lens *LogLensImpl) Search(ctx context.Context, query Query) (*SearchResult, error) {
	memResults, err := lens.buffer.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	results, err := lens.indexManager.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	for _, match := range results.Matches {
		for _, position := range match {
			if position.BatchPath == "" {
				panic("something very bad just happened")
			}
		}
	}

	logs, err := lens.batchManager.RetrieveLogs(ctx, results.Matches)
	if err != nil {
		return nil, fmt.Errorf("loglens failed to retrieve logs from the batch manager: %v", err)
	}

	results.Total += memResults.Total
	results.SearchTime += memResults.SearchTime
	results.RetrievalTime += memResults.RetrievalTime
	results.Logs = append(logs, memResults.Logs...)
	return results, nil
}

// Maintenance
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
