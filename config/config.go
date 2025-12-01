package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	ServerPort string
	ServerHost string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// JWT
	JWTSecret      string
	JWTExpireHours int

	// Rate Limiting
	RateLimitRPS   int // Requests per second
	RateLimitBurst int // Burst size

	// WebSocket
	WSPingInterval  int // Ping interval in seconds
	WSPongTimeout   int // Pong timeout in seconds
	WSWriteTimeout  int // Write timeout in seconds
	WSMaxMessageSize int64 // Max message size in bytes

	// Relay
	RelayMessageTTL    int // Message TTL in seconds
	RelaySessionTTL    int // Session TTL in seconds
	RelayDeviceTTL     int // Device registration TTL in seconds
	RelayCleanupInterval int // Cleanup interval in seconds

	// Environment
	Environment string // "development", "production"
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	return &Config{
		// Server
		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),

		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tss_wallet"),
		DBPassword: getEnv("DB_PASSWORD", "tss_wallet_secret"),
		DBName:     getEnv("DB_NAME", "tss_wallet_db"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		// JWT
		JWTSecret:      getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		JWTExpireHours: getEnvInt("JWT_EXPIRE_HOURS", 24),

		// Rate Limiting
		RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 10),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 20),

		// WebSocket
		WSPingInterval:   getEnvInt("WS_PING_INTERVAL", 30),
		WSPongTimeout:    getEnvInt("WS_PONG_TIMEOUT", 10),
		WSWriteTimeout:   getEnvInt("WS_WRITE_TIMEOUT", 10),
		WSMaxMessageSize: getEnvInt64("WS_MAX_MESSAGE_SIZE", 65536),

		// Relay
		RelayMessageTTL:      getEnvInt("RELAY_MESSAGE_TTL", 3600),      // 1 hour
		RelaySessionTTL:      getEnvInt("RELAY_SESSION_TTL", 86400),     // 24 hours
		RelayDeviceTTL:       getEnvInt("RELAY_DEVICE_TTL", 604800),     // 7 days
		RelayCleanupInterval: getEnvInt("RELAY_CLEANUP_INTERVAL", 300), // 5 minutes

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// GetDSN returns the PostgreSQL connection string
func (c *Config) GetDSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

// GetRedisAddr returns the Redis connection address
func (c *Config) GetRedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Global config instance
var AppConfig *Config

func init() {
	AppConfig = LoadConfig()
}
