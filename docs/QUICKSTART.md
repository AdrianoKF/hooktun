# Quick Start Guide

Get started with Hooktun in 5 minutes.

## Installation

```bash
# Clone and build
git clone https://github.com/adrianokf/hooktun.git
cd hooktun
make build-all
```

## Basic Usage

### 1. Start the Relay Server

```bash
./bin/relay --port=8080
```

You should see:
```
8:19AM INF Starting hooktun server log_level=info port=8080
8:19AM INF Starting relay server port=8080
```

### 2. Start Your Local Service

Start any HTTP service on a local port. For testing:

```bash
python3 -m http.server 3000
```

### 3. Start the Client

```bash
./bin/client \
  --relay-url=http://localhost:8080 \
  --channel-id=my-channel \
  --target-url=http://localhost:3000
```

You should see:
```
8:19AM INF Starting hooktun client channel_id=my-channel relay_url=http://localhost:8080 target_url=http://localhost:3000
8:19AM INF Connecting to relay server channel_id=my-channel relay_url=http://localhost:8080
8:19AM INF Connected to relay server channel_id=my-channel
```

### 4. Send a Webhook

```bash
curl -X POST http://localhost:8080/webhook/my-channel/test \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, World!"}'
```

You should see:
- Relay logs: `Webhook accepted`
- Client logs: `Forwarding webhook` → `Webhook forwarded successfully`
- Target logs: Received POST request

## Real-World Example: GitHub Webhooks

### 1. Deploy Relay to Production

```bash
# Using Fly.io
fly launch
fly deploy

# Your relay will be at: https://your-app.fly.dev
```

### 2. Start Local Client

```bash
./bin/client \
  --relay-url=https://your-app.fly.dev \
  --channel-id=github-dev \
  --target-url=http://localhost:4000
```

### 3. Configure GitHub Webhook

Go to your GitHub repository → Settings → Webhooks → Add webhook:

- **Payload URL**: `https://your-app.fly.dev/webhook/github-dev/github/events`
- **Content type**: `application/json`
- **Events**: Select events you want

### 4. Test It

Make a commit to your repository. You should see:
1. GitHub sends webhook to your public relay
2. Relay forwards to your local client via SSE
3. Client forwards to your local development server at `localhost:4000`

## Using Environment Variables

Instead of flags, use environment variables:

```bash
# Relay
export RELAY_PORT=8080
export RELAY_LOG_LEVEL=info
./bin/hooktun

# Client (note the CLIENT_ prefix)
export CLIENT_RELAY_URL=http://localhost:8080
export CLIENT_CHANNEL_ID=my-channel
export CLIENT_TARGET_URL=http://localhost:3000
export CLIENT_LOG_LEVEL=debug
./bin/hooktun-client
```

## Multiple Clients

Each client needs a unique channel ID:

```bash
# Client 1: GitHub webhooks
./bin/client \
  --relay-url=https://relay.example.com \
  --channel-id=github-prod \
  --target-url=http://localhost:3000

# Client 2: Stripe webhooks
./bin/client \
  --relay-url=https://relay.example.com \
  --channel-id=stripe-dev \
  --target-url=http://localhost:4000
```

Configure webhooks:
- GitHub: `https://relay.example.com/webhook/github-prod/events`
- Stripe: `https://relay.example.com/webhook/stripe-dev/events`

## Troubleshooting

### Client not receiving webhooks

Check that:
1. Relay server is running: `curl http://localhost:8080/health`
2. Client is connected: Look for "Connected to relay server" in logs
3. Channel ID matches: URL uses same channel ID as client

### Connection keeps dropping

This is normal. The client automatically reconnects with exponential backoff. Look for:
```
8:20AM ERR Connection failed, retrying error="scanner error: unexpected EOF" retry_in=1000
8:20AM INF Connecting to relay server channel_id=test123 relay_url=http://localhost:8080
8:20AM INF Connected to relay server channel_id=test123
```

### Webhooks are lost

If no client is connected when a webhook arrives, it will be logged but not stored. This is by design. Ensure your client is running before webhooks arrive, or check relay logs for missed webhooks.

## What's Next?

- Read the [full documentation](README.md)
- Run the [test suite](scripts/local-test.sh)
- Deploy to [production](README.md#deployment)
