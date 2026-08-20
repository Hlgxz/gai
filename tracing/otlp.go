package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type otlpSpan struct {
	Name       string         `json:"name"`
	TraceID    string         `json:"traceId"`
	SpanID     string         `json:"spanId"`
	ParentID   string         `json:"parentSpanId,omitempty"`
	DurationNs int64          `json:"durationNano"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Error      string         `json:"error,omitempty"`
}

// EnableOTLP posts completed spans as JSON to endpoint (OTLP HTTP collector
// compatible enough for a sidecar that accepts application/json).
func EnableOTLP(endpoint string) {
	if endpoint == "" {
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	SetExporter(func(name, traceID, spanID, parentID string, duration time.Duration, attrs map[string]any, err error) {
		payload := otlpSpan{
			Name:       name,
			TraceID:    traceID,
			SpanID:     spanID,
			ParentID:   parentID,
			DurationNs: duration.Nanoseconds(),
			Attributes: attrs,
		}
		if err != nil {
			payload.Error = err.Error()
		}
		raw, merr := json.Marshal(payload)
		if merr != nil {
			return
		}
		req, rerr := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(raw))
		if rerr != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, derr := client.Do(req)
		if derr != nil {
			return
		}
		_ = resp.Body.Close()
	})
}
