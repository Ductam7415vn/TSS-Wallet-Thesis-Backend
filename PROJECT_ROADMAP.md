# TSS Wallet - Project Roadmap

**Last Updated:** November 30, 2025
**Current Phase:** MVP Complete

---

## Project Overview

TSS Wallet is a threshold signature wallet system consisting of:
- **Backend:** Go-based TSS server (this repository)
- **Android Client:** Kotlin/Jetpack Compose mobile app

---

## Phase 1: MVP (Weeks 1-4) - COMPLETE

### Week 1: KeyGen Protocol
| Task | Status | Description |
|------|--------|-------------|
| TSS Library Integration | Done | bnb-chain/tss-lib v2 |
| KeyGen Service | Done | 2-of-3 threshold key generation |
| REST API | Done | `/keygen/start`, `/keygen/:id/status`, `/keygen/:id/share/:partyId` |
| Ethereum Address Derivation | Done | Keccak256 from public key |

### Week 2-3: Signing Protocol
| Task | Status | Description |
|------|--------|-------------|
| Signing Service | Done | Threshold signing with key shares |
| REST API | Done | `/signing/start`, `/signing/:id/status`, `/signing/:id/signature` |
| Signature Verification | Done | `/signing/verify` endpoint |
| Ethereum Format | Done | R, S, V components (65 bytes) |

### Week 4: Ethereum Integration
| Task | Status | Description |
|------|--------|-------------|
| Multi-network Support | Done | Sepolia, Goerli, Mainnet |
| Transaction Building | Done | EIP-155, gas estimation |
| TSS Signature Application | Done | Apply R,S,V to transactions |
| Broadcasting | Done | Send to Ethereum networks |
| Query APIs | Done | Balance, nonce, gas price, receipts |
| Quick Send | Done | All-in-one endpoint |

**MVP Deliverables:**
- 17 REST API endpoints
- Full TSS key generation and signing
- Ethereum transaction support
- Ready for Android integration

---

## Phase 2: Production Readiness (Planned)

### 2.1 Database Persistence
| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| PostgreSQL Setup | Pending | High | Replace in-memory storage |
| Session Storage | Pending | High | Persist KeyGen/Signing sessions |
| Key Share Encryption | Pending | High | Encrypt shares at rest |
| Migration Scripts | Pending | Medium | Database schema versioning |

### 2.2 Security Enhancements
| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| API Authentication | Pending | High | JWT or API keys |
| Rate Limiting | Pending | High | Prevent abuse |
| HTTPS/TLS | Pending | High | Secure transport |
| Input Validation | Pending | Medium | Strengthen request validation |
| Audit Logging | Pending | Medium | Track all operations |

### 2.3 DevOps & Deployment
| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| Docker Container | Pending | High | Containerize backend |
| Docker Compose | Pending | Medium | Backend + PostgreSQL |
| CI/CD Pipeline | Pending | Medium | GitHub Actions |
| Health Monitoring | Pending | Medium | Prometheus/Grafana |
| Cloud Deployment | Pending | Low | AWS/GCP/Azure |

---

## Phase 3: Feature Extensions (Future)

### 3.1 Multi-Chain Support
| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| Bitcoin Integration | Pending | Medium | UTXO management, P2WPKH |
| ERC-20 Tokens | Pending | Medium | Token transfers |
| Other EVM Chains | Pending | Low | Polygon, BSC, Arbitrum |

### 3.2 Advanced Features
| Task | Status | Priority | Description |
|------|--------|----------|-------------|
| WebSocket Support | Pending | Medium | Real-time updates |
| Transaction History | Pending | Medium | Track past transactions |
| Key Refresh | Pending | Low | Rotate key shares |
| Recovery Protocol | Pending | Low | Recover from lost shares |

---

## Integration Checklist

### Backend + Android Integration Testing

```
Pre-requisites:
[ ] Backend running on accessible IP (not localhost)
[ ] Android emulator or device on same network
[ ] Sepolia testnet ETH in wallet

Test Flow:
[ ] 1. Health check from Android
[ ] 2. Create wallet (KeyGen)
[ ] 3. Store key shares on device
[ ] 4. Check balance
[ ] 5. Build transaction
[ ] 6. Sign with TSS
[ ] 7. Broadcast transaction
[ ] 8. Verify on Sepolia explorer

Quick Send Test:
[ ] 1. POST /ethereum/quick-send from Android
[ ] 2. Verify transaction on explorer
```

### Test Commands

```bash
# Start backend
cd tss-wallet-backend
./tss-backend

# Health check
curl http://localhost:8080/health

# Create wallet
curl -X POST http://localhost:8080/keygen/start \
  -H "Content-Type: application/json" \
  -d '{"threshold": 2, "parties": 3, "partyIds": ["device1", "device2", "device3"]}'

# Quick send (after keygen)
curl -X POST http://localhost:8080/ethereum/quick-send \
  -H "Content-Type: application/json" \
  -d '{
    "network": "sepolia",
    "keygenSessionId": "SESSION_ID",
    "signerIds": ["device1", "device2"],
    "to": "0xRECIPIENT",
    "valueEth": "0.001"
  }'
```

---

## Architecture Decision: Android Integration

**Decision:** Option 1 - Android Direct (Recommended)

```
┌─────────────┐     TSS Only      ┌─────────────┐
│   Android   │◄────────────────►│   Backend   │
│    Client   │  KeyGen, Signing  │   (Go)      │
└──────┬──────┘                   └─────────────┘
       │
       │ Direct RPC
       ▼
┌─────────────┐
│  Ethereum   │
│   Network   │
└─────────────┘
```

**Rationale:**
- Simpler architecture
- Fewer network hops
- Better performance
- Android handles Ethereum queries directly
- Backend focuses on TSS operations

---

## File Structure

```
tss-wallet-backend/
├── main.go                 # Entry point
├── keygen.go               # KeyGen HTTP handlers
├── keygen_service.go       # KeyGen TSS logic
├── signing.go              # Signing HTTP handlers
├── signing_service.go      # Signing TSS logic
├── ethereum.go             # Ethereum HTTP handlers
├── ethereum_service.go     # Ethereum operations
├── go.mod                  # Dependencies
├── go.sum                  # Checksums
├── CLAUDE.md               # Development docs
├── BACKEND_REPORT.md       # Client report
├── PROJECT_ROADMAP.md      # This file
└── tss-backend             # Compiled binary
```

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0-mvp | Nov 2025 | Week 1 - KeyGen complete |
| 2.0.0-mvp | Nov 2025 | Week 2-3 - Signing complete |
| 3.0.0-mvp | Nov 30, 2025 | Week 4 - Ethereum Integration complete |

---

## Contact & Resources

- **TSS Library:** [bnb-chain/tss-lib](https://github.com/bnb-chain/tss-lib)
- **Ethereum:** [go-ethereum](https://github.com/ethereum/go-ethereum)
- **Sepolia Faucet:** https://sepoliafaucet.com/

---

*End of Roadmap*
