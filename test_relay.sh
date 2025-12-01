#!/bin/bash

# TSS Wallet Backend - Relay API Test Script
# Usage: ./test_relay.sh

BASE_URL="http://localhost:8080"
DEVICE1="device_$(date +%s)_1"
DEVICE2="device_$(date +%s)_2"

echo "================================================"
echo "   TSS Wallet Backend - Relay API Tests"
echo "================================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper function
test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"

    echo -e "${YELLOW}Testing: $name${NC}"
    echo "  $method $endpoint"

    if [ "$method" == "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint")
    fi

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo -e "  ${GREEN}✓ Status: $http_code${NC}"
    else
        echo -e "  ${RED}✗ Status: $http_code${NC}"
    fi
    echo "  Response: $body"
    echo ""
}

# ============================================================
# 1. Health Check
# ============================================================
echo "=============================================="
echo "1. HEALTH CHECK"
echo "=============================================="

test_endpoint "Health Check" "GET" "/health"

# ============================================================
# 2. Device Registration
# ============================================================
echo "=============================================="
echo "2. DEVICE REGISTRATION"
echo "=============================================="

# Generate fake X25519 public keys (base64 encoded 32 bytes)
PUBKEY1=$(openssl rand -base64 32)
PUBKEY2=$(openssl rand -base64 32)

# Correct endpoint: POST /relay/register (not /relay/device/register)
test_endpoint "Register Device 1" "POST" "/relay/register" \
    "{\"deviceId\": \"$DEVICE1\", \"publicKey\": \"$PUBKEY1\"}"

test_endpoint "Register Device 2" "POST" "/relay/register" \
    "{\"deviceId\": \"$DEVICE2\", \"publicKey\": \"$PUBKEY2\"}"

# ============================================================
# 3. Get Device Public Key
# ============================================================
echo "=============================================="
echo "3. GET DEVICE PUBLIC KEY"
echo "=============================================="

test_endpoint "Get Device 1 Public Key" "GET" "/relay/device/$DEVICE1/publickey"
test_endpoint "Get Device 2 Public Key" "GET" "/relay/device/$DEVICE2/publickey"

# ============================================================
# 4. Create Session
# ============================================================
echo "=============================================="
echo "4. CREATE MPC SESSION"
echo "=============================================="

SESSION_ID="session_$(date +%s)"

test_endpoint "Create KeyGen Session" "POST" "/relay/session" \
    "{\"sessionId\": \"$SESSION_ID\", \"parties\": [\"$DEVICE1\", \"$DEVICE2\"], \"type\": \"keygen\", \"threshold\": 2}"

# ============================================================
# 5. Join Session
# ============================================================
echo "=============================================="
echo "5. JOIN SESSION"
echo "=============================================="

test_endpoint "Device 1 Join Session" "POST" "/relay/session/$SESSION_ID/join" \
    "{\"deviceId\": \"$DEVICE1\"}"

test_endpoint "Device 2 Join Session" "POST" "/relay/session/$SESSION_ID/join" \
    "{\"deviceId\": \"$DEVICE2\"}"

# ============================================================
# 6. Get Session Status
# ============================================================
echo "=============================================="
echo "6. GET SESSION STATUS"
echo "=============================================="

test_endpoint "Get Session Status" "GET" "/relay/session/$SESSION_ID"

# ============================================================
# 7. Send Message
# ============================================================
echo "=============================================="
echo "7. SEND ENCRYPTED MESSAGE"
echo "=============================================="

# Fake encrypted payload (in real scenario, this would be E2E encrypted)
PAYLOAD=$(echo "test message round 1" | base64)

# Correct endpoint: POST /relay/send (not /relay/message)
test_endpoint "Send Message (Device1 -> Device2)" "POST" "/relay/send" \
    "{\"sessionId\": \"$SESSION_ID\", \"from\": \"$DEVICE1\", \"to\": [\"$DEVICE2\"], \"payload\": \"$PAYLOAD\", \"round\": 1, \"isBroadcast\": false}"

# ============================================================
# 8. Poll Messages
# ============================================================
echo "=============================================="
echo "8. POLL MESSAGES"
echo "=============================================="

# Correct endpoint: GET /relay/messages?deviceId=xxx (not /relay/message/:deviceId)
test_endpoint "Get Messages for Device 2" "GET" "/relay/messages?deviceId=$DEVICE2&sessionId=$SESSION_ID&limit=100"

# ============================================================
# 9. Update Session Status
# ============================================================
echo "=============================================="
echo "9. UPDATE SESSION STATUS"
echo "=============================================="

test_endpoint "Update Session to round 2" "PATCH" "/relay/session/$SESSION_ID" \
    "{\"status\": \"in_progress\", \"currentRound\": 2}"

# ============================================================
# 10. Acknowledge Message
# ============================================================
echo "=============================================="
echo "10. ACKNOWLEDGE MESSAGE"
echo "=============================================="

# Note: You need to get the actual messageId from step 8 to ack
# For demo, we'll use a fake one which will return error
test_endpoint "Ack Message (demo)" "POST" "/relay/ack" \
    "{\"messageId\": \"fake-message-id\", \"deviceId\": \"$DEVICE2\"}"

# ============================================================
# 11. Delete Session (Cleanup)
# ============================================================
echo "=============================================="
echo "11. DELETE SESSION (CLEANUP)"
echo "=============================================="

test_endpoint "Delete Session" "DELETE" "/relay/session/$SESSION_ID"

# ============================================================
# Summary
# ============================================================
echo "=============================================="
echo "TEST COMPLETE"
echo "=============================================="
echo ""
echo "Devices used:"
echo "  - Device 1: $DEVICE1"
echo "  - Device 2: $DEVICE2"
echo "  - Session:  $SESSION_ID"
echo ""
echo "API Endpoints Tested:"
echo "  POST /relay/register           - Register device"
echo "  GET  /relay/device/:id/publickey - Get public key"
echo "  POST /relay/session            - Create session"
echo "  GET  /relay/session/:id        - Get session"
echo "  POST /relay/session/:id/join   - Join session"
echo "  PATCH /relay/session/:id       - Update session"
echo "  DELETE /relay/session/:id      - Delete session"
echo "  POST /relay/send               - Send message"
echo "  GET  /relay/messages           - Poll messages"
echo "  POST /relay/ack                - Acknowledge message"
echo ""
echo "For WebSocket testing, use wscat or Postman:"
echo "  wscat -c ws://localhost:8080/relay/ws"
