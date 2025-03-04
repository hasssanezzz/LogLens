package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/hasssanezzz/try-bleve/cmd/api"
)

func parseArgs() (string, string) {
	if len(os.Args) < 3 {
		panic("no enough arguments passes, required: (addr, homepath)")
	}

	return os.Args[1], os.Args[2]
}

func main() {
	addr, homepath := parseArgs()

	api, err := api.New(homepath)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	api.SetupRoutes(mux)

	server := &http.Server{
		Addr:    addr,
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
