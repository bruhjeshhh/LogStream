package main

import (
	"LogStream/health"
	"LogStream/internal/api"
	"LogStream/internal/buffer"
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
	ptr.HandleFunc("POST /ingest", api.DecodeIngestions)
	buffer.StartIngester()
	log.Printf("we ballin")
	log.Fatal(srv.ListenAndServe())
}
