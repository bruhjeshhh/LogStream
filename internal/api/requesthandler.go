package api

import (
	"LogStream/internal/models"
	"LogStream/internal/service"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

func IngestionRequest(w http.ResponseWriter, r *http.Request) {
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

	var allowedLevels = []string{"trace", "debug", "info", "warn", "error", "fatal"}

	validPayloads := payload[:0]

	for _, pld := range payload {
		if pld.Level != "" || pld.Message != "" || pld.Service != "" || isJSONObject(pld.Metadata) && slices.Contains(allowedLevels, strings.ToLower(pld.Level)) {
			validPayloads = append(validPayloads, pld)
		}
	}

	service.Ingest(validPayloads)

}

func isJSONObject(data []byte) bool {
	var m map[string]any
	// json.Unmarshal returns nil error only if it fits the target type
	return json.Unmarshal(data, &m) == nil
}
