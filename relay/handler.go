package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RegisterRelayRoutes registers all relay API routes
func RegisterRelayRoutes(router *gin.Engine) {
	relay := router.Group("/relay")
	{
		// Device management
		relay.POST("/register", HandleRegisterDevice)
		relay.GET("/device/:deviceId/publickey", HandleGetDevicePublicKey)

		// Session management
		relay.POST("/session", HandleCreateSession)
		relay.GET("/session/:sessionId", HandleGetSession)
		relay.POST("/session/:sessionId/join", HandleJoinSession)
		relay.PATCH("/session/:sessionId", HandleUpdateSession)
		relay.DELETE("/session/:sessionId", HandleDeleteSession)

		// Message handling
		relay.POST("/send", HandleSendMessage)
		relay.GET("/messages", HandleGetMessages)
		relay.POST("/ack", HandleAckMessage)

		// WebSocket
		relay.GET("/ws", HandleWebSocket)
	}
}

// ============================================================
// DEVICE HANDLERS
// ============================================================

// HandleRegisterDevice - POST /relay/register
func HandleRegisterDevice(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RegisterResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	// Check if device already exists
	existing, _ := Redis.GetDevice(ctx, req.DeviceID)
	if existing != nil {
		// Update existing device
		existing.PublicKey = req.PublicKey
		existing.PushToken = req.PushToken
		existing.LastSeenAt = time.Now()

		if err := Redis.RegisterDevice(ctx, existing); err != nil {
			c.JSON(http.StatusInternalServerError, RegisterResponse{
				Success: false,
				Message: "Failed to update device: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, RegisterResponse{
			Success:   true,
			DeviceID:  req.DeviceID,
			ExpiresAt: time.Now().Add(time.Duration(Redis.cfg.RelayDeviceTTL) * time.Second).Unix(),
			Message:   "Device registration updated",
		})
		return
	}

	// Create new device
	device := &RelayDevice{
		DeviceID:   req.DeviceID,
		PublicKey:  req.PublicKey,
		PushToken:  req.PushToken,
		IsOnline:   false,
		LastSeenAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	if err := Redis.RegisterDevice(ctx, device); err != nil {
		c.JSON(http.StatusInternalServerError, RegisterResponse{
			Success: false,
			Message: "Failed to register device: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		Success:   true,
		DeviceID:  req.DeviceID,
		ExpiresAt: time.Now().Add(time.Duration(Redis.cfg.RelayDeviceTTL) * time.Second).Unix(),
		Message:   "Device registered successfully",
	})
}

// HandleGetDevicePublicKey - GET /relay/device/:deviceId/publickey
func HandleGetDevicePublicKey(c *gin.Context) {
	deviceID := c.Param("deviceId")

	ctx := context.Background()
	device, err := Redis.GetDevice(ctx, deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, GetDevicePublicKeyResponse{
			Success: false,
			Message: "Device not found",
		})
		return
	}

	c.JSON(http.StatusOK, GetDevicePublicKeyResponse{
		Success:   true,
		DeviceID:  device.DeviceID,
		PublicKey: device.PublicKey,
	})
}

// ============================================================
// SESSION HANDLERS
// ============================================================

// HandleCreateSession - POST /relay/session
func HandleCreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateSessionResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	// Check if session already exists
	existing, _ := Redis.GetSession(ctx, req.SessionID)
	if existing != nil {
		c.JSON(http.StatusConflict, CreateSessionResponse{
			Success: false,
			Message: "Session already exists",
		})
		return
	}

	// Validate parties exist
	for _, partyID := range req.Parties {
		_, err := Redis.GetDevice(ctx, partyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, CreateSessionResponse{
				Success: false,
				Message: "Party device not found: " + partyID,
			})
			return
		}
	}

	// Determine session type
	sessionType := SessionTypeKeyGen
	if req.Type == "signing" {
		sessionType = SessionTypeSigning
	}

	// Create session
	session := &RelaySession{
		SessionID:     req.SessionID,
		SessionType:   sessionType,
		Parties:       req.Parties,
		JoinedParties: []string{},
		Threshold:     req.Threshold,
		Status:        SessionStatusWaiting,
		CurrentRound:  0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := Redis.CreateSession(ctx, session); err != nil {
		c.JSON(http.StatusInternalServerError, CreateSessionResponse{
			Success: false,
			Message: "Failed to create session: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, CreateSessionResponse{
		Success:   true,
		SessionID: req.SessionID,
		Status:    string(SessionStatusWaiting),
		Message:   "Session created successfully",
	})
}

// HandleGetSession - GET /relay/session/:sessionId
func HandleGetSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	ctx := context.Background()
	session, err := Redis.GetSession(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, GetSessionResponse{
			Success: false,
			Message: "Session not found",
		})
		return
	}

	c.JSON(http.StatusOK, GetSessionResponse{
		Success:       true,
		SessionID:     session.SessionID,
		Type:          string(session.SessionType),
		Parties:       session.Parties,
		JoinedParties: session.JoinedParties,
		Threshold:     session.Threshold,
		Status:        string(session.Status),
		CurrentRound:  session.CurrentRound,
		CreatedAt:     session.CreatedAt.Unix(),
	})
}

// HandleJoinSession - POST /relay/session/:sessionId/join
func HandleJoinSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	var req JoinSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, JoinSessionResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	session, err := Redis.GetSession(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, JoinSessionResponse{
			Success: false,
			Message: "Session not found",
		})
		return
	}

	// Check if device is a party in this session
	isParty := false
	for _, party := range session.Parties {
		if party == req.DeviceID {
			isParty = true
			break
		}
	}
	if !isParty {
		c.JSON(http.StatusForbidden, JoinSessionResponse{
			Success: false,
			Message: "Device is not a party in this session",
		})
		return
	}

	// Check if already joined
	for _, joined := range session.JoinedParties {
		if joined == req.DeviceID {
			c.JSON(http.StatusOK, JoinSessionResponse{
				Success:       true,
				SessionID:     session.SessionID,
				Parties:       session.Parties,
				JoinedParties: session.JoinedParties,
				Status:        string(session.Status),
				Message:       "Already joined",
			})
			return
		}
	}

	// Add to joined parties
	session.JoinedParties = append(session.JoinedParties, req.DeviceID)
	session.UpdatedAt = time.Now()

	// Check if all parties have joined
	if len(session.JoinedParties) == len(session.Parties) {
		session.Status = SessionStatusInProgress
	}

	if err := Redis.UpdateSession(ctx, session); err != nil {
		c.JSON(http.StatusInternalServerError, JoinSessionResponse{
			Success: false,
			Message: "Failed to update session: " + err.Error(),
		})
		return
	}

	// Notify other parties via WebSocket
	if Hub != nil {
		event := WSSessionEvent{
			Type:      WSTypeSessionEvent,
			SessionID: sessionID,
			Event:     "party_joined",
			PartyID:   req.DeviceID,
		}
		eventData, _ := json.Marshal(event)
		Hub.SendToSessionSubscribers(sessionID, eventData)
	}

	c.JSON(http.StatusOK, JoinSessionResponse{
		Success:       true,
		SessionID:     session.SessionID,
		Parties:       session.Parties,
		JoinedParties: session.JoinedParties,
		Status:        string(session.Status),
		Message:       "Joined session successfully",
	})
}

// HandleUpdateSession - PATCH /relay/session/:sessionId
func HandleUpdateSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	var req UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	session, err := Redis.GetSession(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Session not found",
		})
		return
	}

	// Update fields
	if req.Status != "" {
		session.Status = SessionStatus(req.Status)
	}
	if req.CurrentRound > 0 {
		session.CurrentRound = req.CurrentRound
	}
	session.UpdatedAt = time.Now()

	if err := Redis.UpdateSession(ctx, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update session: " + err.Error(),
		})
		return
	}

	// Notify via WebSocket
	if Hub != nil {
		event := WSSessionEvent{
			Type:      WSTypeSessionEvent,
			SessionID: sessionID,
			Event:     "session_updated",
			Round:     session.CurrentRound,
		}
		eventData, _ := json.Marshal(event)
		Hub.SendToSessionSubscribers(sessionID, eventData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session updated",
	})
}

// HandleDeleteSession - DELETE /relay/session/:sessionId
func HandleDeleteSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	ctx := context.Background()
	if err := Redis.DeleteSession(ctx, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, DeleteSessionResponse{
			Success: false,
			Message: "Failed to delete session: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, DeleteSessionResponse{
		Success: true,
		Message: "Session deleted",
	})
}

// ============================================================
// MESSAGE HANDLERS
// ============================================================

// HandleSendMessage - POST /relay/send
func HandleSendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, SendMessageResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	// Verify session exists
	session, err := Redis.GetSession(ctx, req.SessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, SendMessageResponse{
			Success: false,
			Message: "Session not found",
		})
		return
	}

	// Verify sender is a party
	isParty := false
	for _, party := range session.Parties {
		if party == req.From {
			isParty = true
			break
		}
	}
	if !isParty {
		c.JSON(http.StatusForbidden, SendMessageResponse{
			Success: false,
			Message: "Sender is not a party in this session",
		})
		return
	}

	// Determine recipients
	recipients := req.To
	if req.IsBroadcast {
		recipients = []string{}
		for _, party := range session.Parties {
			if party != req.From {
				recipients = append(recipients, party)
			}
		}
	}

	// Generate message ID
	messageID := uuid.New().String()

	// Store message for each recipient
	for _, recipient := range recipients {
		msg := &RelayMessageDB{
			MessageID:   messageID + "-" + recipient,
			SessionID:   req.SessionID,
			FromDevice:  req.From,
			ToDevice:    recipient,
			Payload:     req.Payload,
			Round:       req.Round,
			IsDelivered: false,
			CreatedAt:   time.Now(),
		}

		if err := Redis.StoreMessage(ctx, msg); err != nil {
			c.JSON(http.StatusInternalServerError, SendMessageResponse{
				Success: false,
				Message: "Failed to store message: " + err.Error(),
			})
			return
		}

		// Try to deliver via WebSocket
		if Hub != nil {
			notification := WSMessageNotification{
				Type:      WSTypeMessage,
				MessageID: msg.MessageID,
				SessionID: msg.SessionID,
				From:      msg.FromDevice,
				Payload:   msg.Payload,
				Round:     msg.Round,
				Timestamp: msg.CreatedAt.Unix(),
			}
			notifData, _ := json.Marshal(notification)
			Hub.SendToDevice(recipient, notifData)
		}
	}

	c.JSON(http.StatusOK, SendMessageResponse{
		Success:   true,
		MessageID: messageID,
		Status:    "sent",
		Message:   fmt.Sprintf("Message sent to %d recipients", len(recipients)),
	})
}

// HandleGetMessages - GET /relay/messages
func HandleGetMessages(c *gin.Context) {
	var req GetMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	// Get messages for device
	messages, err := Redis.GetMessagesForDevice(ctx, req.DeviceID, req.SessionID, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get messages: " + err.Error(),
		})
		return
	}

	// Convert to response format
	var relayMessages []RelayMessage
	for _, msg := range messages {
		relayMessages = append(relayMessages, RelayMessage{
			MessageID: msg.MessageID,
			SessionID: msg.SessionID,
			From:      msg.FromDevice,
			To:        msg.ToDevice,
			Payload:   msg.Payload,
			Round:     msg.Round,
			Timestamp: msg.CreatedAt.Unix(),
		})
	}

	c.JSON(http.StatusOK, MessagesResponse{
		Success:  true,
		Messages: relayMessages,
		HasMore:  len(relayMessages) == req.Limit,
	})
}

// HandleAckMessage - POST /relay/ack
func HandleAckMessage(c *gin.Context) {
	var req AckMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AckMessageResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := Redis.AcknowledgeMessage(ctx, req.MessageID, req.DeviceID); err != nil {
		status := http.StatusInternalServerError
		if err == ErrMessageNotFound {
			status = http.StatusNotFound
		} else if err == ErrUnauthorized {
			status = http.StatusForbidden
		}

		c.JSON(status, AckMessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AckMessageResponse{
		Success: true,
		Message: "Message acknowledged",
	})
}
