package client

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/adrianokf/go-webhook-relay/internal/shared"
	"github.com/rs/zerolog/log"
)

// SSEReader reads events from SSE stream
type SSEReader struct {
	relayURL  string
	channelID string
	events    chan *shared.WebhookEvent
	done      chan struct{}
}

// NewSSEReader creates a new SSE reader
func NewSSEReader(relayURL, channelID string) *SSEReader {
	return &SSEReader{
		relayURL:  relayURL,
		channelID: channelID,
		events:    make(chan *shared.WebhookEvent),
		done:      make(chan struct{}),
	}
}

// Start begins reading from the SSE stream with reconnection
func (r *SSEReader) Start() <-chan *shared.WebhookEvent {
	go r.connectWithRetry()
	return r.events
}

// Stop stops the SSE reader
func (r *SSEReader) Stop() {
	close(r.done)
	close(r.events)
}

// connectWithRetry handles connection with exponential backoff
func (r *SSEReader) connectWithRetry() {
	delay := 1 * time.Second
	maxDelay := 30 * time.Second

	for {
		select {
		case <-r.done:
			return
		default:
		}

		log.Info().
			Str("relay_url", r.relayURL).
			Str("channel_id", r.channelID).
			Msg("Connecting to relay server")

		err := r.connect()
		if err != nil {
			log.Error().
				Err(err).
				Dur("retry_in", delay).
				Msg("Connection failed, retrying")

			select {
			case <-time.After(delay):
				// Exponential backoff
				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
			case <-r.done:
				return
			}
		} else {
			// Connection successful, reset delay
			delay = 1 * time.Second
		}
	}
}

// connect establishes SSE connection and reads events
func (r *SSEReader) connect() error {
	url := fmt.Sprintf("%s/connect/%s", r.relayURL, r.channelID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{
		Timeout: 0, // No timeout for SSE
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Info().
		Str("channel_id", r.channelID).
		Msg("Connected to relay server")

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var dataLines []string

	for scanner.Scan() {
		select {
		case <-r.done:
			return nil
		default:
		}

		line := scanner.Text()

		// Skip empty lines and comments (keep-alive)
		if line == "" {
			// Empty line signals end of event
			if len(dataLines) > 0 {
				// Parse accumulated data
				eventData := strings.Join(dataLines, "\n")
				r.processEvent(eventData)
				dataLines = nil
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			// Comment (keep-alive), skip
			continue
		}

		// Parse SSE field
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			dataLines = append(dataLines, data)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return fmt.Errorf("connection closed")
}

// processEvent parses and sends the event
func (r *SSEReader) processEvent(data string) {
	event, err := shared.FromJSON([]byte(data))
	if err != nil {
		log.Error().
			Err(err).
			Str("data", data).
			Msg("Failed to parse event")
		return
	}

	log.Debug().
		Str("event_id", event.ID).
		Str("method", event.Method).
		Str("path", event.Path).
		Msg("Event received")

	select {
	case r.events <- event:
	case <-r.done:
	}
}
