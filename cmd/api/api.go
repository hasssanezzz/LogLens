package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hasssanezzz/try-bleve/internal"
)

type API struct {
	LogLens internal.LogLens
}

func New(source string) (*API, error) {
	logLens, err := internal.NewLogLens(source)
	if err != nil {
		return nil, err
	}
	return &API{LogLens: logLens}, nil
}

func (api *API) getHandler(w http.ResponseWriter, r *http.Request) {
	var q internal.Query
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid query format"))
		return
	}

	result, err := api.LogLens.Search(context.Background(), q)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("server failed to process the query"))
		log.Printf("bad query recieved %+v: %v\n", q, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (api *API) postHandler(w http.ResponseWriter, r *http.Request) {
	KV := map[string]string{}
	for key, value := range r.Header {
		if strings.HasPrefix(strings.ToLower(key), "kv-") {
			KV[key[2:]] = strings.Join(value, "")
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log := internal.NewLogEntry(KV, string(body))
	api.LogLens.Ingest(r.Context(), *log)
	w.WriteHeader(http.StatusOK)
}

func (api *API) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", api.getHandler)
	mux.HandleFunc("POST /", api.postHandler)
}
