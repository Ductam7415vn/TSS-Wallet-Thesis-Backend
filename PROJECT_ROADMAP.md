# TSS Wallet Backend - Project Roadmap

**Last Updated:** November 30, 2025
**Current Phase:** Phase 2 Complete - Preparing for True MPC Architecture
**Synced With:** Android Production Roadmap v1.0.0

---

## Project Vision

TSS Wallet is a threshold signature wallet system that will evolve from:
- **MVP (Current):** Server-Side Computation - Backend processes TSS
- **Production (Future):** Client-Side MPC - Android devices compute, Backend only relays

### Why This Matters

| Aspect | MVP (Current) | Production (Target) |
|--------|---------------|---------------------|
| Security Model | Trust the Server | Zero Trust / Zero Knowledge |
| Key Exposure | Server sees all shares | Server sees nothing |
| Compliance | Demo only | SOC2, PCI-DSS ready |
| User Trust | "Trust us" | "Don't trust, verify" |

---

## Architecture Evolution

### Current Architecture (MVP - Server-Side)

```
┌─────────────────────────────────────────────────────────────┐
│                    MVP ARCHITECTURE                          │
│                  (Server-Side Computation)                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌──────────┐         HTTP/REST         ┌──────────────┐  │
│   │ Android  │◄─────────────────────────►│   Backend    │  │
│   │  Client  │  Send shares to server    │   (Go)       │  │
│   └──────────┘                           └──────┬───────┘  │
│                                                  │          │
│                                          TSS Computation    │
│                                                  │          │
│                                          ┌──────▼───────┐  │
│                                          │  tss-lib     │  │
│                                          │  KeyGen      │  │
│                                          │  Signing     │  │
│                                          └──────────────┘  │
│                                                             │
│   Security: Server knows all key shares (NOT IDEAL)        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Target Architecture (Production - True MPC)

```
┌─────────────────────────────────────────────────────────────┐
│                 PRODUCTION ARCHITECTURE                      │
│                (Client-Side MPC / True MPC)                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌──────────┐                           ┌──────────┐      │
│   │ Device 1 │◄─────────────────────────►│ Device 2 │      │
│   │ (Phone)  │    Encrypted Messages     │ (Tablet) │      │
│   │          │    (E2E: X25519 +         │          │      │
│   │  TSS     │     ChaCha20-Poly1305)    │  TSS     │      │
│   │  Engine  │                           │  Engine  │      │
│   └────┬─────┘           ▲               └────┬─────┘      │
│        │                 │                    │             │
│        │    ┌────────────┴────────────┐      │             │
│        │    │                         │      │             │
│        ▼    ▼                         ▼      ▼             │
│   ┌─────────────────────────────────────────────────┐      │
│   │              RELAY SERVER (Dumb Pipe)           │      │
│   │  • Routes encrypted messages between devices    │      │
│   │  • Cannot decrypt or read message content       │      │
│   │  • Zero Knowledge of key shares                 │      │
│   │  • WebSocket for real-time delivery             │      │
│   └─────────────────────────────────────────────────┘      │
│                                                             │
│   Security: Server knows NOTHING (IDEAL)                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## MVP to Production Gap Analysis

| Category | Criteria | MVP Status | Production Target | Change Effort |
|----------|----------|------------|-------------------|---------------|
| **Architecture** | | | | |
| | TSS Computation Location | Server | Client (Android) | 🔴 Major |
| | Communication Pattern | HTTP Polling | WebSocket + Push | 🟡 Medium |
| | State Management | Server RAM | Client + Persistent | 🟡 Medium |
| **Security** | | | | |
| | Key Share Storage | Server Memory | Android Keystore | 🔴 Major |
| | Message Encryption | TLS only | E2E + TLS | 🟡 Medium |
| | Authentication | Basic JWT | JWT + Device Attestation | 🟡 Medium |
| **Infrastructure** | | | | |
| | Backend Role | Compute + Store | Relay only | 🔴 Major |
| | Database | PostgreSQL | PostgreSQL + Redis | 🟡 Medium |
| | Scalability | Single server | Horizontal scaling | 🟢 Minor |

### Migration Advantage: Gomobile

Since we chose **Go + bnb-chain/tss-lib**, we can use **Gomobile** to:
- Package TSS logic as Android library (.aar)
- Reuse 70% of existing Go code
- No need to rewrite crypto in Java/Kotlin

---

## Completed Phases

### Phase 1: MVP (Weeks 1-4) - COMPLETE ✅

#### Week 1: KeyGen Protocol
| Task | Status | Description |
|------|--------|-------------|
| TSS Library Integration | ✅ Done | bnb-chain/tss-lib v2 |
| KeyGen Service | ✅ Done | 2-of-3 threshold key generation |
| REST API | ✅ Done | `/keygen/start`, `/keygen/:id/status`, `/keygen/:id/share/:partyId` |
| Ethereum Address Derivation | ✅ Done | Keccak256 from public key |

#### Week 2-3: Signing Protocol
| Task | Status | Description |
|------|--------|-------------|
| Signing Service | ✅ Done | Threshold signing with key shares |
| REST API | ✅ Done | `/signing/start`, `/signing/:id/status`, `/signing/:id/signature` |
| Signature Verification | ✅ Done | `/signing/verify` endpoint |
| Ethereum Format | ✅ Done | R, S, V components (65 bytes) |

#### Week 4: Ethereum Integration
| Task | Status | Description |
|------|--------|-------------|
| Multi-network Support | ✅ Done | Sepolia, Goerli, Mainnet |
| Transaction Building | ✅ Done | EIP-155, gas estimation |
| TSS Signature Application | ✅ Done | Apply R,S,V to transactions |
| Broadcasting | ✅ Done | Send to Ethereum networks |
| Query APIs | ✅ Done | Balance, nonce, gas price, receipts |
| Quick Send | ✅ Done | All-in-one endpoint |

**MVP Deliverables:**
- 17 REST API endpoints
- Full TSS key generation and signing
- Ethereum transaction support
- Ready for Android integration

---

### Phase 2: Production Infrastructure - COMPLETE ✅

#### 2.1 Database Persistence
| Task | Status | Description |
|------|--------|-------------|
| PostgreSQL Setup | ✅ Done | Replace in-memory storage |
| Session Storage | ✅ Done | Persist KeyGen/Signing sessions |
| Repository Pattern | ✅ Done | Clean data access layer |
| Auto-Migration | ✅ Done | Schema auto-creates on startup |

#### 2.2 Security Enhancements
| Task | Status | Description |
|------|--------|-------------|
| API Key Authentication | ✅ Done | SHA-256 hashed storage |
| JWT Authentication | ✅ Done | Configurable expiry |
| Rate Limiting | ✅ Done | Token bucket algorithm |
| Audit Logging | ✅ Done | All operations tracked |

#### 2.3 DevOps & Deployment
| Task | Status | Description |
|------|--------|-------------|
| Dockerfile | ✅ Done | Multi-stage build |
| Docker Compose | ✅ Done | Backend + PostgreSQL |
| Health Checks | ✅ Done | Container health monitoring |
| Environment Config | ✅ Done | .env based configuration |

---

## Phase 3: True MPC Architecture - PLANNED 🚧

### Timeline Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    MIGRATION TIMELINE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Phase 0        Phase 1         Phase 2         Phase 3         │
│  ────────       ────────        ────────        ────────        │
│  Preparation    Backend         Android         Security        │
│  (Gomobile PoC) Refactor        TSS Engine      Hardening       │
│                                                                  │
│  ▓▓▓▓░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
│  Week 1-2       Week 3-4        Week 5-8        Week 9-10       │
│                                                                  │
│  Backend:       Backend:        Android:        Both:           │
│  - PoC only     - Relay API     - Gomobile      - HTTPS/TLS     │
│                 - WebSocket     - TSS Engine    - Cert Pinning  │
│                 - Remove TSS    - E2E Encrypt   - Pen Testing   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

### Phase 3.0: Preparation (Week 1-2) - Backend Tasks

| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| Gomobile PoC | ⬜ Pending | High | Verify tss-lib works with Gomobile |
| E2E Encryption Design | ⬜ Pending | High | X25519 + ChaCha20-Poly1305 spec |
| Relay API Design | ⬜ Pending | High | Finalize API contracts |
| Database Schema Update | ⬜ Pending | Medium | Add relay_messages, device_sessions tables |

---

### Phase 3.1: Backend "Dumb Pipe" Relay Server (Week 3-4)

#### Objective
Transform backend from "Smart Compute Engine" to "Dumb Message Relay"

```
CURRENT BACKEND                    TARGET BACKEND
───────────────                    ──────────────

┌─────────────────────┐            ┌─────────────────────┐
│   TSS Engine        │            │   Message Router    │
│   ───────────       │            │   ──────────────    │
│   - KeyGen Logic    │  REMOVE    │                     │
│   - Signing Logic   │ ────────►  │   (No crypto logic) │
│   - Share Compute   │            │                     │
└─────────────────────┘            └─────────────────────┘

┌─────────────────────┐            ┌─────────────────────┐
│   PostgreSQL Only   │            │   PostgreSQL+Redis  │
│   ───────────────   │    ADD     │   ─────────────     │
│   - Sessions        │ ────────►  │   - Message Queue   │
│   - Audit Logs      │            │   - Pub/Sub         │
└─────────────────────┘            └─────────────────────┘

┌─────────────────────┐            ┌─────────────────────┐
│   HTTP Only         │            │   HTTP + WebSocket  │
│   ──────────────    │    ADD     │   ────────────────  │
│   - REST endpoints  │ ────────►  │   - Real-time push  │
│                     │            │   - REST for compat │
└─────────────────────┘            └─────────────────────┘
```

#### New Relay API Design

```go
// ============================================================
// RELAY API - No crypto operations, just message routing
// ============================================================

// POST /relay/register
// Register device for receiving messages
type RegisterRequest struct {
    DeviceID    string `json:"deviceId"`
    PublicKey   string `json:"publicKey"`   // X25519 public key for E2E
    PushToken   string `json:"pushToken"`   // FCM token (optional)
}

type RegisterResponse struct {
    Success   bool   `json:"success"`
    DeviceID  string `json:"deviceId"`
    ExpiresAt int64  `json:"expiresAt"`
}

// POST /relay/session
// Create a new MPC session (coordination only, no computation)
type CreateSessionRequest struct {
    SessionID   string   `json:"sessionId"`
    Parties     []string `json:"parties"`    // Device IDs
    Type        string   `json:"type"`       // "keygen" or "signing"
    Threshold   int      `json:"threshold"`
}

type CreateSessionResponse struct {
    SessionID string `json:"sessionId"`
    Status    string `json:"status"`
}

// POST /relay/send
// Send encrypted message to recipient(s)
type SendMessageRequest struct {
    SessionID   string   `json:"sessionId"`
    From        string   `json:"from"`
    To          []string `json:"to"`         // Recipient device IDs
    Payload     string   `json:"payload"`    // E2E encrypted, base64
    Round       int      `json:"round"`      // Protocol round number
    IsBroadcast bool     `json:"isBroadcast"`
}

type SendMessageResponse struct {
    MessageID string `json:"messageId"`
    Status    string `json:"status"`
}

// GET /relay/messages?deviceId=xxx&sessionId=xxx
// Poll for messages (fallback if WebSocket unavailable)
type MessagesResponse struct {
    Messages []RelayMessage `json:"messages"`
}

type RelayMessage struct {
    MessageID string `json:"messageId"`
    SessionID string `json:"sessionId"`
    From      string `json:"from"`
    Payload   string `json:"payload"`  // E2E encrypted
    Round     int    `json:"round"`
    Timestamp int64  `json:"timestamp"`
}

// DELETE /relay/session/:id
// Clean up session after completion
type DeleteSessionResponse struct {
    Success bool `json:"success"`
}
```

#### WebSocket Protocol

```go
// WebSocket /relay/ws?deviceId=xxx&token=xxx

// Client → Server: Authentication
type WSAuthMessage struct {
    Type      string `json:"type"`      // "auth"
    DeviceID  string `json:"deviceId"`
    AuthToken string `json:"authToken"` // JWT
}

// Server → Client: Auth Response
type WSAuthResponse struct {
    Type    string `json:"type"`    // "auth_response"
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}

// Client → Server: Subscribe to session
type WSSubscribe struct {
    Type      string `json:"type"`      // "subscribe"
    SessionID string `json:"sessionId"`
}

// Server → Client: New message notification
type WSMessageNotification struct {
    Type      string `json:"type"`      // "message"
    SessionID string `json:"sessionId"`
    From      string `json:"from"`
    Payload   string `json:"payload"`   // E2E encrypted
    Round     int    `json:"round"`
    Timestamp int64  `json:"timestamp"`
}

// Client → Server: Acknowledge receipt
type WSAck struct {
    Type      string `json:"type"`      // "ack"
    MessageID string `json:"messageId"`
}

// Server → Client: Session event
type WSSessionEvent struct {
    Type      string `json:"type"`      // "session_event"
    SessionID string `json:"sessionId"`
    Event     string `json:"event"`     // "party_joined", "party_left", "completed", "failed"
    PartyID   string `json:"partyId,omitempty"`
}

// Heartbeat (both directions)
type WSPing struct {
    Type string `json:"type"` // "ping"
}

type WSPong struct {
    Type string `json:"type"` // "pong"
}
```

#### Backend Tasks (Week 3-4)

| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| Redis Integration | ⬜ Pending | Critical | Add Redis for pub/sub message routing |
| WebSocket Server | ⬜ Pending | Critical | gorilla/websocket implementation |
| POST /relay/register | ⬜ Pending | Critical | Device registration endpoint |
| POST /relay/session | ⬜ Pending | Critical | Session creation endpoint |
| POST /relay/send | ⬜ Pending | Critical | Message sending endpoint |
| GET /relay/messages | ⬜ Pending | High | Polling fallback endpoint |
| WS /relay/ws | ⬜ Pending | High | WebSocket handler |
| Archive TSS Logic | ⬜ Pending | Medium | Move keygen_service.go, signing_service.go to archive/ |
| Backward Compat Layer | ⬜ Pending | Medium | Keep old API with feature flag |
| Update Docker Compose | ⬜ Pending | Medium | Add Redis service |

#### New Database Schema

```sql
-- Device registration for relay
CREATE TABLE relay_devices (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(64) UNIQUE NOT NULL,
    public_key TEXT NOT NULL,           -- X25519 public key
    push_token TEXT,                    -- FCM token (optional)
    is_online BOOLEAN DEFAULT false,
    last_seen_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- MPC sessions for coordination
CREATE TABLE relay_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) UNIQUE NOT NULL,
    session_type VARCHAR(20) NOT NULL,  -- 'keygen' or 'signing'
    parties TEXT[] NOT NULL,            -- Device IDs
    threshold INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,        -- 'waiting', 'in_progress', 'completed', 'failed'
    current_round INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Message queue for relay
CREATE TABLE relay_messages (
    id SERIAL PRIMARY KEY,
    message_id VARCHAR(64) UNIQUE NOT NULL,
    session_id VARCHAR(64) NOT NULL,
    from_device VARCHAR(64) NOT NULL,
    to_device VARCHAR(64) NOT NULL,
    payload TEXT NOT NULL,              -- E2E encrypted, base64
    round INTEGER NOT NULL,
    is_delivered BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    delivered_at TIMESTAMP,

    INDEX idx_relay_messages_to_device (to_device, is_delivered),
    INDEX idx_relay_messages_session (session_id, round)
);
```

---

### Phase 3.2: Android TSS Engine (Week 5-8)

**Note:** This is primarily Android work. Backend supports via relay API.

| Task | Owner | Description |
|------|-------|-------------|
| Gomobile Integration | Android | Package tss-lib as .aar |
| Local KeyGen | Android | Run KeyGen on device |
| Local Signing | Android | Run Signing on device |
| E2E Encryption | Android | X25519 + ChaCha20-Poly1305 |
| WebSocket Client | Android | Connect to backend relay |
| Android Keystore | Android | Hardware-backed key storage |

---

### Phase 3.3: Security Hardening (Week 9-10)

| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| HTTPS/TLS | ⬜ Pending | Critical | SSL certificate, force HTTPS |
| Certificate Pinning Support | ⬜ Pending | High | Provide cert hash for Android |
| Device Attestation | ⬜ Pending | High | Verify SafetyNet/Play Integrity |
| Enhanced JWT | ⬜ Pending | Medium | Refresh token rotation |
| Rate Limiting Update | ⬜ Pending | Medium | Per-device limits for relay |
| Security Audit | ⬜ Pending | High | Code review, pen testing |
| Monitoring | ⬜ Pending | Medium | Prometheus metrics, alerting |

---

## Phase 4: Feature Extensions - FUTURE 🚀

### 4.1 Multi-Chain Support
| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| Bitcoin Integration | ⬜ Pending | Medium | UTXO management, P2WPKH |
| ERC-20 Tokens | ⬜ Pending | Medium | Token transfers |
| Other EVM Chains | ⬜ Pending | Low | Polygon, BSC, Arbitrum |

### 4.2 Advanced Features
| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| Transaction History | ⬜ Pending | Medium | Track past transactions |
| Key Refresh | ⬜ Pending | Low | Rotate key shares periodically |
| Recovery Protocol | ⬜ Pending | Low | Recover from lost device |
| Social Recovery | ⬜ Pending | Low | Trusted contacts help recover |

### 4.3 Enterprise Features
| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| Multi-tenant | ⬜ Pending | Low | Support multiple organizations |
| Approval Workflows | ⬜ Pending | Low | Multi-level transaction approval |
| Compliance Reporting | ⬜ Pending | Low | Audit trails for regulators |
| Kubernetes Deployment | ⬜ Pending | Low | Production orchestration |

---

## File Structure

### Current (Phase 2)

```
tss-wallet-backend/
├── main.go                 # Entry point
├── keygen.go               # KeyGen HTTP handlers
├── keygen_service.go       # KeyGen TSS logic (WILL BE ARCHIVED)
├── signing.go              # Signing HTTP handlers
├── signing_service.go      # Signing TSS logic (WILL BE ARCHIVED)
├── ethereum.go             # Ethereum HTTP handlers
├── ethereum_service.go     # Ethereum operations
│
├── config/                 # Configuration
│   └── config.go
│
├── database/               # Database layer
│   ├── database.go
│   └── repository.go
│
├── middleware/             # Middleware
│   ├── auth.go
│   ├── ratelimit.go
│   └── audit.go
│
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── go.mod
├── go.sum
├── CLAUDE.md
├── BACKEND_REPORT.md
└── PROJECT_ROADMAP.md
```

### Target (Phase 3)

```
tss-wallet-backend/
├── main.go                 # Entry point (updated for relay)
│
├── relay/                  # NEW: Relay Server
│   ├── handler.go          # HTTP handlers for relay API
│   ├── websocket.go        # WebSocket server
│   ├── session.go          # Session coordination
│   ├── message.go          # Message routing logic
│   └── types.go            # Request/Response types
│
├── ethereum.go             # Ethereum HTTP handlers (KEEP)
├── ethereum_service.go     # Ethereum operations (KEEP)
│
├── archive/                # NEW: Archived MVP code
│   ├── keygen.go
│   ├── keygen_service.go
│   ├── signing.go
│   └── signing_service.go
│
├── config/
│   └── config.go           # Updated for Redis, WebSocket
│
├── database/
│   ├── database.go         # Updated with relay tables
│   ├── repository.go
│   └── relay_repository.go # NEW: Relay data access
│
├── middleware/
│   ├── auth.go             # Updated for device attestation
│   ├── ratelimit.go
│   └── audit.go
│
├── Dockerfile
├── docker-compose.yml      # Updated with Redis
├── .env.example
└── ...
```

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0-mvp | Nov 2025 | Week 1 - KeyGen complete |
| 2.0.0-mvp | Nov 2025 | Week 2-3 - Signing complete |
| 3.0.0-mvp | Nov 30, 2025 | Week 4 - Ethereum Integration complete |
| 4.0.0-production | Nov 30, 2025 | Phase 2 - Production Infrastructure complete |
| 5.0.0-relay | Planned | Phase 3.1 - Relay Server |
| 6.0.0-truempc | Planned | Phase 3 Complete - True MPC Architecture |

---

## Risk Assessment

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **WebSocket scaling** | Medium | High | Redis pub/sub, horizontal scaling |
| **Message delivery reliability** | Medium | High | ACK system, retry logic, polling fallback |
| **Redis single point of failure** | Low | High | Redis Sentinel/Cluster |
| **Backward compat breaks** | Medium | High | Feature flags, gradual rollout |

### Security Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Replay attacks** | Medium | Medium | Nonces, timestamps, session binding |
| **DoS on relay** | Medium | Medium | Rate limiting, DDoS protection |
| **Device impersonation** | Low | High | Device attestation, JWT |

---

## Success Metrics

### Phase 3.1 Complete When:
- [ ] Relay API fully functional (register, session, send, messages)
- [ ] WebSocket server operational with pub/sub
- [ ] Messages persist in database
- [ ] Old API still works (backward compat)
- [ ] Load test: 100 concurrent WebSocket connections

### Phase 3 Complete When:
- [ ] Android KeyGen works entirely on-device
- [ ] Android Signing works entirely on-device
- [ ] E2E encryption on all relay messages
- [ ] No key material ever sent to server
- [ ] Security audit passed

---

## Resources

- **TSS Library:** [bnb-chain/tss-lib](https://github.com/bnb-chain/tss-lib)
- **Gomobile:** [golang.org/x/mobile](https://pkg.go.dev/golang.org/x/mobile)
- **gorilla/websocket:** [github.com/gorilla/websocket](https://github.com/gorilla/websocket)
- **Redis Go Client:** [github.com/redis/go-redis](https://github.com/redis/go-redis)
- **Ethereum:** [go-ethereum](https://github.com/ethereum/go-ethereum)

---

## Academic Reference

For thesis/dissertation:

> *"The system architecture evolves from Server-Side Computation (MVP) to Client-Side Multi-Party Computation (Production). While the MVP demonstrates functional correctness with the server orchestrating TSS protocols, the production architecture implements true MPC where cryptographic computations occur on client devices, and the server acts only as an encrypted message relay with zero knowledge of key shares. End-to-end encryption using X25519 key exchange and ChaCha20-Poly1305 authenticated encryption ensures that even the relay server cannot read the MPC protocol messages. This architectural choice ensures that no single point of compromise can expose the complete private key, achieving the fundamental security guarantee of threshold cryptography."*

---

*End of Roadmap*
