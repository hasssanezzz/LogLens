package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type DiskStorageManager struct {
	homepath    string
	openBatches map[string]*os.File
}

func NewDiskStorageManager(homepath string) StorageManager {
	return &DiskStorageManager{
		homepath: homepath,
	}
}

func (m *DiskStorageManager) GenerateBatchPath(ctx context.Context, start time.Time, end time.Time) string {
	return filepath.Join(
		m.homepath,
		fmt.Sprintf("%04d", start.Year()),
		fmt.Sprintf("%02d", int(start.Month())),
		fmt.Sprintf("%02d", start.Day()),
		fmt.Sprintf("%d-%d.lens", start.UnixMicro(), end.UnixMicro()),
	)
}

func (m *DiskStorageManager) WriteBatch(ctx context.Context, path string, data []byte) (string, error) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		log.Println("can not create data directory")
		return "", err
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Println("can not create batch file")
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
		m.openBatches[path] = reader
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("storage manager failed to read batch %q: %v", path, err)
	}

	return data, nil
}

func (m *DiskStorageManager) ReadFirstNBytesFromBatch(ctx context.Context, batchPath string, n uint32) ([]byte, error) {
	batch, ok := m.openBatches[batchPath]
	if !ok {
		var err error
		batch, err = os.Open(batchPath)
		if err != nil {
			return nil, err
		}
	}

	buff := make([]byte, n)
	readn, err := batch.Read(buff)
	if err != nil {
		return nil, err
	}

	if readn != int(n) {
		return nil, fmt.Errorf("failed to read %d only %d can be read", n, readn)
	}

	return buff, nil
}

func (m *DiskStorageManager) DeleteBatch(ctx context.Context, batchPath string) (int64, error) {
	file, ok := m.openBatches[batchPath]
	if !ok {
		var err error
		file, err = os.Open(batchPath)
		if err != nil {
			return 0, err
		}
		defer file.Close()
	}

	stats, err := file.Stat()
	if err != nil {
		return 0, err
	}

	delete(m.openBatches, batchPath)

	return stats.Size(), os.Remove(batchPath)
}

func (m *DiskStorageManager) Close(ctx context.Context) error {
	for batchPath, reader := range m.openBatches {
		if err := reader.Close(); err != nil {
			return fmt.Errorf("storage manager failed to close batch %q: %v", batchPath, err)
		}
	}
	return nil
}
