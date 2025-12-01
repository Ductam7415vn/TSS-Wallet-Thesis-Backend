// Package archive contains the MVP TSS logic (server-side computation).
// See keygen.go for package documentation.
package archive

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

/**
 * TSS KeyGen Service - Simplified MVP Version
 * Uses bnb-chain/tss-lib v2 for threshold ECDSA
 *
 * Key differences from production version:
 * - Batch initialization: all parties created at once
 * - Synchronous execution: ExecuteKeyGen() runs entire protocol
 * - Backend orchestrates message routing (not client)
 * - PartyID-based identification (device1, device2, device3)
 */

// TSSParty wraps a LocalParty with its channels
type TSSParty struct {
	Party    tss.Party
	OutCh    chan tss.Message
	EndCh    chan *keygen.LocalPartySaveData
	ErrCh    chan *tss.Error
	PartyID  *tss.PartyID
	SaveData *keygen.LocalPartySaveData
	Done     bool
}

// KeyGenService manages TSS keygen sessions
type KeyGenService struct {
	sessions map[string]*KeyGenSession
	mu       sync.RWMutex
}

// KeyGenSession holds all parties for a keygen session (simplified)
type KeyGenSession struct {
	SessionID  string
	Threshold  int                   // 2 for 2-of-3 (actual number needed to sign)
	Parties    int                   // 3 (total parties)
	PartyIDs   []string              // ["device1", "device2", "device3"]
	Status     string                // "initialized", "in_progress", "completed", "failed"
	Error      string                // Error message if failed
	CreatedAt  time.Time

	// Internal TSS data
	TssPartyIDs  tss.SortedPartyIDs
	PeerCtx      *tss.PeerContext
	TssParties   map[string]*TSSParty // partyId -> party
	PartyIdToIdx map[string]int       // partyId -> index mapping
	mu           sync.RWMutex
}

func NewKeyGenService() *KeyGenService {
	return &KeyGenService{
		sessions: make(map[string]*KeyGenSession),
	}
}

// CreateAndInitSession creates a new session and initializes ALL parties at once
func (s *KeyGenService) CreateAndInitSession(sessionID string, threshold, parties int, partyIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sessionID]; exists {
		return fmt.Errorf("session %s already exists", sessionID)
	}

	// Validate
	if len(partyIDs) != parties {
		return fmt.Errorf("partyIDs count (%d) must match parties (%d)", len(partyIDs), parties)
	}
	if threshold < 2 || threshold > parties {
		return fmt.Errorf("invalid threshold: must be 2 <= threshold <= parties")
	}

	// Generate TSS party IDs
	tssPartyIDs := make([]*tss.PartyID, parties)
	partyIdToIdx := make(map[string]int)
	for i := 0; i < parties; i++ {
		key := big.NewInt(int64(i + 1))
		tssPartyIDs[i] = tss.NewPartyID(
			partyIDs[i],                // ID (semantic name like "device1")
			fmt.Sprintf("Party %d", i), // Moniker
			key,
		)
		partyIdToIdx[partyIDs[i]] = i
	}
	sortedPartyIDs := tss.SortPartyIDs(tssPartyIDs)
	peerCtx := tss.NewPeerContext(sortedPartyIDs)

	session := &KeyGenSession{
		SessionID:    sessionID,
		Threshold:    threshold,
		Parties:      parties,
		PartyIDs:     partyIDs,
		Status:       "initialized",
		CreatedAt:    time.Now(),
		TssPartyIDs:  sortedPartyIDs,
		PeerCtx:      peerCtx,
		TssParties:   make(map[string]*TSSParty),
		PartyIdToIdx: partyIdToIdx,
	}

	// Initialize ALL parties at once
	fmt.Printf("[KeyGen] Creating session %s with %d parties (threshold=%d)\n", sessionID, parties, threshold)
	for i, partyId := range partyIDs {
		if err := s.initPartyInternal(session, partyId, i); err != nil {
			return fmt.Errorf("failed to init party %s: %w", partyId, err)
		}
		fmt.Printf("[KeyGen] Party %s (index %d) initialized\n", partyId, i)
	}

	s.sessions[sessionID] = session
	return nil
}

// initPartyInternal initializes a single party (internal use)
func (s *KeyGenService) initPartyInternal(session *KeyGenSession, partyId string, partyIndex int) error {
	// Create channels
	outCh := make(chan tss.Message, 100)
	endCh := make(chan *keygen.LocalPartySaveData, 1)
	errCh := make(chan *tss.Error, 1)

	// Get party ID from sorted list
	partyID := session.TssPartyIDs[partyIndex]

	// Create TSS parameters
	// Note: threshold in tss-lib is t where t+1 parties are needed
	// For 2-of-3: threshold = 1 (need 1+1 = 2 parties)
	tssThreshold := session.Threshold - 1
	params := tss.NewParameters(
		tss.S256(), // secp256k1 curve
		session.PeerCtx,
		partyID,
		session.Parties,
		tssThreshold,
	)

	// Generate pre-params (can be slow, ~30s first time)
	fmt.Printf("[KeyGen] Generating pre-params for %s (this may take ~30s)...\n", partyId)
	preParams, err := keygen.GeneratePreParams(2 * time.Minute)
	if err != nil {
		return fmt.Errorf("failed to generate pre-params: %w", err)
	}
	fmt.Printf("[KeyGen] Pre-params generated for %s\n", partyId)

	// Create local party
	party := keygen.NewLocalParty(params, outCh, endCh, *preParams)

	tssParty := &TSSParty{
		Party:   party,
		OutCh:   outCh,
		EndCh:   endCh,
		ErrCh:   errCh,
		PartyID: partyID,
		Done:    false,
	}

	session.TssParties[partyId] = tssParty
	return nil
}

// ExecuteKeyGen runs the entire keygen protocol synchronously
// This is the simplified MVP approach - backend handles all message routing
func (s *KeyGenService) ExecuteKeyGen(sessionID string) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	if session.Status != "initialized" {
		session.mu.Unlock()
		return fmt.Errorf("session already %s", session.Status)
	}
	session.Status = "in_progress"
	session.mu.Unlock()

	fmt.Printf("[KeyGen] Starting protocol for session %s\n", sessionID)

	// Start all parties
	var wg sync.WaitGroup
	errChan := make(chan error, session.Parties)

	for partyId, tssParty := range session.TssParties {
		wg.Add(1)
		go func(pid string, tp *TSSParty) {
			defer wg.Done()
			if err := tp.Party.Start(); err != nil {
				errChan <- fmt.Errorf("party %s failed to start: %w", pid, err.Cause())
			}
		}(partyId, tssParty)
	}

	// Message routing loop
	done := make(chan bool)
	go func() {
		s.routeMessages(session)
		done <- true
	}()

	// Wait for completion or error
	go func() {
		wg.Wait()
		// Give some time for final messages
		time.Sleep(100 * time.Millisecond)
	}()

	// Monitor for completion
	completedCount := 0
	timeout := time.After(5 * time.Minute)

	for completedCount < session.Parties {
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
			session.Error = "timeout waiting for keygen completion"
			session.mu.Unlock()
			return fmt.Errorf("timeout waiting for keygen completion")
		default:
			// Check completion status
			completedCount = 0
			for _, tp := range session.TssParties {
				if tp.Done {
					completedCount++
				}
			}
			if completedCount < session.Parties {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	session.mu.Lock()
	session.Status = "completed"
	session.mu.Unlock()

	fmt.Printf("[KeyGen] Protocol completed for session %s. All %d parties done.\n", sessionID, session.Parties)
	return nil
}

// routeMessages handles message exchange between all parties
func (s *KeyGenService) routeMessages(session *KeyGenSession) {
	// Create a map of index to partyId for reverse lookup
	idxToPartyId := make(map[int]string)
	for partyId, idx := range session.PartyIdToIdx {
		idxToPartyId[idx] = partyId
	}

	// Keep routing until all parties are done
	for {
		allDone := true
		for _, tp := range session.TssParties {
			if !tp.Done {
				allDone = false
				break
			}
		}
		if allDone {
			return
		}

		// Collect messages from all parties
		for partyId, tssParty := range session.TssParties {
			select {
			case msg := <-tssParty.OutCh:
				if msg == nil {
					continue
				}
				// Route message to recipients
				s.deliverMessage(session, partyId, msg, idxToPartyId)

			case saveData := <-tssParty.EndCh:
				// Party completed
				tssParty.SaveData = saveData
				tssParty.Done = true
				fmt.Printf("[KeyGen] Party %s completed keygen\n", partyId)

			case err := <-tssParty.ErrCh:
				fmt.Printf("[KeyGen] Party %s error: %v\n", partyId, err)
				return

			default:
				// No message, continue
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// deliverMessage routes a message to its recipients
func (s *KeyGenService) deliverMessage(session *KeyGenSession, fromPartyId string, msg tss.Message, idxToPartyId map[int]string) {
	// Get wire bytes
	msgBytes, _, err := msg.WireBytes()
	if err != nil {
		fmt.Printf("[KeyGen] Failed to serialize message: %v\n", err)
		return
	}

	dest := msg.GetTo()
	isBroadcast := dest == nil

	fromTssParty := session.TssParties[fromPartyId]

	if isBroadcast {
		// Send to all except sender
		for toPartyId, toTssParty := range session.TssParties {
			if toPartyId == fromPartyId {
				continue
			}
			parsedMsg, err := tss.ParseWireMessage(msgBytes, fromTssParty.PartyID, true)
			if err != nil {
				fmt.Printf("[KeyGen] Failed to parse broadcast message: %v\n", err)
				continue
			}
			if _, err := toTssParty.Party.Update(parsedMsg); err != nil {
				fmt.Printf("[KeyGen] Failed to update party %s: %v\n", toPartyId, err)
			}
		}
		fmt.Printf("[KeyGen] %s -> broadcast: %s\n", fromPartyId, msg.Type())
	} else {
		// Send to specific recipients
		for _, destPartyID := range dest {
			toPartyId := idxToPartyId[destPartyID.Index]
			toTssParty := session.TssParties[toPartyId]
			parsedMsg, err := tss.ParseWireMessage(msgBytes, fromTssParty.PartyID, false)
			if err != nil {
				fmt.Printf("[KeyGen] Failed to parse p2p message: %v\n", err)
				continue
			}
			if _, err := toTssParty.Party.Update(parsedMsg); err != nil {
				fmt.Printf("[KeyGen] Failed to update party %s: %v\n", toPartyId, err)
			}
		}
		fmt.Printf("[KeyGen] %s -> party %d: %s\n", fromPartyId, dest[0].Index, msg.Type())
	}
}

// GetSessionStatus returns the status of a session
func (s *KeyGenService) GetSessionStatus(sessionID string) (map[string]interface{}, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	completedCount := 0
	for _, tp := range session.TssParties {
		if tp.Done {
			completedCount++
		}
	}

	status := map[string]interface{}{
		"sessionId":        session.SessionID,
		"threshold":        session.Threshold,
		"parties":          session.Parties,
		"partyIds":         session.PartyIDs,
		"status":           session.Status,
		"completedParties": completedCount,
		"createdAt":        session.CreatedAt.Format(time.RFC3339),
	}

	if session.Error != "" {
		status["error"] = session.Error
	}

	return status, nil
}

// GetShare returns the key share for a specific party
func (s *KeyGenService) GetShare(sessionID string, partyId string) (map[string]interface{}, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.RLock()
	tssParty, exists := session.TssParties[partyId]
	session.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("party %s not found", partyId)
	}

	if !tssParty.Done || tssParty.SaveData == nil {
		return nil, fmt.Errorf("keygen not complete for party %s", partyId)
	}

	// Get public key (uncompressed format)
	pubKey := tssParty.SaveData.ECDSAPub
	pubKeyBytes := make([]byte, 65)
	pubKeyBytes[0] = 0x04
	pubKey.X().FillBytes(pubKeyBytes[1:33])
	pubKey.Y().FillBytes(pubKeyBytes[33:65])

	// Serialize share data
	shareData, err := json.Marshal(tssParty.SaveData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize share: %w", err)
	}

	return map[string]interface{}{
		"partyId":    partyId,
		"publicKey":  pubKeyBytes,
		"shareData":  shareData,
		"partyIndex": session.PartyIdToIdx[partyId],
	}, nil
}

// GetSession returns the keygen session (for signing service)
func (s *KeyGenService) GetSession(sessionID string) (*KeyGenSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	return session, nil
}

// KeygenServiceInstance is the global keygen service instance
var KeygenServiceInstance = NewKeyGenService()
