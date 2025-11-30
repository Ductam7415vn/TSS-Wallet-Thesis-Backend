package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// KeyGenSessionRow represents a keygen session in the database
type KeyGenSessionRow struct {
	ID          int64
	SessionID   string
	Threshold   int
	Parties     int
	PartyIDs    []string
	Status      string
	Error       string
	PublicKey   string
	EthAddress  string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// KeyShareRow represents a key share in the database
type KeyShareRow struct {
	ID                 int64
	SessionID          string
	PartyID            string
	PartyIndex         int
	ShareDataEncrypted string
	CreatedAt          time.Time
}

// SigningSessionRow represents a signing session in the database
type SigningSessionRow struct {
	ID              int64
	SigningID       string
	KeygenSessionID string
	MessageHash     string
	SignerIDs       []string
	Status          string
	Error           string
	SignatureR      string
	SignatureS      string
	SignatureV      int
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

// EthTransactionRow represents an Ethereum transaction in the database
type EthTransactionRow struct {
	ID               int64
	TxID             string
	KeygenSessionID  string
	SigningSessionID string
	Network          string
	FromAddress      string
	ToAddress        string
	ValueWei         string
	GasLimit         int64
	GasPriceWei      string
	Nonce            int64
	ChainID          int64
	TxHash           string
	Status           string
	RawTx            string
	CreatedAt        time.Time
	BroadcastAt      *time.Time
	ConfirmedAt      *time.Time
}

// APIKeyRow represents an API key in the database
type APIKeyRow struct {
	ID           int64
	KeyHash      string
	Name         string
	Description  string
	IsActive     bool
	RateLimitRPS int
	CreatedAt    time.Time
	LastUsedAt   *time.Time
	ExpiresAt    *time.Time
}

// ==================== KeyGen Session Repository ====================

// SaveKeyGenSession saves a new keygen session
func SaveKeyGenSession(session *KeyGenSessionRow) error {
	partyIDsJSON, err := json.Marshal(session.PartyIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal party IDs: %w", err)
	}

	query := `
		INSERT INTO keygen_sessions (session_id, threshold, parties, party_ids, status, error, public_key, eth_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (session_id) DO UPDATE SET
			status = EXCLUDED.status,
			error = EXCLUDED.error,
			public_key = EXCLUDED.public_key,
			eth_address = EXCLUDED.eth_address,
			completed_at = CASE WHEN EXCLUDED.status = 'completed' THEN CURRENT_TIMESTAMP ELSE keygen_sessions.completed_at END
		RETURNING id, created_at`

	return DB.QueryRow(query,
		session.SessionID,
		session.Threshold,
		session.Parties,
		partyIDsJSON,
		session.Status,
		session.Error,
		session.PublicKey,
		session.EthAddress,
	).Scan(&session.ID, &session.CreatedAt)
}

// GetKeyGenSession retrieves a keygen session by ID
func GetKeyGenSession(sessionID string) (*KeyGenSessionRow, error) {
	query := `
		SELECT id, session_id, threshold, parties, party_ids, status, error,
		       COALESCE(public_key, ''), COALESCE(eth_address, ''), created_at, completed_at
		FROM keygen_sessions WHERE session_id = $1`

	session := &KeyGenSessionRow{}
	var partyIDsJSON []byte
	var completedAt sql.NullTime

	err := DB.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.SessionID,
		&session.Threshold,
		&session.Parties,
		&partyIDsJSON,
		&session.Status,
		&session.Error,
		&session.PublicKey,
		&session.EthAddress,
		&session.CreatedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get keygen session: %w", err)
	}

	if err := json.Unmarshal(partyIDsJSON, &session.PartyIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal party IDs: %w", err)
	}

	if completedAt.Valid {
		session.CompletedAt = &completedAt.Time
	}

	return session, nil
}

// UpdateKeyGenSessionStatus updates the status of a keygen session
func UpdateKeyGenSessionStatus(sessionID, status, errMsg, publicKey, ethAddress string) error {
	query := `
		UPDATE keygen_sessions
		SET status = $2, error = $3, public_key = $4, eth_address = $5,
		    completed_at = CASE WHEN $2 = 'completed' THEN CURRENT_TIMESTAMP ELSE completed_at END
		WHERE session_id = $1`

	_, err := DB.Exec(query, sessionID, status, errMsg, publicKey, ethAddress)
	return err
}

// ==================== Key Share Repository ====================

// SaveKeyShare saves an encrypted key share
func SaveKeyShare(share *KeyShareRow) error {
	query := `
		INSERT INTO key_shares (session_id, party_id, party_index, share_data_encrypted)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id, party_id) DO UPDATE SET
			share_data_encrypted = EXCLUDED.share_data_encrypted
		RETURNING id, created_at`

	return DB.QueryRow(query,
		share.SessionID,
		share.PartyID,
		share.PartyIndex,
		share.ShareDataEncrypted,
	).Scan(&share.ID, &share.CreatedAt)
}

// GetKeyShare retrieves a key share by session ID and party ID
func GetKeyShare(sessionID, partyID string) (*KeyShareRow, error) {
	query := `
		SELECT id, session_id, party_id, party_index, share_data_encrypted, created_at
		FROM key_shares WHERE session_id = $1 AND party_id = $2`

	share := &KeyShareRow{}
	err := DB.QueryRow(query, sessionID, partyID).Scan(
		&share.ID,
		&share.SessionID,
		&share.PartyID,
		&share.PartyIndex,
		&share.ShareDataEncrypted,
		&share.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key share: %w", err)
	}

	return share, nil
}

// GetAllKeyShares retrieves all key shares for a session
func GetAllKeyShares(sessionID string) ([]*KeyShareRow, error) {
	query := `
		SELECT id, session_id, party_id, party_index, share_data_encrypted, created_at
		FROM key_shares WHERE session_id = $1 ORDER BY party_index`

	rows, err := DB.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query key shares: %w", err)
	}
	defer rows.Close()

	var shares []*KeyShareRow
	for rows.Next() {
		share := &KeyShareRow{}
		if err := rows.Scan(
			&share.ID,
			&share.SessionID,
			&share.PartyID,
			&share.PartyIndex,
			&share.ShareDataEncrypted,
			&share.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan key share: %w", err)
		}
		shares = append(shares, share)
	}

	return shares, nil
}

// ==================== Signing Session Repository ====================

// SaveSigningSession saves a new signing session
func SaveSigningSession(session *SigningSessionRow) error {
	signerIDsJSON, err := json.Marshal(session.SignerIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal signer IDs: %w", err)
	}

	query := `
		INSERT INTO signing_sessions (signing_id, keygen_session_id, message_hash, signer_ids, status, error, signature_r, signature_s, signature_v)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (signing_id) DO UPDATE SET
			status = EXCLUDED.status,
			error = EXCLUDED.error,
			signature_r = EXCLUDED.signature_r,
			signature_s = EXCLUDED.signature_s,
			signature_v = EXCLUDED.signature_v,
			completed_at = CASE WHEN EXCLUDED.status = 'completed' THEN CURRENT_TIMESTAMP ELSE signing_sessions.completed_at END
		RETURNING id, created_at`

	return DB.QueryRow(query,
		session.SigningID,
		session.KeygenSessionID,
		session.MessageHash,
		signerIDsJSON,
		session.Status,
		session.Error,
		session.SignatureR,
		session.SignatureS,
		session.SignatureV,
	).Scan(&session.ID, &session.CreatedAt)
}

// GetSigningSession retrieves a signing session by ID
func GetSigningSession(signingID string) (*SigningSessionRow, error) {
	query := `
		SELECT id, signing_id, keygen_session_id, message_hash, signer_ids, status, error,
		       COALESCE(signature_r, ''), COALESCE(signature_s, ''), COALESCE(signature_v, 0),
		       created_at, completed_at
		FROM signing_sessions WHERE signing_id = $1`

	session := &SigningSessionRow{}
	var signerIDsJSON []byte
	var completedAt sql.NullTime

	err := DB.QueryRow(query, signingID).Scan(
		&session.ID,
		&session.SigningID,
		&session.KeygenSessionID,
		&session.MessageHash,
		&signerIDsJSON,
		&session.Status,
		&session.Error,
		&session.SignatureR,
		&session.SignatureS,
		&session.SignatureV,
		&session.CreatedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get signing session: %w", err)
	}

	if err := json.Unmarshal(signerIDsJSON, &session.SignerIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signer IDs: %w", err)
	}

	if completedAt.Valid {
		session.CompletedAt = &completedAt.Time
	}

	return session, nil
}

// ==================== API Key Repository ====================

// SaveAPIKey saves a new API key
func SaveAPIKey(apiKey *APIKeyRow) error {
	query := `
		INSERT INTO api_keys (key_hash, name, description, is_active, rate_limit_rps, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return DB.QueryRow(query,
		apiKey.KeyHash,
		apiKey.Name,
		apiKey.Description,
		apiKey.IsActive,
		apiKey.RateLimitRPS,
		apiKey.ExpiresAt,
	).Scan(&apiKey.ID, &apiKey.CreatedAt)
}

// GetAPIKeyByHash retrieves an API key by its hash
func GetAPIKeyByHash(keyHash string) (*APIKeyRow, error) {
	query := `
		SELECT id, key_hash, name, description, is_active, rate_limit_rps, created_at, last_used_at, expires_at
		FROM api_keys WHERE key_hash = $1 AND is_active = true`

	apiKey := &APIKeyRow{}
	var lastUsedAt, expiresAt sql.NullTime

	err := DB.QueryRow(query, keyHash).Scan(
		&apiKey.ID,
		&apiKey.KeyHash,
		&apiKey.Name,
		&apiKey.Description,
		&apiKey.IsActive,
		&apiKey.RateLimitRPS,
		&apiKey.CreatedAt,
		&lastUsedAt,
		&expiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if lastUsedAt.Valid {
		apiKey.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}

	return apiKey, nil
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp
func UpdateAPIKeyLastUsed(keyHash string) error {
	query := `UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE key_hash = $1`
	_, err := DB.Exec(query, keyHash)
	return err
}

// ==================== Audit Log Repository ====================

// SaveAuditLog saves an audit log entry
func SaveAuditLog(apiKeyID *int64, action, resourceType, resourceID, ipAddress, userAgent string, requestBody interface{}, responseStatus int) error {
	var requestBodyJSON []byte
	var err error
	if requestBody != nil {
		requestBodyJSON, err = json.Marshal(requestBody)
		if err != nil {
			requestBodyJSON = []byte("{}")
		}
	}

	query := `
		INSERT INTO audit_logs (api_key_id, action, resource_type, resource_id, ip_address, user_agent, request_body, response_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = DB.Exec(query, apiKeyID, action, resourceType, resourceID, ipAddress, userAgent, requestBodyJSON, responseStatus)
	return err
}

// ==================== Ethereum Transaction Repository ====================

// SaveEthTransaction saves an Ethereum transaction
func SaveEthTransaction(tx *EthTransactionRow) error {
	query := `
		INSERT INTO eth_transactions (tx_id, keygen_session_id, signing_session_id, network, from_address, to_address, value_wei, gas_limit, gas_price_wei, nonce, chain_id, tx_hash, status, raw_tx)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tx_id) DO UPDATE SET
			signing_session_id = EXCLUDED.signing_session_id,
			tx_hash = EXCLUDED.tx_hash,
			status = EXCLUDED.status,
			raw_tx = EXCLUDED.raw_tx,
			broadcast_at = CASE WHEN EXCLUDED.status = 'broadcasted' THEN CURRENT_TIMESTAMP ELSE eth_transactions.broadcast_at END,
			confirmed_at = CASE WHEN EXCLUDED.status = 'confirmed' THEN CURRENT_TIMESTAMP ELSE eth_transactions.confirmed_at END
		RETURNING id, created_at`

	return DB.QueryRow(query,
		tx.TxID,
		tx.KeygenSessionID,
		tx.SigningSessionID,
		tx.Network,
		tx.FromAddress,
		tx.ToAddress,
		tx.ValueWei,
		tx.GasLimit,
		tx.GasPriceWei,
		tx.Nonce,
		tx.ChainID,
		tx.TxHash,
		tx.Status,
		tx.RawTx,
	).Scan(&tx.ID, &tx.CreatedAt)
}

// GetEthTransaction retrieves an Ethereum transaction by ID
func GetEthTransaction(txID string) (*EthTransactionRow, error) {
	query := `
		SELECT id, tx_id, keygen_session_id, COALESCE(signing_session_id, ''), network, from_address, to_address,
		       value_wei, gas_limit, gas_price_wei, nonce, chain_id, COALESCE(tx_hash, ''), status, COALESCE(raw_tx, ''),
		       created_at, broadcast_at, confirmed_at
		FROM eth_transactions WHERE tx_id = $1`

	tx := &EthTransactionRow{}
	var broadcastAt, confirmedAt sql.NullTime

	err := DB.QueryRow(query, txID).Scan(
		&tx.ID,
		&tx.TxID,
		&tx.KeygenSessionID,
		&tx.SigningSessionID,
		&tx.Network,
		&tx.FromAddress,
		&tx.ToAddress,
		&tx.ValueWei,
		&tx.GasLimit,
		&tx.GasPriceWei,
		&tx.Nonce,
		&tx.ChainID,
		&tx.TxHash,
		&tx.Status,
		&tx.RawTx,
		&tx.CreatedAt,
		&broadcastAt,
		&confirmedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get eth transaction: %w", err)
	}

	if broadcastAt.Valid {
		tx.BroadcastAt = &broadcastAt.Time
	}
	if confirmedAt.Valid {
		tx.ConfirmedAt = &confirmedAt.Time
	}

	return tx, nil
}
