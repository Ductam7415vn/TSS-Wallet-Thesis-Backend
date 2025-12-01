package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"tss-wallet-backend/config"
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB(cfg *config.Config) error {
	var err error
	DB, err = sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("[Database] Connected to PostgreSQL successfully")

	// Run migrations
	if err := RunMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("[Database] Connection closed")
	}
}

// RunMigrations creates the database schema
func RunMigrations() error {
	migrations := []string{
		// API Keys table
		`CREATE TABLE IF NOT EXISTS api_keys (
			id SERIAL PRIMARY KEY,
			key_hash VARCHAR(64) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			is_active BOOLEAN DEFAULT true,
			rate_limit_rps INTEGER DEFAULT 10,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP,
			expires_at TIMESTAMP
		)`,

		// KeyGen Sessions table
		`CREATE TABLE IF NOT EXISTS keygen_sessions (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(64) UNIQUE NOT NULL,
			threshold INTEGER NOT NULL,
			parties INTEGER NOT NULL,
			party_ids JSONB NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'initialized',
			error TEXT,
			public_key TEXT,
			eth_address VARCHAR(42),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		)`,

		// Key Shares table (encrypted)
		`CREATE TABLE IF NOT EXISTS key_shares (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(64) NOT NULL REFERENCES keygen_sessions(session_id) ON DELETE CASCADE,
			party_id VARCHAR(64) NOT NULL,
			party_index INTEGER NOT NULL,
			share_data_encrypted TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(session_id, party_id)
		)`,

		// Signing Sessions table
		`CREATE TABLE IF NOT EXISTS signing_sessions (
			id SERIAL PRIMARY KEY,
			signing_id VARCHAR(64) UNIQUE NOT NULL,
			keygen_session_id VARCHAR(64) NOT NULL REFERENCES keygen_sessions(session_id),
			message_hash VARCHAR(64) NOT NULL,
			signer_ids JSONB NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'initialized',
			error TEXT,
			signature_r TEXT,
			signature_s TEXT,
			signature_v INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		)`,

		// Ethereum Transactions table
		`CREATE TABLE IF NOT EXISTS eth_transactions (
			id SERIAL PRIMARY KEY,
			tx_id VARCHAR(64) UNIQUE NOT NULL,
			keygen_session_id VARCHAR(64) NOT NULL REFERENCES keygen_sessions(session_id),
			signing_session_id VARCHAR(64) REFERENCES signing_sessions(signing_id),
			network VARCHAR(32) NOT NULL,
			from_address VARCHAR(42) NOT NULL,
			to_address VARCHAR(42) NOT NULL,
			value_wei TEXT NOT NULL,
			gas_limit BIGINT NOT NULL,
			gas_price_wei TEXT NOT NULL,
			nonce BIGINT NOT NULL,
			chain_id BIGINT NOT NULL,
			tx_hash VARCHAR(66),
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			raw_tx TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			broadcast_at TIMESTAMP,
			confirmed_at TIMESTAMP
		)`,

		// Audit Log table
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id SERIAL PRIMARY KEY,
			api_key_id INTEGER REFERENCES api_keys(id),
			action VARCHAR(64) NOT NULL,
			resource_type VARCHAR(32),
			resource_id VARCHAR(64),
			ip_address VARCHAR(45),
			user_agent TEXT,
			request_body JSONB,
			response_status INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ============================================================
		// RELAY SERVER TABLES (Phase 3 - True MPC)
		// ============================================================

		// Relay Devices table - registered devices for message relay
		`CREATE TABLE IF NOT EXISTS relay_devices (
			id SERIAL PRIMARY KEY,
			device_id VARCHAR(64) UNIQUE NOT NULL,
			public_key TEXT NOT NULL,
			push_token TEXT,
			is_online BOOLEAN DEFAULT false,
			last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Relay Sessions table - MPC session coordination
		`CREATE TABLE IF NOT EXISTS relay_sessions (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(64) UNIQUE NOT NULL,
			session_type VARCHAR(32) NOT NULL,
			parties JSONB NOT NULL,
			joined_parties JSONB DEFAULT '[]',
			threshold INTEGER NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'waiting',
			current_round INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Relay Messages table - encrypted messages between devices
		`CREATE TABLE IF NOT EXISTS relay_messages (
			id SERIAL PRIMARY KEY,
			message_id VARCHAR(128) UNIQUE NOT NULL,
			session_id VARCHAR(64) NOT NULL,
			from_device VARCHAR(64) NOT NULL,
			to_device VARCHAR(64) NOT NULL,
			payload TEXT NOT NULL,
			round INTEGER DEFAULT 0,
			is_delivered BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			delivered_at TIMESTAMP
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_keygen_sessions_status ON keygen_sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_keygen_sessions_eth_address ON keygen_sessions(eth_address)`,
		`CREATE INDEX IF NOT EXISTS idx_signing_sessions_status ON signing_sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_signing_sessions_keygen ON signing_sessions(keygen_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_eth_transactions_status ON eth_transactions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_eth_transactions_network ON eth_transactions(network)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash)`,

		// Relay indexes
		`CREATE INDEX IF NOT EXISTS idx_relay_devices_device_id ON relay_devices(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_relay_sessions_session_id ON relay_sessions(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_relay_sessions_status ON relay_sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_relay_messages_session_id ON relay_messages(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_relay_messages_to_device ON relay_messages(to_device)`,
		`CREATE INDEX IF NOT EXISTS idx_relay_messages_delivered ON relay_messages(is_delivered)`,
	}

	for _, migration := range migrations {
		if _, err := DB.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, migration)
		}
	}

	log.Println("[Database] Migrations completed successfully")
	return nil
}

// HealthCheck checks if the database is reachable
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Ping()
}
