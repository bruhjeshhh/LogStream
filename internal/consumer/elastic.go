package consumer

import (
	"LogStream/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

var es *elasticsearch.Client

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

	es = client
	return nil
}

func index(ctx context.Context, entry models.Log) error {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	res, err := es.Index(
		"logs",
		bytes.NewReader(jsonData),
		es.Index.WithDocumentID(entry.ID.String()),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch returned %s", res.Status())
	}

	return nil
}
