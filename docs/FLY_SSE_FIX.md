# Fly.io SSE Connection Fix

## Problem

Fly.io's proxy was killing SSE connections every ~30 seconds with error:
```
[PU02] could not complete HTTP request to instance: connection closed before message completed
```

## Root Causes

1. **Keep-alive interval too long**: Original 30s interval was at or beyond Fly's timeout threshold
2. **Server WriteTimeout**: Go's http.Server WriteTimeout (30s) was killing long-lived SSE connections
3. **Missing proxy hints**: Headers not optimally configured for SSE

## Fixes Applied

### 1. Reduced Keep-Alive Interval (internal/relay/sse.go)

**Changed from:**
```go
ticker := time.NewTicker(30 * time.Second)
```

**Changed to:**
```go
ticker := time.NewTicker(15 * time.Second)
```

**Why:** Sending keep-alive pings every 15 seconds (2x more frequent) ensures the connection stays active well within Fly's timeout window.

### 2. Added X-Accel-Buffering Header (internal/relay/sse.go)

**Added:**
```go
w.Header().Set("X-Accel-Buffering", "no")
```

**Why:** Prevents reverse proxies (like Fly's) from buffering SSE responses, ensuring real-time delivery.

### 3. Removed Server WriteTimeout (internal/relay/server.go)

**Changed from:**
```go
WriteTimeout: 30 * time.Second,
```

**Changed to:**
```go
WriteTimeout: 0,  // Must be 0 for SSE
```

**Why:** WriteTimeout applies to the entire response duration. SSE connections are long-lived and must remain open indefinitely. The IdleTimeout (120s) still protects against truly idle connections.

### 4. Updated Fly.io Configuration (deployments/fly.toml)

**Added:**
```toml
[http_service.concurrency]
  type = "connections"
  hard_limit = 1000
  soft_limit = 500
```

**Why:** Properly configures Fly.io's connection handling for long-lived SSE connections.

## Verification

After deploying these changes:

1. **SSE connections should stay alive indefinitely**
2. **Keep-alive pings visible every 15s**: `: keepalive\n\n`
3. **No more "connection closed" errors** in Fly logs
4. **Client auto-reconnects still work** if disconnections do occur

## Testing

```bash
# Deploy changes
fly deploy

# Monitor logs
fly logs

# Connect a client
./client --relay-url=https://your-app.fly.dev --channel-id=test --target-url=http://localhost:3000

# Send test webhooks
curl -X POST https://your-app.fly.dev/webhook/test/api/hook -d '{"test":"data"}'

# Connection should stay stable for hours
```

## If Issues Persist

If connections still drop:

1. **Check Fly logs** for specific timeout errors
2. **Verify keep-alives are sent**: Look for `: keepalive` in client-side logs (if verbose)
3. **Try even more frequent pings**: Reduce to 10s if needed
4. **Consider Fly regions**: Some regions may have different timeout characteristics

## Additional Notes

- The IdleTimeout (120s) handles connections with no activity
- ReadTimeout (30s) still applies to initial request headers
- This configuration trades off some server-side timeout protection for SSE compatibility
- Consider rate limiting to prevent abuse of long-lived connections
