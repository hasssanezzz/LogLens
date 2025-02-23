package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/hasssanezzz/try-bleve/cmd/api"
	"github.com/hasssanezzz/try-bleve/internal"
)

func ProcessLogFile(filePath string) ([]internal.LogEntry, error) {
	var logEntries []internal.LogEntry // Slice to store the log entries
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		var logEntry internal.LogEntry
		err := json.Unmarshal([]byte(line), &logEntry)
		if err != nil {
			log.Printf("Error unmarshaling JSON on line %d: %v, Line Content: %s", lineNumber, err, line)
			continue // Skip to the next line if there's an error
		}

		logEntry.Timestamp = int64(logEntry.Timestamp)
		logEntries = append(logEntries, logEntry) // Append the log entry to the slice
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return logEntries, nil // Return the slice of log entries and nil error
}

func generateRandomAttributes() map[string]string {
	attributes := map[string]string{
		"level":    []string{"INFO", "WARN", "ERROR", "DEBUG"}[rand.Intn(4)],
		"service":  []string{"auth", "database", "cache", "api"}[rand.Intn(4)],
		"instance": fmt.Sprintf("instance-%d", rand.Intn(10)),
	}
	return attributes
}

func GenerateLogs(count int) []internal.LogEntry {
	logs := []internal.LogEntry{}
	for i := 0; i < count; i++ {
		logs = append(logs, internal.LogEntry{
			Timestamp: time.Now().UnixNano(),
			KV:        generateRandomAttributes(),
			Line:      fmt.Sprintf("Log message %d: Something happened!", i),
			// position: internal.LogPos{
			// 	Size:   i,
			// 	Offset: i * 10,
			// },
		})
	}
	return logs
}

func IndexNLogs(idx internal.IndexManager, n int) {
	logs := GenerateLogs(n)
	println("generated logs of count:", len(logs))

	bsize := 1000
	for i := 0; i < n; i += bsize {
		println(i)
		err := idx.IndexBatch(context.Background(), logs[i:i+bsize])
		if err != nil {
			println("index error")
			panic(err)
		}
	}
}

func Search(idx internal.IndexManager, q internal.Query) {
	result, err := idx.Search(context.Background(), q)
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(result.Matches); i++ {
		fmt.Println(result.Matches)
	}

	for batchPath, position := range result.Matches {
		fmt.Printf("[%s] -> %+v\n", batchPath, position)
	}
}

func main() {
	api, err := api.New("./.lens")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	api.SetupRoutes(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("starting pprof server on :6060")
		log.Fatal(http.ListenAndServe(":6060", nil))
	}()

	log.Println("server is listening on", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("error starting server: %v", err)
	}
}
