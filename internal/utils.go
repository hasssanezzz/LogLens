package internal

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func positionToId(log *LogPosition) string {
	return fmt.Sprintf("%s|%d|%d", log.BatchPath, log.Offset, log.Size)
}

func idToPosition(id string) LogPosition {
	elements := strings.Split(id, "|")
	offset, _ := strconv.Atoi(elements[1])
	size, _ := strconv.Atoi(elements[2])
	return LogPosition{
		BatchPath: elements[0],
		Offset:    offset,
		Size:      size,
	}
}

func unixMicroToTime(timestamp int64) time.Time {
	return time.Unix(timestamp/1000000, (timestamp%1000000)*1000)
}

func isSameCalendarDay(a, b int64) bool {
	aToTime, bToTime := unixMicroToTime(a), unixMicroToTime(b)
	return aToTime.Year() == bToTime.Year() && aToTime.Month() == bToTime.Month() && aToTime.Day() == bToTime.Day()
}

func logsToIds(logs []LogEntry) []string {
	arr := make([]string, len(logs))
	for i, log := range logs {
		arr[i] = positionToId(&log.position)
	}
	return arr
}
