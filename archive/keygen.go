// Package archive contains the MVP TSS logic (server-side computation).
// This code is ARCHIVED and should NOT be used in production.
// In production, TSS computations should happen on client devices.
//
// SECURITY WARNING:
// This implementation exposes key shares to the server, which violates
// the fundamental security principle of threshold cryptography.
//
// Use this code ONLY for:
// - Reference for Gomobile Android implementation
// - Testing and development purposes
// - Understanding the TSS protocol flow
package archive

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/sha3"
)

/**
 * KeyGen HTTP Handlers - Simplified MVP Version
 *
 * API Design:
 *   POST /keygen/start           - Create session, init all parties, execute protocol
 *   GET  /keygen/:sessionId/status - Get session status
 *   GET  /keygen/:sessionId/share/:partyId - Get key share for a party
 *
 * This is the simplified MVP approach where:
 * - All parties are initialized at once
 * - Backend handles all message routing
 * - Protocol runs synchronously
 * - PartyID is semantic (device1, device2, device3)
 */

// StartKeyGenRequest represents the request to start keygen
type StartKeyGenRequest struct {
	Threshold int      `json:"threshold" binding:"required"` // 2 for 2-of-3
	Parties   int      `json:"parties" binding:"required"`   // 3 for 2-of-3
	PartyIDs  []string `json:"partyIds" binding:"required"`  // ["device1", "device2", "device3"]
}

// POST /keygen/start
// Creates session, initializes all parties, and executes the keygen protocol
func StartKeyGen(c *gin.Context) {
	var req StartKeyGenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate
	if req.Threshold < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "threshold must be at least 2",
		})
		return
	}
	if req.Parties < 2 || req.Parties > 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "parties must be between 2 and 10",
		})
		return
	}
	if req.Threshold > req.Parties {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "threshold cannot exceed parties",
		})
		return
	}
	if len(req.PartyIDs) != req.Parties {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("partyIds count (%d) must match parties (%d)", len(req.PartyIDs), req.Parties),
		})
		return
	}

	// Generate session ID
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	// Create session and initialize all parties
	if err := KeygenServiceInstance.CreateAndInitSession(sessionID, req.Threshold, req.Parties, req.PartyIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Execute the keygen protocol (synchronous - blocks until done)
	if err := KeygenServiceInstance.ExecuteKeyGen(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     err.Error(),
			"sessionId": sessionID,
			"status":    "failed",
		})
		return
	}

	// Get status after completion
	status, _ := KeygenServiceInstance.GetSessionStatus(sessionID)

	c.JSON(http.StatusOK, gin.H{
		"sessionId":        sessionID,
		"threshold":        req.Threshold,
		"parties":          req.Parties,
		"partyIds":         req.PartyIDs,
		"status":           "completed",
		"completedParties": status["completedParties"],
		"createdAt":        status["createdAt"],
		"message":          "KeyGen completed successfully. Use GET /keygen/{sessionId}/share/{partyId} to retrieve key shares.",
	})
}

// GET /keygen/:sessionId/status
// Returns the current status of a keygen session
func GetSessionStatus(c *gin.Context) {
	sessionID := c.Param("sessionId")

	status, err := KeygenServiceInstance.GetSessionStatus(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GET /keygen/:sessionId/share/:partyId
// Returns the key share for a specific party after keygen completes
func GetKeyShare(c *gin.Context) {
	sessionID := c.Param("sessionId")
	partyId := c.Param("partyId")

	shareData, err := KeygenServiceInstance.GetShare(sessionID, partyId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get public key bytes and derive Ethereum address
	pubKeyBytes := shareData["publicKey"].([]byte)
	ethAddress := deriveEthAddress(pubKeyBytes)

	// Encode for JSON response
	c.JSON(http.StatusOK, gin.H{
		"partyId":    partyId,
		"partyIndex": shareData["partyIndex"],
		"publicKey":  hex.EncodeToString(pubKeyBytes),
		"ethAddress": ethAddress,
		"shareData":  base64.StdEncoding.EncodeToString(shareData["shareData"].([]byte)),
		"status":     "ready",
	})
}

// deriveEthAddress derives Ethereum address from uncompressed public key
// Uses Keccak256 hash as per Ethereum specification
// Address = last 20 bytes of Keccak256(pubKey[1:]) (skip 0x04 prefix)
func deriveEthAddress(pubKeyBytes []byte) string {
	if len(pubKeyBytes) != 65 {
		return "0x0000000000000000000000000000000000000000"
	}

	// Skip the 0x04 prefix byte, hash the remaining 64 bytes (X || Y coordinates)
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubKeyBytes[1:]) // Skip first byte (0x04)
	hashBytes := hash.Sum(nil)

	// Take last 20 bytes as address
	address := hashBytes[12:32]
	return "0x" + hex.EncodeToString(address)
}

// RegisterKeyGenRoutes registers all keygen routes (simplified MVP)
func RegisterKeyGenRoutes(router *gin.Engine) {
	keygen := router.Group("/keygen")
	{
		// Simplified MVP endpoints
		keygen.POST("/start", StartKeyGen)                    // Create + init + execute
		keygen.GET("/:sessionId/status", GetSessionStatus)    // Check status
		keygen.GET("/:sessionId/share/:partyId", GetKeyShare) // Get key share
	}
}
