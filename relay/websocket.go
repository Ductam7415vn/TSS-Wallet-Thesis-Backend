package relay

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"tss-wallet-backend/config"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	conn         *websocket.Conn
	deviceID     string
	authenticated bool
	subscriptions map[string]bool // sessionID -> subscribed
	send         chan []byte
	hub          *WSHub
	mu           sync.RWMutex
}

// WSHub manages WebSocket connections
type WSHub struct {
	clients    map[string]*WSClient // deviceID -> client
	register   chan *WSClient
	unregister chan *WSClient
	broadcast  chan []byte
	mu         sync.RWMutex
	cfg        *config.Config
}

var Hub *WSHub

// NewWSHub creates a new WebSocket hub
func NewWSHub(cfg *config.Config) *WSHub {
	return &WSHub{
		clients:    make(map[string]*WSClient),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		broadcast:  make(chan []byte),
		cfg:        cfg,
	}
}

// InitWebSocket initializes the WebSocket hub
func InitWebSocket(cfg *config.Config) {
	Hub = NewWSHub(cfg)
	go Hub.Run()
	go Hub.subscribeToRedisMessages()
	log.Println("[WebSocket] Hub initialized")
}

// Run starts the WebSocket hub
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.deviceID] = client
			h.mu.Unlock()
			log.Printf("[WebSocket] Client registered: %s", client.deviceID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.deviceID]; ok {
				delete(h.clients, client.deviceID)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[WebSocket] Client unregistered: %s", client.deviceID)

			// Update device online status in Redis
			ctx := context.Background()
			if Redis != nil {
				Redis.UpdateDeviceOnlineStatus(ctx, client.deviceID, false)
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client.deviceID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// subscribeToRedisMessages subscribes to Redis pub/sub for message delivery
func (h *WSHub) subscribeToRedisMessages() {
	if Redis == nil {
		log.Println("[WebSocket] Redis not available, skipping pub/sub")
		return
	}

	ctx := context.Background()

	// Subscribe to session events
	sessionSub := Redis.SubscribeToSessionEvents(ctx)
	go func() {
		ch := sessionSub.Channel()
		for msg := range ch {
			h.handleSessionEvent([]byte(msg.Payload))
		}
	}()

	log.Println("[WebSocket] Subscribed to Redis channels")
}

// handleSessionEvent processes session events from Redis
func (h *WSHub) handleSessionEvent(data []byte) {
	var event map[string]interface{}
	if err := json.Unmarshal(data, &event); err != nil {
		return
	}

	// Broadcast to all clients subscribed to this session
	sessionID, _ := event["sessionId"].(string)
	if sessionID == "" {
		return
	}

	wsEvent := WSSessionEvent{
		Type:      WSTypeSessionEvent,
		SessionID: sessionID,
		Event:     event["type"].(string),
	}
	if partyID, ok := event["partyId"].(string); ok {
		wsEvent.PartyID = partyID
	}

	eventData, _ := json.Marshal(wsEvent)
	h.SendToSessionSubscribers(sessionID, eventData)
}

// GetClient returns a client by device ID
func (h *WSHub) GetClient(deviceID string) *WSClient {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[deviceID]
}

// SendToDevice sends a message to a specific device
func (h *WSHub) SendToDevice(deviceID string, message []byte) bool {
	h.mu.RLock()
	client, ok := h.clients[deviceID]
	h.mu.RUnlock()

	if !ok || client == nil {
		return false
	}

	select {
	case client.send <- message:
		return true
	default:
		return false
	}
}

// SendToSessionSubscribers sends a message to all clients subscribed to a session
func (h *WSHub) SendToSessionSubscribers(sessionID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		client.mu.RLock()
		subscribed := client.subscriptions[sessionID]
		client.mu.RUnlock()

		if subscribed {
			select {
			case client.send <- message:
			default:
			}
		}
	}
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	client := &WSClient{
		conn:          conn,
		subscriptions: make(map[string]bool),
		send:          make(chan []byte, 256),
		hub:           Hub,
	}

	// Start read/write pumps
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages from the WebSocket
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(c.hub.cfg.WSMaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.hub.cfg.WSPongTimeout+c.hub.cfg.WSPingInterval) * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.hub.cfg.WSPongTimeout+c.hub.cfg.WSPingInterval) * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Read error: %v", err)
			}
			break
		}
		c.handleMessage(message)
	}
}

// writePump handles outgoing messages to the WebSocket
func (c *WSClient) writePump() {
	ticker := time.NewTicker(time.Duration(c.hub.cfg.WSPingInterval) * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.hub.cfg.WSWriteTimeout) * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.hub.cfg.WSWriteTimeout) * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *WSClient) handleMessage(data []byte) {
	var baseMsg WSBaseMessage
	if err := json.Unmarshal(data, &baseMsg); err != nil {
		c.sendError("INVALID_MESSAGE", "Invalid message format")
		return
	}

	switch baseMsg.Type {
	case WSTypeAuth:
		c.handleAuth(data)
	case WSTypeSubscribe:
		c.handleSubscribe(data)
	case WSTypeUnsubscribe:
		c.handleUnsubscribe(data)
	case WSTypeAck:
		c.handleAck(data)
	case WSTypePing:
		c.handlePing(data)
	default:
		c.sendError("UNKNOWN_TYPE", "Unknown message type")
	}
}

// handleAuth processes authentication messages
func (c *WSClient) handleAuth(data []byte) {
	var authMsg WSAuthMessage
	if err := json.Unmarshal(data, &authMsg); err != nil {
		c.sendAuthResponse(false, "Invalid auth message")
		return
	}

	// Verify device exists
	ctx := context.Background()
	if Redis != nil {
		_, err := Redis.GetDevice(ctx, authMsg.DeviceID)
		if err != nil {
			c.sendAuthResponse(false, "Device not registered")
			return
		}
	}

	// TODO: Verify JWT token in production
	// For now, we accept any registered device

	c.mu.Lock()
	c.deviceID = authMsg.DeviceID
	c.authenticated = true
	c.mu.Unlock()

	// Register with hub
	c.hub.register <- c

	// Update online status
	if Redis != nil {
		Redis.UpdateDeviceOnlineStatus(ctx, c.deviceID, true)
	}

	// Subscribe to device messages from Redis
	go c.subscribeToDeviceMessages()

	c.sendAuthResponse(true, "")
	log.Printf("[WebSocket] Device authenticated: %s", c.deviceID)
}

// subscribeToDeviceMessages subscribes to Redis messages for this device
func (c *WSClient) subscribeToDeviceMessages() {
	if Redis == nil {
		return
	}

	ctx := context.Background()
	sub := Redis.SubscribeToDeviceMessages(ctx, c.deviceID)
	ch := sub.Channel()

	for msg := range ch {
		c.mu.RLock()
		authenticated := c.authenticated
		c.mu.RUnlock()

		if !authenticated {
			break
		}

		select {
		case c.send <- []byte(msg.Payload):
		default:
		}
	}
}

// handleSubscribe processes session subscription requests
func (c *WSClient) handleSubscribe(data []byte) {
	if !c.authenticated {
		c.sendError("UNAUTHORIZED", "Not authenticated")
		return
	}

	var subMsg WSSubscribe
	if err := json.Unmarshal(data, &subMsg); err != nil {
		c.sendError("INVALID_MESSAGE", "Invalid subscribe message")
		return
	}

	// Verify session exists and device is a party
	ctx := context.Background()
	if Redis != nil {
		session, err := Redis.GetSession(ctx, subMsg.SessionID)
		if err != nil {
			c.sendError("SESSION_NOT_FOUND", "Session not found")
			return
		}

		// Check if device is a party in this session
		isParty := false
		for _, party := range session.Parties {
			if party == c.deviceID {
				isParty = true
				break
			}
		}
		if !isParty {
			c.sendError("INVALID_PARTY", "Device is not a party in this session")
			return
		}
	}

	c.mu.Lock()
	c.subscriptions[subMsg.SessionID] = true
	c.mu.Unlock()

	log.Printf("[WebSocket] Device %s subscribed to session %s", c.deviceID, subMsg.SessionID)
}

// handleUnsubscribe processes session unsubscription requests
func (c *WSClient) handleUnsubscribe(data []byte) {
	if !c.authenticated {
		c.sendError("UNAUTHORIZED", "Not authenticated")
		return
	}

	var unsubMsg WSUnsubscribe
	if err := json.Unmarshal(data, &unsubMsg); err != nil {
		c.sendError("INVALID_MESSAGE", "Invalid unsubscribe message")
		return
	}

	c.mu.Lock()
	delete(c.subscriptions, unsubMsg.SessionID)
	c.mu.Unlock()

	log.Printf("[WebSocket] Device %s unsubscribed from session %s", c.deviceID, unsubMsg.SessionID)
}

// handleAck processes message acknowledgments
func (c *WSClient) handleAck(data []byte) {
	if !c.authenticated {
		c.sendError("UNAUTHORIZED", "Not authenticated")
		return
	}

	var ackMsg WSAck
	if err := json.Unmarshal(data, &ackMsg); err != nil {
		c.sendError("INVALID_MESSAGE", "Invalid ack message")
		return
	}

	ctx := context.Background()
	if Redis != nil {
		if err := Redis.AcknowledgeMessage(ctx, ackMsg.MessageID, c.deviceID); err != nil {
			log.Printf("[WebSocket] Failed to acknowledge message: %v", err)
		}
	}
}

// handlePing processes ping messages
func (c *WSClient) handlePing(data []byte) {
	var pingMsg WSPing
	if err := json.Unmarshal(data, &pingMsg); err != nil {
		return
	}

	pongMsg := WSPong{
		Type:      WSTypePong,
		Timestamp: time.Now().Unix(),
	}
	pongData, _ := json.Marshal(pongMsg)
	c.send <- pongData
}

// sendAuthResponse sends an authentication response
func (c *WSClient) sendAuthResponse(success bool, errMsg string) {
	resp := WSAuthResponse{
		Type:    WSTypeAuthResponse,
		Success: success,
		Error:   errMsg,
	}
	data, _ := json.Marshal(resp)
	c.send <- data
}

// sendError sends an error message
func (c *WSClient) sendError(code, message string) {
	errMsg := WSError{
		Type:    WSTypeError,
		Code:    code,
		Message: message,
	}
	data, _ := json.Marshal(errMsg)
	c.send <- data
}
