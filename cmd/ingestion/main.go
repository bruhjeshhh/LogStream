package main

import (
	"LogStream/health"
	"LogStream/internal/api"
	"log"
	"net/http"
)

func main() {
	ptr := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: ptr,
	}

	ptr.HandleFunc("GET /api/healthz", health.App)
	ptr.HandleFunc("POST /ingest", api.IngestionRequest)

	log.Printf("we ballin")
	log.Fatal(srv.ListenAndServe())
}
