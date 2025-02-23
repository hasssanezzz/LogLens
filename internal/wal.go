package internal

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"sync"
)

type DiskWAL struct {
	path  string
	count int
	file  *os.File

	mu sync.Mutex
}

func NewDiskWAL(path string) (WAL, error) {
	wfile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &DiskWAL{
		path: path,
		file: wfile,
	}, nil
}

func (w *DiskWAL) Append(ctx context.Context, log LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	bytes, err := log.Encode()
	if err != nil {
		return fmt.Errorf("disk WAL failed to append serialize log to append it: %v", err)
	}

	logSizeBuffer := make([]byte, 4)
	binary.LittleEndian.PutUint32(logSizeBuffer, uint32(len(bytes)))

	if _, err := w.file.Write(append(logSizeBuffer, bytes...)); err != nil {
		return fmt.Errorf("disk WAL failed to append serialized log: %v", err)
	}

	// TODO: add this to goldb
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync disk WAL: %v", err)
	}
	w.count += 1
	return nil
}

func (w *DiskWAL) Read(ctx context.Context, maxEntries int) ([]LogEntry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	rfile, err := os.Open(w.path)
	if err != nil {
		return nil, fmt.Errorf("disk WAL failed to open WAL file for reaeding: %v", err)
	}
	defer rfile.Close()

	var logs []LogEntry

	for {
		logSizeBuffer := make([]byte, 4)
		if _, err := io.ReadFull(rfile, logSizeBuffer); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read log size: %v", err)
		}

		logSize := binary.LittleEndian.Uint32(logSizeBuffer)
		logBytes := make([]byte, logSize)
		if _, err := io.ReadFull(rfile, logBytes); err != nil {
			return nil, fmt.Errorf("failed to read log entry: %v", err)
		}

		var log LogEntry
		if err := gob.NewDecoder(bytes.NewReader(logBytes)).Decode(&log); err != nil {
			return nil, fmt.Errorf("failed to decode log: %v", err)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

func (w *DiskWAL) Clear(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.Truncate(w.path, 0); err != nil {
		w.count = 0
		return err
	}
	return nil
}

func (w *DiskWAL) Close(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
