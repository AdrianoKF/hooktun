#!/bin/bash

# Local testing script for webhook relay

set -e

echo "=== Webhook Relay Local Test ==="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if relay is running
echo -e "${BLUE}Step 1: Checking if relay server is running on port 8080...${NC}"
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "Relay server not running. Starting it..."
    echo "Run in a separate terminal: make run-relay"
    exit 1
fi
echo -e "${GREEN}✓ Relay server is running${NC}"
echo ""

# Check if target server is running
echo -e "${BLUE}Step 2: Checking if target server is running on port 3000...${NC}"
if ! curl -s http://localhost:3000 > /dev/null 2>&1; then
    echo "Target server not running. Starting it..."
    echo "Run in a separate terminal: python3 -m http.server 3000"
    exit 1
fi
echo -e "${GREEN}✓ Target server is running${NC}"
echo ""

# Check if client is running
echo -e "${BLUE}Step 3: Make sure client is connected...${NC}"
echo "Run in a separate terminal: make run-client"
echo "Press Enter when client is connected..."
read

# Send test webhook
echo -e "${BLUE}Step 4: Sending test webhook...${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    http://localhost:8080/webhook/test123/api/hook \
    -H "Content-Type: application/json" \
    -H "X-Test-Header: TestValue" \
    -d '{"test": "data", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 202 ]; then
    echo -e "${GREEN}✓ Webhook accepted (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
else
    echo "✗ Unexpected response (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi
echo ""

# Test with query parameters
echo -e "${BLUE}Step 5: Sending webhook with query parameters...${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    "http://localhost:8080/webhook/test123/api/events?event=push&repo=test" \
    -H "Content-Type: application/json" \
    -d '{"action": "push", "data": "example"}')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" -eq 202 ]; then
    echo -e "${GREEN}✓ Webhook with query params accepted${NC}"
else
    echo "✗ Unexpected response (HTTP $HTTP_CODE)"
    exit 1
fi
echo ""

# Test webhook when no client is connected
echo -e "${BLUE}Step 6: Testing webhook without client (optional)...${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    http://localhost:8080/webhook/noclient/test \
    -H "Content-Type: application/json" \
    -d '{"test": "orphan webhook"}')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" -eq 202 ]; then
    echo -e "${GREEN}✓ Webhook accepted even without client (logged only)${NC}"
else
    echo "✗ Unexpected response (HTTP $HTTP_CODE)"
    exit 1
fi
echo ""

echo -e "${GREEN}=== All tests passed! ===${NC}"
echo ""
echo "Check client logs to verify webhook delivery."
