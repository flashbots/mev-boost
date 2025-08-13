package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const (
	MEVBoostURL   = "http://localhost:18550"
	BeaconNodeURL = "http://localhost:3500"
	RelayURL      = "http://localhost:5555"
	ExecutionURL  = "http://localhost:8545"

	RelaySecretKey       = "0x5eae315483f028b5cdd5d1090ff0c7618b18737ea9bf3c35047189db22835c48"
	ValidationPublickKey = "0x80a2be2c7dbce8ddc2eba03522697587c375a5a9e92d4b31ed9e3c34bee047095d93e3c70b1662b3faa301f5b19978e5" // Real validator from playground
	ValidationSignature  = "0x920daae6298681069a3cb7e1ff8cfd8dab0593eca11298388e2fae3eabe66249ba0d8218ff4df29b448506b006163e240dbe8fd9dd3a71439dd727406981a38a626fb883b19778bd2a5b1ae3d6ccaaf37079bfe3f572292b136f8f739c3f36a3"
	FeeRecipient         = "0x690b9a9e9aa1c9db991c7721a92d351db4fac990"

	// Test transaction parameters
	TestPrivateKey = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
	TestTimeout    = 30 * time.Second
)

type BeaconNodeClient struct {
	baseURL string
	client  *http.Client
}

func NewBeaconNodeClient(baseURL string) *BeaconNodeClient {
	return &BeaconNodeClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *BeaconNodeClient) GetCurrentSlot(ctx context.Context) (phase0.Slot, error) {
	resp, err := c.client.Get(c.baseURL + "/eth/v1/beacon/headers/head")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Header struct {
				Message struct {
					Slot string `json:"slot"`
				} `json:"message"`
			} `json:"header"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	var slot phase0.Slot
	if _, err := fmt.Sscanf(result.Data.Header.Message.Slot, "%d", &slot); err != nil {
		return 0, err
	}

	return slot, nil
}

func (c *BeaconNodeClient) GetBlockHeader(ctx context.Context, slot phase0.Slot) (*phase0.BeaconBlockHeader, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/headers/%d", c.baseURL, slot)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Header struct {
				Message *phase0.BeaconBlockHeader `json:"message"`
			} `json:"header"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data.Header.Message, nil
}

type MEVBoostClient struct {
	baseURL string
	client  *http.Client
}

func NewMEVBoostClient(baseURL string) *MEVBoostClient {
	return &MEVBoostClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *MEVBoostClient) CheckStatus(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/eth/v1/builder/status", nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// sendTestTransaction sends a test transaction to the execution layer
func sendTestTransaction(t *testing.T, ctx context.Context) common.Hash {
	t.Helper()

	// Connect to the execution layer
	client, err := ethclient.Dial(ExecutionURL)
	require.NoError(t, err, "Should be able to connect to execution layer")
	defer client.Close()

	// Parse private key
	privateKey, err := crypto.HexToECDSA(TestPrivateKey)
	require.NoError(t, err, "Should be able to parse private key")

	// Get public key and address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	require.True(t, ok, "Should be able to cast public key")
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	require.NoError(t, err, "Should be able to get nonce")

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(ctx)
	require.NoError(t, err, "Should be able to get gas price")

	// Create transaction (simple transfer to a different address)
	toAddress := common.HexToAddress("0x8ba1f109551bD432803012645Hac136c22C177c9")
	value := big.NewInt(1000000000000000) // 0.001 ETH
	gasLimit := uint64(21000)

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)

	// Get chain ID
	chainID, err := client.NetworkID(ctx)
	require.NoError(t, err, "Should be able to get chain ID")

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	require.NoError(t, err, "Should be able to sign transaction")

	// Send transaction
	err = client.SendTransaction(ctx, signedTx)
	require.NoError(t, err, "Should be able to send transaction")

	t.Logf("📤 Sent test transaction: %s", signedTx.Hash().Hex())
	return signedTx.Hash()
}

// waitForTransactionReceipt waits for a transaction receipt
func waitForTransactionReceipt(t *testing.T, ctx context.Context, txHash common.Hash) *types.Receipt {
	t.Helper()

	// Connect to the execution layer
	client, err := ethclient.Dial(ExecutionURL)
	require.NoError(t, err, "Should be able to connect to execution layer")
	defer client.Close()

	// Poll for receipt
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(60 * time.Second)

	for {
		select {
		case <-timeout:
			t.Fatalf("Transaction receipt not found after timeout: %s", txHash.Hex())
		case <-ticker.C:
			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err == nil && receipt != nil {
				t.Logf("📥 Transaction confirmed in block %d: %s", receipt.BlockNumber.Uint64(), txHash.Hex())
				return receipt
			}
			t.Logf("⏳ Waiting for transaction receipt: %s", txHash.Hex())
		}
	}
}

// waitForMEVBoost waits for MEV-boost to be available
func waitForMEVBoost(t *testing.T, timeout time.Duration) {
	t.Helper()

	client := NewMEVBoostClient(MEVBoostURL)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("MEV-boost not available at %s after %v", MEVBoostURL, timeout)
		case <-ticker.C:
			if err := client.CheckStatus(ctx); err == nil {
				t.Logf("MEV-boost is available at %s", MEVBoostURL)
				return
			}
		}
	}
}

// TestMEVBoostIntegration tests MEV-boost by observing the live system
func TestMEVBoostIntegration(t *testing.T) {
	// Skip this test if we're not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	// Wait for MEV-boost to be available
	waitForMEVBoost(t, 10*time.Second)

	// Initialize clients for observation
	beaconClient := NewBeaconNodeClient(BeaconNodeURL)
	relayClient := &http.Client{Timeout: 10 * time.Second}

	t.Logf("Starting MEV-boost integration test by observing live system...")
	t.Logf("Services: Beacon (%s), MEV-boost (%s), Relay (%s)", BeaconNodeURL, MEVBoostURL, RelayURL)

	// Test 1: Verify all services are healthy
	t.Run("Service health checks", func(t *testing.T) {
		// Check MEV-boost
		resp, err := relayClient.Get(MEVBoostURL + "/eth/v1/builder/status")
		require.NoError(t, err, "MEV-boost should be accessible")
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "MEV-boost should be healthy")
		t.Logf("MEV-boost is healthy")

		// Check Relay
		resp, err = relayClient.Get(RelayURL + "/eth/v1/builder/status")
		require.NoError(t, err, "Relay should be accessible")
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "Relay should be healthy")
		t.Logf("Relay is healthy")

		// Check Beacon Node
		currentSlot, err := beaconClient.GetCurrentSlot(ctx)
		require.NoError(t, err, "Beacon node should be accessible")
		require.Greater(t, uint64(currentSlot), uint64(0), "Beacon chain should be progressing")
		t.Logf("Beacon node is healthy (current slot: %d)", currentSlot)
	})

	// Test 2: Observe beacon chain block production
	t.Run("Beacon chain block production", func(t *testing.T) {
		initialSlot, err := beaconClient.GetCurrentSlot(ctx)
		require.NoError(t, err, "Should be able to get initial slot")

		t.Logf("Observing beacon chain from slot %d...", initialSlot)

		// wait to check if  new blocks are being produced
		time.Sleep(15 * time.Second)

		newSlot, err := beaconClient.GetCurrentSlot(ctx)
		require.NoError(t, err, "Should be able to get new slot")

		require.Greater(t, uint64(newSlot), uint64(initialSlot), "Beacon chain should be progressing")
		t.Logf("Beacon chain is producing blocks (slot %d → %d)", initialSlot, newSlot)

		// Try to get the block for the latest slot
		header, err := beaconClient.GetBlockHeader(ctx, newSlot)
		if err == nil && header != nil {
			t.Logf("Successfully retrieved block header for slot %d", newSlot)
		} else {
			t.Logf("⚠️ Could not retrieve block header for slot %d: %v", newSlot, err)
		}
	})

	// Test 3: Send test transaction and validate activity
	t.Run("Send test transaction and validate activity", func(t *testing.T) {
		t.Logf("🔍 Sending test transaction to create MEV opportunities...")
		
		// Send a test transaction to create activity
		txHash := sendTestTransaction(t, ctx)
		
		// Wait for the transaction to be confirmed
		receipt := waitForTransactionReceipt(t, ctx, txHash)
		require.NotNil(t, receipt, "Transaction should be confirmed")
		require.Equal(t, uint64(1), receipt.Status, "Transaction should be successful")
		
		t.Logf("✅ Test transaction confirmed in block %d", receipt.BlockNumber.Uint64())
		
		// Now check if relay has delivered payloads (should be active)
		t.Logf("🔍 Validating active builder and relay activity...")
		resp, err := relayClient.Get(RelayURL + "/relay/v1/data/bidtraces/proposer_payload_delivered")
		require.NoError(t, err, "Relay should be reachable for payload delivery data")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "Relay should return payload delivery data")

		var payloads []map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payloads))
		require.Greater(t, len(payloads), 0, "Relay should be actively delivering payloads in builder-playground environment")
		t.Logf("Relay has delivered %d payloads", len(payloads))

		// Check validator registrations (MEV-boost → relay communication)
		resp, err = relayClient.Get(RelayURL + "/relay/v1/data/validators")
		require.NoError(t, err, "Relay should be reachable for validator data")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "Relay should return validator data")

		var validators []map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&validators))
		require.Greater(t, len(validators), 0, "MEV-boost should have registered validators with the relay")
		t.Logf("Relay has %d registered validators", len(validators))
	})

	// Test 4: Validate MEV-boost is consistently building all blocks
	t.Run("MEV-boost consistent block building", func(t *testing.T) {
		t.Logf("Validating that MEV-boost is building all blocks (builder-playground environment)...")

		currentSlot, err := beaconClient.GetCurrentSlot(ctx)
		require.NoError(t, err, "Should be able to get current slot")

		fmt.Println("current slot ==>", currentSlot)
		mevBoostBlocks := 0
		totalBlocks := 0

		for i := 1; i <= 5; i++ {
			slotToCheck := currentSlot - phase0.Slot(i)
			if slotToCheck <= 0 {
				t.Logf("slot less then equal to zero")
				continue
			}

			// Get block details from beacon node
			resp, err := relayClient.Get(fmt.Sprintf("%s/eth/v2/beacon/blocks/%d", BeaconNodeURL, slotToCheck))
			if err != nil {
				continue
			}
			resp.Body.Close()

			require.Equal(t, resp.StatusCode, http.StatusOK)
			totalBlocks++
			t.Logf("Slot %d: Block found", slotToCheck)

			// In builder-playground, this block MUST have been delivered by the relay
			resp, err = relayClient.Get(fmt.Sprintf("%s/relay/v1/data/bidtraces/proposer_payload_delivered?slot=%d", RelayURL, slotToCheck))
			require.NoError(t, err, "Relay should be reachable for slot %d", slotToCheck)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "Relay should return payload delivery data for slot %d", slotToCheck)

			var deliveries []map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&deliveries))
			require.Greater(t, len(deliveries), 0, "Slot %d should have been built via MEV-boost (builder-playground should provide bids)", slotToCheck)

			mevBoostBlocks++
			t.Logf("Slot %d: Block was built via MEV-boost as expected", slotToCheck)
		}

		require.Greater(t, totalBlocks, 0, "Should have at least one block in recent slots")
		require.Equal(t, mevBoostBlocks, totalBlocks, "All blocks should be built via MEV-boost in builder-playground environment (no local fallback expected)")
	})
}
