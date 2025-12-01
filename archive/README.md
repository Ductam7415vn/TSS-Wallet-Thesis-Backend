# Archive - MVP TSS Logic (Server-Side Computation)

This directory contains the **archived MVP code** from Phase 1-2 where TSS computations were performed server-side.

## Why Archived?

In Phase 3, we transitioned to **True MPC Architecture** where:
- **Before (MVP):** Server computed KeyGen and Signing
- **After (Production):** Clients compute, Server only relays encrypted messages

## Files

| File | Description |
|------|-------------|
| `keygen.go` | HTTP handlers for KeyGen endpoints |
| `keygen_service.go` | TSS KeyGen logic using bnb-chain/tss-lib |
| `signing.go` | HTTP handlers for Signing endpoints |
| `signing_service.go` | TSS Signing logic using bnb-chain/tss-lib |

## Important Notes

1. **DO NOT USE IN PRODUCTION** - These files expose key shares to the server
2. **For Reference Only** - Use as reference for Android Gomobile implementation
3. **Security Risk** - Server-side TSS means server can reconstruct private keys

## Migration Path

The TSS logic from these files will be:
1. Packaged using **Gomobile** as Android .aar library
2. Run entirely on client devices
3. Communicate via **Relay Server** (encrypted messages only)

## Related Documentation

- See `PROJECT_ROADMAP.md` for migration details
- See `relay/` directory for the new Relay Server implementation
