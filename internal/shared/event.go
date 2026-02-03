package shared

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// WebhookEvent represents a captured webhook request
type WebhookEvent struct {
	ID          string              `json:"id"`
	Timestamp   time.Time           `json:"timestamp"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	QueryParams string              `json:"query_params"`
	Headers     map[string][]string `json:"headers"`
	Body        string              `json:"body"` // base64 encoded
	ChannelID   string              `json:"channel_id"`
}

// NewWebhookEvent creates a WebhookEvent from an HTTP request
func NewWebhookEvent(r *http.Request, channelID string) (*WebhookEvent, error) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Encode body as base64
	encodedBody := base64.StdEncoding.EncodeToString(body)

	// Build query string
	queryParams := r.URL.RawQuery

	event := &WebhookEvent{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		Method:      r.Method,
		Path:        r.URL.Path,
		QueryParams: queryParams,
		Headers:     r.Header,
		Body:        encodedBody,
		ChannelID:   channelID,
	}

	return event, nil
}

// ToJSON serializes the event to JSON
func (e *WebhookEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON deserializes the event from JSON
func FromJSON(data []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// DecodeBody decodes the base64 body
func (e *WebhookEvent) DecodeBody() ([]byte, error) {
	return base64.StdEncoding.DecodeString(e.Body)
}
