package api

import (
	"LogStream/internal/models"
	"LogStream/internal/service"
	"encoding/json"
	"net/http"
)

func decodeIngestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload []models.Ingestion
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	service.Ingest(payload)

}
