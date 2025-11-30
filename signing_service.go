package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

/**
 * TSS Signing Service - Week 2-3 Implementation
 * Uses bnb-chain/tss-lib v2 for threshold ECDSA signing
 *
 * Key features:
 * - Loads key shares from completed KeyGen session
 * - Runs threshold signing protocol (2-of-3)
 * - Returns (R, S, V) signature components
 * - Supports Ethereum signature format (65 bytes)
 */

// SigningParty wraps a signing LocalParty with its channels
type SigningParty struct {
	Party   tss.Party
	OutCh   chan tss.Message
	EndCh   chan *common.SignatureData
	ErrCh   chan *tss.Error
	PartyID *tss.PartyID
	Done    bool
}

// SigningSession holds all data for a signing session
type SigningSession struct {
	SigningID       string
	KeygenSessionID string
	Message         []byte   // 32-byte hash to sign
	SignerIDs       []string // Parties participating
	Status          string   // "initialized", "in_progress", "completed", "failed"
	Error           string
	CreatedAt       time.Time

	// Internal TSS data
	tssPartyIDs  tss.SortedPartyIDs
	peerCtx      *tss.PeerContext
	sigParties   map[string]*SigningParty
	partyIdToIdx map[string]int

	// Result
	Signature *SignatureResult
	mu        sync.RWMutex
}

// SignatureResult holds the final signature
type SignatureResult struct {
	R []byte // 32 bytes
	S []byte // 32 bytes
	V int    // Recovery ID (27 or 28 for Ethereum)
}

// SigningService manages TSS signing sessions
type SigningService struct {
	sessions map[string]*SigningSession
	mu       sync.RWMutex
}

func NewSigningService() *SigningService {
	return &SigningService{
		sessions: make(map[string]*SigningSession),
	}
}

// CreateAndExecuteSigning creates a new signing session and runs the protocol
func (s *SigningService) CreateAndExecuteSigning(
	signingID string,
	keygenSessionID string,
	message []byte,
	signerIDs []string,
) error {
	s.mu.Lock()

	if _, exists := s.sessions[signingID]; exists {
		s.mu.Unlock()
		return fmt.Errorf("signing session %s already exists", signingID)
	}

	// Get keygen session to load key shares
	keygenSession, err := keygenService.getSession(keygenSessionID)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("keygen session not found: %w", err)
	}

	if keygenSession.Status != "completed" {
		s.mu.Unlock()
		return fmt.Errorf("keygen session not completed (status: %s)", keygenSession.Status)
	}

	// Validate signerIDs exist in keygen session
	for _, signerID := range signerIDs {
		if _, exists := keygenSession.tssParties[signerID]; !exists {
			s.mu.Unlock()
			return fmt.Errorf("signer %s not found in keygen session", signerID)
		}
	}

	// Validate threshold
	if len(signerIDs) < keygenSession.Threshold {
		s.mu.Unlock()
		return fmt.Errorf("need at least %d signers (got %d)", keygenSession.Threshold, len(signerIDs))
	}

	// Create signing party IDs (subset of keygen parties)
	signingPartyIDs := make([]*tss.PartyID, len(signerIDs))
	partyIdToIdx := make(map[string]int)
	for i, signerID := range signerIDs {
		// Get the original party ID from keygen
		keygenParty := keygenSession.tssParties[signerID]
		signingPartyIDs[i] = keygenParty.PartyID
		partyIdToIdx[signerID] = i
	}
	sortedPartyIDs := tss.SortPartyIDs(signingPartyIDs)
	peerCtx := tss.NewPeerContext(sortedPartyIDs)

	session := &SigningSession{
		SigningID:       signingID,
		KeygenSessionID: keygenSessionID,
		Message:         message,
		SignerIDs:       signerIDs,
		Status:          "initialized",
		CreatedAt:       time.Now(),
		tssPartyIDs:     sortedPartyIDs,
		peerCtx:         peerCtx,
		sigParties:      make(map[string]*SigningParty),
		partyIdToIdx:    partyIdToIdx,
	}

	// Initialize signing parties
	fmt.Printf("[Signing] Creating session %s with %d signers\n", signingID, len(signerIDs))
	for i, signerID := range signerIDs {
		if err := s.initSigningParty(session, keygenSession, signerID, i); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to init signing party %s: %w", signerID, err)
		}
		fmt.Printf("[Signing] Party %s initialized\n", signerID)
	}

	s.sessions[signingID] = session
	s.mu.Unlock()

	// Execute signing protocol
	return s.executeSigning(session)
}

// initSigningParty initializes a single signing party
func (s *SigningService) initSigningParty(
	session *SigningSession,
	keygenSession *KeyGenSession,
	signerID string,
	partyIndex int,
) error {
	// Create channels
	outCh := make(chan tss.Message, 100)
	endCh := make(chan *common.SignatureData, 1)
	errCh := make(chan *tss.Error, 1)

	// Get party ID from sorted list
	partyID := session.tssPartyIDs[partyIndex]

	// Get key share from keygen session
	keygenParty := keygenSession.tssParties[signerID]
	if keygenParty.SaveData == nil {
		return fmt.Errorf("no key share data for party %s", signerID)
	}

	// Create TSS parameters for signing
	// Note: threshold in tss-lib is t where t+1 parties are needed
	tssThreshold := keygenSession.Threshold - 1
	params := tss.NewParameters(
		tss.S256(), // secp256k1 curve
		session.peerCtx,
		partyID,
		len(session.SignerIDs),
		tssThreshold,
	)

	// Convert message to big.Int
	msgBigInt := new(big.Int).SetBytes(session.Message)

	// Create signing party with key share
	party := signing.NewLocalParty(msgBigInt, params, *keygenParty.SaveData, outCh, endCh)

	sigParty := &SigningParty{
		Party:   party,
		OutCh:   outCh,
		EndCh:   endCh,
		ErrCh:   errCh,
		PartyID: partyID,
		Done:    false,
	}

	session.sigParties[signerID] = sigParty
	return nil
}

// executeSigning runs the signing protocol
func (s *SigningService) executeSigning(session *SigningSession) error {
	session.mu.Lock()
	session.Status = "in_progress"
	session.mu.Unlock()

	fmt.Printf("[Signing] Starting protocol for session %s\n", session.SigningID)

	// Start all parties
	var wg sync.WaitGroup
	errChan := make(chan error, len(session.SignerIDs))

	for signerID, sigParty := range session.sigParties {
		wg.Add(1)
		go func(sid string, sp *SigningParty) {
			defer wg.Done()
			if err := sp.Party.Start(); err != nil {
				errChan <- fmt.Errorf("party %s failed to start: %w", sid, err.Cause())
			}
		}(signerID, sigParty)
	}

	// Message routing loop
	done := make(chan bool)
	go func() {
		s.routeSigningMessages(session)
		done <- true
	}()

	// Wait for completion or error
	go func() {
		wg.Wait()
		time.Sleep(100 * time.Millisecond)
	}()

	// Monitor for completion
	completedCount := 0
	timeout := time.After(3 * time.Minute)

	for completedCount < len(session.SignerIDs) {
		select {
		case err := <-errChan:
			session.mu.Lock()
			session.Status = "failed"
			session.Error = err.Error()
			session.mu.Unlock()
			return err
		case <-timeout:
			session.mu.Lock()
			session.Status = "failed"
			session.Error = "timeout waiting for signing completion"
			session.mu.Unlock()
			return fmt.Errorf("timeout waiting for signing completion")
		default:
			completedCount = 0
			for _, sp := range session.sigParties {
				if sp.Done {
					completedCount++
				}
			}
			if completedCount < len(session.SignerIDs) {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	session.mu.Lock()
	session.Status = "completed"
	session.mu.Unlock()

	fmt.Printf("[Signing] Protocol completed for session %s\n", session.SigningID)
	return nil
}

// routeSigningMessages handles message exchange between signing parties
func (s *SigningService) routeSigningMessages(session *SigningSession) {
	// Create index to partyId map
	idxToPartyId := make(map[int]string)
	for partyId, idx := range session.partyIdToIdx {
		idxToPartyId[idx] = partyId
	}

	for {
		allDone := true
		for _, sp := range session.sigParties {
			if !sp.Done {
				allDone = false
				break
			}
		}
		if allDone {
			return
		}

		// Collect messages from all parties
		for signerID, sigParty := range session.sigParties {
			select {
			case msg := <-sigParty.OutCh:
				if msg == nil {
					continue
				}
				s.deliverSigningMessage(session, signerID, msg, idxToPartyId)

			case sigData := <-sigParty.EndCh:
				// Signing completed for this party
				sigParty.Done = true

				// Store signature result (only need to do once)
				session.mu.Lock()
				if session.Signature == nil {
					// Extract R, S components
					rBytes := sigData.R
					sBytes := sigData.S

					// Pad to 32 bytes if needed
					r := make([]byte, 32)
					s := make([]byte, 32)
					copy(r[32-len(rBytes):], rBytes)
					copy(s[32-len(sBytes):], sBytes)

					// Calculate V (recovery ID)
					// For Ethereum: V = 27 + recoveryID
					v := 27
					if sigData.SignatureRecovery != nil && len(sigData.SignatureRecovery) > 0 {
						v = 27 + int(sigData.SignatureRecovery[0])
					}

					session.Signature = &SignatureResult{
						R: r,
						S: s,
						V: v,
					}
				}
				session.mu.Unlock()

				fmt.Printf("[Signing] Party %s completed signing\n", signerID)

			case err := <-sigParty.ErrCh:
				fmt.Printf("[Signing] Party %s error: %v\n", signerID, err)
				return

			default:
				// No message, continue
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// deliverSigningMessage routes a message to recipients
func (s *SigningService) deliverSigningMessage(
	session *SigningSession,
	fromSignerID string,
	msg tss.Message,
	idxToPartyId map[int]string,
) {
	msgBytes, _, err := msg.WireBytes()
	if err != nil {
		fmt.Printf("[Signing] Failed to serialize message: %v\n", err)
		return
	}

	dest := msg.GetTo()
	isBroadcast := dest == nil

	fromSigParty := session.sigParties[fromSignerID]

	if isBroadcast {
		for toSignerID, toSigParty := range session.sigParties {
			if toSignerID == fromSignerID {
				continue
			}
			parsedMsg, err := tss.ParseWireMessage(msgBytes, fromSigParty.PartyID, true)
			if err != nil {
				fmt.Printf("[Signing] Failed to parse broadcast message: %v\n", err)
				continue
			}
			if _, err := toSigParty.Party.Update(parsedMsg); err != nil {
				fmt.Printf("[Signing] Failed to update party %s: %v\n", toSignerID, err)
			}
		}
		fmt.Printf("[Signing] %s -> broadcast: %s\n", fromSignerID, msg.Type())
	} else {
		for _, destPartyID := range dest {
			toSignerID := idxToPartyId[destPartyID.Index]
			toSigParty := session.sigParties[toSignerID]
			if toSigParty == nil {
				continue
			}
			parsedMsg, err := tss.ParseWireMessage(msgBytes, fromSigParty.PartyID, false)
			if err != nil {
				fmt.Printf("[Signing] Failed to parse p2p message: %v\n", err)
				continue
			}
			if _, err := toSigParty.Party.Update(parsedMsg); err != nil {
				fmt.Printf("[Signing] Failed to update party %s: %v\n", toSignerID, err)
			}
		}
		if len(dest) > 0 {
			fmt.Printf("[Signing] %s -> party %d: %s\n", fromSignerID, dest[0].Index, msg.Type())
		}
	}
}

// GetSigningStatus returns the status of a signing session
func (s *SigningService) GetSigningStatus(signingID string) (map[string]interface{}, error) {
	s.mu.RLock()
	session, exists := s.sessions[signingID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("signing session %s not found", signingID)
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	completedCount := 0
	for _, sp := range session.sigParties {
		if sp.Done {
			completedCount++
		}
	}

	status := map[string]interface{}{
		"signingId":       session.SigningID,
		"keygenSessionId": session.KeygenSessionID,
		"signerIds":       session.SignerIDs,
		"status":          session.Status,
		"completedSigners": completedCount,
		"createdAt":       session.CreatedAt.Format(time.RFC3339),
	}

	if session.Error != "" {
		status["error"] = session.Error
	}

	return status, nil
}

// GetSignature returns the signature from a completed signing session
func (s *SigningService) GetSignature(signingID string) (*SignatureResult, error) {
	s.mu.RLock()
	session, exists := s.sessions[signingID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("signing session %s not found", signingID)
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	if session.Status != "completed" {
		return nil, fmt.Errorf("signing not completed (status: %s)", session.Status)
	}

	if session.Signature == nil {
		return nil, fmt.Errorf("signature not available")
	}

	return session.Signature, nil
}

// VerifySignature verifies a signature against the public key from keygen session
func (s *SigningService) VerifySignature(keygenSessionID string, message []byte, signature []byte) (bool, error) {
	// Get keygen session
	keygenSession, err := keygenService.getSession(keygenSessionID)
	if err != nil {
		return false, fmt.Errorf("keygen session not found: %w", err)
	}

	// Get public key from any party (all have same public key)
	var pubKey *ecdsa.PublicKey
	for _, party := range keygenSession.tssParties {
		if party.SaveData != nil {
			pubKey = party.SaveData.ECDSAPub.ToECDSAPubKey()
			break
		}
	}

	if pubKey == nil {
		return false, fmt.Errorf("could not get public key from keygen session")
	}

	// Parse signature
	if len(signature) < 64 {
		return false, fmt.Errorf("signature too short")
	}

	r := new(big.Int).SetBytes(signature[0:32])
	sigS := new(big.Int).SetBytes(signature[32:64])

	// Verify using ECDSA
	valid := ecdsa.Verify(pubKey, message, r, sigS)
	return valid, nil
}

// Helper: get keygen session (add this method to KeyGenService)
func (s *KeyGenService) getSession(sessionID string) (*KeyGenSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	return session, nil
}

// Global signing service instance
var signingService = NewSigningService()
