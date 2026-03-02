package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adrianokf/hooktun/internal/relay"
	"github.com/adrianokf/hooktun/internal/shared"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	port           int
	logLevel       string
	logFormat      string
	channelSecrets string
)

var rootCmd = &cobra.Command{
	Use:   "hooktun",
	Short: "Hooktun relay server",
	Long:  `Hooktun - A tunnel server that receives webhooks and forwards them to connected clients via SSE`,
	Run:   run,
}

func init() {
	rootCmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&logFormat, "log-format", "auto", "Log format (auto, json, console)")
	rootCmd.Flags().StringVar(&channelSecrets, "channel-secrets", "", "Channel secrets (format: channel1:secret1,channel2:secret2)")

	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("log_level", rootCmd.Flags().Lookup("log-level"))
	viper.BindPFlag("log_format", rootCmd.Flags().Lookup("log-format"))
	viper.BindPFlag("channel_secrets", rootCmd.Flags().Lookup("channel-secrets"))

	viper.SetEnvPrefix("RELAY")
	viper.AutomaticEnv()
}

func run(cmd *cobra.Command, args []string) {
	// Get config from viper (respects env vars)
	port = viper.GetInt("port")
	logLevel = viper.GetString("log_level")
	logFormat = viper.GetString("log_format")
	channelSecrets = viper.GetString("channel_secrets")

	// Setup logger
	resolvedLogFormat := shared.SetupLogger(logLevel, logFormat)

	log.Info().
		Int("port", port).
		Str("log_level", logLevel).
		Str("log_format", resolvedLogFormat).
		Bool("auth_enabled", channelSecrets != "").
		Msg("Starting hooktun relay server")

	// Create server
	server := relay.NewServer(port, channelSecrets)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt or error
	select {
	case <-sigChan:
		log.Info().Msg("Received interrupt signal")
	case err := <-errChan:
		log.Error().Err(err).Msg("Server error")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
		os.Exit(1)
	}

	log.Info().Msg("Server stopped gracefully")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}
