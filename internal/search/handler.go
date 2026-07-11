package search

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var allowedLevels = map[string]bool{
	"trace": true,
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
	"fatal": true,
}

func Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()

	req := SearchRequest{
		Service: q.Get("service"),
		Level:   q.Get("level"),
		Q:       q.Get("q"),
	}

	if fromStr := q.Get("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			http.Error(w, "invalid 'from' timestamp, use RFC3339 format", http.StatusBadRequest)
			return
		}
		req.From = &t
	}

	if toStr := q.Get("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			http.Error(w, "invalid 'to' timestamp, use RFC3339 format", http.StatusBadRequest)
			return
		}
		req.To = &t
	}

	if req.From != nil && req.To != nil && req.To.Before(*req.From) {
		http.Error(w, "'to' must not be before 'from'", http.StatusBadRequest)
		return
	}

	if req.Level != "" {
		if !allowedLevels[strings.ToLower(req.Level)] {
			http.Error(w, "invalid 'level', must be one of: trace, debug, info, warn, error, fatal", http.StatusBadRequest)
			return
		}
		req.Level = strings.ToLower(req.Level)
	}

	req.Page = 1
	if pageStr := q.Get("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			http.Error(w, "invalid 'page', must be >= 1", http.StatusBadRequest)
			return
		}
		req.Page = p
	}

	req.Size = 20
	if sizeStr := q.Get("size"); sizeStr != "" {
		s, err := strconv.Atoi(sizeStr)
		if err != nil || s < 1 {
			http.Error(w, "invalid 'size', must be >= 1", http.StatusBadRequest)
			return
		}
		if s > 100 {
			s = 100
		}
		req.Size = s
	}

	resp, err := repo.Search(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
