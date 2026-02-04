package relay

import (
	"net/http"
	"strings"

	"github.com/adrianokf/hooktun/internal/shared"
	"github.com/rs/zerolog/log"
)

// HandleWebhook handles incoming webhook requests
func HandleWebhook(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract channel ID from path: /webhook/{channel-id}/...
		path := r.URL.Path

		// Remove leading /webhook/
		if !strings.HasPrefix(path, "/webhook/") {
			http.Error(w, "Invalid webhook path", http.StatusBadRequest)
			return
		}

		pathParts := strings.SplitN(strings.TrimPrefix(path, "/webhook/"), "/", 2)
		if len(pathParts) == 0 || pathParts[0] == "" {
			http.Error(w, "Missing channel_id in path", http.StatusBadRequest)
			return
		}

		channelID := pathParts[0]

		// Reconstruct the path after channel ID
		remainingPath := "/"
		if len(pathParts) > 1 {
			remainingPath = "/" + pathParts[1]
		}

		// Create a modified request with the remaining path
		r.URL.Path = remainingPath

		// Create webhook event
		event, err := shared.NewWebhookEvent(r, channelID)
		if err != nil {
			log.Error().
				Err(err).
				Str("channel_id", channelID).
				Msg("Failed to create webhook event")
			http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
			return
		}

		// Broadcast to hub
		hub.Broadcast(event)

		// Always return 202 Accepted
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Webhook received"))

		log.Info().
			Str("channel_id", channelID).
			Str("event_id", event.ID).
			Str("method", event.Method).
			Str("path", remainingPath).
			Msg("Webhook accepted")
	}
}
