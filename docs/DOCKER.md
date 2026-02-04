# Docker Deployment Guide

This guide covers deploying Hooktun components using Docker and Docker Compose.

## Quick Start with Docker Compose

The easiest way to add Hooktun client to your existing Docker Compose stack:

### Basic Example

```yaml
version: '3.8'

services:
  # Your existing service that receives webhooks
  app:
    image: your-app:latest
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=development

  # Hooktun client - tunnels webhooks to your app
  hooktun-client:
    image: ghcr.io/adrianokf/hooktun-client:latest
    # Or use Docker Hub:
    # image: arumpold/hooktun-client:latest
    restart: unless-stopped
    environment:
      - CLIENT_RELAY_URL=https://your-hooktun-server.fly.dev
      - CLIENT_CHANNEL_ID=my-channel
      - CLIENT_TARGET_URL=http://app:3000
      - CLIENT_LOG_LEVEL=info
    depends_on:
      - app
```

### With Authentication

If your Hooktun server has authentication enabled:

```yaml
version: '3.8'

services:
  app:
    image: your-app:latest
    ports:
      - "3000:3000"

  hooktun-client:
    image: ghcr.io/adrianokf/hooktun-client:latest
    restart: unless-stopped
    environment:
      - CLIENT_RELAY_URL=https://your-hooktun-server.fly.dev
      - CLIENT_CHANNEL_ID=production
      - CLIENT_TARGET_URL=http://app:3000
      - CLIENT_TOKEN=${HOOKTUN_TOKEN}  # Set in .env file
      - CLIENT_LOG_LEVEL=info
    depends_on:
      - app
```

Create a `.env` file:
```bash
HOOKTUN_TOKEN=your-secret-token-here
```

### Full Stack Example

Complete example with a web application and Hooktun client:

```yaml
version: '3.8'

services:
  # PostgreSQL database
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
    volumes:
      - db_data:/var/lib/postgresql/data

  # Your web application
  web:
    image: myapp:latest
    build: .
    ports:
      - "3000:3000"
    environment:
      - DATABASE_URL=postgresql://user:password@db:5432/myapp
      - NODE_ENV=development
    depends_on:
      - db

  # Hooktun client - receives webhooks from relay server
  hooktun:
    image: ghcr.io/adrianokf/hooktun-client:latest
    restart: unless-stopped
    environment:
      - CLIENT_RELAY_URL=https://hooktun.fly.dev
      - CLIENT_CHANNEL_ID=github-webhooks
      - CLIENT_TARGET_URL=http://web:3000/webhooks
      - CLIENT_TOKEN=${HOOKTUN_TOKEN}
      - CLIENT_LOG_LEVEL=info
    depends_on:
      - web
    # Optional: Override command for custom arguments
    # command: ["--relay-url=https://hooktun.fly.dev", "--channel-id=github-webhooks"]

volumes:
  db_data:
```

## Environment Variables

The client uses the `CLIENT_` prefix for environment variables:

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `CLIENT_RELAY_URL` | Yes | Hooktun relay server URL | `https://hooktun.fly.dev` |
| `CLIENT_CHANNEL_ID` | Yes | Unique channel identifier | `my-channel` |
| `CLIENT_TARGET_URL` | Yes | Local service URL to forward to | `http://app:3000` |
| `CLIENT_TOKEN` | No | Authentication token (if server requires it) | `secret-token-123` |
| `CLIENT_LOG_LEVEL` | No | Logging level (debug, info, warn, error) | `info` |

## Using Command-Line Arguments

You can also use command-line arguments instead of environment variables:

```yaml
services:
  hooktun-client:
    image: ghcr.io/adrianokf/hooktun-client:latest
    command:
      - --relay-url=https://hooktun.fly.dev
      - --channel-id=my-channel
      - --target-url=http://app:3000
      - --token=secret-token-123
      - --log-level=info
    depends_on:
      - app
```

## Multi-Channel Setup

Run multiple Hooktun clients for different channels:

```yaml
version: '3.8'

services:
  app:
    image: myapp:latest
    ports:
      - "3000:3000"

  # GitHub webhooks
  hooktun-github:
    image: ghcr.io/adrianokf/hooktun-client:latest
    restart: unless-stopped
    environment:
      - CLIENT_RELAY_URL=https://hooktun.fly.dev
      - CLIENT_CHANNEL_ID=github-prod
      - CLIENT_TARGET_URL=http://app:3000/github/webhook
      - CLIENT_TOKEN=${GITHUB_HOOKTUN_TOKEN}

  # Stripe webhooks
  hooktun-stripe:
    image: ghcr.io/adrianokf/hooktun-client:latest
    restart: unless-stopped
    environment:
      - CLIENT_RELAY_URL=https://hooktun.fly.dev
      - CLIENT_CHANNEL_ID=stripe-prod
      - CLIENT_TARGET_URL=http://app:3000/stripe/webhook
      - CLIENT_TOKEN=${STRIPE_HOOKTUN_TOKEN}
```

## Health Checks

Add health checks to monitor the client:

```yaml
services:
  hooktun-client:
    image: ghcr.io/adrianokf/hooktun-client:latest
    restart: unless-stopped
    environment:
      - CLIENT_RELAY_URL=https://hooktun.fly.dev
      - CLIENT_CHANNEL_ID=my-channel
      - CLIENT_TARGET_URL=http://app:3000
    healthcheck:
      test: ["CMD", "pgrep", "-x", "hooktun-client"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

## Networking

### Internal Service Communication

When your target service is in the same Docker network, use the service name:

```yaml
services:
  api:
    image: api:latest
    # No need to expose ports externally

  hooktun-client:
    environment:
      - CLIENT_TARGET_URL=http://api:8080  # Use service name
```

### External Service

To forward to a service outside Docker:

```yaml
services:
  hooktun-client:
    image: ghcr.io/adrianokf/hooktun-client:latest
    network_mode: host  # Required to access host services
    environment:
      - CLIENT_TARGET_URL=http://localhost:3000  # Host service
      - RELAY_URL=https://hooktun.fly.dev
      - CLIENT_CHANNEL_ID=my-channel
```

**Note:** Using `network_mode: host` makes other Docker services unreachable by service name.

## Logging

### View Logs

```bash
# Follow logs
docker-compose logs -f hooktun-client

# Last 100 lines
docker-compose logs --tail=100 hooktun-client

# With timestamps
docker-compose logs -f -t hooktun-client
```

### Configure Log Level

```yaml
services:
  hooktun-client:
    environment:
      - CLIENT_LOG_LEVEL=debug  # debug, info, warn, error
```

### JSON Logs

Hooktun uses structured JSON logging (zerolog). Parse with jq:

```bash
docker-compose logs hooktun-client | jq -r '.message'
```

## Production Considerations

### Use Specific Versions

Don't use `:latest` in production:

```yaml
services:
  hooktun-client:
    image: ghcr.io/adrianokf/hooktun-client:v1.0.0
```

### Restart Policy

```yaml
services:
  hooktun-client:
    restart: unless-stopped  # or 'always'
```

### Resource Limits

```yaml
services:
  hooktun-client:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 128M
        reservations:
          memory: 64M
```

### Secrets Management

Use Docker secrets for sensitive data:

```yaml
version: '3.8'

services:
  hooktun-client:
    image: ghcr.io/adrianokf/hooktun-client:latest
    secrets:
      - hooktun_token
    environment:
      - CLIENT_RELAY_URL=https://hooktun.fly.dev
      - CLIENT_CHANNEL_ID=production
      - CLIENT_TARGET_URL=http://app:3000
    command: ["--token=/run/secrets/hooktun_token"]

secrets:
  hooktun_token:
    file: ./secrets/hooktun_token.txt
```

## Troubleshooting

### Client Won't Connect

Check logs for connection errors:
```bash
docker-compose logs hooktun-client | grep -i error
```

Common issues:
- Wrong `RELAY_URL` - verify server is accessible
- Invalid `TOKEN` - check authentication credentials
- Network issues - ensure DNS resolution works

### Target Service Unreachable

Test connectivity from client container:
```bash
docker-compose exec hooktun-client wget -O- http://app:3000/health
```

### Connection Keeps Dropping

This is normal - SSE connections timeout periodically. The client automatically reconnects. Look for:
```
{"level":"info","message":"Connecting to relay server"}
{"level":"info","message":"Connected to relay server"}
```

## Complete Example

Save as `docker-compose.yml`:

```yaml
version: '3.8'

services:
  # Example Node.js app
  app:
    image: node:20-alpine
    working_dir: /app
    command: npx http-server -p 3000
    volumes:
      - ./app:/app
    environment:
      - NODE_ENV=development

  # Hooktun client
  hooktun:
    image: ghcr.io/adrianokf/hooktun-client:latest
    restart: unless-stopped
    environment:
      - CLIENT_RELAY_URL=https://hooktun.fly.dev
      - CLIENT_CHANNEL_ID=dev-channel
      - CLIENT_TARGET_URL=http://app:3000
      - CLIENT_LOG_LEVEL=info
    depends_on:
      - app

  # Optional: Local relay server for testing
  hooktun-server:
    image: ghcr.io/adrianokf/hooktun:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - CLIENT_LOG_LEVEL=info
```

Run with:
```bash
docker-compose up -d
docker-compose logs -f
```

## Next Steps

- [Authentication Guide](AUTHENTICATION.md) - Secure your channels
- [Quick Start](QUICKSTART.md) - Local development setup
- [Main README](../README.md) - Full documentation
