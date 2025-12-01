#!/bin/bash

# TSS Wallet Backend - Legacy API Test Script (KeyGen + Signing)
# Usage: ./test_legacy.sh
# Note: These APIs run TSS on server (MVP mode, not production)

BASE_URL="http://localhost:8080"

echo "================================================"
echo "   TSS Wallet Backend - Legacy API Tests"
echo "   (Server-side TSS computation - MVP mode)"
echo "================================================"
echo ""
echo "⚠️  WARNING: These endpoints compute TSS on server"
echo "   In production, use Relay API instead"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ============================================================
# 1. Health Check
# ============================================================
echo "=============================================="
echo "1. HEALTH CHECK"
echo "=============================================="

echo -e "${YELLOW}GET /health${NC}"
curl -s $BASE_URL/health | jq .
echo ""

# ============================================================
# 2. KeyGen (⚠️ Takes 2-3 minutes due to pre-param generation)
# ============================================================
echo "=============================================="
echo "2. KEYGEN (2-of-3 Threshold)"
echo "=============================================="
echo ""
echo "⏳ This will take 2-3 minutes..."
echo "   (Generating cryptographic pre-parameters)"
echo ""

echo -e "${YELLOW}POST /keygen/start${NC}"
KEYGEN_RESPONSE=$(curl -s -X POST $BASE_URL/keygen/start \
    -H "Content-Type: application/json" \
    -d '{
        "threshold": 2,
        "parties": 3,
        "partyIds": ["device1", "device2", "device3"]
    }')

echo "$KEYGEN_RESPONSE" | jq .

# Extract sessionId
SESSION_ID=$(echo "$KEYGEN_RESPONSE" | jq -r '.sessionId')

if [ "$SESSION_ID" == "null" ] || [ -z "$SESSION_ID" ]; then
    echo -e "${RED}✗ KeyGen failed, cannot continue${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ KeyGen completed! Session ID: $SESSION_ID${NC}"
echo ""

# ============================================================
# 3. Get Key Shares
# ============================================================
echo "=============================================="
echo "3. GET KEY SHARES"
echo "=============================================="

echo -e "${YELLOW}GET /keygen/$SESSION_ID/share/device1${NC}"
SHARE1=$(curl -s "$BASE_URL/keygen/$SESSION_ID/share/device1")
echo "$SHARE1" | jq '{partyId, partyIndex, ethAddress, status}'
echo ""

echo -e "${YELLOW}GET /keygen/$SESSION_ID/share/device2${NC}"
SHARE2=$(curl -s "$BASE_URL/keygen/$SESSION_ID/share/device2")
echo "$SHARE2" | jq '{partyId, partyIndex, ethAddress, status}'
echo ""

# Extract Ethereum address
ETH_ADDRESS=$(echo "$SHARE1" | jq -r '.ethAddress')
echo -e "${GREEN}✓ Ethereum Address: $ETH_ADDRESS${NC}"
echo ""

# ============================================================
# 4. Signing (2-of-3)
# ============================================================
echo "=============================================="
echo "4. SIGNING (2-of-3 Threshold)"
echo "=============================================="

# Create a test message hash (32 bytes hex)
# This is SHA256("hello world")
MESSAGE_HASH="b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

echo "Message hash to sign: $MESSAGE_HASH"
echo ""

echo -e "${YELLOW}POST /signing/start${NC}"
SIGNING_RESPONSE=$(curl -s -X POST $BASE_URL/signing/start \
    -H "Content-Type: application/json" \
    -d "{
        \"keygenSessionId\": \"$SESSION_ID\",
        \"message\": \"$MESSAGE_HASH\",
        \"signerIds\": [\"device1\", \"device2\"]
    }")

echo "$SIGNING_RESPONSE" | jq .

# Extract signingId
SIGNING_ID=$(echo "$SIGNING_RESPONSE" | jq -r '.signingId')

if [ "$SIGNING_ID" == "null" ] || [ -z "$SIGNING_ID" ]; then
    echo -e "${RED}✗ Signing failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Signing completed! Signing ID: $SIGNING_ID${NC}"
echo ""

# ============================================================
# 5. Get Signature
# ============================================================
echo "=============================================="
echo "5. GET SIGNATURE"
echo "=============================================="

echo -e "${YELLOW}GET /signing/$SIGNING_ID/signature${NC}"
SIGNATURE=$(curl -s "$BASE_URL/signing/$SIGNING_ID/signature")
echo "$SIGNATURE" | jq .
echo ""

# Extract signature components
R=$(echo "$SIGNATURE" | jq -r '.r')
S=$(echo "$SIGNATURE" | jq -r '.s')
V=$(echo "$SIGNATURE" | jq -r '.v')
FULL_SIG=$(echo "$SIGNATURE" | jq -r '.signature')

echo "Signature Components:"
echo "  R: ${R:0:20}..."
echo "  S: ${S:0:20}..."
echo "  V: $V"
echo "  Full (65 bytes): ${FULL_SIG:0:40}..."
echo ""

# ============================================================
# 6. Verify Signature
# ============================================================
echo "=============================================="
echo "6. VERIFY SIGNATURE"
echo "=============================================="

echo -e "${YELLOW}POST /signing/verify${NC}"
VERIFY_RESPONSE=$(curl -s -X POST $BASE_URL/signing/verify \
    -H "Content-Type: application/json" \
    -d "{
        \"keygenSessionId\": \"$SESSION_ID\",
        \"message\": \"$MESSAGE_HASH\",
        \"signature\": \"$FULL_SIG\"
    }")

echo "$VERIFY_RESPONSE" | jq .

VALID=$(echo "$VERIFY_RESPONSE" | jq -r '.valid')
if [ "$VALID" == "true" ]; then
    echo -e "${GREEN}✓ Signature is VALID!${NC}"
else
    echo -e "${RED}✗ Signature verification FAILED${NC}"
fi
echo ""

# ============================================================
# Summary
# ============================================================
echo "=============================================="
echo "TEST SUMMARY"
echo "=============================================="
echo ""
echo "KeyGen Session:  $SESSION_ID"
echo "Signing Session: $SIGNING_ID"
echo "Ethereum Address: $ETH_ADDRESS"
echo "Signature Valid: $VALID"
echo ""
echo "Note: Key shares are stored in server memory"
echo "      They will be lost when server restarts"
echo ""
