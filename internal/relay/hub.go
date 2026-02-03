package relay

import (
	"sync"

	"github.com/adrianokf/go-webhook-relay/internal/shared"
	"github.com/rs/zerolog/log"
)

// Hub manages client connections and message broadcasting
type Hub struct {
	// Registered clients (1:1 mapping: channel -> client)
	clients map[string]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to clients
	broadcast chan *shared.WebhookEvent

	// Mutex for thread-safe access
	mu sync.RWMutex

	// Done channel for shutdown
	done chan struct{}
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *shared.WebhookEvent),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// Check if there's an existing client on this channel
			if existingClient, exists := h.clients[client.ChannelID]; exists {
				log.Info().
					Str("channel_id", client.ChannelID).
					Msg("Replacing existing client connection")
				// Close the existing client
				close(existingClient.Done)
				close(existingClient.Send)
			}
			// Register the new client
			h.clients[client.ChannelID] = client
			log.Info().
				Str("channel_id", client.ChannelID).
				Msg("Client registered")
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, exists := h.clients[client.ChannelID]; exists {
				delete(h.clients, client.ChannelID)
				close(client.Done)
				close(client.Send)
				log.Info().
					Str("channel_id", client.ChannelID).
					Msg("Client unregistered")
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.RLock()
			client, exists := h.clients[event.ChannelID]
			h.mu.RUnlock()

			if exists {
				select {
				case client.Send <- event:
					log.Debug().
						Str("channel_id", event.ChannelID).
						Str("event_id", event.ID).
						Msg("Event sent to client")
				default:
					// Client's send channel is full, skip
					log.Warn().
						Str("channel_id", event.ChannelID).
						Str("event_id", event.ID).
						Msg("Client send channel full, dropping event")
				}
			} else {
				log.Info().
					Str("channel_id", event.ChannelID).
					Str("event_id", event.ID).
					Str("method", event.Method).
					Str("path", event.Path).
					Msg("Webhook received but no client connected")
			}

		case <-h.done:
			log.Info().Msg("Hub shutting down")
			h.mu.Lock()
			for _, client := range h.clients {
				close(client.Done)
				close(client.Send)
			}
			h.clients = make(map[string]*Client)
			h.mu.Unlock()
			return
		}
	}
}

// Register registers a client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast broadcasts an event to the appropriate client
func (h *Hub) Broadcast(event *shared.WebhookEvent) {
	h.broadcast <- event
}

// Close shuts down the hub
func (h *Hub) Close() {
	close(h.done)
}
