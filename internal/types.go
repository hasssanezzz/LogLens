package internal

type (
	DateFrequencryMap  map[string]int           // "2006-02-01" -> number of logs ingested in this date
	MappedLogPositions map[string][]LogPosition // "batch_path" -> []LogPosition

	LogPosition struct {
		BatchPath string `json:"batch_path"`
		Offset    int    `json:"offset"`
	}

	Query struct {
		Text       string            `json:"text"`
		Filters    map[string]string `json:"filters"`
		TimeRange  TimeWindow        `json:"time_range"`
		Page       int               `json:"page"`
		MaxResults int               `json:"max_results"`
	}

	TimeWindow struct {
		Start int64 `json:"start"`
		End   int64 `json:"end"`
	}

	SearchResult struct {
		Matches       MappedLogPositions `json:"-"`
		Positions     []LogPosition      `json:"-"`
		Logs          []LogEntry         `json:"logs"`
		FreqMap       DateFrequencryMap  `json:"freq"`
		Count         int                `json:"count"`
		Page          int                `json:"page"`
		Total         int                `json:"total"`
		SearchTime    int64              `json:"searchTime"`
		RetrievalTime int64              `json:"retrievalTime"`
	}

	RangeCountResult struct {
		FreqMap    DateFrequencryMap `json:"freq"`
		Total      int               `json:"total"`
		SearchTime int64             `json:"searchTime"`
	}

	CleanupReport struct {
		DeletedBatches int     `json:"deletedBatches"`
		DeletedLogs    int     `json:"deletedLogs"`
		FreedSpaceMB   float32 `json:"freedSpaceMB"`
	}

	RetentionStats struct {
		DetectedLogs int64
	}

	SystemStats struct {
		IngestedLogs  int64   `json:"ingestedLogs"`
		IndexSize     uint64  `json:"indexSize"`
		BufferSize    int     `json:"bufferSize"`
		UptimeMS      int64   `json:"uptime"`
		ActiveBatches int     `json:"activeBatches"`
		StorageUsedMB float64 `json:"storageUsedMB"`
	}
)
