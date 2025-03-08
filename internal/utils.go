package internal

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
)

const BatchHeaderSize = 9

func parseBatchMetadata(data []byte) (byte, uint32, error) {
	if string(data[:4]) != "LENS" {
		return 0, 0, fmt.Errorf("reading a batch with invalid magin code %q", string(data[:4]))
	}
	return data[4], binary.LittleEndian.Uint32(data[5:9]), nil
}

func positionToId(log *LogPosition) string {
	return fmt.Sprintf("%s|%d", log.BatchPath, log.Offset)
}

func idToPosition(id string) LogPosition {
	elements := strings.Split(id, "|")
	offset, _ := strconv.Atoi(elements[1])
	return LogPosition{
		BatchPath: elements[0],
		Offset:    offset,
	}
}

func unixMicroToTime(timestamp int64) time.Time {
	return time.Unix(timestamp/1000000, (timestamp%1000000)*1000)
}

func isSameCalendarDay(a, b int64) bool {
	aToTime, bToTime := unixMicroToTime(a), unixMicroToTime(b)
	return aToTime.Year() == bToTime.Year() && aToTime.Month() == bToTime.Month() && aToTime.Day() == bToTime.Day()
}

func logsToIds(logs []*LogEntry) []string {
	arr := make([]string, len(logs))
	for i, log := range logs {
		arr[i] = positionToId(&log.position)
	}
	return arr
}

func createSearchRequest(q *Query) *bleve.SearchRequest {
	searchQuery := bleve.NewConjunctionQuery()

	if q.Text != "" {
		textQuery := bleve.NewMatchQuery(q.Text)
		textQuery.SetField("line")
		searchQuery.AddQuery(textQuery)
	}

	startAsFloat := float64(q.TimeRange.Start)
	endAsFloat := float64(q.TimeRange.End)

	if q.TimeRange.Start != 0 || q.TimeRange.End != 0 {
		timeQuery := bleve.NewNumericRangeQuery(&startAsFloat, &endAsFloat)
		timeQuery.SetField("timestamp")
		searchQuery.AddQuery(timeQuery)
	}

	for key, value := range q.Filters {
		termQuery := bleve.NewTermQuery(value)
		termQuery.SetField(fmt.Sprintf("kv.%s", key))
		searchQuery.AddQuery(termQuery)
	}

	searchRequest := bleve.NewSearchRequest(searchQuery)
	searchRequest.Size = int(q.MaxResults)
	searchRequest.SortBy([]string{"timestamp"})

	return searchRequest
}
