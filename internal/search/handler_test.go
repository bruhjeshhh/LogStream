package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
)

func setupTestRepo(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	esServer := httptest.NewServer(handler)
	t.Cleanup(esServer.Close)

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{esServer.URL},
	})
	if err != nil {
		t.Fatalf("failed to create ES client: %v", err)
	}

	repo = NewRepository(client)
}

func cannedESResponse(totalHits int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		resp := map[string]any{
			"hits": map[string]any{
				"total": map[string]any{"value": totalHits},
				"hits":  []any{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func TestSearch_DefaultParams(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SearchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if resp.Size != 20 {
		t.Errorf("expected size 20, got %d", resp.Size)
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
	if len(resp.Hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(resp.Hits))
	}
}

func TestSearch_AllParams(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?service=api&level=error&q=timeout&from=2025-01-01T00:00:00Z&to=2025-12-31T23:59:59Z&page=2&size=10", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SearchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Page)
	}
	if resp.Size != 10 {
		t.Errorf("expected size 10, got %d", resp.Size)
	}
}

func TestSearch_InvalidFromTimestamp(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?from=not-a-timestamp", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_InvalidToTimestamp(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?to=2025/01/01", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_ToBeforeFrom(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?from=2025-12-31T00:00:00Z&to=2025-01-01T00:00:00Z", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_InvalidLevel(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?level=invalid", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_LevelCaseInsensitive(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?level=ERROR", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSearch_InvalidPage(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?page=0", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_InvalidSize(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?size=0", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_SizeCappedAt100(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?size=500", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SearchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Size != 100 {
		t.Errorf("expected size capped at 100, got %d", resp.Size)
	}
}

func TestSearch_WrongMethod(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodPost, "/search", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestSearch_NonNumericPage(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?page=abc", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_NonNumericSize(t *testing.T) {
	setupTestRepo(t, cannedESResponse(0))

	req := httptest.NewRequest(http.MethodGet, "/search?size=abc", nil)
	w := httptest.NewRecorder()

	Search(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_AllLevelsAccepted(t *testing.T) {
	levels := []string{"trace", "debug", "info", "warn", "error", "fatal"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			setupTestRepo(t, cannedESResponse(0))

			req := httptest.NewRequest(http.MethodGet, "/search?level="+level, nil)
			w := httptest.NewRecorder()

			Search(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200 for level %q, got %d", level, w.Code)
			}
		})
	}
}
