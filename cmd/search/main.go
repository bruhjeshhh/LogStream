package main

import (
	"LogStream/internal/search"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := search.InitElastic(); err != nil {
		log.Fatalf("failed to init elasticsearch: %v", err)
	}

	ptr := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":8084",
		Handler: ptr,
	}

	ptr.HandleFunc("GET /search", search.Search)

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	log.Printf("search is on")
	log.Fatal(srv.ListenAndServe())
}
