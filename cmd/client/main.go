package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/adrianokf/hooktun/internal/client"
	"github.com/adrianokf/hooktun/internal/shared"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	relayURL  string
	channelID string
	targetURL string
	token     string
	logLevel  string
)

var rootCmd = &cobra.Command{
	Use:   "hooktun-client",
	Short: "Hooktun client",
	Long:  `Hooktun client - Connects to a tunnel server and forwards webhooks to a local target`,
	Run:   run,
}

func init() {
	rootCmd.Flags().StringVar(&relayURL, "relay-url", "", "Relay server URL (required)")
	rootCmd.Flags().StringVar(&channelID, "channel-id", "", "Unique channel identifier (required)")
	rootCmd.Flags().StringVar(&targetURL, "target-url", "", "Local target URL to forward webhooks to (required)")
	rootCmd.Flags().StringVar(&token, "token", "", "Authentication token for the channel")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	// Don't mark flags as required - we'll validate after viper loads env vars
	// This allows environment variables to satisfy requirements

	viper.BindPFlag("relay_url", rootCmd.Flags().Lookup("relay-url"))
	viper.BindPFlag("channel_id", rootCmd.Flags().Lookup("channel-id"))
	viper.BindPFlag("target_url", rootCmd.Flags().Lookup("target-url"))
	viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))
	viper.BindPFlag("log_level", rootCmd.Flags().Lookup("log-level"))

	// Enable environment variable support
	// Environment variables should be prefixed with CLIENT_ (e.g., CLIENT_RELAY_URL)
	viper.SetEnvPrefix("CLIENT")
	// Replace dashes with underscores in env var names (relay-url -> RELAY_URL)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func run(cmd *cobra.Command, args []string) {
	// Get config from viper (respects env vars)
	relayURL = viper.GetString("relay_url")
	channelID = viper.GetString("channel_id")
	targetURL = viper.GetString("target_url")
	token = viper.GetString("token")
	logLevel = viper.GetString("log_level")

	// Validate required fields after viper loads config
	var missing []string
	if relayURL == "" {
		missing = append(missing, "relay-url (or CLIENT_RELAY_URL)")
	}
	if channelID == "" {
		missing = append(missing, "channel-id (or CLIENT_CHANNEL_ID)")
	}
	if targetURL == "" {
		missing = append(missing, "target-url (or CLIENT_TARGET_URL)")
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Error: Missing required configuration: %s\n", strings.Join(missing, ", "))
		os.Exit(1)
	}

	// Setup logger
	shared.SetupLogger(logLevel)

	log.Info().
		Str("relay_url", relayURL).
		Str("channel_id", channelID).
		Str("target_url", targetURL).
		Bool("auth_enabled", token != "").
		Str("log_level", logLevel).
		Msg("Starting hooktun client")

	// Create client config
	config := &client.Config{
		RelayURL:  relayURL,
		ChannelID: channelID,
		TargetURL: targetURL,
		Token:     token,
		LogLevel:  logLevel,
	}

	// Create client
	c, err := client.NewClient(config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create client")
		os.Exit(1)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start client in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := c.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt or error
	select {
	case <-sigChan:
		log.Info().Msg("Received interrupt signal")
		c.Stop()
	case err := <-errChan:
		log.Error().Err(err).Msg("Client error")
		os.Exit(1)
	}

	log.Info().Msg("Client stopped gracefully")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}
