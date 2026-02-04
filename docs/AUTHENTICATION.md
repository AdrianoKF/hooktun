# Authentication

Hooktun supports channel-based authentication using pre-shared secrets.

## Overview

- **Optional**: Authentication is disabled by default for backwards compatibility
- **Bearer Token**: Uses standard `Authorization: Bearer <token>` headers
- **Per-Channel**: Each channel has its own secret token
- **Secure**: Uses constant-time comparison to prevent timing attacks

## Enabling Authentication

### Relay Server Configuration

Configure channel secrets via environment variable or CLI flag:

**Format**: `channel1:secret1,channel2:secret2`

#### Via Environment Variable

```bash
export RELAY_CHANNEL_SECRETS="prod-channel:super-secret-token-123,dev-channel:dev-token-456"
./relay
```

#### Via CLI Flag

```bash
./relay --channel-secrets="prod-channel:super-secret-token-123,dev-channel:dev-token-456"
```

#### Via fly.toml (Fly.io Deployment)

```toml
[env]
  RELAY_CHANNEL_SECRETS = "prod-channel:super-secret-token-123,dev-channel:dev-token-456"
```

**Security Note**: For production, use Fly.io secrets instead:

```bash
fly secrets set RELAY_CHANNEL_SECRETS="prod-channel:super-secret-token-123,dev-channel:dev-token-456"
```

### Client Configuration

Provide the token when connecting:

#### Via Environment Variable

```bash
export TOKEN="super-secret-token-123"
./client --relay-url=https://relay.example.com \
         --channel-id=prod-channel \
         --target-url=http://localhost:3000
```

#### Via CLI Flag

```bash
./client --relay-url=https://relay.example.com \
         --channel-id=prod-channel \
         --target-url=http://localhost:3000 \
         --token=super-secret-token-123
```

## How It Works

### Connection Flow with Authentication

1. **Client Connects**: Sends `Authorization: Bearer <token>` header
2. **Relay Validates**:
   - Checks if channel is configured
   - Compares token using constant-time comparison
   - Allows or denies connection
3. **SSE Stream**: If valid, establishes SSE connection
4. **Webhook Delivery**: Normal operation continues

### Without Authentication (Default)

If no secrets are configured:
- Authentication is **disabled**
- All channels are allowed
- No token validation occurs
- Backwards compatible with existing deployments

## Security Considerations

### Token Requirements

- **Minimum Length**: Use at least 32 characters
- **Randomness**: Generate cryptographically random tokens
- **Uniqueness**: Each channel should have a unique token

### Generating Secure Tokens

```bash
# Generate a secure random token (macOS/Linux)
openssl rand -base64 32

# Or using Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"

# Example output
dQw4w9WgXcQ_r7yN-kL9mP8vB2xZ5tA3hJ6fE1sC4uG0
```

### Best Practices

1. **Use Secrets Management**: Store secrets in environment variables or secret managers (Fly.io secrets, AWS Secrets Manager, etc.)
2. **Rotate Regularly**: Change tokens periodically
3. **Limit Exposure**: Don't commit tokens to version control
4. **Use HTTPS**: Always use HTTPS in production to protect tokens in transit
5. **Monitor Access**: Review logs for authentication failures

### What's Protected

✅ **SSE Connection**: Authentication required to establish connection
✅ **Channel Access**: Only configured channels can connect
✅ **Timing Attacks**: Constant-time comparison prevents timing-based attacks

### What's NOT Protected

❌ **Webhook Endpoint**: The `/webhook/{channel-id}/*` endpoint does NOT require authentication
- Anyone with the channel ID can send webhooks
- This is by design for simplicity
- If channel ID is secret, this provides obscurity-based security

**Why?** Webhook sources (GitHub, Stripe, etc.) typically don't support custom authentication headers. They rely on webhook secrets/signatures instead.

## Migration Guide

### Enabling Auth on Existing Deployment

1. **Generate Tokens**: Create secure tokens for each channel
2. **Update Relay**: Add `RELAY_CHANNEL_SECRETS` environment variable
3. **Update Clients**: Add `--token` flag to each client
4. **Test**: Verify clients can connect
5. **Monitor**: Check logs for authentication errors

### Example Migration

**Before (No Auth):**
```bash
# Relay
./relay --port=8080

# Client
./client --relay-url=http://localhost:8080 \
         --channel-id=prod \
         --target-url=http://localhost:3000
```

**After (With Auth):**
```bash
# Relay
./relay --port=8080 \
        --channel-secrets="prod:dQw4w9WgXcQ_r7yN-kL9mP8vB2xZ5tA3"

# Client
./client --relay-url=http://localhost:8080 \
         --channel-id=prod \
         --target-url=http://localhost:3000 \
         --token=dQw4w9WgXcQ_r7yN-kL9mP8vB2xZ5tA3
```

## Troubleshooting

### "Unauthorized" Error

**Symptoms**: Client can't connect, receives HTTP 401

**Causes**:
1. Token is missing or incorrect
2. Channel not configured in relay
3. Token format is invalid

**Solutions**:
```bash
# Check relay logs for authentication errors
fly logs | grep -i auth

# Verify channel is configured
# Look for "Channel secrets loaded" in relay startup logs

# Test with verbose logging
./relay --log-level=debug
./client --log-level=debug
```

### "Missing Authorization header"

**Cause**: Client not sending token

**Solution**: Add `--token` flag to client command

### Authentication Not Enforced

**Cause**: No secrets configured on relay

**Solution**:
1. Check relay logs for "No channel secrets configured"
2. Add `--channel-secrets` flag or `RELAY_CHANNEL_SECRETS` env var
3. Restart relay

## API Reference

### Authentication Header Format

```http
GET /connect/{channel_id} HTTP/1.1
Authorization: Bearer <token>
Accept: text/event-stream
```

### Error Responses

#### 401 Unauthorized
```http
HTTP/1.1 401 Unauthorized

Unauthorized
```

Reasons:
- Missing Authorization header
- Invalid token format
- Token doesn't match channel secret
- Channel not configured

#### 400 Bad Request
```http
HTTP/1.1 400 Bad Request

Missing channel_id
```

Reason: Channel ID not provided in URL

## Examples

### Single Channel

```bash
# Relay
export RELAY_CHANNEL_SECRETS="github-prod:prod-token-123"
./relay

# Client
./client --relay-url=https://relay.example.com \
         --channel-id=github-prod \
         --token=prod-token-123 \
         --target-url=http://localhost:3000
```

### Multiple Channels

```bash
# Relay
export RELAY_CHANNEL_SECRETS="github-prod:token1,stripe-prod:token2,dev:token3"
./relay

# Client 1 (GitHub)
./client --channel-id=github-prod --token=token1 ...

# Client 2 (Stripe)
./client --channel-id=stripe-prod --token=token2 ...

# Client 3 (Development)
./client --channel-id=dev --token=token3 ...
```

### Production Deployment (Fly.io)

```bash
# Set secrets (not visible in fly.toml)
fly secrets set RELAY_CHANNEL_SECRETS="prod:$(openssl rand -base64 32)"

# Deploy
fly deploy

# Get the relay URL
fly info

# Connect client
./client --relay-url=https://your-app.fly.dev \
         --channel-id=prod \
         --token=<your-token> \
         --target-url=http://localhost:3000
```
