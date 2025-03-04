package internal

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	glog "log"
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

func (m *BatchManagerImpl) CreateBatch(ctx context.Context, logs []*LogEntry) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(logs) == 0 {
		return "", fmt.Errorf("cannot create block with zero logs")
	}

	var (
		positions  = make([]LogPosition, len(logs))
		buffer     bytes.Buffer
		currOffset int
	)

	path := m.storage.GenerateBatchPath(ctx, logs[0].GetTime(), logs[len(logs)-1].GetTime())

	for i, log := range logs {
		serializedLog, err := log.Encode()
		if err != nil {
			glog.Println("failed to serialize log:", err)
			continue
		}

		positions[i] = LogPosition{
			BatchPath: path,
			Offset:    currOffset,
		}
		logs[i].position = positions[i]

		// write log size
		buffer.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(serializedLog))))
		// write log bytes
		buffer.Write(serializedLog)

		currOffset += 1
	}

	compressedData := m.encoder.EncodeAll(buffer.Bytes(), nil)
	_, err := m.storage.WriteBatch(ctx, path, compressedData)
	if err != nil {
		return "", fmt.Errorf("can not write batch data to disk: %v", err)
	}

	return path, nil
}

func (m *BatchManagerImpl) RetrieveLogs(ctx context.Context, positions LogPositions) ([]LogEntry, error) {
	allLogs := []LogEntry{}

	for batchPath, positionsItr := range positions {
		logs, err := m.ReadBatchEntries(ctx, batchPath, positionsItr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse logs from batch %q: %v", batchPath, err)
		}
		allLogs = append(allLogs, logs...)
	}

	return allLogs, nil
}

func (m *BatchManagerImpl) ReadBatchEntries(ctx context.Context, batchPath string, positions []LogPosition) ([]LogEntry, error) {
	allLogs := []LogEntry{}

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

	reader := bytes.NewReader(batchBytes)
	for {
		logSize := make([]byte, 4)
		if _, err := reader.Read(logSize); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		logSizeAsInt := binary.LittleEndian.Uint32(logSize)

		logBuff := make([]byte, logSizeAsInt)
		if _, err := reader.Read(logBuff); err != nil {
			return nil, err
		}

		var entry LogEntry
		if err := gob.NewDecoder(bytes.NewReader(logBuff)).Decode(&entry); err != nil {
			panic(err)
		}

		allLogs = append(allLogs, entry)
	}

	// return all logs if no positions is passed
	if len(positions) == 0 {
		return allLogs, nil
	}

	logs := make([]LogEntry, len(positions))
	for i, position := range positions {
		logs[i] = allLogs[position.Offset]
	}

	return logs, nil
}

func (m *BatchManagerImpl) DeleteBatch(ctx context.Context, batchPath string) error {
	// TODO: delete log entries from the index
	return m.storage.DeleteBatch(ctx, batchPath)
}
