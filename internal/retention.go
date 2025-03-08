package internal

import (
	"context"
	"math"
	"time"
)

type RetentionManagerImpl struct {
	storage StorageManager
	index   IndexManager
	stats   RetentionStats
}

func NewDiskRetentionManager(storage StorageManager, index IndexManager) RetentionManager {
	return &RetentionManagerImpl{
		storage: storage,
		index:   index,
	}
}

func (m *RetentionManagerImpl) Run(ctx context.Context) {
	// TODO: create a cron job to call scan every 24 hours
}

func (m *RetentionManagerImpl) Scan(ctx context.Context) (CleanupReport, error) {
	results, err := m.index.Search(ctx, Query{
		TimeRange: TimeWindow{
			Start: 0,
			End:   time.Now().UnixMicro() - int64((RetentionThresholdInDays+1)*24*60*60*1000000),
		},
		MaxResults: 1e18,
	})
	if err != nil {
		return CleanupReport{}, err
	}

	freedBytes := 0.0
	for batchPath := range results.Matches {
		fileSize, err := m.storage.DeleteBatch(ctx, batchPath)
		if err != nil {
			return CleanupReport{}, err
		}

		println("deleting:", batchPath)

		freedBytes += float64(fileSize)
	}

	return CleanupReport{
		DeletedBatches: len(results.Matches),
		DeletedLogs:    len(results.Matches) * BufferThreshold,
		FreedSpaceMB:   float32(math.Round(float64(freedBytes/(1024*1024))*1000) / 1000),
	}, nil
}

func (m *RetentionManagerImpl) Stats(ctx context.Context) RetentionStats {
	return m.stats
}
