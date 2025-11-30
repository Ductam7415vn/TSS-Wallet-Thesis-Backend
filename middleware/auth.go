package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"tss-wallet-backend/config"
	"tss-wallet-backend/database"
)

// Claims represents JWT claims
type Claims struct {
	APIKeyID int64  `json:"api_key_id"`
	Name     string `json:"name"`
	jwt.RegisteredClaims
}

// AuthMiddleware checks for valid API key or JWT token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth in development mode if configured
		if config.AppConfig.IsDevelopment() {
			// Check if auth is explicitly disabled for development
			if c.GetHeader("X-Skip-Auth") == "development" {
				c.Next()
				return
			}
		}

		// Try API Key first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			if validateAPIKey(c, apiKey) {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_api_key",
				"message": "The provided API key is invalid or expired",
			})
			return
		}

		// Try JWT Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if validateJWT(c, token) {
					c.Next()
					return
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_token",
					"message": "The provided JWT token is invalid or expired",
				})
				return
			}
		}

		// No valid authentication provided
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required. Provide X-API-Key header or Bearer token.",
		})
	}
}

// validateAPIKey validates an API key against the database
func validateAPIKey(c *gin.Context, apiKey string) bool {
	// Hash the API key
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// Look up in database
	apiKeyRow, err := database.GetAPIKeyByHash(keyHash)
	if err != nil || apiKeyRow == nil {
		return false
	}

	// Check if expired
	if apiKeyRow.ExpiresAt != nil && apiKeyRow.ExpiresAt.Before(time.Now()) {
		return false
	}

	// Update last used
	go database.UpdateAPIKeyLastUsed(keyHash)

	// Store API key info in context
	c.Set("api_key_id", apiKeyRow.ID)
	c.Set("api_key_name", apiKeyRow.Name)
	c.Set("rate_limit_rps", apiKeyRow.RateLimitRPS)

	return true
}

// validateJWT validates a JWT token
func validateJWT(c *gin.Context, tokenString string) bool {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return false
	}

	// Store claims in context
	c.Set("api_key_id", claims.APIKeyID)
	c.Set("api_key_name", claims.Name)

	return true
}

// GenerateJWT generates a JWT token for an API key
func GenerateJWT(apiKeyID int64, name string) (string, error) {
	expirationTime := time.Now().Add(time.Duration(config.AppConfig.JWTExpireHours) * time.Hour)

	claims := &Claims{
		APIKeyID: apiKeyID,
		Name:     name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tss-wallet-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// HashAPIKey creates a SHA256 hash of an API key
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// GenerateAPIKey generates a new random API key
func GenerateAPIKey() string {
	// Generate 32 random bytes
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() % 256)
		time.Sleep(time.Nanosecond)
	}
	return "tss_" + hex.EncodeToString(b)
}

// SecureCompare performs a constant-time comparison of two strings
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// OptionalAuthMiddleware allows unauthenticated requests but still processes auth if provided
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to authenticate but don't block if not provided
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			validateAPIKey(c, apiKey)
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			validateJWT(c, token)
		}

		c.Next()
	}
}
