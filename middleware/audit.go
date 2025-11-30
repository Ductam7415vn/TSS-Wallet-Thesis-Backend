package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"

	"github.com/gin-gonic/gin"
	"tss-wallet-backend/database"
)

// AuditMiddleware logs all API requests to the audit log
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		var requestBody interface{}
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// Parse JSON body if present
			if len(bodyBytes) > 0 {
				json.Unmarshal(bodyBytes, &requestBody)
			}
		}

		// Process request
		c.Next()

		// Get API key ID if authenticated
		var apiKeyID *int64
		if id, exists := c.Get("api_key_id"); exists {
			keyID := id.(int64)
			apiKeyID = &keyID
		}

		// Determine action and resource from path
		action := c.Request.Method + " " + c.FullPath()
		resourceType := extractResourceType(c.FullPath())
		resourceID := extractResourceID(c)

		// Save audit log asynchronously
		go func() {
			err := database.SaveAuditLog(
				apiKeyID,
				action,
				resourceType,
				resourceID,
				c.ClientIP(),
				c.Request.UserAgent(),
				sanitizeRequestBody(requestBody),
				c.Writer.Status(),
			)
			if err != nil {
				log.Printf("[Audit] Failed to save audit log: %v", err)
			}
		}()
	}
}

// extractResourceType extracts the resource type from the path
func extractResourceType(path string) string {
	if path == "" {
		return "unknown"
	}

	// Map paths to resource types
	switch {
	case contains(path, "/keygen"):
		return "keygen"
	case contains(path, "/signing"):
		return "signing"
	case contains(path, "/ethereum"):
		return "ethereum"
	case contains(path, "/health"):
		return "health"
	case contains(path, "/api-keys"):
		return "api_key"
	default:
		return "other"
	}
}

// extractResourceID extracts the resource ID from context params
func extractResourceID(c *gin.Context) string {
	// Try common ID parameter names
	if id := c.Param("sessionId"); id != "" {
		return id
	}
	if id := c.Param("signingId"); id != "" {
		return id
	}
	if id := c.Param("txHash"); id != "" {
		return id
	}
	if id := c.Param("address"); id != "" {
		return id
	}
	return ""
}

// sanitizeRequestBody removes sensitive fields from request body before logging
func sanitizeRequestBody(body interface{}) interface{} {
	if body == nil {
		return nil
	}

	bodyMap, ok := body.(map[string]interface{})
	if !ok {
		return body
	}

	// Create a copy to avoid modifying original
	sanitized := make(map[string]interface{})
	for k, v := range bodyMap {
		// Mask sensitive fields
		switch k {
		case "shareData", "signature", "privateKey", "secret", "password":
			sanitized[k] = "[REDACTED]"
		default:
			sanitized[k] = v
		}
	}

	return sanitized
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
