package internal

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sort"
	"sync"

	z "github.com/klauspost/compress/zstd"
)

type BatchManagerImpl struct {
	storage StorageManager
	encoder *z.Encoder
	decoder *z.Decoder
	mu      sync.RWMutex
}

func NewBatchManager(storageManger StorageManager) (BatchManager, error) {
	encoder, err := z.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("can not create encoder in batch manager: %v", err)
	}

	decoder, err := z.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("can not create reader in batch manager: %v", err)
	}

	return &BatchManagerImpl{
		storage: storageManger,
		encoder: encoder,
		decoder: decoder,
		mu:      sync.RWMutex{},
	}, nil
}

func (m *BatchManagerImpl) CreateBatch(ctx context.Context, logs []LogEntry) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(logs) == 0 {
		return "", fmt.Errorf("cannot create block with zero logs")
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp > logs[j].Timestamp
	})

	var (
		positions  = make([]LogPosition, len(logs))
		buffer     bytes.Buffer
		currOffset int
	)

	path := m.storage.GenerateBatchPath(ctx, logs[0].GetTime(), logs[len(logs)-1].GetTime())

	for i, log := range logs {
		serializedLog, _ := log.Encode()
		positions[i] = LogPosition{
			BatchPath: path,
			Offset:    currOffset,
			Size:      len(serializedLog),
		}
		buffer.Write(serializedLog)
		currOffset += len(serializedLog)
	}

	compressedData := m.encoder.EncodeAll(buffer.Bytes(), nil)
	_, err := m.storage.WriteBatch(ctx, path, compressedData)
	if err != nil {
		return "", fmt.Errorf("can not write batch data to disk: %v", err)
	}

	return path, nil
}

func (m *BatchManagerImpl) ParseBatch(ctx context.Context, batchPath string, positions []LogPosition) ([]LogEntry, error) {
	logs := []LogEntry{}

	// read compressed bytes
	compressedBatchBytes, err := m.storage.ReadBatch(ctx, batchPath)
	if err != nil {
		return nil, fmt.Errorf("batch manager failed to read batch bytes: %v", err)
	}

	// decompress bytes
	batchBytes, err := m.decoder.DecodeAll(compressedBatchBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("batch manager failed to decompress batch bytes: %v", err)
	}

	// deserialize the wanted logs using the log positions
	for _, position := range positions {
		var log LogEntry
		serializedLog := batchBytes[position.Offset : position.Offset+position.Size]
		if err := gob.NewDecoder(bytes.NewReader(serializedLog)).Decode(&log); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func (m *BatchManagerImpl) RetrieveLogs(ctx context.Context, positions LogPositions) ([]LogEntry, error) {
	allLogs := []LogEntry{}

	for batchPath, pos := range positions {
		logs, err := m.ParseBatch(ctx, batchPath, pos)
		if err != nil {
			return nil, fmt.Errorf("failed to parse logs from batch %q: %v", batchPath, err)
		}
		allLogs = append(allLogs, logs...)
	}

	return allLogs, nil
}

func (m *BatchManagerImpl) DeleteBatch(ctx context.Context, batchPath string) error {
	// TODO: delete log entries from the index
	return m.storage.DeleteBatch(ctx, batchPath)
}
