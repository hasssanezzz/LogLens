package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/hasssanezzz/try-bleve/cmd/api"
	"github.com/rs/cors"
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

	corsOptions := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handlerWithCORS := corsOptions.Handler(mux)

	server := &http.Server{
		Addr:    addr,
		Handler: handlerWithCORS,
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
