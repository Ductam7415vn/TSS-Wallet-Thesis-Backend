# TSS Wallet Backend - Technical Report

**Project:** Threshold Signature Scheme (TSS) Wallet Backend
**Version:** 4.0.0-Production
**Date:** November 30, 2025
**Status:** Phase 2 - Production Readiness Complete ✅

---

## Executive Summary

This document provides a comprehensive technical report of the TSS Wallet Backend implementation. The backend serves as the core cryptographic engine for a distributed key generation and signing system, enabling secure multi-party computation for cryptocurrency wallet management.

### Key Achievements

#### Week 1 - KeyGen ✅
- ✅ Fully functional 2-of-3 threshold key generation
- ✅ Secure key share distribution
- ✅ Ethereum address derivation
- ✅ RESTful API ready for mobile integration

#### Week 2-3 - Signing ✅
- ✅ Threshold signing protocol (2-of-3)
- ✅ ECDSA signature generation
- ✅ Ethereum-compatible signatures (R, S, V)
- ✅ Signature verification endpoint

#### Week 4 - Ethereum Integration ✅
- ✅ Multi-network support (Sepolia, Goerli, Mainnet)
- ✅ Transaction building with automatic gas estimation
- ✅ TSS signature application to transactions
- ✅ Transaction broadcasting to Ethereum networks
- ✅ Balance, nonce, and gas price queries
- ✅ Quick-send endpoint for one-step transactions

#### Phase 2 - Production Readiness ✅ (NEW)
- ✅ PostgreSQL database persistence
- ✅ Session storage (KeyGen/Signing sessions)
- ✅ API authentication (JWT + API Keys)
- ✅ Rate limiting (token bucket algorithm)
- ✅ Audit logging
- ✅ Docker containerization
- ✅ Docker Compose (Backend + PostgreSQL)

---

## 1. Project Overview

### 1.1 What is TSS (Threshold Signature Scheme)?

TSS is a cryptographic protocol that allows multiple parties to jointly generate and use a private key without any single party ever having access to the complete key. This provides:

- **Enhanced Security:** No single point of failure
- **Distributed Trust:** Multiple devices must cooperate to sign transactions
- **Recovery Options:** Lost devices can be replaced without compromising security

### 1.2 Use Case

```
┌─────────────────────────────────────────────────────────────┐
│                    TSS Wallet System                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   User has 3 devices:                                       │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│   │ Phone 1  │  │ Phone 2  │  │ Tablet   │                │
│   │ (Share1) │  │ (Share2) │  │ (Share3) │                │
│   └──────────┘  └──────────┘  └──────────┘                │
│                                                             │
│   To sign a transaction:                                    │
│   - Need ANY 2 of 3 devices                                │
│   - No single device can sign alone                        │
│   - If 1 device lost → still can sign with other 2         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Complete Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    COMPLETE TSS FLOW                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  WEEK 1: KEY GENERATION                                     │
│  ─────────────────────                                      │
│  1. User requests new wallet                                │
│  2. Backend creates 3 key shares                            │
│  3. Each device receives its share                          │
│  4. Public key & Ethereum address generated                 │
│                                                             │
│  WEEK 2-3: SIGNING                                          │
│  ─────────────────────                                      │
│  1. User wants to sign transaction                          │
│  2. 2 devices participate (threshold = 2)                   │
│  3. Backend orchestrates signing protocol                   │
│  4. Valid ECDSA signature produced                          │
│  5. Signature can be used on Ethereum/Bitcoin               │
│                                                             │
│  WEEK 4: ETHEREUM INTEGRATION ✅                            │
│  ─────────────────────                                      │
│  1. Build Ethereum transaction                              │
│  2. Sign transaction hash with TSS                          │
│  3. Apply signature to transaction                          │
│  4. Broadcast to Ethereum network (Sepolia/Mainnet)         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Technical Architecture

### 2.1 System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CLIENT LAYER                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │  Android    │  │    iOS      │  │    Web      │        │
│  │    App      │  │    App      │  │    App      │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
└─────────┼────────────────┼────────────────┼────────────────┘
          │                │                │
          └────────────────┼────────────────┘
                           │ HTTP/REST
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                     API LAYER (Gin)                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  main.go - Router, CORS, Health Check               │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  keygen.go - KeyGen HTTP Handlers (Week 1)          │   │
│  │  • POST /keygen/start                               │   │
│  │  • GET  /keygen/:sessionId/status                   │   │
│  │  • GET  /keygen/:sessionId/share/:partyId           │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  signing.go - Signing HTTP Handlers (Week 2-3)      │   │
│  │  • POST /signing/start                              │   │
│  │  • GET  /signing/:signingId/status                  │   │
│  │  • GET  /signing/:signingId/signature               │   │
│  │  • POST /signing/verify                             │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  ethereum.go - Ethereum HTTP Handlers (Week 4) NEW  │   │
│  │  • GET  /ethereum/networks                          │   │
│  │  • GET  /ethereum/:network/balance/:address         │   │
│  │  • GET  /ethereum/:network/gas-price                │   │
│  │  • POST /ethereum/build-tx                          │   │
│  │  • POST /ethereum/sign-tx                           │   │
│  │  • POST /ethereum/broadcast                         │   │
│  │  • POST /ethereum/quick-send                        │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   SERVICE LAYER                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  keygen_service.go - TSS KeyGen Orchestration       │   │
│  │  • Session Management                               │   │
│  │  • Party Initialization                             │   │
│  │  • Message Routing                                  │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  signing_service.go - TSS Signing Orchestration     │   │
│  │  • Load Key Shares from KeyGen                      │   │
│  │  • Threshold Signing Protocol                       │   │
│  │  • Signature Verification                           │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  ethereum_service.go - Ethereum Operations (Week 4) │   │
│  │  • Network connections (RPC)                        │   │
│  │  • Transaction building (EIP-155)                   │   │
│  │  • Signature application                            │   │
│  │  • Broadcasting to networks                         │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                 CRYPTOGRAPHY LAYER                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  bnb-chain/tss-lib v2                               │   │
│  │  • ECDSA Key Generation                             │   │
│  │  • ECDSA Threshold Signing                          │   │
│  │  • secp256k1 Curve                                  │   │
│  │  • Threshold Cryptography                           │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  go-ethereum v1.13.5                                │   │
│  │  • Ethereum RPC Client                              │   │
│  │  • Transaction Types (EIP-155)                      │   │
│  │  • RLP Encoding                                     │   │
│  │  • Crypto utilities                                 │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Technology Stack

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| Language | Go | 1.21+ | Backend development |
| Web Framework | Gin | 1.9.1 | HTTP routing, middleware |
| TSS Library | bnb-chain/tss-lib | v2.0.2 | Threshold cryptography |
| Ethereum | go-ethereum | v1.13.5 | Ethereum integration |
| Database | PostgreSQL | 15+ | Persistent storage |
| Auth | golang-jwt/jwt | v5.2.0 | JWT authentication |
| Crypto | golang.org/x/crypto | 0.13.0 | Keccak256 (Ethereum) |
| Curve | secp256k1 | - | Bitcoin/Ethereum compatible |
| Container | Docker | - | Containerization |

### 2.3 File Structure

```
tss-wallet-backend/
├── main.go                 # Server entry point
│   ├── Router setup
│   ├── CORS middleware
│   └── Health check endpoint
│
├── keygen.go               # KeyGen HTTP handlers (Week 1)
│   ├── StartKeyGen()
│   ├── GetSessionStatus()
│   ├── GetKeyShare()
│   └── deriveEthAddress()
│
├── keygen_service.go       # Core TSS KeyGen logic (Week 1)
│   ├── KeyGenService
│   ├── KeyGenSession
│   ├── CreateAndInitSession()
│   ├── ExecuteKeyGen()
│   └── routeMessages()
│
├── signing.go              # Signing HTTP handlers (Week 2-3)
│   ├── StartSigning()
│   ├── GetSigningStatus()
│   ├── GetSignature()
│   └── VerifySignature()
│
├── signing_service.go      # Core TSS Signing logic (Week 2-3)
│   ├── SigningService
│   ├── SigningSession
│   ├── CreateAndExecuteSigning()
│   ├── routeSigningMessages()
│   └── VerifySignature()
│
├── ethereum.go             # Ethereum HTTP handlers (Week 4)
│   ├── GetNetworks()
│   ├── GetBalance()
│   ├── BuildTransaction()
│   ├── SignTransaction()
│   ├── BroadcastTransaction()
│   └── QuickSend()
│
├── ethereum_service.go     # Ethereum service logic (Week 4)
│   ├── EthereumService
│   ├── BuildUnsignedTransaction()
│   ├── ApplySignature()
│   └── BroadcastTransaction()
│
├── config/                 # Configuration (Phase 2) ← NEW
│   └── config.go           # Environment-based configuration
│       ├── Config struct
│       ├── LoadConfig()
│       └── GetDSN()
│
├── database/               # Database layer (Phase 2) ← NEW
│   ├── database.go         # PostgreSQL connection & migrations
│   │   ├── InitDB()
│   │   ├── RunMigrations()
│   │   └── HealthCheck()
│   └── repository.go       # Data access layer
│       ├── SaveKeyGenSession()
│       ├── GetKeyGenSession()
│       ├── SaveSigningSession()
│       ├── SaveAPIKey()
│       └── SaveAuditLog()
│
├── middleware/             # Middleware (Phase 2) ← NEW
│   ├── auth.go             # Authentication
│   │   ├── AuthMiddleware()
│   │   ├── validateAPIKey()
│   │   ├── validateJWT()
│   │   └── GenerateJWT()
│   ├── ratelimit.go        # Rate limiting
│   │   ├── RateLimiter
│   │   ├── RateLimitMiddleware()
│   │   └── KeyGenRateLimitMiddleware()
│   └── audit.go            # Audit logging
│       ├── AuditMiddleware()
│       └── sanitizeRequestBody()
│
├── Dockerfile              # Multi-stage Docker build (Phase 2) ← NEW
├── docker-compose.yml      # Backend + PostgreSQL (Phase 2) ← NEW
├── .env.example            # Environment template (Phase 2) ← NEW
│
├── go.mod                  # Dependencies
├── go.sum                  # Checksums
├── CLAUDE.md               # Technical documentation
├── BACKEND_REPORT.md       # This report
├── PROJECT_ROADMAP.md      # Project roadmap
└── tss-backend             # Compiled binary (~25MB)
```

---

## 3. API Documentation

### 3.1 Base URL

```
Development: http://localhost:8080
Production:  https://api.tsswallet.com (future)
```

### 3.2 Endpoints Overview

| # | Method | Endpoint | Description | Week |
|---|--------|----------|-------------|------|
| 1 | GET | `/health` | Health check | 1 |
| 2 | POST | `/keygen/start` | Start KeyGen session | 1 |
| 3 | GET | `/keygen/:sessionId/status` | Get KeyGen status | 1 |
| 4 | GET | `/keygen/:sessionId/share/:partyId` | Get key share | 1 |
| 5 | POST | `/signing/start` | Start signing session | 2-3 |
| 6 | GET | `/signing/:signingId/status` | Get signing status | 2-3 |
| 7 | GET | `/signing/:signingId/signature` | Get signature | 2-3 |
| 8 | POST | `/signing/verify` | Verify signature | 2-3 |
| 9 | GET | `/ethereum/networks` | List supported networks | 4 |
| 10 | GET | `/ethereum/:network/balance/:address` | Get ETH balance | 4 |
| 11 | GET | `/ethereum/:network/nonce/:address` | Get account nonce | 4 |
| 12 | GET | `/ethereum/:network/gas-price` | Get gas price | 4 |
| 13 | POST | `/ethereum/build-tx` | Build unsigned transaction | 4 |
| 14 | POST | `/ethereum/sign-tx` | Sign with TSS | 4 |
| 15 | POST | `/ethereum/broadcast` | Broadcast to network | 4 |
| 16 | POST | `/ethereum/quick-send` | One-step send | 4 |
| 17 | GET | `/ethereum/:network/tx/:txHash` | Get tx receipt | 4 |

---

### 3.3 Week 1 Endpoints (KeyGen)

#### 3.3.1 Health Check

```http
GET /health
```

**Response:**
```json
{
    "status": "ok",
    "service": "tss-wallet-backend",
    "version": "3.0.0-mvp"
}
```

#### 3.3.2 Start KeyGen

```http
POST /keygen/start
Content-Type: application/json
```

**Request Body:**
```json
{
    "threshold": 2,
    "parties": 3,
    "partyIds": ["device1", "device2", "device3"]
}
```

**Response (200 OK):**
```json
{
    "sessionId": "session_1732869123456789",
    "threshold": 2,
    "parties": 3,
    "partyIds": ["device1", "device2", "device3"],
    "status": "completed",
    "completedParties": 3,
    "createdAt": "2025-11-29T15:30:00Z",
    "message": "KeyGen completed successfully..."
}
```

**Timing:** ~2-3 minutes

#### 3.3.3 Get KeyGen Status

```http
GET /keygen/:sessionId/status
```

**Response:**
```json
{
    "sessionId": "session_1732869123456789",
    "status": "completed",
    "threshold": 2,
    "parties": 3,
    "partyIds": ["device1", "device2", "device3"],
    "completedParties": 3,
    "createdAt": "2025-11-29T15:30:00Z"
}
```

#### 3.3.4 Get Key Share

```http
GET /keygen/:sessionId/share/:partyId
```

**Response:**
```json
{
    "partyId": "device1",
    "partyIndex": 0,
    "publicKey": "04a1b2c3d4e5f6...65bytes...hex",
    "ethAddress": "0x742d35Cc6634C0532925a3b844Bc9e7595f8dE12",
    "shareData": "base64EncodedShareData...",
    "status": "ready"
}
```

---

### 3.4 Week 2-3 Endpoints (Signing) ← NEW

#### 3.4.1 Start Signing

```http
POST /signing/start
Content-Type: application/json
```

**Request Body:**
```json
{
    "keygenSessionId": "session_1732869123456789",
    "message": "a1b2c3d4e5f6...32bytes...hex",
    "signerIds": ["device1", "device2"]
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| keygenSessionId | string | Yes | Session ID from completed KeyGen |
| message | string | Yes | 32-byte hash to sign (hex encoded) |
| signerIds | string[] | Yes | Parties participating (min: threshold) |

**Response (200 OK):**
```json
{
    "signingId": "signing_1732869200000000",
    "keygenSessionId": "session_1732869123456789",
    "signerIds": ["device1", "device2"],
    "status": "completed",
    "createdAt": "2025-11-29T16:00:00Z",
    "message": "Signing completed successfully..."
}
```

**Timing:** ~30-60 seconds

**Error Responses:**

| HTTP Code | Error | Cause |
|-----------|-------|-------|
| 400 | message must be hex-encoded | Invalid hex string |
| 400 | message must be 32 bytes | Wrong message length |
| 400 | signerIds must have at least 2 parties | Not enough signers |
| 400 | keygen session not found | Invalid keygenSessionId |
| 400 | keygen session not completed | KeyGen not finished |
| 400 | signer not found in keygen session | Invalid signer ID |

#### 3.4.2 Get Signing Status

```http
GET /signing/:signingId/status
```

**Response:**
```json
{
    "signingId": "signing_1732869200000000",
    "keygenSessionId": "session_1732869123456789",
    "signerIds": ["device1", "device2"],
    "status": "completed",
    "completedSigners": 2,
    "createdAt": "2025-11-29T16:00:00Z"
}
```

**Status Values:**

| Status | Description |
|--------|-------------|
| initialized | Session created, not started |
| in_progress | Signing protocol running |
| completed | Signature generated successfully |
| failed | An error occurred |

#### 3.4.3 Get Signature

```http
GET /signing/:signingId/signature
```

**Response:**
```json
{
    "signingId": "signing_1732869200000000",
    "r": "a1b2c3d4...32bytes...hex",
    "s": "e5f6a7b8...32bytes...hex",
    "v": 27,
    "signature": "a1b2c3d4...65bytes...hex",
    "status": "ready"
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| signingId | string | Signing session ID |
| r | string | R component (32 bytes, hex) |
| s | string | S component (32 bytes, hex) |
| v | int | Recovery ID (27 or 28 for Ethereum) |
| signature | string | Full signature R \|\| S \|\| V (65 bytes, hex) |
| status | string | Always "ready" when available |

#### 3.4.4 Verify Signature

```http
POST /signing/verify
Content-Type: application/json
```

**Request Body:**
```json
{
    "keygenSessionId": "session_1732869123456789",
    "message": "a1b2c3d4...32bytes...hex",
    "signature": "a1b2c3d4...65bytes...hex"
}
```

**Response:**
```json
{
    "valid": true,
    "message": "a1b2c3d4...32bytes...hex"
}
```

---

### 3.5 Week 4 Endpoints (Ethereum) ← NEW

#### 3.5.1 Get Supported Networks

```http
GET /ethereum/networks
```

**Response:**
```json
{
    "networks": [
        {
            "name": "Sepolia Testnet",
            "network": "sepolia",
            "chainId": 11155111,
            "symbol": "ETH",
            "explorer": "https://sepolia.etherscan.io",
            "isTestnet": true
        },
        {
            "name": "Ethereum Mainnet",
            "network": "mainnet",
            "chainId": 1,
            "symbol": "ETH",
            "explorer": "https://etherscan.io",
            "isTestnet": false
        }
    ]
}
```

#### 3.5.2 Get Balance

```http
GET /ethereum/:network/balance/:address
```

**Example:**
```bash
GET /ethereum/sepolia/balance/0x742d35Cc6634C0532925a3b844Bc9e7595f8dE12
```

**Response:**
```json
{
    "network": "sepolia",
    "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f8dE12",
    "balanceWei": "1000000000000000000",
    "balanceEth": "1.000000000000000000",
    "symbol": "ETH"
}
```

#### 3.5.3 Get Gas Price

```http
GET /ethereum/:network/gas-price
```

**Response:**
```json
{
    "network": "sepolia",
    "gasPriceWei": "20000000000",
    "gasPriceGwei": "20.00"
}
```

#### 3.5.4 Build Transaction

```http
POST /ethereum/build-tx
Content-Type: application/json
```

**Request Body:**
```json
{
    "network": "sepolia",
    "keygenSessionId": "session_1732869123456789",
    "to": "0xRecipientAddress...",
    "valueEth": "0.1",
    "gasPriceGwei": 20
}
```

**Response:**
```json
{
    "txId": "tx_0",
    "network": "sepolia",
    "from": "0xYourAddress...",
    "to": "0xRecipientAddress...",
    "valueWei": "100000000000000000",
    "valueEth": "0.100000000000000000",
    "gasLimit": 21000,
    "gasPriceWei": "20000000000",
    "gasPriceGwei": "20.00",
    "nonce": 0,
    "chainId": 11155111,
    "hashToSign": "abc123...32bytes...hex",
    "message": "Use the hashToSign with /signing/start to get the signature"
}
```

#### 3.5.5 Sign Transaction with TSS

```http
POST /ethereum/sign-tx
Content-Type: application/json
```

**Request Body:**
```json
{
    "txId": "tx_0",
    "keygenSessionId": "session_1732869123456789",
    "signerIds": ["device1", "device2"]
}
```

**Response:**
```json
{
    "txId": "tx_0",
    "signingId": "ethsign_tx_0",
    "rawTx": "0xf86c...signed_transaction_hex...",
    "txHash": "0xTransactionHash...",
    "from": "0xYourAddress...",
    "to": "0xRecipientAddress...",
    "valueWei": "100000000000000000",
    "gasLimit": 21000,
    "gasPriceWei": "20000000000",
    "nonce": 0,
    "chainId": 11155111,
    "status": "signed",
    "message": "Transaction signed. Use /ethereum/broadcast to send to network."
}
```

#### 3.5.6 Broadcast Transaction

```http
POST /ethereum/broadcast
Content-Type: application/json
```

**Request Body:**
```json
{
    "network": "sepolia",
    "rawTx": "0xf86c...signed_transaction_hex..."
}
```

**Response:**
```json
{
    "network": "sepolia",
    "txHash": "0xTransactionHash...",
    "explorerUrl": "https://sepolia.etherscan.io/tx/0xTransactionHash...",
    "status": "broadcasted",
    "message": "Transaction submitted to network. Check explorer for confirmation."
}
```

#### 3.5.7 Quick Send (All-in-One)

```http
POST /ethereum/quick-send
Content-Type: application/json
```

**Request Body:**
```json
{
    "network": "sepolia",
    "keygenSessionId": "session_1732869123456789",
    "signerIds": ["device1", "device2"],
    "to": "0xRecipientAddress...",
    "valueEth": "0.1"
}
```

**Response:**
```json
{
    "network": "sepolia",
    "txHash": "0xTransactionHash...",
    "from": "0xYourAddress...",
    "to": "0xRecipientAddress...",
    "valueEth": "0.100000000000000000",
    "valueWei": "100000000000000000",
    "gasLimit": 21000,
    "nonce": 0,
    "explorerUrl": "https://sepolia.etherscan.io/tx/0xTransactionHash...",
    "status": "broadcasted",
    "message": "Transaction sent successfully!"
}
```

#### 3.5.8 Get Transaction Receipt

```http
GET /ethereum/:network/tx/:txHash
```

**Response (Success):**
```json
{
    "network": "sepolia",
    "txHash": "0xTransactionHash...",
    "blockNumber": 12345678,
    "gasUsed": 21000,
    "status": "success",
    "explorerUrl": "https://sepolia.etherscan.io/tx/0xTransactionHash..."
}
```

**Response (Pending):**
```json
{
    "error": "failed to get receipt: not found",
    "status": "pending",
    "message": "Transaction may still be pending"
}
```

---

## 4. Signing Protocol Details

### 4.1 How Threshold Signing Works

```
┌─────────────────────────────────────────────────────────────┐
│                  THRESHOLD SIGNING FLOW                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Prerequisites:                                             │
│  • Completed KeyGen session with 3 key shares              │
│  • 2 of 3 parties available for signing                    │
│                                                             │
│  Step 1: Initiation                                         │
│  ┌──────────┐     POST /signing/start      ┌──────────┐    │
│  │  Client  │ ──────────────────────────→  │  Backend │    │
│  └──────────┘   keygenSessionId, message   └──────────┘    │
│                 signerIds: [device1, device2]              │
│                                                             │
│  Step 2: Load Key Shares                                    │
│  ┌──────────┐                                              │
│  │  Backend │ → Load device1 share from KeyGen session     │
│  │          │ → Load device2 share from KeyGen session     │
│  └──────────┘                                              │
│                                                             │
│  Step 3: MPC Signing Protocol                               │
│  ┌──────────┐         ┌──────────┐                        │
│  │ Party 1  │ ←─────→ │ Party 2  │  (9 rounds of MPC)     │
│  │(device1) │  msgs   │(device2) │                        │
│  └──────────┘         └──────────┘                        │
│                                                             │
│  Step 4: Signature Assembly                                 │
│  • Combine partial signatures                              │
│  • Produce (R, S, V) components                            │
│  • Format for Ethereum compatibility                        │
│                                                             │
│  Step 5: Return Signature                                   │
│  ┌──────────┐     GET /signature          ┌──────────┐    │
│  │  Client  │ ←────────────────────────── │  Backend │    │
│  └──────────┘   {r, s, v, signature}      └──────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Signature Format

The signature is returned in Ethereum-compatible format:

```
┌────────────────────────────────────────────────────────────┐
│                    SIGNATURE FORMAT                         │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  Full Signature (65 bytes):                                │
│  ┌────────────────┬────────────────┬───────┐               │
│  │    R (32)      │    S (32)      │ V (1) │               │
│  └────────────────┴────────────────┴───────┘               │
│                                                             │
│  Individual Components:                                     │
│  • R: First 32 bytes (x-coordinate of point R)             │
│  • S: Next 32 bytes (signature proof)                      │
│  • V: Last byte (recovery ID: 27 or 28)                    │
│                                                             │
│  Ethereum Usage:                                            │
│  • Use with ecrecover() to verify                          │
│  • Compatible with web3.eth.accounts.recover()             │
│  • Can be used for transaction signing                     │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

### 4.3 Message Format

The message to sign must be a 32-byte hash:

```go
// For Ethereum transactions:
messageHash := crypto.Keccak256(rawTransaction)  // 32 bytes

// For typed data (EIP-712):
messageHash := crypto.Keccak256(typedDataHash)   // 32 bytes

// For personal sign:
prefix := "\x19Ethereum Signed Message:\n32"
messageHash := crypto.Keccak256([]byte(prefix + message))
```

---

## 5. Security Considerations

### 5.1 Cryptographic Security

| Aspect | Implementation | Security Level |
|--------|----------------|----------------|
| Curve | secp256k1 | 128-bit security |
| Key Generation | Distributed (no single point) | High |
| Signing | Threshold (2-of-3) | High |
| Threshold | 2-of-3 | Tolerates 1 compromised party |
| Private Key | Never reconstructed | Maximum |

### 5.2 Signing Security

```
┌─────────────────────────────────────────────────────────────┐
│                   SIGNING SECURITY                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ✅ Private Key Protection:                                 │
│     • Full private key NEVER exists in memory              │
│     • Each party only has its share                        │
│     • Shares combined cryptographically, not literally     │
│                                                             │
│  ✅ Threshold Security:                                     │
│     • Need 2 parties to sign                               │
│     • 1 compromised party = still secure                   │
│     • Attacker needs majority of parties                   │
│                                                             │
│  ✅ Message Integrity:                                      │
│     • Message hash verified before signing                 │
│     • Signature bound to specific message                  │
│     • Cannot be reused for different messages              │
│                                                             │
│  ⚠️ Production Recommendations:                             │
│     • Use HTTPS in production (TLS termination)            │
│     • Store secrets in environment variables               │
│     • Enable audit logging for compliance                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 5.3 Phase 2 - Production Security Features

```
┌─────────────────────────────────────────────────────────────┐
│              PRODUCTION SECURITY (Phase 2)                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ✅ Authentication:                                          │
│     • API Key authentication (SHA-256 hashed storage)       │
│     • JWT tokens with configurable expiry                   │
│     • Header-based: Authorization or X-API-Key              │
│                                                             │
│  ✅ Rate Limiting:                                           │
│     • Token bucket algorithm                                 │
│     • Per-IP rate limiting (100 req/s default)              │
│     • Per-API-key rate limiting                             │
│     • Special limits for expensive operations (KeyGen)      │
│                                                             │
│  ✅ Audit Logging:                                           │
│     • All API requests logged                               │
│     • Request body sanitization (secrets redacted)          │
│     • IP address and User-Agent tracking                    │
│     • Async logging (non-blocking)                          │
│                                                             │
│  ✅ Database Security:                                       │
│     • Connection string via environment variables           │
│     • Prepared statements (SQL injection prevention)        │
│     • Password hashing for sensitive data                   │
│                                                             │
│  ✅ Docker Security:                                         │
│     • Multi-stage builds (minimal attack surface)           │
│     • Non-root user execution                               │
│     • Health checks enabled                                 │
│     • Secrets via environment variables                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 5.4 Database Schema

```sql
-- API Keys for authentication
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) UNIQUE NOT NULL,  -- SHA-256 hash
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP
);

-- KeyGen session persistence
CREATE TABLE keygen_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) UNIQUE NOT NULL,
    threshold INTEGER NOT NULL,
    parties INTEGER NOT NULL,
    party_ids TEXT[] NOT NULL,
    status VARCHAR(20) NOT NULL,
    error TEXT,
    eth_address VARCHAR(42),
    public_key TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Key shares storage
CREATE TABLE key_shares (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL,
    party_id VARCHAR(64) NOT NULL,
    share_data TEXT NOT NULL,           -- Base64 encoded, encrypted
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(session_id, party_id)
);

-- Signing session persistence
CREATE TABLE signing_sessions (
    id SERIAL PRIMARY KEY,
    signing_id VARCHAR(64) UNIQUE NOT NULL,
    keygen_session_id VARCHAR(64) NOT NULL,
    message_hash VARCHAR(64) NOT NULL,
    signer_ids TEXT[] NOT NULL,
    status VARCHAR(20) NOT NULL,
    signature_r VARCHAR(64),
    signature_s VARCHAR(64),
    signature_v INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Ethereum transaction tracking
CREATE TABLE eth_transactions (
    id SERIAL PRIMARY KEY,
    tx_id VARCHAR(64) UNIQUE NOT NULL,
    keygen_session_id VARCHAR(64) NOT NULL,
    network VARCHAR(20) NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42) NOT NULL,
    value_wei VARCHAR(78) NOT NULL,
    gas_limit BIGINT NOT NULL,
    gas_price_wei VARCHAR(78) NOT NULL,
    nonce BIGINT NOT NULL,
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR(66),
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Audit logs for compliance
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    api_key_id INTEGER REFERENCES api_keys(id),
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_body JSONB,
    response_status INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 6. Performance Metrics

### 6.1 KeyGen Performance (Week 1)

| Metric | Value | Notes |
|--------|-------|-------|
| Pre-param generation | ~30s per party | One-time |
| Protocol execution | ~10-30s | 3 parties |
| Total KeyGen time | ~2-3 minutes | For 3 parties |

### 6.2 Signing Performance (Week 2-3)

| Metric | Value | Notes |
|--------|-------|-------|
| Party initialization | <1s | Load key shares |
| Protocol execution | ~30-60s | 2 parties |
| Signature assembly | <100ms | Combine results |
| Verification | <10ms | ECDSA verify |

### 6.3 API Response Times

| Endpoint | Average | Notes |
|----------|---------|-------|
| GET /health | <10ms | Simple response |
| POST /keygen/start | 2-3 min | Synchronous |
| POST /signing/start | 30-60s | Synchronous |
| GET /signature | <10ms | Memory lookup |
| POST /signing/verify | <10ms | ECDSA verify |

---

## 7. Testing Guide

### 7.1 Complete Test Flow

```bash
# Terminal 1: Start backend
cd /path/to/tss-wallet-backend
go build -o tss-backend .
./tss-backend

# Terminal 2: Run tests
```

### 7.2 Step 1: Health Check

```bash
curl http://localhost:8080/health
```

Expected:
```json
{"service":"tss-wallet-backend","status":"ok","version":"2.0.0-mvp"}
```

### 7.3 Step 2: KeyGen (Wait 2-3 minutes)

```bash
curl -X POST http://localhost:8080/keygen/start \
  -H "Content-Type: application/json" \
  -d '{"threshold": 2, "parties": 3, "partyIds": ["device1", "device2", "device3"]}'
```

Save the `sessionId` from response (e.g., `session_1732869123456789`).

### 7.4 Step 3: Start Signing (Wait 30-60 seconds)

```bash
# Create a test message (32-byte hash)
MESSAGE="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

# Sign with 2 parties
curl -X POST http://localhost:8080/signing/start \
  -H "Content-Type: application/json" \
  -d '{
    "keygenSessionId": "SESSION_ID_HERE",
    "message": "'$MESSAGE'",
    "signerIds": ["device1", "device2"]
  }'
```

Save the `signingId` from response.

### 7.5 Step 4: Get Signature

```bash
curl http://localhost:8080/signing/SIGNING_ID_HERE/signature
```

Expected:
```json
{
  "signingId": "signing_xxx",
  "r": "...",
  "s": "...",
  "v": 27,
  "signature": "...",
  "status": "ready"
}
```

### 7.6 Step 5: Verify Signature

```bash
curl -X POST http://localhost:8080/signing/verify \
  -H "Content-Type: application/json" \
  -d '{
    "keygenSessionId": "SESSION_ID_HERE",
    "message": "'$MESSAGE'",
    "signature": "SIGNATURE_HEX_HERE"
  }'
```

Expected:
```json
{"valid": true, "message": "..."}
```

---

## 8. Deployment

### 8.1 Local Development (without Docker)

```bash
# Prerequisites
go version  # Requires Go 1.21+

# Build
cd tss-wallet-backend
go mod tidy
go build -o tss-backend .

# Verify
ls -lh tss-backend       # ~22MB binary

# Run
./tss-backend
```

### 8.2 Docker Deployment (Recommended for Production)

```bash
# Prerequisites
docker --version         # Docker 20+
docker-compose --version # Docker Compose 2.0+

# 1. Copy and configure environment
cp .env.example .env
# Edit .env with your settings (especially secrets!)

# 2. Build and start services
docker-compose up -d

# 3. View logs
docker-compose logs -f backend

# 4. Check health
curl http://localhost:8080/health
```

### 8.3 Environment Configuration

Create a `.env` file with the following variables:

```bash
# Server Configuration
SERVER_PORT=8080
GIN_MODE=release

# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=tss_user
DB_PASSWORD=<strong-password-here>
DB_NAME=tss_wallet

# Authentication
JWT_SECRET=<32-character-random-string>
JWT_EXPIRY_HOURS=24

# Rate Limiting
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200
KEYGEN_RATE_LIMIT_PER_HOUR=10
```

### 8.4 Docker Services Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Docker Compose Stack                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   tss-backend                        │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │  Port: 8080                                  │    │   │
│  │  │  Image: tss-wallet-backend:latest           │    │   │
│  │  │  Depends: postgres                          │    │   │
│  │  │  Health: /health endpoint                   │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                     postgres                         │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │  Port: 5432                                  │    │   │
│  │  │  Image: postgres:15-alpine                  │    │   │
│  │  │  Volume: postgres_data                      │    │   │
│  │  │  Health: pg_isready                         │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 8.5 Production Checklist

```
┌─────────────────────────────────────────────────────────────┐
│              PRODUCTION DEPLOYMENT CHECKLIST                 │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Security:                                                   │
│  [ ] Generate strong JWT_SECRET (32+ chars)                 │
│  [ ] Generate strong DB_PASSWORD                            │
│  [ ] Configure HTTPS/TLS termination                        │
│  [ ] Set GIN_MODE=release                                   │
│  [ ] Review and adjust rate limits                          │
│                                                             │
│  Database:                                                   │
│  [ ] Set up PostgreSQL backup strategy                      │
│  [ ] Configure connection pooling                           │
│  [ ] Set appropriate resource limits                        │
│                                                             │
│  Monitoring:                                                 │
│  [ ] Configure log aggregation                              │
│  [ ] Set up health check monitoring                         │
│  [ ] Configure alerting for errors                          │
│                                                             │
│  Infrastructure:                                             │
│  [ ] Set up load balancer (if needed)                       │
│  [ ] Configure container resource limits                    │
│  [ ] Set up container orchestration (Kubernetes, etc.)      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 8.6 Docker Commands Reference

```bash
# Build
docker-compose build

# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Stop and remove volumes (CAUTION: deletes data)
docker-compose down -v

# Restart backend only
docker-compose restart backend

# Execute command in container
docker-compose exec backend sh

# View running containers
docker-compose ps
```

---

## 9. Roadmap

### 9.1 Phase 1 - MVP (Completed ✅)

#### Week 1 - KeyGen ✅
- [x] KeyGen protocol implementation
- [x] 2-of-3 threshold support
- [x] REST API endpoints
- [x] Ethereum address derivation
- [x] CORS support
- [x] Session management

#### Week 2-3 - Signing ✅
- [x] Signing protocol implementation
- [x] Load key shares from KeyGen
- [x] Threshold signing (2-of-3)
- [x] ECDSA signature generation
- [x] Ethereum-compatible format (R, S, V)
- [x] Signature verification endpoint

#### Week 4 - Ethereum Integration ✅
- [x] Multi-network support (Sepolia, Goerli, Mainnet)
- [x] Balance and nonce queries
- [x] Gas price estimation
- [x] Transaction building with EIP-155
- [x] TSS signature application
- [x] Transaction broadcasting
- [x] Quick-send endpoint
- [x] Transaction receipt queries

### 9.2 Phase 2 - Production Readiness (Completed ✅)

#### Database Persistence ✅
- [x] PostgreSQL database integration
- [x] Connection management with health checks
- [x] Auto-migration on startup
- [x] Repository pattern for data access
- [x] KeyGen session persistence
- [x] Signing session persistence
- [x] Ethereum transaction tracking

#### API Authentication ✅
- [x] API Key authentication
- [x] JWT token authentication
- [x] SHA-256 key hashing
- [x] Configurable token expiry
- [x] Header-based auth (Authorization, X-API-Key)

#### Rate Limiting ✅
- [x] Token bucket algorithm
- [x] Per-IP rate limiting
- [x] Per-API-key rate limiting
- [x] Special limits for expensive operations (KeyGen)
- [x] Configurable RPS and burst limits

#### Audit Logging ✅
- [x] All API requests logged
- [x] Request body sanitization
- [x] IP and User-Agent tracking
- [x] Async non-blocking logging
- [x] Compliance-ready audit trail

#### Docker Containerization ✅
- [x] Multi-stage Dockerfile
- [x] Docker Compose (Backend + PostgreSQL)
- [x] Health check integration
- [x] Non-root user execution
- [x] Environment-based configuration

### 9.3 Phase 3 - Extensions (Future)

- [ ] Bitcoin integration (UTXO management)
- [ ] WebSocket for real-time updates
- [ ] ERC-20 token transfers
- [ ] Transaction history tracking
- [ ] Multi-currency support
- [ ] Hardware wallet integration
- [ ] Key refresh protocol
- [ ] Kubernetes deployment manifests

---

## 10. API Quick Reference

### KeyGen (Week 1)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/keygen/start` | Create & execute KeyGen |
| GET | `/keygen/:sessionId/status` | Check status |
| GET | `/keygen/:sessionId/share/:partyId` | Get key share |

### Signing (Week 2-3)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/signing/start` | Create & execute signing |
| GET | `/signing/:signingId/status` | Check status |
| GET | `/signing/:signingId/signature` | Get signature |
| POST | `/signing/verify` | Verify signature |

### Ethereum (Week 4) - NEW

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/ethereum/networks` | List supported networks |
| GET | `/ethereum/:network/balance/:address` | Get ETH balance |
| GET | `/ethereum/:network/nonce/:address` | Get account nonce |
| GET | `/ethereum/:network/gas-price` | Get current gas price |
| POST | `/ethereum/build-tx` | Build unsigned transaction |
| POST | `/ethereum/sign-tx` | Sign with TSS |
| POST | `/ethereum/broadcast` | Broadcast to network |
| POST | `/ethereum/quick-send` | One-step send |
| GET | `/ethereum/:network/tx/:txHash` | Get transaction receipt |

---

## Appendix A: Complete Test Script

```bash
#!/bin/bash
# test_tss.sh - Complete TSS test script

BASE_URL="http://localhost:8080"

echo "=== TSS Wallet Backend Test ==="
echo ""

# 1. Health check
echo "1. Health check..."
curl -s $BASE_URL/health | jq .
echo ""

# 2. KeyGen
echo "2. Starting KeyGen (this takes 2-3 minutes)..."
KEYGEN_RESPONSE=$(curl -s -X POST $BASE_URL/keygen/start \
  -H "Content-Type: application/json" \
  -d '{"threshold": 2, "parties": 3, "partyIds": ["device1", "device2", "device3"]}')
echo $KEYGEN_RESPONSE | jq .
SESSION_ID=$(echo $KEYGEN_RESPONSE | jq -r '.sessionId')
echo "Session ID: $SESSION_ID"
echo ""

# 3. Get key share
echo "3. Getting key share for device1..."
curl -s $BASE_URL/keygen/$SESSION_ID/share/device1 | jq .
echo ""

# 4. Signing
MESSAGE="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
echo "4. Starting signing (this takes 30-60 seconds)..."
SIGNING_RESPONSE=$(curl -s -X POST $BASE_URL/signing/start \
  -H "Content-Type: application/json" \
  -d "{\"keygenSessionId\": \"$SESSION_ID\", \"message\": \"$MESSAGE\", \"signerIds\": [\"device1\", \"device2\"]}")
echo $SIGNING_RESPONSE | jq .
SIGNING_ID=$(echo $SIGNING_RESPONSE | jq -r '.signingId')
echo "Signing ID: $SIGNING_ID"
echo ""

# 5. Get signature
echo "5. Getting signature..."
SIGNATURE_RESPONSE=$(curl -s $BASE_URL/signing/$SIGNING_ID/signature)
echo $SIGNATURE_RESPONSE | jq .
SIGNATURE=$(echo $SIGNATURE_RESPONSE | jq -r '.signature')
echo ""

# 6. Verify
echo "6. Verifying signature..."
curl -s -X POST $BASE_URL/signing/verify \
  -H "Content-Type: application/json" \
  -d "{\"keygenSessionId\": \"$SESSION_ID\", \"message\": \"$MESSAGE\", \"signature\": \"$SIGNATURE\"}" | jq .

echo ""
echo "=== Test Complete ==="
```

---

## Appendix B: Error Codes

| HTTP Code | Error | Cause |
|-----------|-------|-------|
| 400 | threshold must be at least 2 | threshold < 2 |
| 400 | message must be hex-encoded | Invalid hex |
| 400 | message must be 32 bytes | Wrong length |
| 400 | signerIds must have at least 2 parties | Not enough signers |
| 404 | session not found | Invalid session ID |
| 404 | signing session not found | Invalid signing ID |
| 400 | signing not completed | Signature not ready |
| 500 | Internal server error | Protocol failure |

---

## Appendix C: Glossary

| Term | Definition |
|------|------------|
| TSS | Threshold Signature Scheme |
| KeyGen | Key Generation protocol |
| Signing | Threshold signing protocol |
| Party | A participant (device) |
| Threshold | Minimum parties needed (2 of 3) |
| Key Share | A party's portion of private key |
| R, S, V | ECDSA signature components |
| secp256k1 | Elliptic curve for Bitcoin/Ethereum |

---

---

## Appendix D: Ethereum Transaction Flow

```
┌─────────────────────────────────────────────────────────────┐
│              ETHEREUM TRANSACTION FLOW                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Option 1: Step-by-Step (More Control)                      │
│  ─────────────────────────────────────                      │
│                                                             │
│  Step 1: Build Transaction                                  │
│  POST /ethereum/build-tx                                    │
│  → Returns: txId, hashToSign                                │
│                                                             │
│  Step 2: Sign Transaction                                   │
│  POST /ethereum/sign-tx                                     │
│  → Returns: rawTx, txHash                                   │
│                                                             │
│  Step 3: Broadcast                                          │
│  POST /ethereum/broadcast                                   │
│  → Returns: txHash, explorerUrl                             │
│                                                             │
│  ─────────────────────────────────────                      │
│                                                             │
│  Option 2: Quick Send (All-in-One)                          │
│  ─────────────────────────────────────                      │
│                                                             │
│  POST /ethereum/quick-send                                  │
│  → Build + Sign + Broadcast in one call                     │
│  → Returns: txHash, explorerUrl                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Appendix E: Supported Networks

| Network | Chain ID | Type | RPC Endpoint |
|---------|----------|------|--------------|
| Sepolia | 11155111 | Testnet | https://rpc.sepolia.org |
| Goerli | 5 | Testnet | https://rpc.ankr.com/eth_goerli |
| Mainnet | 1 | Mainnet | https://eth.llamarpc.com |

**Getting Testnet ETH:**
- Sepolia: https://sepoliafaucet.com/
- Goerli: https://goerlifaucet.com/

---

**End of Report**

*Generated: November 30, 2025*
*Version: 4.0.0-Production (Phase 2 - Production Readiness Complete)*
