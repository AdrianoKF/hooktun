# Go Webhook Relay

A lightweight webhook relay system that forwards webhook events from a public relay server to local clients via Server-Sent Events (SSE).

## Overview

Go Webhook Relay consists of two components:

1. **Relay Server**: A public server that receives webhooks and forwards them to connected clients via SSE
2. **Client**: A local application that connects to the relay server and forwards received webhooks to a target service

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐         ┌──────────┐
│   Webhook   │         │    Relay     │         │   Client    │         │  Local   │
│   Source    │────────▶│    Server    │────────▶│  (via SSE)  │────────▶│  Service │
│  (GitHub,   │  POST   │  (Public)    │   SSE   │  (Private)  │  POST   │  (Port   │
│   etc.)     │         │              │         │             │         │   3000)  │
└─────────────┘         └──────────────┘         └─────────────┘         └──────────┘
```

## Features

- **Simple Channel-Based Routing**: Route webhooks using channel IDs in the URL
- **Real-Time Delivery**: SSE provides instant webhook delivery to connected clients
- **Automatic Reconnection**: Clients reconnect automatically with exponential backoff
- **1:1 Channel Mapping**: One client per channel (new clients replace existing ones)
- **Full Request Preservation**: Headers, body, query parameters, and paths are preserved
- **No Authentication Required**: Simple MVP design with channel IDs only
- **Graceful Handling**: Accepts webhooks even when no client is connected (logged only)

## Quick Start

### Prerequisites

- Go 1.25 or later
- Make (optional, for convenience)

### Installation

```bash
# Clone the repository
git clone https://github.com/adrianokf/go-webhook-relay.git
cd go-webhook-relay

# Install dependencies
go mod download
```

### Running Locally

#### Terminal 1: Start the Relay Server

```bash
make run-relay
# or
go run ./cmd/relay --port=8080 --log-level=info
```

#### Terminal 2: Start a Test HTTP Server

```bash
# Simple Python HTTP server
python3 -m http.server 3000
```

#### Terminal 3: Start the Client

```bash
make run-client
# or
go run ./cmd/client \
  --relay-url=http://localhost:8080 \
  --channel-id=test123 \
  --target-url=http://localhost:3000 \
  --log-level=info
```

#### Terminal 4: Send a Test Webhook

```bash
curl -X POST http://localhost:8080/webhook/test123/api/hook \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

You should see the webhook appear in the client logs and be forwarded to your local service.

### Automated Testing

```bash
./scripts/local-test.sh
```

## Usage

### Relay Server

The relay server accepts webhooks and forwards them to connected clients.

**Command:**
```bash
./bin/relay [flags]
```

**Flags:**
- `--port`: Port to listen on (default: 8080)
- `--log-level`: Log level: debug, info, warn, error (default: info)

**Environment Variables:**
- `RELAY_PORT`: Port to listen on
- `LOG_LEVEL`: Log level

**Webhook URL Format:**
```
POST http://your-relay-server.com/webhook/{channel-id}/{path}
```

**Example:**
```bash
# This webhook will be sent to the client connected to channel "abc123"
# The client will receive it with path "/github/push"
curl -X POST https://relay.example.com/webhook/abc123/github/push \
  -H "Content-Type: application/json" \
  -d '{"event": "push"}'
```

**SSE Connection Endpoint:**
```
GET http://your-relay-server.com/connect/{channel-id}
```

**Health Check:**
```
GET http://your-relay-server.com/health
```

### Client

The client connects to a relay server and forwards webhooks to a local target.

**Command:**
```bash
./bin/client [flags]
```

**Flags:**
- `--relay-url`: Relay server URL (required)
- `--channel-id`: Unique channel identifier (required)
- `--target-url`: Local target URL to forward webhooks to (required)
- `--log-level`: Log level: debug, info, warn, error (default: info)

**Environment Variables:**
- `RELAY_URL`: Relay server URL
- `CHANNEL_ID`: Channel identifier
- `TARGET_URL`: Target URL
- `LOG_LEVEL`: Log level

**Example:**
```bash
./bin/client \
  --relay-url=https://relay.example.com \
  --channel-id=my-channel-123 \
  --target-url=http://localhost:3000 \
  --log-level=info
```

## Building

### Build Both Components

```bash
make build-all
```

This creates:
- `bin/relay` - Relay server binary
- `bin/client` - Client binary

### Build Individually

```bash
# Build relay server
make build-relay

# Build client
make build-client
```

## Deployment

### Docker

Build and run the relay server in Docker:

```bash
make docker-build
make docker-run
```

Or manually:

```bash
docker build -f deployments/Dockerfile -t go-webhook-relay:latest .
docker run -p 8080:8080 go-webhook-relay:latest
```

### Fly.io

Deploy the relay server to Fly.io:

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login
fly auth login

# Launch (first time)
fly launch

# Deploy
fly deploy

# Check status
fly status

# View logs
fly logs
```

The relay will be available at `https://your-app.fly.dev`.

### Other Platforms

The relay server is a standard Go HTTP application and can be deployed to:
- AWS (ECS, Lambda, EC2)
- Google Cloud (Cloud Run, GKE, Compute Engine)
- Azure (Container Instances, AKS, App Service)
- DigitalOcean (App Platform, Kubernetes)
- Heroku
- Railway
- Render

## Architecture

### Components

#### Relay Server

- **Hub**: Manages client connections (1:1 channel-to-client mapping)
- **SSE Handler**: Establishes SSE connections with clients
- **Webhook Handler**: Receives webhooks and broadcasts them to appropriate clients
- **HTTP Server**: Chi router with middleware (logging, recovery, request ID)

#### Client

- **SSE Reader**: Connects to relay server, reads events, handles reconnection
- **Forwarder**: Reconstructs HTTP requests and forwards them to target
- **Client Orchestrator**: Coordinates SSE reader and forwarder

### Data Flow

1. Webhook source sends POST to `https://relay.example.com/webhook/abc123/github/push`
2. Relay extracts channel ID (`abc123`) and path (`/github/push`)
3. Relay creates `WebhookEvent` with full request details
4. Hub broadcasts event to client connected on channel `abc123`
5. SSE handler writes event as `data: {JSON}\n\n`
6. Client's SSE reader receives and deserializes event
7. Forwarder reconstructs request to `http://localhost:3000/github/push`
8. Client forwards request and logs response

### WebhookEvent Structure

```json
{
  "id": "uuid-v4",
  "timestamp": "2024-01-15T10:30:00Z",
  "method": "POST",
  "path": "/github/push",
  "query_params": "foo=bar",
  "headers": {
    "Content-Type": ["application/json"],
    "X-GitHub-Event": ["push"]
  },
  "body": "base64-encoded-body",
  "channel_id": "abc123"
}
```

## Configuration Reference

### Relay Server

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--port` | `RELAY_PORT` | 8080 | Port to listen on |
| `--log-level` | `LOG_LEVEL` | info | Log level (debug, info, warn, error) |

### Client

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--relay-url` | `RELAY_URL` | - | Relay server URL (required) |
| `--channel-id` | `CHANNEL_ID` | - | Unique channel ID (required) |
| `--target-url` | `TARGET_URL` | - | Local target URL (required) |
| `--log-level` | `LOG_LEVEL` | info | Log level (debug, info, warn, error) |

## Error Handling

### Relay Server

- **Webhook with no client**: Logs the webhook, returns 202 Accepted
- **Panic recovery**: Chi middleware catches panics
- **Graceful shutdown**: Closes all connections and stops hub

### Client

- **Connection failure**: Exponential backoff reconnection (1s, 2s, 4s, 8s, max 30s)
- **Forwarding failure**: Logs error, continues processing
- **Invalid event**: Logs warning, skips event
- **Startup validation**: Fails fast if configuration is invalid

## Limitations

This is an MVP implementation with the following limitations:

- **No authentication**: Channel IDs are the only security mechanism
- **No persistence**: Webhooks are not stored; missed webhooks are lost
- **1:1 client mapping**: Only one client per channel at a time
- **No replay**: Webhooks cannot be replayed or retrieved
- **No web UI**: Configuration and monitoring are CLI-only

## Future Enhancements

Potential improvements (out of scope for MVP):

- Authentication tokens for channels
- Multi-client broadcast per channel
- Webhook persistence and replay
- Web UI for monitoring and management
- Request/response inspection dashboard
- Webhook signature verification
- Rate limiting per channel
- Metrics and observability (Prometheus)

## Dependencies

- [Chi](https://github.com/go-chi/chi) - HTTP router
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Zerolog](https://github.com/rs/zerolog) - Structured logging
- [UUID](https://github.com/google/uuid) - UUID generation

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Support

For issues, questions, or feature requests, please open an issue on GitHub.
