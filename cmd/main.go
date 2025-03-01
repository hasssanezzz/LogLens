package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/hasssanezzz/try-bleve/cmd/api"
)

func main() {
	api, err := api.New("./.lens/2")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	api.SetupRoutes(mux)

	server := &http.Server{
		Addr:    ":3000",
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
