package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type DiskStorageManager struct {
	homepath    string
	openBatches map[string]io.ReadCloser
}

func NewDiskStorageManager(homepath string) StorageManager {
	return &DiskStorageManager{
		homepath: homepath,
	}
}

func (m *DiskStorageManager) GenerateBatchPath(ctx context.Context, start time.Time, end time.Time) string {
	return filepath.Join(
		m.homepath,
		fmt.Sprintf("%d", start.Year()),
		fmt.Sprintf("%d", int(start.Month())),
		fmt.Sprintf("%d", start.Day()),
		fmt.Sprintf("%02d%02d-%02d%02d.zstd", start.Hour(), start.Minute(), end.Hour(), end.Minute()),
	)
}

func (m *DiskStorageManager) WriteBatch(ctx context.Context, path string, data []byte) (string, error) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		println("can not create zstd folder path")
		return "", err
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		println("can not create zstd file")
		return "", err
	}

	return path, nil
}

func (m *DiskStorageManager) ReadBatch(ctx context.Context, path string) ([]byte, error) {
	reader, ok := m.openBatches[path]
	if !ok {
		var err error
		reader, err = os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("storage manager failed to open batch %q: %v", path, err)
		}
		defer reader.Close()
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("storage manager failed to read batch %q: %v", path, err)
	}

	return data, nil
}

func (m *DiskStorageManager) DeleteBatch(ctx context.Context, batchPath string) error {
	return os.Remove(batchPath)
}

func (m *DiskStorageManager) Close(ctx context.Context) error {
	for batchPath, reader := range m.openBatches {
		if err := reader.Close(); err != nil {
			return fmt.Errorf("storage manager failed to close batch %q: %v", batchPath, err)
		}
	}
	return nil
}
