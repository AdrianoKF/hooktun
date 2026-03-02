package shared

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SetupLogger configures zerolog with the specified log level
func SetupLogger(level, format string) string {
	resolvedFormat := resolveLogFormat(format)
	zerolog.TimeFieldFormat = time.RFC3339

	switch resolvedFormat {
	case "json":
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	default:
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:     os.Stdout,
			NoColor: true,
		})
	}

	// Set log level
	switch strings.ToLower(level) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	return resolvedFormat
}

func resolveLogFormat(format string) string {
	switch strings.ToLower(format) {
	case "json", "console":
		return strings.ToLower(format)
	case "auto", "":
		// Cloud Run sets K_SERVICE in the runtime environment.
		if os.Getenv("K_SERVICE") != "" {
			return "json"
		}
		return "console"
	default:
		if os.Getenv("K_SERVICE") != "" {
			return "json"
		}
		return "console"
	}
}
