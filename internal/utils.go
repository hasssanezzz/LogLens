package internal

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
)

const BatchHeaderSize = 9

func parseBatchMetadata(data []byte) (byte, uint32, error) {
	if string(data[:4]) != "LENS" {
		return 0, 0, fmt.Errorf("magic number mismatch %q", string(data[:4]))
	}
	return data[4], binary.LittleEndian.Uint32(data[5:9]), nil
}

func getDateFromBatchPath(path string) string {
	base := filepath.Dir(filepath.ToSlash(path))
	return strings.ReplaceAll(base[len(base)-10:], "/", "-")
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

func createSearchRequest(q *Query, ignoreMaxResult ...bool) *bleve.SearchRequest {
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
	searchRequest.SortBy([]string{"-timestamp"})
	if len(ignoreMaxResult) <= 0 {
		searchRequest.Size = q.MaxResults
	} else {
		searchRequest.Size = 1e18
	}

	return searchRequest
}

func createFrequencyMap(matches MappedLogPositions) DateFrequencryMap {
	freqMap := DateFrequencryMap{}
	for path, positions := range matches {
		freqMap[getDateFromBatchPath(path)] += len(positions)
	}
	return freqMap
}

func mapLogPositions(p []LogPosition) MappedLogPositions {
	m := MappedLogPositions{}
	for _, i := range p {
		m[i.BatchPath] = append(m[i.BatchPath], i)
	}
	return m
}
