package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"tss-wallet-backend/config"
	"tss-wallet-backend/relay"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	router := gin.Default()

	// CORS middleware for Android/Web clients
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		// Check Redis health
		redisStatus := "connected"
		if relay.Redis != nil {
			if err := relay.Redis.HealthCheck(c.Request.Context()); err != nil {
				redisStatus = "error: " + err.Error()
			}
		} else {
			redisStatus = "not initialized"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "tss-wallet-backend",
			"version": "4.0.0-relay",
			"redis":   redisStatus,
		})
	})

	// Initialize Redis for Relay Server
	if err := relay.InitRedis(cfg); err != nil {
		log.Printf("[Warning] Failed to initialize Redis: %v", err)
		log.Println("[Warning] Relay server features will be limited")
	} else {
		// Initialize WebSocket hub
		relay.InitWebSocket(cfg)
	}

	// Register KeyGen routes
	RegisterKeyGenRoutes(router)

	// Register Signing routes (Week 2-3)
	RegisterSigningRoutes(router)

	// Register Ethereum routes (Week 4)
	RegisterEthereumRoutes(router)

	// Register Relay routes (Phase 3)
	relay.RegisterRelayRoutes(router)

	// Print available routes
	log.Println("==============================================")
	log.Println("   TSS Wallet Backend - Phase 3 Relay Server")
	log.Println("   True MPC Architecture - Dumb Pipe Mode")
	log.Println("==============================================")
	log.Println("")
	log.Println("Available endpoints:")
	log.Println("")
	log.Println("  GET  /health                              - Health check")
	log.Println("")
	log.Println("KeyGen (Week 1):")
	log.Println("  POST /keygen/start                        - Create session & auto-execute")
	log.Println("  GET  /keygen/:sessionId/status            - Get session status")
	log.Println("  GET  /keygen/:sessionId/share/:partyId    - Get key share for device")
	log.Println("")
	log.Println("Signing (Week 2-3):")
	log.Println("  POST /signing/start                       - Create signing session & execute")
	log.Println("  GET  /signing/:signingId/status           - Get signing session status")
	log.Println("  GET  /signing/:signingId/signature        - Get signature (R, S, V)")
	log.Println("  POST /signing/verify                      - Verify a signature")
	log.Println("")
	log.Println("Ethereum (Week 4):")
	log.Println("  GET  /ethereum/networks                   - List supported networks")
	log.Println("  GET  /ethereum/:network/balance/:address  - Get ETH balance")
	log.Println("  POST /ethereum/build-tx                   - Build unsigned transaction")
	log.Println("  POST /ethereum/sign-tx                    - Sign transaction with TSS")
	log.Println("  POST /ethereum/broadcast                  - Broadcast signed transaction")
	log.Println("")
	log.Println("Relay Server (Phase 3 - True MPC):")
	log.Println("  POST /relay/register                      - Register device for relay")
	log.Println("  GET  /relay/device/:deviceId/publickey    - Get device public key")
	log.Println("  POST /relay/session                       - Create MPC session")
	log.Println("  GET  /relay/session/:sessionId            - Get session details")
	log.Println("  POST /relay/session/:sessionId/join       - Join session")
	log.Println("  PATCH /relay/session/:sessionId           - Update session status")
	log.Println("  DELETE /relay/session/:sessionId          - Delete session")
	log.Println("  POST /relay/send                          - Send encrypted message")
	log.Println("  GET  /relay/messages                      - Poll messages")
	log.Println("  POST /relay/ack                           - Acknowledge message")
	log.Println("  GET  /relay/ws                            - WebSocket connection")
	log.Println("")
	log.Println("==============================================")
	log.Printf("Starting server on :%s...", cfg.ServerPort)

	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
