package relay

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// SecretsStore manages channel authentication secrets
type SecretsStore struct {
	secrets map[string]string
	enabled bool
}

// NewSecretsStore creates a new secrets store from a secrets string
// Format: "channel1:secret1,channel2:secret2"
func NewSecretsStore(secretsConfig string) *SecretsStore {
	store := &SecretsStore{
		secrets: make(map[string]string),
		enabled: false,
	}

	if secretsConfig == "" {
		log.Info().Msg("No channel secrets configured, authentication disabled")
		return store
	}

	// Parse secrets
	pairs := strings.Split(secretsConfig, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			log.Warn().
				Str("pair", pair).
				Msg("Invalid secret format, skipping")
			continue
		}

		channelID := strings.TrimSpace(parts[0])
		secret := strings.TrimSpace(parts[1])

		if channelID == "" || secret == "" {
			log.Warn().
				Str("pair", pair).
				Msg("Empty channel ID or secret, skipping")
			continue
		}

		store.secrets[channelID] = secret
	}

	if len(store.secrets) > 0 {
		store.enabled = true
		log.Info().
			Int("channels", len(store.secrets)).
			Msg("Channel secrets loaded, authentication enabled")
	}

	return store
}

// Validate checks if the provided token matches the channel's secret
// Returns true if valid or if authentication is disabled
func (s *SecretsStore) Validate(channelID, token string) bool {
	if !s.enabled {
		// Auth disabled, allow all
		return true
	}

	expectedSecret, exists := s.secrets[channelID]
	if !exists {
		// Channel not configured, deny
		log.Warn().
			Str("channel_id", channelID).
			Msg("Authentication failed: channel not configured")
		return false
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(token), []byte(expectedSecret)) != 1 {
		log.Warn().
			Str("channel_id", channelID).
			Msg("Authentication failed: invalid token")
		return false
	}

	return true
}

// IsEnabled returns whether authentication is enabled
func (s *SecretsStore) IsEnabled() bool {
	return s.enabled
}

// HasChannel returns whether a channel is configured
func (s *SecretsStore) HasChannel(channelID string) bool {
	if !s.enabled {
		return true // If auth disabled, all channels are "configured"
	}
	_, exists := s.secrets[channelID]
	return exists
}

// ExtractBearerToken extracts the token from an Authorization header
// Expected format: "Bearer <token>"
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	if strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("unsupported authentication scheme: %s", parts[0])
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	return token, nil
}
