package internal

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"
)

const (
	BufferThreshold          = 2000
	BufferChanSize           = BufferThreshold * 3
	TemporalIndexBatchSize   = BufferThreshold / 3
	RetentionThresholdInDays = 100
)

type LogLensImpl struct {
	// subcomponents
	batchManager     BatchManager
	buffer           LogBuffer
	wal              WAL
	retentionManager RetentionManager
	indexManager     IndexManager

	// attributes
	startDate int64
	entryChan chan LogEntry
}

// TODO: run a cron job to flush the buffer as new days passes

func NewLogLens(homepath string) (LogLens, error) {
	ctx := context.Background()

	wal, err := NewDiskWAL(filepath.Join(homepath, "wal"))
	if err != nil {
		return nil, err
	}

	indexManager, err := NewBleveIndexManager(filepath.Join(homepath, "indexes"))
	if err != nil {
		return nil, err
	}

	storage := NewDiskStorageManager(homepath)
	batchManager, err := NewBatchManager(storage)
	if err != nil {
		return nil, err
	}

	buffer, err := NewLogBuffer()
	if err != nil {
		return nil, err
	}

	retentionManager := NewDiskRetentionManager(storage, indexManager)

	walEntries, err := wal.Read(ctx, 1e5)
	if err != nil {
		return nil, err
	}

	for _, entry := range walEntries {
		buffer.Add(ctx, &entry)
	}
	buffer.Index(ctx)

	lens := &LogLensImpl{
		wal:              wal,
		buffer:           buffer,
		indexManager:     indexManager,
		batchManager:     batchManager,
		startDate:        time.Now().UnixMilli(),
		entryChan:        make(chan LogEntry, BufferChanSize),
		retentionManager: retentionManager,
	}

	log.Println("[LOGLENS INITIATED]")
	stats, _ := lens.Stats(ctx)
	log.Println(stats)
	log.Println("current buffer size:", stats.BufferSize)
	log.Println("current index size: ", stats.IndexSize)
	log.Println("all logs count:     ", stats.IngestedLogs)
	log.Println("=====================")

	go lens.consumeLogsFromChan()
	go retentionManager.Run(ctx)

	return lens, nil
}

func (lens *LogLensImpl) Ingest(ctx context.Context, entry LogEntry) {
	start := time.Now()
	defer func() {
		s := time.Since(start).Milliseconds()
		if s >= 1000 && lens.buffer.Size(ctx)%100 == 0 {
			log.Printf("[LogLens.Ingest] %dms\n", s)
		}
	}()

	lens.entryChan <- entry
}

func (lens *LogLensImpl) IngestBatch(ctx context.Context, logs []*LogEntry) error {
	_, err := lens.batchManager.CreateBatch(ctx, logs)
	if err != nil {
		return fmt.Errorf("failed to create batch: %v", err)
	}

	lens.indexManager.Consume(ctx, logs)

	return nil
}

// TODO: refacor: remove the whole idea of the MappedLogPositions things
// by doing this exclusively in the BatchManager.RetrieveLogs
// to reduce overhead :)

func (lens *LogLensImpl) Search(ctx context.Context, query Query) (*SearchResult, error) {
	memResults, err := lens.buffer.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	query.MaxResults -= len(memResults.Logs)
	if query.MaxResults <= 0 && len(memResults.Logs) > 0 {
		memResults.FreqMap = DateFrequencryMap{time.Now().String()[:10]: len(memResults.Logs)}
		return memResults, nil
	}

	retrievalTimeStart := time.Now()
	diskResults, err := lens.indexManager.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	logs, err := lens.batchManager.RetrieveLogs(ctx, diskResults.Matches)
	if err != nil {
		return nil, fmt.Errorf("loglens failed to retrieve logs from the batch manager: %v", err)
	}

	// aggregate the results
	diskResults.Total += memResults.Total
	diskResults.SearchTime += memResults.SearchTime
	diskResults.RetrievalTime += memResults.RetrievalTime + time.Since(retrievalTimeStart).Milliseconds()
	diskResults.Logs = append(logs, memResults.Logs...)

	diskResults.FreqMap = createFrequencyMap(diskResults.Matches)
	if len(memResults.Logs) > 0 {
		diskResults.FreqMap[time.Now().String()[:10]] += len(memResults.Logs)
	}

	return diskResults, nil
}

func (lens *LogLensImpl) RangeCountSearch(ctx context.Context, start, end int64) (*RangeCountResult, error) {
	query := Query{
		TimeRange: TimeWindow{
			Start: start,
			End:   end,
		},
		MaxResults: 1e18,
	}

	indexResults, err := lens.indexManager.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	results := &RangeCountResult{
		FreqMap:    DateFrequencryMap{},
		SearchTime: indexResults.SearchTime,
	}

	for path, positions := range indexResults.Matches {
		results.FreqMap[getDateFromBatchPath(path)] += len(positions)
		results.Total += len(positions)
	}

	// add buffer size if today is in range
	now := time.Now()
	nowf := now.String()[:10]
	startf, endf := unixMicroToTime(start).String()[:10], unixMicroToTime(end).String()[:10]
	if (now.UnixMicro() >= start && now.UnixMicro() <= end) || startf == nowf || endf == nowf {
		results.FreqMap[now.String()[:10]] += lens.buffer.Size(ctx)
	}
	return results, nil
}

// Maintenance
func (lens *LogLensImpl) Cleanup(ctx context.Context, retentionDays int) (CleanupReport, error) {
	return CleanupReport{}, nil
}

func (lens *LogLensImpl) Stats(ctx context.Context) (SystemStats, error) {
	bufferSize := lens.buffer.Size(ctx)
	indexSize := lens.indexManager.Size(ctx)
	return SystemStats{
		IngestedLogs:  int64(bufferSize) + int64(indexSize),
		IndexSize:     indexSize,
		BufferSize:    bufferSize,
		UptimeMS:      time.Now().UnixMilli() - lens.startDate,
		ActiveBatches: 0,
		StorageUsedMB: 0,
	}, nil
}

// Lifecycle
func (lens *LogLensImpl) Shutdown(ctx context.Context) error {
	return nil
}

func (lens *LogLensImpl) consumeLogsFromChan() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			l := len(lens.entryChan)
			if l > 0 {
				log.Println("buffen channel size:", l)
			}
		}
	}()

	for entry := range lens.entryChan {
		ctx := context.Background()

		if lens.buffer.Size(ctx) >= BufferThreshold ||
			(!isSameCalendarDay(lens.buffer.LatestEntryTimestamp(ctx), entry.Timestamp) &&
				lens.buffer.LatestEntryTimestamp(ctx) != 0 &&
				lens.buffer.Size(ctx) > 0) {
			println("flushing the buffer with length:", lens.buffer.Size(ctx))
			if err := lens.flushBuffer(); err != nil {
				panic(err)
			}
		}

		// add to the buffer
		lens.buffer.Add(ctx, &entry)
		if err := lens.wal.Append(ctx, entry); err != nil {
			err = fmt.Errorf("failed to append entry to WAL: %v", err)
			log.Println(err)
		}
	}
}

func (lens *LogLensImpl) flushBuffer() error {
	ctx := context.Background()

	// panic if buffer is empty
	if lens.buffer.Size(ctx) <= 0 {
		panic("something went wrong here")
	}

	log.Println("triggering a buffer flush, SIZE:", lens.buffer.Size(ctx))

	// TODO: make these operations atomic

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
