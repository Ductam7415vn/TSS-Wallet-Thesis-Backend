package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// CORS middleware for Android/Web clients
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "tss-wallet-backend",
			"version": "3.0.0-mvp",
		})
	})

	// Register KeyGen routes
	RegisterKeyGenRoutes(router)

	// Register Signing routes (Week 2-3)
	RegisterSigningRoutes(router)

	// Register Ethereum routes (Week 4)
	RegisterEthereumRoutes(router)

	// Print available routes
	log.Println("==============================================")
	log.Println("   TSS Wallet Backend - MVP Week 4")
	log.Println("   Ethereum Integration Complete!")
	log.Println("==============================================")
	log.Println("")
	log.Println("Available endpoints:")
	log.Println("")
	log.Println("  GET  /health                              - Health check")
	log.Println("")
	log.Println("KeyGen (Week 1):")
	log.Println("  POST /keygen/start                        - Create session & auto-execute")
	log.Println("       Body: {\"threshold\": 2, \"parties\": 3, \"partyIds\": [\"device1\", \"device2\", \"device3\"]}")
	log.Println("")
	log.Println("  GET  /keygen/:sessionId/status            - Get session status")
	log.Println("  GET  /keygen/:sessionId/share/:partyId    - Get key share for device")
	log.Println("")
	log.Println("Signing (Week 2-3):")
	log.Println("  POST /signing/start                       - Create signing session & execute")
	log.Println("       Body: {\"keygenSessionId\": \"...\", \"message\": \"hex32bytes\", \"signerIds\": [\"device1\", \"device2\"]}")
	log.Println("")
	log.Println("  GET  /signing/:signingId/status           - Get signing session status")
	log.Println("  GET  /signing/:signingId/signature        - Get signature (R, S, V)")
	log.Println("  POST /signing/verify                      - Verify a signature")
	log.Println("")
	log.Println("Ethereum (Week 4):")
	log.Println("  GET  /ethereum/networks                   - List supported networks")
	log.Println("  GET  /ethereum/:network/balance/:address  - Get ETH balance")
	log.Println("  GET  /ethereum/:network/nonce/:address    - Get account nonce")
	log.Println("  GET  /ethereum/:network/gas-price         - Get current gas price")
	log.Println("  POST /ethereum/build-tx                   - Build unsigned transaction")
	log.Println("  POST /ethereum/sign-tx                    - Sign transaction with TSS")
	log.Println("  POST /ethereum/broadcast                  - Broadcast signed transaction")
	log.Println("  POST /ethereum/quick-send                 - One-step send (build+sign+broadcast)")
	log.Println("  GET  /ethereum/:network/tx/:txHash        - Get transaction receipt")
	log.Println("")
	log.Println("==============================================")
	log.Println("Starting server on :8080...")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
