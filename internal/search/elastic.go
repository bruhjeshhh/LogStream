package search

import (
	"LogStream/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

var repo *Repository

func InitElastic() error {
	address := os.Getenv("ELASTICSEARCH_URL")
	if address == "" {
		address = "http://localhost:9200"
	}
	cfg := elasticsearch.Config{
		Addresses: []string{address},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return err
	}

	repo = NewRepository(client)
	return nil
}

type SearchRequest struct {
	Service string
	Level   string
	From    *time.Time
	To      *time.Time
	Q       string
	Page    int
	Size    int
}

type SearchResponse struct {
	Hits  []models.Log `json:"hits"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Size  int          `json:"size"`
}

type Repository struct {
	client *elasticsearch.Client
}

type esSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source models.Log `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func NewRepository(client *elasticsearch.Client) *Repository {
	return &Repository{
		client: client,
	}
}

func SetRepository(r *Repository) {
	repo = r
}

func (r *Repository) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	query := BuildQuery(req)
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("could not marshal query: %w", err)
	}

	from := (req.Page - 1) * req.Size

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex("logs"),
		r.client.Search.WithBody(bytes.NewReader(body)),
		r.client.Search.WithFrom(from),
		r.client.Search.WithSize(req.Size),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch returned %s", res.Status())
	}

	var esResp esSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		return nil, err
	}

	hits := make([]models.Log, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		hits = append(hits, hit.Source)
	}

	return &SearchResponse{
		Hits:  hits,
		Total: esResp.Hits.Total.Value,
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

func BuildQuery(req SearchRequest) map[string]any {
	var must []map[string]any
	var filter []map[string]any

	if req.Service != "" {
		filter = append(filter, map[string]any{
			"term": map[string]any{"service": req.Service},
		})
	}

	if req.Level != "" {
		filter = append(filter, map[string]any{
			"term": map[string]any{"level": req.Level},
		})
	}

	if req.From != nil || req.To != nil {
		rangeMap := map[string]any{}
		if req.From != nil {
			rangeMap["gte"] = req.From.Format(time.RFC3339)
		}
		if req.To != nil {
			rangeMap["lte"] = req.To.Format(time.RFC3339)
		}
		filter = append(filter, map[string]any{
			"range": map[string]any{"timestamp": rangeMap},
		})
	}

	if req.Q != "" {
		must = append(must, map[string]any{
			"match": map[string]any{"message": req.Q},
		})
	}

	if len(must) == 0 && len(filter) == 0 {
		return map[string]any{
			"query": map[string]any{
				"match_all": map[string]any{},
			},
		}
	}

	boolQuery := map[string]any{}
	if len(must) > 0 {
		boolQuery["must"] = must
	}
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}

	return map[string]any{
		"query": map[string]any{
			"bool": boolQuery,
		},
	}
}
