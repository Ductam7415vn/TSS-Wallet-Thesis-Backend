package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

/**
 * Signing HTTP Handlers - DEPRECATED (Legacy MVP Version)
 *
 * ⚠️ DEPRECATION NOTICE:
 * These endpoints run TSS signing on the SERVER, which is INSECURE for production.
 * The server has access to ALL key shares and can sign transactions without user consent.
 *
 * For production, use the Relay API (/relay/*) which:
 * - Only routes encrypted messages between devices
 * - Has ZERO knowledge of key shares
 * - Devices compute signatures locally
 *
 * API Design:
 *   POST /signing/start                    - Create signing session and execute
 *   GET  /signing/:signingId/status        - Get signing session status
 *   GET  /signing/:signingId/signature     - Get the final signature
 *
 * Flow:
 * 1. Client calls POST /signing/start with keygenSessionId and message
 * 2. Backend loads key shares from keygen session
 * 3. Backend runs threshold signing protocol (2-of-3)
 * 4. Client retrieves signature via GET /signing/:signingId/signature
 *
 * Set LEGACY_MODE_ENABLED=false to disable these endpoints.
 */

// StartSigningRequest represents the request to start signing
type StartSigningRequest struct {
	KeygenSessionID string   `json:"keygenSessionId" binding:"required"` // Session from keygen
	Message         string   `json:"message" binding:"required"`         // Message to sign (hex encoded hash)
	SignerIDs       []string `json:"signerIds" binding:"required"`       // Parties participating in signing (min: threshold)
}

// SignatureResponse represents the final signature
type SignatureResponse struct {
	SigningID string `json:"signingId"`
	R         string `json:"r"`         // R component (hex)
	S         string `json:"s"`         // S component (hex)
	V         int    `json:"v"`         // Recovery ID (27 or 28 for Ethereum)
	Signature string `json:"signature"` // Full signature (hex) - R || S || V
	Status    string `json:"status"`
}

// POST /signing/start
// Creates signing session, loads key shares, and executes the signing protocol
// DEPRECATED: Use /relay/* endpoints for production
func StartSigning(c *gin.Context) {
	// Add deprecation warning header
	c.Header("X-Deprecation-Warning", "This endpoint is deprecated. Use /relay/* for production.")
	c.Header("X-Deprecated-Since", "v5.0.0")
	c.Header("X-Sunset-Date", "2025-03-01")

	var req StartSigningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate message format (should be hex-encoded 32-byte hash)
	messageBytes, err := hex.DecodeString(req.Message)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "message must be hex-encoded",
		})
		return
	}
	if len(messageBytes) != 32 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("message must be 32 bytes (got %d)", len(messageBytes)),
		})
		return
	}

	// Validate signerIds count (must be >= threshold)
	if len(req.SignerIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "signerIds must have at least 2 parties (threshold)",
		})
		return
	}

	// Generate signing session ID
	signingID := fmt.Sprintf("signing_%d", time.Now().UnixNano())

	// Create and execute signing session
	if err := signingService.CreateAndExecuteSigning(
		signingID,
		req.KeygenSessionID,
		messageBytes,
		req.SignerIDs,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     err.Error(),
			"signingId": signingID,
			"status":    "failed",
		})
		return
	}

	// Get status after completion
	status, _ := signingService.GetSigningStatus(signingID)

	c.JSON(http.StatusOK, gin.H{
		"signingId":       signingID,
		"keygenSessionId": req.KeygenSessionID,
		"signerIds":       req.SignerIDs,
		"status":          status["status"],
		"createdAt":       status["createdAt"],
		"message":         "Signing completed successfully. Use GET /signing/{signingId}/signature to retrieve the signature.",
	})
}

// GET /signing/:signingId/status
// Returns the current status of a signing session
// DEPRECATED: Use /relay/* endpoints for production
func GetSigningStatus(c *gin.Context) {
	// Add deprecation warning header
	c.Header("X-Deprecation-Warning", "This endpoint is deprecated. Use /relay/* for production.")

	signingID := c.Param("signingId")

	status, err := signingService.GetSigningStatus(signingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GET /signing/:signingId/signature
// Returns the final signature after signing completes
// DEPRECATED: Use /relay/* endpoints for production
func GetSignature(c *gin.Context) {
	// Add deprecation warning header
	c.Header("X-Deprecation-Warning", "This endpoint is deprecated. Use /relay/* for production.")

	signingID := c.Param("signingId")

	sig, err := signingService.GetSignature(signingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Format signature components
	rHex := hex.EncodeToString(sig.R)
	sHex := hex.EncodeToString(sig.S)

	// Create full signature (R || S || V) for Ethereum compatibility
	fullSig := make([]byte, 65)
	copy(fullSig[0:32], sig.R)
	copy(fullSig[32:64], sig.S)
	fullSig[64] = byte(sig.V)

	c.JSON(http.StatusOK, SignatureResponse{
		SigningID: signingID,
		R:         rHex,
		S:         sHex,
		V:         sig.V,
		Signature: hex.EncodeToString(fullSig),
		Status:    "ready",
	})
}

// POST /signing/verify (optional utility endpoint)
// Verifies a signature against the public key from keygen session
// DEPRECATED: Use /relay/* endpoints for production
func VerifySignatureHandler(c *gin.Context) {
	// Add deprecation warning header
	c.Header("X-Deprecation-Warning", "This endpoint is deprecated. Use /relay/* for production.")

	var req struct {
		KeygenSessionID string `json:"keygenSessionId" binding:"required"`
		Message         string `json:"message" binding:"required"`
		Signature       string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Decode message
	messageBytes, err := hex.DecodeString(req.Message)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message hex"})
		return
	}

	// Decode signature
	sigBytes, err := hex.DecodeString(req.Signature)
	if err != nil {
		sigBytes, err = base64.StdEncoding.DecodeString(req.Signature)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature format"})
			return
		}
	}

	// Verify
	valid, err := signingService.VerifySignature(req.KeygenSessionID, messageBytes, sigBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   valid,
		"message": req.Message,
	})
}

// RegisterSigningRoutes registers all signing routes
func RegisterSigningRoutes(router *gin.Engine) {
	signing := router.Group("/signing")
	{
		signing.POST("/start", StartSigning)
		signing.GET("/:signingId/status", GetSigningStatus)
		signing.GET("/:signingId/signature", GetSignature)
		signing.POST("/verify", VerifySignatureHandler)
	}
}
