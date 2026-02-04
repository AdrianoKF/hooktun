package relay

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/rs/zerolog/log"
)

// Server represents the relay server
type Server struct {
	port    int
	hub     *Hub
	secrets *SecretsStore
	server  *http.Server
}

// NewServer creates a new relay server
func NewServer(port int, secretsConfig string) *Server {
	return &Server{
		port:    port,
		hub:     NewHub(),
		secrets: NewSecretsStore(secretsConfig),
	}
}

// Start starts the relay server
func (s *Server) Start() error {
	// Start hub
	go s.hub.Run()

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	// Rate limit: 100 requests per minute per IP, with burst of 20
	// This helps prevent DoS attempts while allowing normal traffic
	r.Use(httprate.LimitByIP(100, 1*time.Minute))

	// Routes
	r.Get("/connect/{channel_id}", HandleSSE(s.hub, s.secrets))
	r.HandleFunc("/webhook/*", HandleWebhook(s.hub))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Configure server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: r,
		// ReadTimeout applies to reading request headers and body
		ReadTimeout: 30 * time.Second,
		// WriteTimeout must be 0 for SSE (long-lived connections with periodic writes)
		// IdleTimeout handles truly idle connections instead
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	log.Info().
		Int("port", s.port).
		Msg("Starting relay server")

	// Start server
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("Shutting down relay server")

	// Close hub
	s.hub.Close()

	// Shutdown HTTP server
	return s.server.Shutdown(ctx)
}
