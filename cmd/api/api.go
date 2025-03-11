package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hasssanezzz/try-bleve/internal"
)

const DateLayout = "2006-01-02"

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

	result, err := api.LogLens.Search(r.Context(), q)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server failed to process the query"))
		log.Printf("bad query recieved \n%+v\nerror: %v\n", q, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (api *API) rangeCountHandler(w http.ResponseWriter, r *http.Request) {
	start, end := r.URL.Query().Get("start"), r.URL.Query().Get("end")
	if start == "" || end == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("please provide start/end date"))
		return
	}

	// parse the start date
	sdate, err := time.Parse(DateLayout, start)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("please provide a valid start date"))
		log.Println(err)
		return
	}

	// parse the end date
	edate, err := time.Parse(DateLayout, end)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("please provide a valid end date"))
		log.Println(err)
		return
	}

	results, err := api.LogLens.RangeCountSearch(r.Context(), sdate.UnixMicro(), edate.UnixMicro())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server failed to process the query"))
		log.Printf("bad query recieved \n%q -> %q\nerror: %v\n", start, end, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(results)
}

func (api *API) postHandler(w http.ResponseWriter, r *http.Request) {
	KV := map[string]string{}
	for key, value := range r.Header {
		if strings.HasPrefix(strings.ToLower(key), "kv-") {
			KV[key[3:]] = strings.Join(value, "")
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
	mux.HandleFunc("GET /range-count", api.rangeCountHandler)
	mux.HandleFunc("POST /", api.postHandler)
}
