package consumer

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/segmentio/kafka-go"
)

// Metrics are deliberately small so the project can be observed without a
// Prometheus client library or an extra service.
var metrics struct {
	processed atomic.Uint64
	failed    atomic.Uint64
	inFlight  atomic.Int64
	lag       atomic.Int64
}

// TrackLag copies kafka-go's latest lag estimate into the metrics endpoint.
func TrackLag(reader *kafka.Reader) {
	metrics.lag.Store(reader.Stats().Lag)
}

func MetricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP logstream_consumer_processed_total Logs indexed successfully.\n# TYPE logstream_consumer_processed_total counter\nlogstream_consumer_processed_total %d\n", metrics.processed.Load())
	fmt.Fprintf(w, "# HELP logstream_consumer_failed_total Logs sent to the dead-letter queue.\n# TYPE logstream_consumer_failed_total counter\nlogstream_consumer_failed_total %d\n", metrics.failed.Load())
	fmt.Fprintf(w, "# HELP logstream_consumer_in_flight Logs currently being written.\n# TYPE logstream_consumer_in_flight gauge\nlogstream_consumer_in_flight %d\n", metrics.inFlight.Load())
	fmt.Fprintf(w, "# HELP logstream_consumer_lag_messages Kafka reader lag estimate.\n# TYPE logstream_consumer_lag_messages gauge\nlogstream_consumer_lag_messages %d\n", metrics.lag.Load())
}

func MetricsHealthHandler(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
