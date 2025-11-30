# TSS Wallet Backend - MVP Implementation

## Overview

This is a **Threshold Signature Scheme (TSS) Wallet Backend** that provides distributed key generation and threshold signing for Ethereum wallets. Uses [bnb-chain/tss-lib](https://github.com/bnb-chain/tss-lib) v2 for cryptographic operations.

**Current Version:** 2.0.0-mvp (Week 2-3 Complete)

### Key Concepts

- **TSS (Threshold Signature Scheme)**: Private key split among multiple parties (devices)
- **2-of-3 Scheme**: Any 2 of 3 devices can sign, but no single device has full key
- **KeyGen**: Generates key shares (secure, no private key ever constructed)
- **Signing**: 2+ devices collaborate to produce valid ECDSA signature

### Implementation Status

| Phase | Feature | Status |
|-------|---------|--------|
| Week 1 | KeyGen Protocol | ✅ Complete |
| Week 2-3 | Signing Protocol | ✅ Complete |
| Week 4 | Blockchain Integration | 🔲 Pending |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  HTTP REST API (Gin)                        │
├─────────────────────────────────────────────────────────────┤
│    keygen.go              │        signing.go               │
│    (KeyGen HTTP handlers) │        (Signing HTTP handlers)  │
├─────────────────────────────────────────────────────────────┤
│    keygen_service.go      │        signing_service.go       │
│    (KeyGen TSS logic)     │        (Signing TSS logic)      │
├─────────────────────────────────────────────────────────────┤
│          bnb-chain/tss-lib v2 (ECDSA, secp256k1)           │
└─────────────────────────────────────────────────────────────┘
```

## File Structure

```
tss-wallet-backend/
├── main.go              # Server entry point, routes
├── keygen.go            # HTTP endpoints for keygen (Week 1)
├── keygen_service.go    # Core TSS keygen logic (Week 1)
├── signing.go           # HTTP endpoints for signing (Week 2-3)
├── signing_service.go   # Core TSS signing logic (Week 2-3)
├── go.mod               # Dependencies
├── go.sum               # Checksums
├── CLAUDE.md            # This documentation
└── BACKEND_REPORT.md    # Client report
```

## Data Structures

### KeyGenSession (keygen_service.go)

```go
type KeyGenSession struct {
    SessionID    string
    Threshold    int                    // 2 for 2-of-3 (actual number needed to sign)
    Parties      int                    // 3 (total parties)
    PartyIDs     []string               // ["device1", "device2", "device3"]
    Status       string                 // "initialized", "in_progress", "completed", "failed"
    Error        string                 // Error message if failed
    CreatedAt    time.Time

    // Internal TSS data
    tssPartyIDs  tss.SortedPartyIDs
    peerCtx      *tss.PeerContext
    tssParties   map[string]*TSSParty   // partyId -> party
    partyIdToIdx map[string]int         // partyId -> index mapping
}

type TSSParty struct {
    Party    tss.Party
    OutCh    chan tss.Message
    EndCh    chan *keygen.LocalPartySaveData
    ErrCh    chan *tss.Error
    PartyID  *tss.PartyID
    SaveData *keygen.LocalPartySaveData
    Done     bool
}
```

### SigningSession (signing_service.go)

```go
type SigningSession struct {
    SigningID       string
    KeygenSessionID string
    Message         []byte           // 32-byte hash to sign
    SignerIDs       []string         // Parties participating (min: threshold)
    Status          string           // "initialized", "in_progress", "completed", "failed"
    Error           string
    CreatedAt       time.Time

    // Internal TSS data
    tssPartyIDs  tss.SortedPartyIDs
    peerCtx      *tss.PeerContext
    sigParties   map[string]*SigningParty
    partyIdToIdx map[string]int

    // Result
    Signature *SignatureResult
}

type SigningParty struct {
    Party   tss.Party
    OutCh   chan tss.Message
    EndCh   chan *common.SignatureData
    ErrCh   chan *tss.Error
    PartyID *tss.PartyID
    Done    bool
}

type SignatureResult struct {
    R []byte // 32 bytes
    S []byte // 32 bytes
    V int    // Recovery ID (27 or 28 for Ethereum)
}
```

## API Endpoints

### KeyGen Endpoints (Week 1)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/keygen/start` | Create session, init all parties, execute protocol |
| GET | `/keygen/:sessionId/status` | Check session status |
| GET | `/keygen/:sessionId/share/:partyId` | Get key share after completion |

### Start KeyGen (Create + Initialize + Execute)

```bash
POST /keygen/start
Content-Type: application/json

{
    "threshold": 2,
    "parties": 3,
    "partyIds": ["device1", "device2", "device3"]
}

Response (success):
{
    "sessionId": "session_1735...",
    "threshold": 2,
    "parties": 3,
    "partyIds": ["device1", "device2", "device3"],
    "status": "completed",
    "completedParties": 3,
    "createdAt": "2025-01-23T...",
    "message": "KeyGen completed successfully..."
}
```

**NOTE:** This endpoint does ALL of the following in one call:
- Creates session
- Initializes all party instances (generates pre-params, ~30s per party)
- Runs the TSS protocol
- Returns only after protocol completes

### Check Status

```bash
GET /keygen/{sessionId}/status

Response:
{
    "sessionId": "session_1735...",
    "status": "completed",
    "threshold": 2,
    "parties": 3,
    "partyIds": ["device1", "device2", "device3"],
    "completedParties": 3,
    "createdAt": "2025-01-23T..."
}
```

### Get Key Share

```bash
GET /keygen/{sessionId}/share/device1

Response:
{
    "partyId": "device1",
    "partyIndex": 0,
    "publicKey": "04abc123...",     # 65-byte uncompressed pubkey (hex)
    "ethAddress": "0x1234...",      # Derived Ethereum address
    "shareData": "base64...",       # Party's key share (save this!)
    "status": "ready"
}
```

### Signing Endpoints (Week 2-3)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/signing/start` | Create signing session, load keys, execute protocol |
| GET | `/signing/:signingId/status` | Check signing session status |
| GET | `/signing/:signingId/signature` | Get the final signature (R, S, V) |
| POST | `/signing/verify` | Verify a signature against public key |

### Start Signing

```bash
POST /signing/start
Content-Type: application/json

{
    "keygenSessionId": "session_1735...",
    "message": "a1b2c3d4...64hex",           # 32-byte hash (hex encoded)
    "signerIds": ["device1", "device2"]      # Min: threshold parties
}

Response (success):
{
    "signingId": "signing_1735...",
    "keygenSessionId": "session_1735...",
    "signerIds": ["device1", "device2"],
    "status": "completed",
    "createdAt": "2025-01-23T...",
    "message": "Signing completed successfully..."
}
```

**NOTE:** This endpoint:
- Loads key shares from the specified KeyGen session
- Validates signerIds exist and meet threshold requirement
- Runs the TSS signing protocol
- Returns only after protocol completes

### Get Signing Status

```bash
GET /signing/{signingId}/status

Response:
{
    "signingId": "signing_1735...",
    "keygenSessionId": "session_1735...",
    "signerIds": ["device1", "device2"],
    "status": "completed",
    "completedSigners": 2,
    "createdAt": "2025-01-23T..."
}
```

### Get Signature

```bash
GET /signing/{signingId}/signature

Response:
{
    "signingId": "signing_1735...",
    "r": "abc123...",                         # R component (32 bytes hex)
    "s": "def456...",                         # S component (32 bytes hex)
    "v": 27,                                  # Recovery ID (27 or 28)
    "signature": "abc123...def456...1b",     # Full signature (65 bytes hex)
    "status": "ready"
}
```

### Verify Signature

```bash
POST /signing/verify
Content-Type: application/json

{
    "keygenSessionId": "session_1735...",
    "message": "a1b2c3d4...64hex",
    "signature": "abc123...65byteshex"
}

Response:
{
    "valid": true,
    "message": "a1b2c3d4...64hex"
}
```

## TSS Protocol Flow (Simplified)

### KeyGen Flow (Week 1)

```
1. Client: POST /keygen/start
   └── Server internally:
       a. Creates session with 3 party instances
       b. Generates pre-params for each party (~30s each)
       c. Starts all parties
       d. Routes messages between parties
       e. Waits for all parties to complete
       f. Returns success response

2. Client: GET /keygen/{sessionId}/share/{partyId}
   └── Server returns key share for the specified party
```

### Signing Flow (Week 2-3)

```
1. Client: POST /signing/start
   └── Server internally:
       a. Loads key shares from referenced KeyGen session
       b. Validates signerIds meet threshold requirement
       c. Creates signing parties with loaded key shares
       d. Starts threshold signing protocol
       e. Routes messages between signing parties
       f. Collects signature components (R, S, V)
       g. Returns success response

2. Client: GET /signing/{signingId}/signature
   └── Server returns:
       - R: 32-byte signature component
       - S: 32-byte signature component
       - V: Recovery ID (27 or 28 for Ethereum)
       - Full 65-byte signature (R || S || V)

3. (Optional) Client: POST /signing/verify
   └── Server verifies signature using public key from KeyGen
```

### Complete Wallet Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    WALLET CREATION                          │
├─────────────────────────────────────────────────────────────┤
│  1. POST /keygen/start                                      │
│     └── Creates 3 key shares (device1, device2, device3)    │
│                                                             │
│  2. GET /keygen/{sessionId}/share/{partyId}                 │
│     └── Each device downloads and securely stores its share │
│                                                             │
│  Result: Ethereum address + 3 key shares (no full key!)     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    TRANSACTION SIGNING                      │
├─────────────────────────────────────────────────────────────┤
│  1. Client builds transaction, creates 32-byte hash         │
│                                                             │
│  2. POST /signing/start                                     │
│     └── 2 of 3 devices collaborate to sign                  │
│                                                             │
│  3. GET /signing/{signingId}/signature                      │
│     └── Returns (R, S, V) for Ethereum transaction          │
│                                                             │
│  4. Client broadcasts signed transaction to blockchain      │
└─────────────────────────────────────────────────────────────┘
```

## Important Details

### Party ID vs Index

- **PartyID:** Semantic identifier ("device1", "device2", "device3")
- **PartyIndex:** Position in array (0, 1, 2) - internal only, returned for reference

### Threshold Semantics

```
threshold=2, parties=3  →  2-of-3 signature scheme
  Any 2 devices needed to sign
  No single device has full key

Note: Internally, tss-lib uses threshold = t where t+1 parties are needed.
      So for 2-of-3, we pass threshold=1 to tss-lib (1+1=2).
      The API uses the intuitive threshold=2 (number of parties needed).
```

### Message Flow (Internal - Handled by Backend)

```
Party device1 ---Start()---> message1 ──┐
                                         ├──> backend routes ──┐
Party device2 ---Start()---> message2 ──┤                      ├──> device1 Update()
                                         │                      ├──> device2 Update()
Party device3 ---Start()---> message3 ──┘                      └──> device3 Update()
                                                                     (repeat until done)
```

The backend handles all message routing internally. Clients don't need to poll for messages.

### Ethereum Address Derivation

```go
func deriveEthAddress(pubKeyBytes []byte) string {
    // pubKeyBytes is 65 bytes: 0x04 || X (32 bytes) || Y (32 bytes)
    hash := sha3.NewLegacyKeccak256()
    hash.Write(pubKeyBytes[1:])  // Skip 0x04 prefix
    hashBytes := hash.Sum(nil)

    // Address = last 20 bytes
    address := hashBytes[12:32]
    return "0x" + hex.EncodeToString(address)
}
```

## Error Handling

### KeyGen Errors
- `"threshold must be at least 2"` - threshold too low
- `"threshold cannot exceed parties"` - invalid configuration
- `"partyIDs count must match parties"` - wrong number of party IDs
- `"session not found"` - sessionId doesn't exist
- `"party not found"` - partyId not in session
- `"keygen not complete for party"` - share requested before completion

### Signing Errors
- `"keygen session not found"` - referenced KeyGen session doesn't exist
- `"keygen session not completed"` - KeyGen must complete before signing
- `"signer not found in keygen session"` - signerIds must match KeyGen partyIds
- `"need at least N signers"` - signerIds count below threshold
- `"message must be hex-encoded"` - invalid message format
- `"message must be 32 bytes"` - hash must be exactly 32 bytes
- `"signing session not found"` - signingId doesn't exist
- `"signing not completed"` - signature requested before completion

## Build & Run

```bash
# Build
go mod tidy
go build -o tss-backend .

# Run
./tss-backend
# Output: Starting server on :8080

# Test (in another terminal)
curl localhost:8080/health
```

## Test Example

### Complete Test Flow (KeyGen + Signing)

```bash
# 1. Health check
curl localhost:8080/health

# 2. Start 2-of-3 keygen (this takes ~2-3 minutes due to pre-param generation)
curl -X POST localhost:8080/keygen/start \
  -H "Content-Type: application/json" \
  -d '{
    "threshold": 2,
    "parties": 3,
    "partyIds": ["device1", "device2", "device3"]
  }'
# Note: Save the sessionId from response!

# 3. Get key shares (replace {sessionId} with actual value)
curl localhost:8080/keygen/{sessionId}/share/device1
curl localhost:8080/keygen/{sessionId}/share/device2
curl localhost:8080/keygen/{sessionId}/share/device3

# 4. Sign a message (32-byte hash, hex-encoded)
# Example hash: SHA256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
curl -X POST localhost:8080/signing/start \
  -H "Content-Type: application/json" \
  -d '{
    "keygenSessionId": "{sessionId}",
    "message": "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
    "signerIds": ["device1", "device2"]
  }'
# Note: Save the signingId from response!

# 5. Get the signature
curl localhost:8080/signing/{signingId}/signature

# 6. Verify the signature (optional)
curl -X POST localhost:8080/signing/verify \
  -H "Content-Type: application/json" \
  -d '{
    "keygenSessionId": "{sessionId}",
    "message": "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
    "signature": "{signatureHex}"
  }'
```

## Session States

- `"initialized"`: Session created, parties being initialized
- `"in_progress"`: TSS protocol running
- `"completed"`: All parties finished successfully
- `"failed"`: An error occurred (check `error` field)

## MVP Limitations (Intentional)

1. **Synchronous Execution:** `/keygen/start` blocks until protocol completes (~2-3 min)
   - Simple to test and debug
   - Production version would be async with WebSocket

2. **In-Memory Sessions:** Sessions lost on server restart
   - Acceptable for MVP
   - Production would use database

3. **No Authentication:** Open to all clients
   - Acceptable for testing on localhost
   - Production needs API keys

4. **Pre-param Generation:** Takes ~30s per party
   - Could be pre-computed for production

## Dependencies

```
github.com/bnb-chain/tss-lib/v2   # TSS cryptography
github.com/gin-gonic/gin          # HTTP framework
golang.org/x/crypto/sha3          # Keccak256 for Ethereum
```

## Security Notes

- Key shares never reconstructed (true TSS)
- Each device holds only partial key
- Shares are serialized and returned to client (client encrypts for storage)
- Use HTTPS in production
- Implement session cleanup for production

## Phase 2 (Signing) - ✅ IMPLEMENTED

- [x] Create `/signing/start` endpoint
- [x] Implement threshold signing protocol
- [x] Return (R, S, V) signature components
- [x] Add signature verification endpoint
- [x] Ethereum-compatible signature format (65 bytes)

## Phase 3 (Blockchain Integration) - Week 4 Pending

- [ ] Ethereum transaction building
- [ ] Sign raw transactions with TSS signatures
- [ ] Broadcast to Ethereum testnet (Sepolia/Goerli)
- [ ] Bitcoin transaction support (optional)
- [ ] UTXO management for Bitcoin

## Signature Format Details

### Ethereum Signature (65 bytes)

```
┌──────────────────────────────────────────────────────────────┐
│  R (32 bytes)  │  S (32 bytes)  │  V (1 byte)                │
├──────────────────────────────────────────────────────────────┤
│  bytes[0:32]   │  bytes[32:64]  │  bytes[64]                 │
│                │                │  27 or 28 (recovery ID)    │
└──────────────────────────────────────────────────────────────┘
```

### Usage with Ethereum

```go
// Client side: build and sign transaction
rawTx := BuildEthereumTx(to, value, gasPrice, gasLimit, nonce)
txHash := Keccak256(rawTx)  // 32 bytes

// Sign via TSS
POST /signing/start { message: hex(txHash), signerIds: [...] }
GET /signing/{id}/signature → { r, s, v, signature }

// Apply signature to transaction
signedTx := ApplySignature(rawTx, r, s, v)
BroadcastToNetwork(signedTx)
```
