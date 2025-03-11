package internal

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
)

type LogEntry struct {
	Timestamp int64             `json:"timestamp"`
	KV        map[string]string `json:"kv"`
	Line      string            `json:"line"`
	position  LogPosition
}

func NewLogEntry(kv map[string]string, message string) *LogEntry {
	// var base int64 = 1735682400000000
	return &LogEntry{
		Timestamp: time.Now().UnixMicro(),
		// Timestamp: base + rand.Int63n(5184000000000),
		KV:   kv,
		Line: message,
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
