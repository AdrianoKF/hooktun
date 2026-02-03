package relay

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// HandleSSE handles SSE connections from clients
func HandleSSE(hub *Hub, secrets *SecretsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := chi.URLParam(r, "channel_id")
		if channelID == "" {
			http.Error(w, "Missing channel_id", http.StatusBadRequest)
			return
		}

		// Validate authentication if enabled
		if secrets.IsEnabled() {
			authHeader := r.Header.Get("Authorization")
			token, err := ExtractBearerToken(authHeader)
			if err != nil {
				log.Warn().
					Err(err).
					Str("channel_id", channelID).
					Str("remote_addr", r.RemoteAddr).
					Msg("Authentication failed: invalid header format")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !secrets.Validate(channelID, token) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			log.Debug().
				Str("channel_id", channelID).
				Msg("Authentication successful")
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Check if we can flush
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Create and register client
		client := NewClient(channelID)
		hub.Register(client)

		// Ensure cleanup on disconnect
		defer func() {
			hub.Unregister(client)
		}()

		// Keep-alive ticker (15s for Fly.io proxy compatibility)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		log.Info().
			Str("channel_id", channelID).
			Str("remote_addr", r.RemoteAddr).
			Msg("SSE connection established")

		for {
			select {
			case event, ok := <-client.Send:
				if !ok {
					// Channel closed
					return
				}

				// Serialize event to JSON
				jsonData, err := event.ToJSON()
				if err != nil {
					log.Error().
						Err(err).
						Str("channel_id", channelID).
						Msg("Failed to serialize event")
					continue
				}

				// Write SSE event
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				flusher.Flush()

				log.Debug().
					Str("channel_id", channelID).
					Str("event_id", event.ID).
					Msg("Event sent via SSE")

			case <-ticker.C:
				// Send keep-alive ping
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()

			case <-client.Done:
				// Client disconnected
				return

			case <-r.Context().Done():
				// Request context cancelled (client disconnected)
				log.Info().
					Str("channel_id", channelID).
					Msg("SSE connection closed by client")
				return
			}
		}
	}
}
