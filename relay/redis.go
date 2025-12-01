package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"tss-wallet-backend/config"
)

// Redis key prefixes
const (
	KeyPrefixDevice      = "relay:device:"
	KeyPrefixSession     = "relay:session:"
	KeyPrefixMessage     = "relay:message:"
	KeyPrefixDeviceMsgs  = "relay:device_msgs:"
	KeyPrefixSessionMsgs = "relay:session_msgs:"
	KeyPrefixOnline      = "relay:online:"
	ChannelMessages      = "relay:messages"
	ChannelSessions      = "relay:sessions"
)

// RedisClient wraps redis client with relay-specific operations
type RedisClient struct {
	client *redis.Client
	cfg    *config.Config
}

var Redis *RedisClient

// InitRedis initializes the Redis connection
func InitRedis(cfg *config.Config) error {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	Redis = &RedisClient{
		client: client,
		cfg:    cfg,
	}

	log.Println("[Redis] Connected to Redis successfully")
	return nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Client returns the underlying redis client
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// ============================================================
// DEVICE OPERATIONS
// ============================================================

// RegisterDevice stores device information
func (r *RedisClient) RegisterDevice(ctx context.Context, device *RelayDevice) error {
	key := KeyPrefixDevice + device.DeviceID
	data, err := json.Marshal(device)
	if err != nil {
		return fmt.Errorf("failed to marshal device: %w", err)
	}

	ttl := time.Duration(r.cfg.RelayDeviceTTL) * time.Second
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store device: %w", err)
	}

	return nil
}

// GetDevice retrieves device information
func (r *RedisClient) GetDevice(ctx context.Context, deviceID string) (*RelayDevice, error) {
	key := KeyPrefixDevice + deviceID
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	var device RelayDevice
	if err := json.Unmarshal(data, &device); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device: %w", err)
	}

	return &device, nil
}

// UpdateDeviceOnlineStatus updates device online status
func (r *RedisClient) UpdateDeviceOnlineStatus(ctx context.Context, deviceID string, online bool) error {
	key := KeyPrefixOnline + deviceID
	if online {
		// Set with TTL slightly longer than ping interval
		ttl := time.Duration(r.cfg.WSPingInterval*2) * time.Second
		return r.client.Set(ctx, key, "1", ttl).Err()
	}
	return r.client.Del(ctx, key).Err()
}

// IsDeviceOnline checks if device is online
func (r *RedisClient) IsDeviceOnline(ctx context.Context, deviceID string) (bool, error) {
	key := KeyPrefixOnline + deviceID
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// DeleteDevice removes device registration
func (r *RedisClient) DeleteDevice(ctx context.Context, deviceID string) error {
	keys := []string{
		KeyPrefixDevice + deviceID,
		KeyPrefixOnline + deviceID,
		KeyPrefixDeviceMsgs + deviceID,
	}
	return r.client.Del(ctx, keys...).Err()
}

// ============================================================
// SESSION OPERATIONS
// ============================================================

// CreateSession stores a new session
func (r *RedisClient) CreateSession(ctx context.Context, session *RelaySession) error {
	key := KeyPrefixSession + session.SessionID
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := time.Duration(r.cfg.RelaySessionTTL) * time.Second
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Publish session creation event
	event := map[string]interface{}{
		"type":      "session_created",
		"sessionId": session.SessionID,
		"parties":   session.Parties,
	}
	eventData, _ := json.Marshal(event)
	r.client.Publish(ctx, ChannelSessions, eventData)

	return nil
}

// GetSession retrieves session information
func (r *RedisClient) GetSession(ctx context.Context, sessionID string) (*RelaySession, error) {
	key := KeyPrefixSession + sessionID
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session RelaySession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// UpdateSession updates session information
func (r *RedisClient) UpdateSession(ctx context.Context, session *RelaySession) error {
	key := KeyPrefixSession + session.SessionID
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Keep existing TTL
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		ttl = time.Duration(r.cfg.RelaySessionTTL) * time.Second
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

// DeleteSession removes a session and its messages
func (r *RedisClient) DeleteSession(ctx context.Context, sessionID string) error {
	keys := []string{
		KeyPrefixSession + sessionID,
		KeyPrefixSessionMsgs + sessionID,
	}
	return r.client.Del(ctx, keys...).Err()
}

// ============================================================
// MESSAGE OPERATIONS
// ============================================================

// StoreMessage stores a message for delivery
func (r *RedisClient) StoreMessage(ctx context.Context, msg *RelayMessageDB) error {
	// Store the message
	key := KeyPrefixMessage + msg.MessageID
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ttl := time.Duration(r.cfg.RelayMessageTTL) * time.Second
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Add to recipient's message queue
	deviceQueueKey := KeyPrefixDeviceMsgs + msg.ToDevice
	if err := r.client.RPush(ctx, deviceQueueKey, msg.MessageID).Err(); err != nil {
		return fmt.Errorf("failed to queue message: %w", err)
	}
	r.client.Expire(ctx, deviceQueueKey, ttl)

	// Add to session message list
	sessionMsgsKey := KeyPrefixSessionMsgs + msg.SessionID
	r.client.RPush(ctx, sessionMsgsKey, msg.MessageID)
	r.client.Expire(ctx, sessionMsgsKey, ttl)

	// Publish message notification for WebSocket delivery
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
	r.client.Publish(ctx, ChannelMessages+":"+msg.ToDevice, notifData)

	return nil
}

// GetMessage retrieves a message by ID
func (r *RedisClient) GetMessage(ctx context.Context, messageID string) (*RelayMessageDB, error) {
	key := KeyPrefixMessage + messageID
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	var msg RelayMessageDB
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// GetMessagesForDevice retrieves pending messages for a device
func (r *RedisClient) GetMessagesForDevice(ctx context.Context, deviceID string, sessionID string, limit int) ([]*RelayMessageDB, error) {
	if limit <= 0 {
		limit = 100
	}

	queueKey := KeyPrefixDeviceMsgs + deviceID
	messageIDs, err := r.client.LRange(ctx, queueKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get message queue: %w", err)
	}

	var messages []*RelayMessageDB
	for _, msgID := range messageIDs {
		msg, err := r.GetMessage(ctx, msgID)
		if err != nil {
			continue // Skip missing messages
		}
		// Filter by session if specified
		if sessionID != "" && msg.SessionID != sessionID {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// AcknowledgeMessage marks a message as delivered and removes from queue
func (r *RedisClient) AcknowledgeMessage(ctx context.Context, messageID, deviceID string) error {
	// Get the message to verify recipient
	msg, err := r.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	if msg.ToDevice != deviceID {
		return ErrUnauthorized
	}

	// Update delivery status
	now := time.Now()
	msg.IsDelivered = true
	msg.DeliveredAt = &now

	// Store updated message
	key := KeyPrefixMessage + messageID
	data, _ := json.Marshal(msg)
	r.client.Set(ctx, key, data, r.client.TTL(ctx, key).Val())

	// Remove from device queue
	queueKey := KeyPrefixDeviceMsgs + deviceID
	r.client.LRem(ctx, queueKey, 1, messageID)

	return nil
}

// ============================================================
// PUB/SUB OPERATIONS
// ============================================================

// Subscribe returns a subscription for device messages
func (r *RedisClient) SubscribeToDeviceMessages(ctx context.Context, deviceID string) *redis.PubSub {
	return r.client.Subscribe(ctx, ChannelMessages+":"+deviceID)
}

// SubscribeToSessionEvents returns a subscription for session events
func (r *RedisClient) SubscribeToSessionEvents(ctx context.Context) *redis.PubSub {
	return r.client.Subscribe(ctx, ChannelSessions)
}

// ============================================================
// HEALTH CHECK
// ============================================================

// HealthCheck checks if Redis is reachable
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// CloseRedis closes the Redis connection
func CloseRedis() {
	if Redis != nil {
		Redis.Close()
		log.Println("[Redis] Connection closed")
	}
}
