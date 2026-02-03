package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/adrianokf/go-webhook-relay/internal/client"
	"github.com/adrianokf/go-webhook-relay/internal/shared"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	relayURL  string
	channelID string
	targetURL string
	logLevel  string
)

var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "Webhook relay client",
	Long:  `A client that connects to a relay server and forwards webhooks to a local target`,
	Run:   run,
}

func init() {
	rootCmd.Flags().StringVar(&relayURL, "relay-url", "", "Relay server URL (required)")
	rootCmd.Flags().StringVar(&channelID, "channel-id", "", "Unique channel identifier (required)")
	rootCmd.Flags().StringVar(&targetURL, "target-url", "", "Local target URL to forward webhooks to (required)")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	rootCmd.MarkFlagRequired("relay-url")
	rootCmd.MarkFlagRequired("channel-id")
	rootCmd.MarkFlagRequired("target-url")

	viper.BindPFlag("relay_url", rootCmd.Flags().Lookup("relay-url"))
	viper.BindPFlag("channel_id", rootCmd.Flags().Lookup("channel-id"))
	viper.BindPFlag("target_url", rootCmd.Flags().Lookup("target-url"))
	viper.BindPFlag("log_level", rootCmd.Flags().Lookup("log-level"))

	viper.AutomaticEnv()
}

func run(cmd *cobra.Command, args []string) {
	// Get config from viper (respects env vars)
	relayURL = viper.GetString("relay_url")
	channelID = viper.GetString("channel_id")
	targetURL = viper.GetString("target_url")
	logLevel = viper.GetString("log_level")

	// Setup logger
	shared.SetupLogger(logLevel)

	log.Info().
		Str("relay_url", relayURL).
		Str("channel_id", channelID).
		Str("target_url", targetURL).
		Str("log_level", logLevel).
		Msg("Starting webhook relay client")

	// Create client config
	config := &client.Config{
		RelayURL:  relayURL,
		ChannelID: channelID,
		TargetURL: targetURL,
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
