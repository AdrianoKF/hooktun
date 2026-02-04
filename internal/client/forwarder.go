package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/adrianokf/hooktun/internal/shared"
	"github.com/rs/zerolog/log"
)

// Forwarder forwards webhook events to the target URL
type Forwarder struct {
	targetURL string
	client    *http.Client
}

// NewForwarder creates a new forwarder
func NewForwarder(targetURL string) *Forwarder {
	return &Forwarder{
		targetURL: targetURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Forward forwards a webhook event to the target
func (f *Forwarder) Forward(event *shared.WebhookEvent) error {
	// Decode body
	body, err := event.DecodeBody()
	if err != nil {
		return fmt.Errorf("failed to decode body: %w", err)
	}

	// Build target URL with path and query params
	targetURL, err := f.buildTargetURL(event.Path, event.QueryParams)
	if err != nil {
		return fmt.Errorf("failed to build target URL: %w", err)
	}

	// Create request
	req, err := http.NewRequest(event.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers (skip connection headers)
	skipHeaders := map[string]bool{
		"host":              true,
		"connection":        true,
		"keep-alive":        true,
		"proxy-connection":  true,
		"transfer-encoding": true,
		"upgrade":           true,
	}

	for key, values := range event.Headers {
		if !skipHeaders[strings.ToLower(key)] {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// Forward request
	log.Info().
		Str("event_id", event.ID).
		Str("method", event.Method).
		Str("url", targetURL).
		Msg("Forwarding webhook")

	resp, err := f.client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("event_id", event.ID).
			Msg("Failed to forward webhook")
		return err
	}
	defer resp.Body.Close()

	// Read response body
	respBody, _ := io.ReadAll(resp.Body)

	log.Info().
		Str("event_id", event.ID).
		Int("status", resp.StatusCode).
		Int("response_size", len(respBody)).
		Msg("Webhook forwarded successfully")

	return nil
}

// buildTargetURL constructs the full target URL
func (f *Forwarder) buildTargetURL(path, queryParams string) (string, error) {
	// Parse base URL
	base, err := url.Parse(f.targetURL)
	if err != nil {
		return "", err
	}

	// Parse path
	pathURL, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	// Combine base and path
	base.Path = pathURL.Path
	base.RawQuery = queryParams

	return base.String(), nil
}
