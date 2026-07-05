package buffer

import (
	"LogStream/internal/kafka"
	"LogStream/internal/models"
	"time"
)

var IngestChan = make(chan models.Log, 1000)

func StartIngester() {

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		batch := make([]models.Log, 0, 100)

		for {
			select {
			case log := <-IngestChan:
				batch = append(batch, log)
				if len(batch) >= 100 {
					kafka.Flush(batch)
					batch = batch[:0]
				}
			case <-ticker.C:
				if len(batch) > 0 {
					kafka.Flush(batch)
					batch = batch[:0]
				}

			}
		}
	}()

}

func Submit(log models.Log) {
	IngestChan <- log
}
