package relay

import "time"

// ============================================================
// RELAY API TYPES
// No crypto operations, just message routing
// ============================================================

// SessionType represents the type of MPC session
type SessionType string

const (
	SessionTypeKeyGen  SessionType = "keygen"
	SessionTypeSigning SessionType = "signing"
)

// SessionStatus represents the status of a relay session
type SessionStatus string

const (
	SessionStatusWaiting    SessionStatus = "waiting"
	SessionStatusInProgress SessionStatus = "in_progress"
	SessionStatusCompleted  SessionStatus = "completed"
	SessionStatusFailed     SessionStatus = "failed"
)

// ============================================================
// HTTP REQUEST/RESPONSE TYPES
// ============================================================

// RegisterRequest - POST /relay/register
// Register device for receiving messages
type RegisterRequest struct {
	DeviceID  string `json:"deviceId" binding:"required"`
	PublicKey string `json:"publicKey" binding:"required"` // X25519 public key for E2E
	PushToken string `json:"pushToken"`                    // FCM token (optional)
}

// RegisterResponse - Response for device registration
type RegisterResponse struct {
	Success   bool   `json:"success"`
	DeviceID  string `json:"deviceId"`
	ExpiresAt int64  `json:"expiresAt"`
	Message   string `json:"message,omitempty"`
}

// CreateSessionRequest - POST /relay/session
// Create a new MPC session (coordination only, no computation)
type CreateSessionRequest struct {
	SessionID string   `json:"sessionId" binding:"required"`
	Parties   []string `json:"parties" binding:"required,min=2"` // Device IDs
	Type      string   `json:"type" binding:"required"`          // "keygen" or "signing"
	Threshold int      `json:"threshold" binding:"required,min=2"`
}

// CreateSessionResponse - Response for session creation
type CreateSessionResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

// JoinSessionRequest - POST /relay/session/:id/join
// Join an existing MPC session
type JoinSessionRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
}

// JoinSessionResponse - Response for joining session
type JoinSessionResponse struct {
	Success       bool     `json:"success"`
	SessionID     string   `json:"sessionId"`
	Parties       []string `json:"parties"`
	JoinedParties []string `json:"joinedParties"`
	Status        string   `json:"status"`
	Message       string   `json:"message,omitempty"`
}

// SendMessageRequest - POST /relay/send
// Send encrypted message to recipient(s)
type SendMessageRequest struct {
	SessionID   string   `json:"sessionId" binding:"required"`
	From        string   `json:"from" binding:"required"`
	To          []string `json:"to" binding:"required,min=1"` // Recipient device IDs
	Payload     string   `json:"payload" binding:"required"`  // E2E encrypted, base64
	Round       int      `json:"round"`                       // Protocol round number
	IsBroadcast bool     `json:"isBroadcast"`                 // Send to all parties
}

// SendMessageResponse - Response for message sending
type SendMessageResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

// GetMessagesRequest - GET /relay/messages
// Query parameters for polling messages
type GetMessagesRequest struct {
	DeviceID  string `form:"deviceId" binding:"required"`
	SessionID string `form:"sessionId"`
	AfterID   string `form:"afterId"` // For pagination
	Limit     int    `form:"limit"`   // Max messages to return
}

// MessagesResponse - Response for polling messages
type MessagesResponse struct {
	Success  bool           `json:"success"`
	Messages []RelayMessage `json:"messages"`
	HasMore  bool           `json:"hasMore"`
}

// RelayMessage - Individual message in relay
type RelayMessage struct {
	MessageID string `json:"messageId"`
	SessionID string `json:"sessionId"`
	From      string `json:"from"`
	To        string `json:"to"`
	Payload   string `json:"payload"` // E2E encrypted
	Round     int    `json:"round"`
	Timestamp int64  `json:"timestamp"`
}

// GetSessionRequest - GET /relay/session/:id
// Get session details
type GetSessionResponse struct {
	Success       bool     `json:"success"`
	SessionID     string   `json:"sessionId"`
	Type          string   `json:"type"`
	Parties       []string `json:"parties"`
	JoinedParties []string `json:"joinedParties"`
	Threshold     int      `json:"threshold"`
	Status        string   `json:"status"`
	CurrentRound  int      `json:"currentRound"`
	CreatedAt     int64    `json:"createdAt"`
	Message       string   `json:"message,omitempty"`
}

// UpdateSessionRequest - PATCH /relay/session/:id
// Update session status
type UpdateSessionRequest struct {
	Status       string `json:"status"`
	CurrentRound int    `json:"currentRound"`
}

// DeleteSessionResponse - DELETE /relay/session/:id
// Clean up session after completion
type DeleteSessionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// AckMessageRequest - POST /relay/ack
// Acknowledge message receipt
type AckMessageRequest struct {
	MessageID string `json:"messageId" binding:"required"`
	DeviceID  string `json:"deviceId" binding:"required"`
}

// AckMessageResponse - Response for message acknowledgment
type AckMessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// GetDevicePublicKeyResponse - GET /relay/device/:id/publickey
// Get device's public key for E2E encryption
type GetDevicePublicKeyResponse struct {
	Success   bool   `json:"success"`
	DeviceID  string `json:"deviceId"`
	PublicKey string `json:"publicKey"`
	Message   string `json:"message,omitempty"`
}

// ============================================================
// WEBSOCKET MESSAGE TYPES
// ============================================================

// WSMessageType represents different WebSocket message types
type WSMessageType string

const (
	WSTypeAuth          WSMessageType = "auth"
	WSTypeAuthResponse  WSMessageType = "auth_response"
	WSTypeSubscribe     WSMessageType = "subscribe"
	WSTypeUnsubscribe   WSMessageType = "unsubscribe"
	WSTypeMessage       WSMessageType = "message"
	WSTypeAck           WSMessageType = "ack"
	WSTypeSessionEvent  WSMessageType = "session_event"
	WSTypePing          WSMessageType = "ping"
	WSTypePong          WSMessageType = "pong"
	WSTypeError         WSMessageType = "error"
)

// WSBaseMessage - Base structure for all WebSocket messages
type WSBaseMessage struct {
	Type WSMessageType `json:"type"`
}

// WSAuthMessage - Client → Server: Authentication
type WSAuthMessage struct {
	Type      WSMessageType `json:"type"` // "auth"
	DeviceID  string        `json:"deviceId"`
	AuthToken string        `json:"authToken"` // JWT
}

// WSAuthResponse - Server → Client: Auth Response
type WSAuthResponse struct {
	Type    WSMessageType `json:"type"` // "auth_response"
	Success bool          `json:"success"`
	Error   string        `json:"error,omitempty"`
}

// WSSubscribe - Client → Server: Subscribe to session
type WSSubscribe struct {
	Type      WSMessageType `json:"type"` // "subscribe"
	SessionID string        `json:"sessionId"`
}

// WSUnsubscribe - Client → Server: Unsubscribe from session
type WSUnsubscribe struct {
	Type      WSMessageType `json:"type"` // "unsubscribe"
	SessionID string        `json:"sessionId"`
}

// WSMessageNotification - Server → Client: New message notification
type WSMessageNotification struct {
	Type      WSMessageType `json:"type"` // "message"
	MessageID string        `json:"messageId"`
	SessionID string        `json:"sessionId"`
	From      string        `json:"from"`
	Payload   string        `json:"payload"` // E2E encrypted
	Round     int           `json:"round"`
	Timestamp int64         `json:"timestamp"`
}

// WSAck - Client → Server: Acknowledge receipt
type WSAck struct {
	Type      WSMessageType `json:"type"` // "ack"
	MessageID string        `json:"messageId"`
}

// WSSessionEvent - Server → Client: Session event
type WSSessionEvent struct {
	Type      WSMessageType `json:"type"` // "session_event"
	SessionID string        `json:"sessionId"`
	Event     string        `json:"event"` // "party_joined", "party_left", "completed", "failed", "round_advanced"
	PartyID   string        `json:"partyId,omitempty"`
	Round     int           `json:"round,omitempty"`
}

// WSPing - Heartbeat ping
type WSPing struct {
	Type      WSMessageType `json:"type"` // "ping"
	Timestamp int64         `json:"timestamp"`
}

// WSPong - Heartbeat pong
type WSPong struct {
	Type      WSMessageType `json:"type"` // "pong"
	Timestamp int64         `json:"timestamp"`
}

// WSError - Server → Client: Error message
type WSError struct {
	Type    WSMessageType `json:"type"` // "error"
	Code    string        `json:"code"`
	Message string        `json:"message"`
}

// ============================================================
// DATABASE MODELS
// ============================================================

// RelayDevice - Database model for registered devices
type RelayDevice struct {
	ID         int64     `json:"id"`
	DeviceID   string    `json:"deviceId"`
	PublicKey  string    `json:"publicKey"`
	PushToken  string    `json:"pushToken,omitempty"`
	IsOnline   bool      `json:"isOnline"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

// RelaySession - Database model for MPC sessions
type RelaySession struct {
	ID            int64         `json:"id"`
	SessionID     string        `json:"sessionId"`
	SessionType   SessionType   `json:"sessionType"`
	Parties       []string      `json:"parties"`
	JoinedParties []string      `json:"joinedParties"`
	Threshold     int           `json:"threshold"`
	Status        SessionStatus `json:"status"`
	CurrentRound  int           `json:"currentRound"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// RelayMessageDB - Database model for messages
type RelayMessageDB struct {
	ID          int64      `json:"id"`
	MessageID   string     `json:"messageId"`
	SessionID   string     `json:"sessionId"`
	FromDevice  string     `json:"fromDevice"`
	ToDevice    string     `json:"toDevice"`
	Payload     string     `json:"payload"`
	Round       int        `json:"round"`
	IsDelivered bool       `json:"isDelivered"`
	CreatedAt   time.Time  `json:"createdAt"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
}

// ============================================================
// ERROR TYPES
// ============================================================

// RelayError - Custom error type for relay operations
type RelayError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *RelayError) Error() string {
	return e.Message
}

// Common relay errors
var (
	ErrDeviceNotFound      = &RelayError{Code: "DEVICE_NOT_FOUND", Message: "Device not found"}
	ErrDeviceAlreadyExists = &RelayError{Code: "DEVICE_EXISTS", Message: "Device already registered"}
	ErrSessionNotFound     = &RelayError{Code: "SESSION_NOT_FOUND", Message: "Session not found"}
	ErrSessionExists       = &RelayError{Code: "SESSION_EXISTS", Message: "Session already exists"}
	ErrSessionFull         = &RelayError{Code: "SESSION_FULL", Message: "Session is full"}
	ErrSessionNotReady     = &RelayError{Code: "SESSION_NOT_READY", Message: "Session not ready"}
	ErrInvalidParty        = &RelayError{Code: "INVALID_PARTY", Message: "Device is not a party in this session"}
	ErrMessageNotFound     = &RelayError{Code: "MESSAGE_NOT_FOUND", Message: "Message not found"}
	ErrUnauthorized        = &RelayError{Code: "UNAUTHORIZED", Message: "Unauthorized"}
	ErrInvalidRequest      = &RelayError{Code: "INVALID_REQUEST", Message: "Invalid request"}
)
