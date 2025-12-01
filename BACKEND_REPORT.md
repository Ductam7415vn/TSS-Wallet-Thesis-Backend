# TSS Wallet Backend - Technical Report

**Project:** Threshold Signature Scheme (TSS) Wallet Backend
**Version:** 5.0.0-Relay
**Date:** December 1, 2025
**Status:** Phase 3.1 - Relay Server Complete ✅

---

## Executive Summary

This document provides a comprehensive technical report of the TSS Wallet Backend implementation. The backend has evolved from a server-side TSS computation engine to a **"Dumb Pipe" Relay Server** that routes encrypted messages between client devices for True MPC Architecture.

### Architecture Evolution

```
┌─────────────────────────────────────────────────────────────────┐
│                    ARCHITECTURE EVOLUTION                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  BEFORE (MVP Phase 1-2):          AFTER (Phase 3.1+):           │
│  ═══════════════════════          ══════════════════════════    │
│                                                                  │
│  ┌─────────────────────┐          ┌─────────────────────┐       │
│  │   TSS Engine        │          │   Message Router    │       │
│  │   ───────────       │          │   ──────────────    │       │
│  │   - KeyGen Logic    │  ────►   │   (No crypto logic) │       │
│  │   - Signing Logic   │          │   - Routes messages │       │
│  │   - Share Compute   │          │   - WebSocket hub   │       │
│  └─────────────────────┘          └─────────────────────┘       │
│                                                                  │
│  Security: Server knows         Security: Server knows          │
│  all key shares (BAD)           NOTHING (GOOD - Zero Knowledge) │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Key Achievements

| Phase | Features | Status |
|-------|----------|--------|
| Week 1 | KeyGen Protocol | ✅ Done (Archived) |
| Week 2-3 | Signing Protocol | ✅ Done (Archived) |
| Week 4 | Ethereum Integration | ✅ Done |
| Phase 2 | Production Infrastructure | ✅ Done |
| **Phase 3.1** | **Relay Server** | **✅ Done (NEW)** |

---

## What's New in Phase 3.1

### 1. Relay Server Architecture

The backend is now a **"Dumb Pipe"** that:
- Routes encrypted messages between devices
- Cannot decrypt or read message content
- Has zero knowledge of key shares
- Uses WebSocket for real-time delivery
- Uses Redis for pub/sub message routing

### 2. New Components

| Component | File | Description |
|-----------|------|-------------|
| **Types** | `relay/types.go` | All request/response types |
| **Redis** | `relay/redis.go` | Redis client for message routing |
| **WebSocket** | `relay/websocket.go` | Real-time message delivery |
| **Handlers** | `relay/handler.go` | HTTP/WS endpoints |
| **Archive** | `archive/` | Old TSS logic (for reference) |

### 3. New Relay API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/relay/register` | Register device with X25519 public key |
| GET | `/relay/device/:deviceId/publickey` | Get device's public key |
| POST | `/relay/session` | Create MPC session |
| GET | `/relay/session/:sessionId` | Get session details |
| POST | `/relay/session/:sessionId/join` | Join session |
| PATCH | `/relay/session/:sessionId` | Update session status |
| DELETE | `/relay/session/:sessionId` | Delete session |
| POST | `/relay/send` | Send E2E encrypted message |
| GET | `/relay/messages` | Poll pending messages |
| POST | `/relay/ack` | Acknowledge message receipt |
| GET | `/relay/ws` | WebSocket connection |

---

## Relay API Documentation (For Android Team)

### Base URL

```
Development: http://localhost:8080
WebSocket:   ws://localhost:8080/relay/ws
```

### 1. Device Registration

Register a device to receive messages via relay.

```http
POST /relay/register
Content-Type: application/json
```

**Request:**
```json
{
    "deviceId": "device_uuid_here",
    "publicKey": "X25519_public_key_base64",
    "pushToken": "FCM_token_optional"
}
```

**Response (201 Created):**
```json
{
    "success": true,
    "deviceId": "device_uuid_here",
    "expiresAt": 1701475200,
    "message": "Device registered successfully"
}
```

### 2. Get Device Public Key

Get another device's public key for E2E encryption.

```http
GET /relay/device/:deviceId/publickey
```

**Response:**
```json
{
    "success": true,
    "deviceId": "device_uuid_here",
    "publicKey": "X25519_public_key_base64"
}
```

### 3. Create MPC Session

Create a new KeyGen or Signing session.

```http
POST /relay/session
Content-Type: application/json
```

**Request:**
```json
{
    "sessionId": "session_unique_id",
    "parties": ["device1", "device2", "device3"],
    "type": "keygen",
    "threshold": 2
}
```

**Response (201 Created):**
```json
{
    "success": true,
    "sessionId": "session_unique_id",
    "status": "waiting",
    "message": "Session created successfully"
}
```

### 4. Join Session

A device joins an existing session.

```http
POST /relay/session/:sessionId/join
Content-Type: application/json
```

**Request:**
```json
{
    "deviceId": "device1"
}
```

**Response:**
```json
{
    "success": true,
    "sessionId": "session_unique_id",
    "parties": ["device1", "device2", "device3"],
    "joinedParties": ["device1"],
    "status": "waiting",
    "message": "Joined session successfully"
}
```

### 5. Get Session Status

```http
GET /relay/session/:sessionId
```

**Response:**
```json
{
    "success": true,
    "sessionId": "session_unique_id",
    "type": "keygen",
    "parties": ["device1", "device2", "device3"],
    "joinedParties": ["device1", "device2"],
    "threshold": 2,
    "status": "in_progress",
    "currentRound": 3,
    "createdAt": 1701388800
}
```

### 6. Send Encrypted Message

Send an E2E encrypted message to other parties.

```http
POST /relay/send
Content-Type: application/json
```

**Request:**
```json
{
    "sessionId": "session_unique_id",
    "from": "device1",
    "to": ["device2", "device3"],
    "payload": "base64_encrypted_message",
    "round": 1,
    "isBroadcast": false
}
```

**Response:**
```json
{
    "success": true,
    "messageId": "msg_uuid",
    "status": "sent",
    "message": "Message sent to 2 recipients"
}
```

### 7. Poll Messages (HTTP Fallback)

For devices that can't use WebSocket.

```http
GET /relay/messages?deviceId=device1&sessionId=session_id&limit=100
```

**Response:**
```json
{
    "success": true,
    "messages": [
        {
            "messageId": "msg_uuid",
            "sessionId": "session_id",
            "from": "device2",
            "to": "device1",
            "payload": "base64_encrypted_message",
            "round": 1,
            "timestamp": 1701388800
        }
    ],
    "hasMore": false
}
```

### 8. Acknowledge Message

Mark a message as delivered.

```http
POST /relay/ack
Content-Type: application/json
```

**Request:**
```json
{
    "messageId": "msg_uuid",
    "deviceId": "device1"
}
```

**Response:**
```json
{
    "success": true,
    "message": "Message acknowledged"
}
```

---

## WebSocket Protocol

### Connection

```
ws://localhost:8080/relay/ws
```

### Message Types

#### 1. Authentication (Client → Server)

```json
{
    "type": "auth",
    "deviceId": "device1",
    "authToken": "JWT_token"
}
```

#### 2. Auth Response (Server → Client)

```json
{
    "type": "auth_response",
    "success": true,
    "error": ""
}
```

#### 3. Subscribe to Session (Client → Server)

```json
{
    "type": "subscribe",
    "sessionId": "session_id"
}
```

#### 4. Unsubscribe (Client → Server)

```json
{
    "type": "unsubscribe",
    "sessionId": "session_id"
}
```

#### 5. Message Notification (Server → Client)

```json
{
    "type": "message",
    "messageId": "msg_uuid",
    "sessionId": "session_id",
    "from": "device2",
    "payload": "base64_encrypted_message",
    "round": 1,
    "timestamp": 1701388800
}
```

#### 6. Acknowledge (Client → Server)

```json
{
    "type": "ack",
    "messageId": "msg_uuid"
}
```

#### 7. Session Event (Server → Client)

```json
{
    "type": "session_event",
    "sessionId": "session_id",
    "event": "party_joined",
    "partyId": "device2",
    "round": 0
}
```

Events: `party_joined`, `party_left`, `completed`, `failed`, `round_advanced`

#### 8. Ping/Pong (Heartbeat)

```json
{"type": "ping", "timestamp": 1701388800}
{"type": "pong", "timestamp": 1701388800}
```

#### 9. Error

```json
{
    "type": "error",
    "code": "UNAUTHORIZED",
    "message": "Not authenticated"
}
```

---

## E2E Encryption Specification (For Android)

### Encryption Scheme

| Component | Algorithm | Purpose |
|-----------|-----------|---------|
| Key Exchange | X25519 | Derive shared secret |
| Encryption | ChaCha20-Poly1305 | Authenticated encryption |
| Nonce | 12 bytes random | Unique per message |

### Android Implementation Flow

```kotlin
// 1. Generate X25519 keypair (once per device)
val keyPair = X25519.generateKeyPair()
val publicKey = keyPair.publicKey

// 2. Register device with server
POST /relay/register {
    deviceId: deviceId,
    publicKey: Base64.encode(publicKey)
}

// 3. Get recipient's public key
GET /relay/device/{recipientId}/publickey
val recipientPublicKey = response.publicKey

// 4. Derive shared secret
val sharedSecret = X25519.computeSharedSecret(
    myPrivateKey,
    recipientPublicKey
)

// 5. Encrypt message
val nonce = SecureRandom().generateBytes(12)
val ciphertext = ChaCha20Poly1305.encrypt(
    key = sharedSecret,
    nonce = nonce,
    plaintext = tssMessage,
    aad = sessionId.toBytes()
)

// 6. Send via relay
POST /relay/send {
    sessionId: sessionId,
    from: myDeviceId,
    to: [recipientId],
    payload: Base64.encode(nonce + ciphertext),
    round: currentRound
}
```

---

## Database Schema (Phase 3.1)

### New Relay Tables

```sql
-- Device registration for relay
CREATE TABLE relay_devices (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(64) UNIQUE NOT NULL,
    public_key TEXT NOT NULL,           -- X25519 public key
    push_token TEXT,                     -- FCM token (optional)
    is_online BOOLEAN DEFAULT false,
    last_seen_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- MPC sessions for coordination
CREATE TABLE relay_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) UNIQUE NOT NULL,
    session_type VARCHAR(32) NOT NULL,   -- 'keygen' or 'signing'
    parties JSONB NOT NULL,              -- Device IDs
    joined_parties JSONB DEFAULT '[]',
    threshold INTEGER NOT NULL,
    status VARCHAR(32) NOT NULL,         -- 'waiting', 'in_progress', 'completed', 'failed'
    current_round INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Message queue for relay
CREATE TABLE relay_messages (
    id SERIAL PRIMARY KEY,
    message_id VARCHAR(128) UNIQUE NOT NULL,
    session_id VARCHAR(64) NOT NULL,
    from_device VARCHAR(64) NOT NULL,
    to_device VARCHAR(64) NOT NULL,
    payload TEXT NOT NULL,               -- E2E encrypted, base64
    round INTEGER DEFAULT 0,
    is_delivered BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    delivered_at TIMESTAMP
);
```

---

## Docker Deployment

### Docker Compose Stack

```yaml
services:
  postgres:
    image: postgres:15-alpine
    # ... PostgreSQL configuration

  redis:                    # NEW in Phase 3.1
    image: redis:7-alpine
    ports:
      - "6379:6379"

  backend:
    build: .
    depends_on:
      - postgres
      - redis             # NEW dependency
    environment:
      REDIS_HOST: redis
      REDIS_PORT: 6379
      # ... other config
```

### Environment Variables (New)

```bash
# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# WebSocket Configuration
WS_PING_INTERVAL=30       # Ping interval in seconds
WS_PONG_TIMEOUT=10        # Pong timeout in seconds
WS_WRITE_TIMEOUT=10       # Write timeout in seconds
WS_MAX_MESSAGE_SIZE=65536 # Max message size in bytes (64KB)

# Relay Configuration
RELAY_MESSAGE_TTL=3600       # Message TTL in seconds (1 hour)
RELAY_SESSION_TTL=86400      # Session TTL in seconds (24 hours)
RELAY_DEVICE_TTL=604800      # Device registration TTL (7 days)
RELAY_CLEANUP_INTERVAL=300   # Cleanup interval (5 minutes)
```

### Quick Start

```bash
# 1. Start services
docker-compose up -d

# 2. Check health
curl http://localhost:8080/health

# Expected response:
{
    "status": "ok",
    "service": "tss-wallet-backend",
    "version": "5.0.0-relay",
    "redis": "connected"
}
```

---

## File Structure (Phase 3.1)

```
tss-wallet-backend/
├── main.go                 # Entry point (updated for relay)
│
├── relay/                  # NEW: Relay Server Package
│   ├── types.go            # Request/Response/WebSocket types
│   ├── redis.go            # Redis client wrapper
│   ├── websocket.go        # WebSocket hub & client
│   └── handler.go          # HTTP/WS handlers
│
├── archive/                # NEW: Archived MVP Code
│   ├── README.md           # Archive documentation
│   ├── keygen.go           # Old KeyGen handlers
│   ├── keygen_service.go   # Old KeyGen TSS logic
│   ├── signing.go          # Old Signing handlers
│   └── signing_service.go  # Old Signing TSS logic
│
├── keygen.go               # Still available (backward compat)
├── keygen_service.go       # Still available (backward compat)
├── signing.go              # Still available (backward compat)
├── signing_service.go      # Still available (backward compat)
│
├── ethereum.go             # Ethereum handlers (kept)
├── ethereum_service.go     # Ethereum service (kept)
│
├── config/
│   └── config.go           # Updated with Redis/WS/Relay config
│
├── database/
│   ├── database.go         # Updated with relay tables
│   └── repository.go
│
├── middleware/
│   ├── auth.go
│   ├── ratelimit.go
│   └── audit.go
│
├── Dockerfile
├── docker-compose.yml      # Updated with Redis
├── .env.example            # Updated with new variables
└── ...
```

---

## Android Integration Guide

### Phase 3.2 Tasks (For Android Team)

| Task | Priority | Description |
|------|----------|-------------|
| Gomobile Integration | Critical | Package tss-lib as .aar |
| X25519 KeyPair | Critical | Generate device key pair |
| ChaCha20-Poly1305 | Critical | Implement E2E encryption |
| WebSocket Client | Critical | Connect to `/relay/ws` |
| Local KeyGen | High | Run KeyGen on device |
| Local Signing | High | Run Signing on device |
| Android Keystore | High | Hardware-backed key storage |
| HTTP Polling | Medium | Fallback if WS unavailable |

### Recommended Libraries

```kotlin
// build.gradle.kts
dependencies {
    // Cryptography
    implementation("org.bouncycastle:bcprov-jdk15on:1.70")
    implementation("com.google.crypto.tink:tink-android:1.7.0")

    // WebSocket
    implementation("com.squareup.okhttp3:okhttp:4.11.0")

    // JSON
    implementation("com.google.code.gson:gson:2.10.1")

    // TSS (Gomobile .aar)
    implementation(files("libs/tss-lib.aar"))
}
```

### Communication Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    TRUE MPC FLOW (Phase 3)                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────┐                           ┌──────────┐           │
│   │ Device 1 │                           │ Device 2 │           │
│   │ (Phone)  │                           │ (Tablet) │           │
│   │          │                           │          │           │
│   │ ┌──────┐ │    E2E Encrypted Msgs     │ ┌──────┐ │           │
│   │ │ TSS  │ │ ◄───────────────────────► │ │ TSS  │ │           │
│   │ │Engine│ │    via Relay Server       │ │Engine│ │           │
│   │ └──────┘ │                           │ └──────┘ │           │
│   └────┬─────┘                           └────┬─────┘           │
│        │                                      │                  │
│        │         ┌────────────────┐          │                  │
│        │         │                │          │                  │
│        └────────►│  RELAY SERVER  │◄─────────┘                  │
│                  │  (Dumb Pipe)   │                             │
│                  │                │                             │
│                  │ • Routes msgs  │                             │
│                  │ • Zero decrypt │                             │
│                  │ • Redis pubsub │                             │
│                  │ • WebSocket    │                             │
│                  └────────────────┘                             │
│                                                                  │
│   Server knows: NOTHING about keys or computations              │
│   Clients compute: ALL TSS operations locally                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## API Quick Reference

### Legacy APIs (Still Working - Backward Compatibility)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/keygen/start` | Server-side KeyGen |
| GET | `/keygen/:sessionId/status` | Check status |
| GET | `/keygen/:sessionId/share/:partyId` | Get key share |
| POST | `/signing/start` | Server-side Signing |
| GET | `/signing/:signingId/signature` | Get signature |
| POST | `/ethereum/quick-send` | Send ETH |

### New Relay APIs (Phase 3.1)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/relay/register` | Register device |
| GET | `/relay/device/:id/publickey` | Get public key |
| POST | `/relay/session` | Create session |
| GET | `/relay/session/:id` | Get session |
| POST | `/relay/session/:id/join` | Join session |
| POST | `/relay/send` | Send message |
| GET | `/relay/messages` | Poll messages |
| POST | `/relay/ack` | Acknowledge |
| GET | `/relay/ws` | WebSocket |

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0-mvp | Nov 2025 | Week 1 - KeyGen complete |
| 2.0.0-mvp | Nov 2025 | Week 2-3 - Signing complete |
| 3.0.0-mvp | Nov 30, 2025 | Week 4 - Ethereum Integration |
| 4.0.0-production | Nov 30, 2025 | Phase 2 - Production Infrastructure |
| **5.0.0-relay** | **Dec 1, 2025** | **Phase 3.1 - Relay Server ✅** |
| 6.0.0-truempc | Planned | Phase 3 Complete - True MPC |

---

## Next Steps (Phase 3.2 - Android)

1. **Gomobile PoC** - Verify tss-lib works with Gomobile
2. **Android TSS Engine** - Package as .aar library
3. **E2E Encryption** - X25519 + ChaCha20-Poly1305
4. **WebSocket Client** - Connect to relay
5. **Android Keystore** - Hardware-backed storage
6. **Local KeyGen/Signing** - All computation on device

---

**End of Report**

*Generated: December 1, 2025*
*Version: 5.0.0-Relay (Phase 3.1 - Relay Server Complete)*
