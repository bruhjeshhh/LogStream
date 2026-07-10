package search

import (
	"LogStream/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

var repo *Repository

func InitElastic() error {
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return err
	}

	repo = NewRepository(client)
	return nil
}

type Repository struct {
	client *elasticsearch.Client
}

type searchResponse struct {
	Hits struct {
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

func (r *Repository) SearchAll(ctx context.Context) ([]models.Log, error) {

	query := map[string]any{
		"query": map[string]any{
			"match_all": map[string]any{},
		},
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("couldnt marshal", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex("logs"), // your index name
		r.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch returned %s", res.Status())
	}

	var resp searchResponse

	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	logs := make([]models.Log, 0, len(resp.Hits.Hits))

	for _, hit := range resp.Hits.Hits {
		logs = append(logs, hit.Source)
	}

	return logs, nil

}
