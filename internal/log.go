package internal

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
)

type LogPosition struct {
	BatchPath string `json:"batch_path"`
	Offset    int    `json:"offset"`
}

type LogPositions map[string][]LogPosition // "batch_path" -> []LogPosition

type LogEntry struct {
	Timestamp int64             `json:"timestamp"`
	KV        map[string]string `json:"kv"`
	Line      string            `json:"line"`
	position  LogPosition

	// mu sync.Mutex
}

func NewLogEntry(kv map[string]string, message string) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now().UnixMicro(),
		KV:        kv,
		Line:      message,
	}
}

func (l *LogEntry) Encode() ([]byte, error) {
	var buff bytes.Buffer
	if err := gob.NewEncoder(&buff).Encode(*l); err != nil {
		return nil, fmt.Errorf("gob can not serialize log: %v", err)
	}
	return buff.Bytes(), nil
}

func (l *LogEntry) GetTime() time.Time {
	return unixMicroToTime(l.Timestamp)
}

func (l *LogEntry) Type() string {
	return "log"
}

// func (l *LogEntry) SetPosition(p LogPosition) {
// 	l.mu.Lock()
// 	defer l.mu.Unlock()
// 	l.position = p
// }
