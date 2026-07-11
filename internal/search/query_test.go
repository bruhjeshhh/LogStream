package search

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildQuery_MatchAll(t *testing.T) {
	q := buildQuery(SearchRequest{})
	assertJSON(t, q, map[string]any{
		"query": map[string]any{
			"match_all": map[string]any{},
		},
	})
}

func TestBuildQuery_ServiceFilter(t *testing.T) {
	q := buildQuery(SearchRequest{Service: "api"})
	assertJSON(t, q, map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{"term": map[string]any{"service": "api"}},
				},
			},
		},
	})
}

func TestBuildQuery_LevelFilter(t *testing.T) {
	q := buildQuery(SearchRequest{Level: "error"})
	assertJSON(t, q, map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{"term": map[string]any{"level": "error"}},
				},
			},
		},
	})
}

func TestBuildQuery_TimeRange(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 30, 23, 59, 59, 0, time.UTC)

	q := buildQuery(SearchRequest{From: &from, To: &to})
	assertJSON(t, q, map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{
						"range": map[string]any{
							"timestamp": map[string]any{
								"gte": "2025-01-01T00:00:00Z",
								"lte": "2025-06-30T23:59:59Z",
							},
						},
					},
				},
			},
		},
	})
}

func TestBuildQuery_FromOnly(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	q := buildQuery(SearchRequest{From: &from})
	boolQuery := q["query"].(map[string]any)["bool"].(map[string]any)
	filters := boolQuery["filter"].([]map[string]any)
	rangeFilter := filters[0]["range"].(map[string]any)
	ts := rangeFilter["timestamp"].(map[string]any)

	if _, ok := ts["lte"]; ok {
		t.Error("expected no lte in range when only from is set")
	}
	if ts["gte"] != "2025-01-01T00:00:00Z" {
		t.Errorf("expected gte, got %v", ts["gte"])
	}
}

func TestBuildQuery_FreeText(t *testing.T) {
	q := buildQuery(SearchRequest{Q: "connection refused"})
	assertJSON(t, q, map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{"match": map[string]any{"message": "connection refused"}},
				},
			},
		},
	})
}

func TestBuildQuery_AllFilters(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 30, 23, 59, 59, 0, time.UTC)

	q := buildQuery(SearchRequest{
		Service: "api",
		Level:   "error",
		From:    &from,
		To:      &to,
		Q:       "timeout",
	})

	boolQ := q["query"].(map[string]any)["bool"].(map[string]any)

	must := boolQ["must"].([]map[string]any)
	if len(must) != 1 {
		t.Fatalf("expected 1 must clause, got %d", len(must))
	}

	filters := boolQ["filter"].([]map[string]any)
	if len(filters) != 3 {
		t.Fatalf("expected 3 filter clauses, got %d", len(filters))
	}
}

func assertJSON(t *testing.T, got, want map[string]any) {
	t.Helper()

	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)

	var gotParsed, wantParsed any
	json.Unmarshal(gotJSON, &gotParsed)
	json.Unmarshal(wantJSON, &wantParsed)

	gotBytes, _ := json.MarshalIndent(gotParsed, "", "  ")
	wantBytes, _ := json.MarshalIndent(wantParsed, "", "  ")

	if string(gotBytes) != string(wantBytes) {
		t.Errorf("query mismatch:\ngot:\n%s\n\nwant:\n%s", gotBytes, wantBytes)
	}
}
