package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
)

/**
 * Ethereum Service - Week 4 Implementation
 *
 * Features:
 * - Connect to Ethereum networks (Sepolia testnet, Mainnet)
 * - Build unsigned transactions
 * - Apply TSS signatures to transactions
 * - Broadcast signed transactions
 * - Query balance, nonce, gas price
 */

// Network configurations
var NetworkConfigs = map[string]NetworkConfig{
	"sepolia": {
		Name:      "Sepolia Testnet",
		ChainID:   11155111,
		RPC:       "https://rpc.sepolia.org",
		Explorer:  "https://sepolia.etherscan.io",
		Symbol:    "ETH",
		IsTestnet: true,
	},
	"goerli": {
		Name:      "Goerli Testnet",
		ChainID:   5,
		RPC:       "https://rpc.ankr.com/eth_goerli",
		Explorer:  "https://goerli.etherscan.io",
		Symbol:    "ETH",
		IsTestnet: true,
	},
	"mainnet": {
		Name:      "Ethereum Mainnet",
		ChainID:   1,
		RPC:       "https://eth.llamarpc.com",
		Explorer:  "https://etherscan.io",
		Symbol:    "ETH",
		IsTestnet: false,
	},
}

type NetworkConfig struct {
	Name      string
	ChainID   int64
	RPC       string
	Explorer  string
	Symbol    string
	IsTestnet bool
}

// EthereumService manages Ethereum operations
type EthereumService struct {
	clients map[string]*ethclient.Client
	mu      sync.RWMutex
}

func NewEthereumService() *EthereumService {
	return &EthereumService{
		clients: make(map[string]*ethclient.Client),
	}
}

// getClient returns or creates an Ethereum client for the network
func (s *EthereumService) getClient(network string) (*ethclient.Client, *NetworkConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, exists := NetworkConfigs[network]
	if !exists {
		return nil, nil, fmt.Errorf("unknown network: %s", network)
	}

	if client, ok := s.clients[network]; ok {
		return client, &config, nil
	}

	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to %s: %w", network, err)
	}

	s.clients[network] = client
	return client, &config, nil
}

// GetBalance returns the balance of an address in Wei
func (s *EthereumService) GetBalance(network, address string) (*big.Int, error) {
	client, _, err := s.getClient(network)
	if err != nil {
		return nil, err
	}

	addr := common.HexToAddress(address)
	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// GetNonce returns the next nonce for an address
func (s *EthereumService) GetNonce(network, address string) (uint64, error) {
	client, _, err := s.getClient(network)
	if err != nil {
		return 0, err
	}

	addr := common.HexToAddress(address)
	nonce, err := client.PendingNonceAt(context.Background(), addr)
	if err != nil {
		return 0, fmt.Errorf("failed to get nonce: %w", err)
	}

	return nonce, nil
}

// GetGasPrice returns the current gas price
func (s *EthereumService) GetGasPrice(network string) (*big.Int, error) {
	client, _, err := s.getClient(network)
	if err != nil {
		return nil, err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	return gasPrice, nil
}

// EstimateGas estimates gas for a transaction
func (s *EthereumService) EstimateGas(network, from, to string, value *big.Int, data []byte) (uint64, error) {
	client, _, err := s.getClient(network)
	if err != nil {
		return 0, err
	}

	fromAddr := common.HexToAddress(from)
	toAddr := common.HexToAddress(to)

	msg := ethereum.CallMsg{
		From:  fromAddr,
		To:    &toAddr,
		Value: value,
		Data:  data,
	}

	gas, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %w", err)
	}

	return gas, nil
}

// UnsignedTransaction represents a transaction to be signed
type UnsignedTransaction struct {
	Network  string   `json:"network"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Value    *big.Int `json:"value"`    // in Wei
	GasLimit uint64   `json:"gasLimit"`
	GasPrice *big.Int `json:"gasPrice"` // in Wei
	Nonce    uint64   `json:"nonce"`
	Data     []byte   `json:"data"`
	ChainID  *big.Int `json:"chainId"`
}

// BuildUnsignedTransaction creates an unsigned transaction
func (s *EthereumService) BuildUnsignedTransaction(
	network string,
	from string,
	to string,
	valueWei *big.Int,
	gasLimit uint64,
	gasPriceWei *big.Int,
	data []byte,
) (*UnsignedTransaction, []byte, error) {
	client, config, err := s.getClient(network)
	if err != nil {
		return nil, nil, err
	}

	// Get nonce
	fromAddr := common.HexToAddress(from)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price if not provided
	if gasPriceWei == nil {
		gasPriceWei, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get gas price: %w", err)
		}
	}

	// Estimate gas if not provided
	if gasLimit == 0 {
		toAddr := common.HexToAddress(to)
		msg := ethereum.CallMsg{
			From:  fromAddr,
			To:    &toAddr,
			Value: valueWei,
			Data:  data,
		}
		gasLimit, err = client.EstimateGas(context.Background(), msg)
		if err != nil {
			// Default gas limit for simple transfer
			gasLimit = 21000
		}
		// Add 20% buffer
		gasLimit = gasLimit * 120 / 100
	}

	chainID := big.NewInt(config.ChainID)

	// Create transaction
	toAddr := common.HexToAddress(to)
	tx := types.NewTransaction(nonce, toAddr, valueWei, gasLimit, gasPriceWei, data)

	// Get the signer hash (EIP-155)
	signer := types.NewEIP155Signer(chainID)
	hash := signer.Hash(tx)

	unsignedTx := &UnsignedTransaction{
		Network:  network,
		From:     from,
		To:       to,
		Value:    valueWei,
		GasLimit: gasLimit,
		GasPrice: gasPriceWei,
		Nonce:    nonce,
		Data:     data,
		ChainID:  chainID,
	}

	return unsignedTx, hash.Bytes(), nil
}

// SignedTransactionResult contains the signed transaction data
type SignedTransactionResult struct {
	RawTx     string `json:"rawTx"`     // RLP encoded signed tx (hex)
	TxHash    string `json:"txHash"`    // Transaction hash
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`     // in Wei
	GasLimit  uint64 `json:"gasLimit"`
	GasPrice  string `json:"gasPrice"`  // in Wei
	Nonce     uint64 `json:"nonce"`
	ChainID   int64  `json:"chainId"`
}

// ApplySignature applies TSS signature to unsigned transaction
func (s *EthereumService) ApplySignature(
	unsignedTx *UnsignedTransaction,
	r, sigS []byte,
	v int,
) (*SignedTransactionResult, error) {
	// Recreate the transaction
	toAddr := common.HexToAddress(unsignedTx.To)
	tx := types.NewTransaction(
		unsignedTx.Nonce,
		toAddr,
		unsignedTx.Value,
		unsignedTx.GasLimit,
		unsignedTx.GasPrice,
		unsignedTx.Data,
	)

	// Create signature bytes (65 bytes: R || S || V)
	// For EIP-155, V = chainId * 2 + 35 + recoveryId
	// recoveryId is 0 or 1
	recoveryId := v - 27 // Convert from Ethereum format (27/28) to raw (0/1)
	if recoveryId < 0 || recoveryId > 1 {
		return nil, fmt.Errorf("invalid recovery id: %d", v)
	}

	// Calculate EIP-155 V value
	chainID := unsignedTx.ChainID.Int64()
	eip155V := chainID*2 + 35 + int64(recoveryId)

	// Create R, S as big.Int
	rInt := new(big.Int).SetBytes(r)
	sInt := new(big.Int).SetBytes(sigS)
	vInt := big.NewInt(eip155V)

	// Apply signature to transaction
	signer := types.NewEIP155Signer(unsignedTx.ChainID)
	signedTx, err := tx.WithSignature(signer, append(append(r, sigS...), byte(recoveryId)))
	if err != nil {
		return nil, fmt.Errorf("failed to apply signature: %w", err)
	}

	// Encode to RLP
	rawTxBytes, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction: %w", err)
	}

	// Verify the signature
	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to recover sender: %w", err)
	}

	expectedFrom := common.HexToAddress(unsignedTx.From)
	if sender != expectedFrom {
		return nil, fmt.Errorf("signature verification failed: expected %s, got %s", expectedFrom.Hex(), sender.Hex())
	}

	// Suppress unused variable warnings
	_ = rInt
	_ = sInt
	_ = vInt

	return &SignedTransactionResult{
		RawTx:    "0x" + hex.EncodeToString(rawTxBytes),
		TxHash:   signedTx.Hash().Hex(),
		From:     unsignedTx.From,
		To:       unsignedTx.To,
		Value:    unsignedTx.Value.String(),
		GasLimit: unsignedTx.GasLimit,
		GasPrice: unsignedTx.GasPrice.String(),
		Nonce:    unsignedTx.Nonce,
		ChainID:  chainID,
	}, nil
}

// BroadcastTransaction sends a signed transaction to the network
func (s *EthereumService) BroadcastTransaction(network string, rawTxHex string) (string, error) {
	client, config, err := s.getClient(network)
	if err != nil {
		return "", err
	}

	// Remove 0x prefix if present
	rawTxHex = strings.TrimPrefix(rawTxHex, "0x")

	// Decode the raw transaction
	rawTxBytes, err := hex.DecodeString(rawTxHex)
	if err != nil {
		return "", fmt.Errorf("invalid raw transaction hex: %w", err)
	}

	// Decode RLP
	var tx types.Transaction
	if err := rlp.DecodeBytes(rawTxBytes, &tx); err != nil {
		return "", fmt.Errorf("failed to decode transaction: %w", err)
	}

	// Send the transaction
	if err := client.SendTransaction(context.Background(), &tx); err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	txHash := tx.Hash().Hex()
	explorerURL := fmt.Sprintf("%s/tx/%s", config.Explorer, txHash)

	fmt.Printf("[Ethereum] Transaction broadcast to %s: %s\n", network, txHash)
	fmt.Printf("[Ethereum] Explorer: %s\n", explorerURL)

	return txHash, nil
}

// GetTransactionReceipt gets the receipt for a transaction
func (s *EthereumService) GetTransactionReceipt(network, txHash string) (*types.Receipt, error) {
	client, _, err := s.getClient(network)
	if err != nil {
		return nil, err
	}

	hash := common.HexToHash(txHash)
	receipt, err := client.TransactionReceipt(context.Background(), hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	return receipt, nil
}

// WeiToEther converts Wei to Ether
func WeiToEther(wei *big.Int) *big.Float {
	fWei := new(big.Float).SetInt(wei)
	ethValue := new(big.Float).Quo(fWei, big.NewFloat(1e18))
	return ethValue
}

// EtherToWei converts Ether to Wei
func EtherToWei(ether *big.Float) *big.Int {
	weiFloat := new(big.Float).Mul(ether, big.NewFloat(1e18))
	wei, _ := weiFloat.Int(nil)
	return wei
}

// GweiToWei converts Gwei to Wei
func GweiToWei(gwei int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(gwei), big.NewInt(1e9))
}

// DeriveAddressFromPublicKey derives Ethereum address from public key bytes
func DeriveAddressFromPublicKey(pubKeyBytes []byte) (string, error) {
	if len(pubKeyBytes) != 65 {
		return "", fmt.Errorf("invalid public key length: expected 65, got %d", len(pubKeyBytes))
	}

	// pubKeyBytes is 65 bytes: 0x04 || X (32 bytes) || Y (32 bytes)
	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	address := crypto.PubkeyToAddress(*pubKey)
	return address.Hex(), nil
}

// RecoverAddressFromSignature recovers the signer address from a message and signature
func RecoverAddressFromSignature(messageHash []byte, signature []byte) (string, error) {
	if len(signature) != 65 {
		return "", fmt.Errorf("invalid signature length: expected 65, got %d", len(signature))
	}

	// Adjust V value if needed (Ethereum uses 27/28, crypto.Ecrecover uses 0/1)
	sig := make([]byte, 65)
	copy(sig, signature)
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	// Recover public key
	pubKey, err := crypto.Ecrecover(messageHash, sig)
	if err != nil {
		return "", fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert to ecdsa.PublicKey
	ecdsaPubKey, err := crypto.UnmarshalPubkey(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	address := crypto.PubkeyToAddress(*ecdsaPubKey)
	return address.Hex(), nil
}

// VerifySignature verifies a signature against an address
func VerifySignature(address string, messageHash []byte, signature []byte) (bool, error) {
	recoveredAddr, err := RecoverAddressFromSignature(messageHash, signature)
	if err != nil {
		return false, err
	}

	expectedAddr := common.HexToAddress(address)
	recoveredAddress := common.HexToAddress(recoveredAddr)

	return expectedAddr == recoveredAddress, nil
}

// Global ethereum service instance
var ethereumService = NewEthereumService()

// Helper to get public key from keygen session
func GetPublicKeyFromSession(keygenSessionID string) (*ecdsa.PublicKey, string, error) {
	session, err := keygenService.getSession(keygenSessionID)
	if err != nil {
		return nil, "", fmt.Errorf("keygen session not found: %w", err)
	}

	if session.Status != "completed" {
		return nil, "", fmt.Errorf("keygen session not completed")
	}

	// Get public key from any party (all have same public key)
	for _, party := range session.tssParties {
		if party.SaveData != nil {
			pubKey := party.SaveData.ECDSAPub.ToECDSAPubKey()
			address := crypto.PubkeyToAddress(*pubKey)
			return pubKey, address.Hex(), nil
		}
	}

	return nil, "", fmt.Errorf("no public key found in session")
}
