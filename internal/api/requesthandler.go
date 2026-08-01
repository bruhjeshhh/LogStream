package api

import (
	"LogStream/internal/models"
	"LogStream/internal/service"
	"encoding/json"
	"net/http"
	"time"
)

func DecodeIngestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()

	var payload []models.Ingestion
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	accepted, rejected := service.Ingest(payload)
	recordRequest(accepted, rejected, time.Since(start))
}
