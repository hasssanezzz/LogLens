package internal

import "context"

type RetentionManagerImpl struct {
	storage StorageManager
	stats   RetentionStats
}

func NewDiskRetentionManager(storage StorageManager) RetentionManager {
	return &RetentionManagerImpl{
		storage: storage,
	}
}

func (rm *RetentionManagerImpl) Run(ctx context.Context, retentionDays int) (CleanupReport, error) {
	return CleanupReport{}, nil
}

func (rm *RetentionManagerImpl) Stats(ctx context.Context) RetentionStats {
	return rm.stats
}
