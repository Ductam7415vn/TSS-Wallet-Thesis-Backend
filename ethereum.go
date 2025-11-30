package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

/**
 * Ethereum HTTP Handlers - Week 4 Implementation
 *
 * API Endpoints:
 *   GET  /ethereum/networks                     - List supported networks
 *   GET  /ethereum/:network/balance/:address    - Get ETH balance
 *   GET  /ethereum/:network/nonce/:address      - Get account nonce
 *   GET  /ethereum/:network/gas-price           - Get current gas price
 *   POST /ethereum/build-tx                     - Build unsigned transaction
 *   POST /ethereum/sign-tx                      - Sign transaction with TSS
 *   POST /ethereum/broadcast                    - Broadcast signed transaction
 *   GET  /ethereum/:network/tx/:txHash          - Get transaction receipt
 *
 * Flow:
 * 1. Build unsigned transaction (get hash to sign)
 * 2. Sign the hash using TSS signing
 * 3. Apply signature to transaction
 * 4. Broadcast to network
 */

// NetworkInfo for API response
type NetworkInfo struct {
	Name      string `json:"name"`
	Network   string `json:"network"`
	ChainID   int64  `json:"chainId"`
	Symbol    string `json:"symbol"`
	Explorer  string `json:"explorer"`
	IsTestnet bool   `json:"isTestnet"`
}

// GET /ethereum/networks
func GetNetworks(c *gin.Context) {
	networks := make([]NetworkInfo, 0)
	for key, config := range NetworkConfigs {
		networks = append(networks, NetworkInfo{
			Name:      config.Name,
			Network:   key,
			ChainID:   config.ChainID,
			Symbol:    config.Symbol,
			Explorer:  config.Explorer,
			IsTestnet: config.IsTestnet,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"networks": networks,
	})
}

// GET /ethereum/:network/balance/:address
func GetBalance(c *gin.Context) {
	network := c.Param("network")
	address := c.Param("address")

	balance, err := ethereumService.GetBalance(network, address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to Ether for display
	etherBalance := WeiToEther(balance)

	c.JSON(http.StatusOK, gin.H{
		"network":     network,
		"address":     address,
		"balanceWei":  balance.String(),
		"balanceEth":  etherBalance.Text('f', 18),
		"symbol":      "ETH",
	})
}

// GET /ethereum/:network/nonce/:address
func GetNonce(c *gin.Context) {
	network := c.Param("network")
	address := c.Param("address")

	nonce, err := ethereumService.GetNonce(network, address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"network": network,
		"address": address,
		"nonce":   nonce,
	})
}

// GET /ethereum/:network/gas-price
func GetGasPrice(c *gin.Context) {
	network := c.Param("network")

	gasPrice, err := ethereumService.GetGasPrice(network)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to Gwei for display
	gwei := new(big.Float).Quo(
		new(big.Float).SetInt(gasPrice),
		big.NewFloat(1e9),
	)

	c.JSON(http.StatusOK, gin.H{
		"network":      network,
		"gasPriceWei":  gasPrice.String(),
		"gasPriceGwei": gwei.Text('f', 2),
	})
}

// BuildTxRequest for building unsigned transaction
type BuildTxRequest struct {
	Network         string `json:"network" binding:"required"`
	KeygenSessionID string `json:"keygenSessionId" binding:"required"`
	To              string `json:"to" binding:"required"`
	ValueEth        string `json:"valueEth"`        // Amount in ETH (e.g., "0.1")
	ValueWei        string `json:"valueWei"`        // Amount in Wei (alternative)
	GasLimit        uint64 `json:"gasLimit"`        // Optional, will estimate
	GasPriceGwei    int64  `json:"gasPriceGwei"`    // Optional, will use suggested
	Data            string `json:"data"`            // Optional, hex encoded
}

// BuildTxResponse contains unsigned tx and hash to sign
type BuildTxResponse struct {
	TxID        string `json:"txId"`          // Unique ID for this unsigned tx
	Network     string `json:"network"`
	From        string `json:"from"`
	To          string `json:"to"`
	ValueWei    string `json:"valueWei"`
	ValueEth    string `json:"valueEth"`
	GasLimit    uint64 `json:"gasLimit"`
	GasPrice    string `json:"gasPriceWei"`
	GasPriceGwei string `json:"gasPriceGwei"`
	Nonce       uint64 `json:"nonce"`
	ChainID     int64  `json:"chainId"`
	HashToSign  string `json:"hashToSign"`    // 32-byte hash to sign (hex)
	Message     string `json:"message"`
}

// In-memory storage for unsigned transactions
var unsignedTxStore = make(map[string]*UnsignedTransaction)

// POST /ethereum/build-tx
func BuildTransaction(c *gin.Context) {
	var req BuildTxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get address from keygen session
	_, fromAddress, err := GetPublicKeyFromSession(req.KeygenSessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse value
	var valueWei *big.Int
	if req.ValueWei != "" {
		valueWei = new(big.Int)
		if _, ok := valueWei.SetString(req.ValueWei, 10); !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valueWei"})
			return
		}
	} else if req.ValueEth != "" {
		ethFloat, _, err := big.ParseFloat(req.ValueEth, 10, 256, big.ToNearestEven)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valueEth"})
			return
		}
		valueWei = EtherToWei(ethFloat)
	} else {
		valueWei = big.NewInt(0)
	}

	// Parse gas price
	var gasPriceWei *big.Int
	if req.GasPriceGwei > 0 {
		gasPriceWei = GweiToWei(req.GasPriceGwei)
	}

	// Parse data
	var data []byte
	if req.Data != "" {
		dataHex := strings.TrimPrefix(req.Data, "0x")
		data, err = hex.DecodeString(dataHex)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data hex"})
			return
		}
	}

	// Build unsigned transaction
	unsignedTx, hashToSign, err := ethereumService.BuildUnsignedTransaction(
		req.Network,
		fromAddress,
		req.To,
		valueWei,
		req.GasLimit,
		gasPriceWei,
		data,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate tx ID and store
	txID := fmt.Sprintf("tx_%d", unsignedTx.Nonce)
	unsignedTxStore[txID] = unsignedTx

	// Calculate display values
	ethValue := WeiToEther(valueWei)
	gasPriceGwei := new(big.Float).Quo(
		new(big.Float).SetInt(unsignedTx.GasPrice),
		big.NewFloat(1e9),
	)

	c.JSON(http.StatusOK, BuildTxResponse{
		TxID:         txID,
		Network:      req.Network,
		From:         fromAddress,
		To:           req.To,
		ValueWei:     valueWei.String(),
		ValueEth:     ethValue.Text('f', 18),
		GasLimit:     unsignedTx.GasLimit,
		GasPrice:     unsignedTx.GasPrice.String(),
		GasPriceGwei: gasPriceGwei.Text('f', 2),
		Nonce:        unsignedTx.Nonce,
		ChainID:      unsignedTx.ChainID.Int64(),
		HashToSign:   hex.EncodeToString(hashToSign),
		Message:      "Use the hashToSign with /signing/start to get the signature",
	})
}

// SignTxRequest for signing a built transaction
type SignTxRequest struct {
	TxID            string   `json:"txId" binding:"required"`
	KeygenSessionID string   `json:"keygenSessionId" binding:"required"`
	SignerIDs       []string `json:"signerIds" binding:"required"`
}

// SignTxResponse contains the signed transaction
type SignTxResponse struct {
	TxID       string `json:"txId"`
	SigningID  string `json:"signingId"`
	RawTx      string `json:"rawTx"`
	TxHash     string `json:"txHash"`
	From       string `json:"from"`
	To         string `json:"to"`
	ValueWei   string `json:"valueWei"`
	GasLimit   uint64 `json:"gasLimit"`
	GasPrice   string `json:"gasPriceWei"`
	Nonce      uint64 `json:"nonce"`
	ChainID    int64  `json:"chainId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// POST /ethereum/sign-tx
func SignTransaction(c *gin.Context) {
	var req SignTxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get unsigned transaction
	unsignedTx, exists := unsignedTxStore[req.TxID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found, build first"})
		return
	}

	// Get the hash to sign
	_, hashToSign, err := ethereumService.BuildUnsignedTransaction(
		unsignedTx.Network,
		unsignedTx.From,
		unsignedTx.To,
		unsignedTx.Value,
		unsignedTx.GasLimit,
		unsignedTx.GasPrice,
		unsignedTx.Data,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create signing session
	signingID := fmt.Sprintf("ethsign_%s", req.TxID)
	if err := signingService.CreateAndExecuteSigning(
		signingID,
		req.KeygenSessionID,
		hashToSign,
		req.SignerIDs,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     err.Error(),
			"signingId": signingID,
			"status":    "failed",
		})
		return
	}

	// Get signature
	sig, err := signingService.GetSignature(signingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Apply signature to transaction
	signedTx, err := ethereumService.ApplySignature(unsignedTx, sig.R, sig.S, sig.V)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SignTxResponse{
		TxID:      req.TxID,
		SigningID: signingID,
		RawTx:     signedTx.RawTx,
		TxHash:    signedTx.TxHash,
		From:      signedTx.From,
		To:        signedTx.To,
		ValueWei:  signedTx.Value,
		GasLimit:  signedTx.GasLimit,
		GasPrice:  signedTx.GasPrice,
		Nonce:     signedTx.Nonce,
		ChainID:   signedTx.ChainID,
		Status:    "signed",
		Message:   "Transaction signed. Use /ethereum/broadcast to send to network.",
	})
}

// BroadcastRequest for broadcasting a signed transaction
type BroadcastRequest struct {
	Network string `json:"network" binding:"required"`
	RawTx   string `json:"rawTx" binding:"required"`
}

// POST /ethereum/broadcast
func BroadcastTransaction(c *gin.Context) {
	var req BroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txHash, err := ethereumService.BroadcastTransaction(req.Network, req.RawTx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	config := NetworkConfigs[req.Network]
	explorerURL := fmt.Sprintf("%s/tx/%s", config.Explorer, txHash)

	c.JSON(http.StatusOK, gin.H{
		"network":     req.Network,
		"txHash":      txHash,
		"explorerUrl": explorerURL,
		"status":      "broadcasted",
		"message":     "Transaction submitted to network. Check explorer for confirmation.",
	})
}

// GET /ethereum/:network/tx/:txHash
func GetTransactionReceipt(c *gin.Context) {
	network := c.Param("network")
	txHash := c.Param("txHash")

	receipt, err := ethereumService.GetTransactionReceipt(network, txHash)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"status":  "pending",
			"message": "Transaction may still be pending",
		})
		return
	}

	config := NetworkConfigs[network]
	explorerURL := fmt.Sprintf("%s/tx/%s", config.Explorer, txHash)

	status := "failed"
	if receipt.Status == 1 {
		status = "success"
	}

	c.JSON(http.StatusOK, gin.H{
		"network":      network,
		"txHash":       txHash,
		"blockNumber":  receipt.BlockNumber.Uint64(),
		"gasUsed":      receipt.GasUsed,
		"status":       status,
		"explorerUrl":  explorerURL,
	})
}

// QuickSendRequest for one-step send (build + sign + broadcast)
type QuickSendRequest struct {
	Network         string   `json:"network" binding:"required"`
	KeygenSessionID string   `json:"keygenSessionId" binding:"required"`
	SignerIDs       []string `json:"signerIds" binding:"required"`
	To              string   `json:"to" binding:"required"`
	ValueEth        string   `json:"valueEth"`
	ValueWei        string   `json:"valueWei"`
	GasPriceGwei    int64    `json:"gasPriceGwei"`
}

// POST /ethereum/quick-send
func QuickSend(c *gin.Context) {
	var req QuickSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get address from keygen session
	_, fromAddress, err := GetPublicKeyFromSession(req.KeygenSessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse value
	var valueWei *big.Int
	if req.ValueWei != "" {
		valueWei = new(big.Int)
		if _, ok := valueWei.SetString(req.ValueWei, 10); !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valueWei"})
			return
		}
	} else if req.ValueEth != "" {
		ethFloat, _, err := big.ParseFloat(req.ValueEth, 10, 256, big.ToNearestEven)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valueEth"})
			return
		}
		valueWei = EtherToWei(ethFloat)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valueEth or valueWei required"})
		return
	}

	// Parse gas price
	var gasPriceWei *big.Int
	if req.GasPriceGwei > 0 {
		gasPriceWei = GweiToWei(req.GasPriceGwei)
	}

	// Step 1: Build transaction
	unsignedTx, hashToSign, err := ethereumService.BuildUnsignedTransaction(
		req.Network,
		fromAddress,
		req.To,
		valueWei,
		0, // auto estimate gas
		gasPriceWei,
		nil, // no data
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "build failed: " + err.Error()})
		return
	}

	// Step 2: Sign with TSS
	signingID := fmt.Sprintf("quicksend_%d", unsignedTx.Nonce)
	if err := signingService.CreateAndExecuteSigning(
		signingID,
		req.KeygenSessionID,
		hashToSign,
		req.SignerIDs,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "signing failed: " + err.Error()})
		return
	}

	sig, err := signingService.GetSignature(signingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get signature failed: " + err.Error()})
		return
	}

	// Step 3: Apply signature
	signedTx, err := ethereumService.ApplySignature(unsignedTx, sig.R, sig.S, sig.V)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "apply signature failed: " + err.Error()})
		return
	}

	// Step 4: Broadcast
	txHash, err := ethereumService.BroadcastTransaction(req.Network, signedTx.RawTx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "broadcast failed: " + err.Error(),
			"rawTx":   signedTx.RawTx,
			"message": "Transaction signed but broadcast failed. You can retry with /ethereum/broadcast",
		})
		return
	}

	config := NetworkConfigs[req.Network]
	explorerURL := fmt.Sprintf("%s/tx/%s", config.Explorer, txHash)
	ethValue := WeiToEther(valueWei)

	c.JSON(http.StatusOK, gin.H{
		"network":     req.Network,
		"txHash":      txHash,
		"from":        fromAddress,
		"to":          req.To,
		"valueEth":    ethValue.Text('f', 18),
		"valueWei":    valueWei.String(),
		"gasLimit":    signedTx.GasLimit,
		"nonce":       signedTx.Nonce,
		"explorerUrl": explorerURL,
		"status":      "broadcasted",
		"message":     "Transaction sent successfully!",
	})
}

// RegisterEthereumRoutes registers all Ethereum routes
func RegisterEthereumRoutes(router *gin.Engine) {
	eth := router.Group("/ethereum")
	{
		// Network info
		eth.GET("/networks", GetNetworks)

		// Account queries
		eth.GET("/:network/balance/:address", GetBalance)
		eth.GET("/:network/nonce/:address", GetNonce)
		eth.GET("/:network/gas-price", GetGasPrice)

		// Transaction flow
		eth.POST("/build-tx", BuildTransaction)
		eth.POST("/sign-tx", SignTransaction)
		eth.POST("/broadcast", BroadcastTransaction)

		// Quick send (all-in-one)
		eth.POST("/quick-send", QuickSend)

		// Transaction status
		eth.GET("/:network/tx/:txHash", GetTransactionReceipt)
	}
}
