package client

import (
	"fmt"
	"net/url"

	"github.com/rs/zerolog/log"
)

// Client represents the webhook relay client
type Client struct {
	config    *Config
	reader    *SSEReader
	forwarder *Forwarder
	done      chan struct{}
}

// NewClient creates a new client
func NewClient(config *Config) (*Client, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return &Client{
		config:    config,
		reader:    NewSSEReader(config.RelayURL, config.ChannelID),
		forwarder: NewForwarder(config.TargetURL),
		done:      make(chan struct{}),
	}, nil
}

// Start starts the client
func (c *Client) Start() error {
	log.Info().
		Str("relay_url", c.config.RelayURL).
		Str("channel_id", c.config.ChannelID).
		Str("target_url", c.config.TargetURL).
		Msg("Starting webhook relay client")

	// Start SSE reader
	events := c.reader.Start()

	// Process events
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return fmt.Errorf("event channel closed")
			}

			// Forward the event
			if err := c.forwarder.Forward(event); err != nil {
				log.Error().
					Err(err).
					Str("event_id", event.ID).
					Msg("Failed to forward event, continuing")
				// Continue processing even if forwarding fails
			}

		case <-c.done:
			log.Info().Msg("Client stopping")
			c.reader.Stop()
			return nil
		}
	}
}

// Stop stops the client
func (c *Client) Stop() {
	close(c.done)
}

// validateConfig validates the client configuration
func validateConfig(config *Config) error {
	if config.RelayURL == "" {
		return fmt.Errorf("relay_url is required")
	}

	if config.ChannelID == "" {
		return fmt.Errorf("channel_id is required")
	}

	if config.TargetURL == "" {
		return fmt.Errorf("target_url is required")
	}

	// Validate URLs
	if _, err := url.Parse(config.RelayURL); err != nil {
		return fmt.Errorf("invalid relay_url: %w", err)
	}

	if _, err := url.Parse(config.TargetURL); err != nil {
		return fmt.Errorf("invalid target_url: %w", err)
	}

	return nil
}
