package relay

import "github.com/adrianokf/go-webhook-relay/internal/shared"

// Client represents an SSE client connection
type Client struct {
	ChannelID string
	Send      chan *shared.WebhookEvent
	Done      chan struct{}
}

// NewClient creates a new client
func NewClient(channelID string) *Client {
	return &Client{
		ChannelID: channelID,
		Send:      make(chan *shared.WebhookEvent, 10), // Buffered to avoid blocking
		Done:      make(chan struct{}),
	}
}
