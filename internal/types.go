package internal

type Query struct {
	Text       string            `json:"text"`        // Full-text search
	Filters    map[string]string `json:"filters"`     // Label filters
	TimeRange  TimeWindow        `json:"time_range"`  // [Start, End]
	MaxResults int               `json:"max_results"` // Pagination
}

type TimeWindow struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type SearchResult struct {
	Matches       LogPositions `json:"-"`
	Logs          []LogEntry   `json:"logs"`
	Total         int          `json:"total"`
	SearchTime    int64        `json:"searchTime"`
	RetrievalTime int64        `json:"retrievalTime"`
}

type CleanupReport struct {
	DeletedBatches int `json:"deletedBatches"`
	DeletedLogs    int `json:"deletedLogs"`
	FreedSpaceMB   int `json:"freedSpaceMB"`
}

type RetentionStats struct {
	DetectedLogs int64
}

type SystemStats struct {
	IngestedLogs  int64   `json:"ingestedLogs"`
	IndexSize     uint64  `json:"indexSize"`
	BufferSize    int     `json:"bufferSize"`
	UptimeMS      int64   `json:"uptime"`
	ActiveBatches int     `json:"activeBatches"`
	StorageUsedMB float64 `json:"storageUsedMB"`
}
