package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/stretchr/testify/require"
)

const (
	MEVBoostURL   = "http://localhost:18550"
	BeaconNodeURL = "http://localhost:3500"
	RelayURL      = "http://localhost:5555"
	ExecutionURL  = "http://localhost:8545"

	RelaySecretKey       = "0x5eae315483f028b5cdd5d1090ff0c7618b18737ea9bf3c35047189db22835c48"
	ValidationPublickKey = "0x80a2be2c7dbce8ddc2eba03522697587c375a5a9e92d4b31ed9e3c34bee047095d93e3c70b1662b3faa301f5b19978e5" // Real validator from playground
	TestTimeout          = 30 * time.Second
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

// getScheduledValidatorForSlot gets the validator public key scheduled to propose for a specific slot
func getScheduledValidatorForSlot(ctx context.Context, client *BeaconNodeClient, slot phase0.Slot) (string, error) {
	// Calculate epoch from slot (32 slots per epoch)
	epoch := slot / 32

	// Get validator duties for the epoch
	url := fmt.Sprintf("%s/eth/v1/validator/duties/proposer/%d", client.baseURL, epoch)
	resp, err := client.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get proposer duties: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("proposer duties request failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Pubkey string `json:"pubkey"`
			Slot   string `json:"slot"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode proposer duties: %w", err)
	}

	// Find the duty for our specific slot
	targetSlotStr := fmt.Sprintf("%d", slot)
	for _, duty := range result.Data {
		if duty.Slot == targetSlotStr {
			return duty.Pubkey, nil
		}
	}

	return "", fmt.Errorf("no proposer found for slot %d", slot)
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

	// Get testing parameters from environment (set by CI matrix)
	testingFork := os.Getenv("TESTING_FORK")
	if testingFork == "" {
		testingFork = "unknown"
	}
	testingTxType := os.Getenv("TESTING_TX_TYPE")
	if testingTxType == "" {
		testingTxType = "unknown"
	}

	t.Logf("Starting MEV-boost integration test by observing live system...")
	t.Logf("Testing Fork: %s", testingFork)
	t.Logf("Testing Transaction Type: %s", testingTxType)
	t.Logf("Services: Beacon (%s), MEV-boost (%s), Relay (%s)", BeaconNodeURL, MEVBoostURL, RelayURL)

	// Initialize validator keys for testing (BLS version for tests)
	initValidatorsWithBLS()

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

	// Test 3: Validate blockchain activity and transaction types
	t.Run("Validate blockchain activity and transaction types", func(t *testing.T) {
		t.Logf("🔍 Validating blockchain activity and transaction type support...")

		// Check for relay activity (should have delivered payloads)
		t.Logf("🔍 Validating active builder and relay activity...")
		resp, err := relayClient.Get(RelayURL + "/relay/v1/data/bidtraces/proposer_payload_delivered")
		require.NoError(t, err, "Relay should be reachable for payload delivery data")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "Relay should return payload delivery data")

		var payloads []map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payloads))
		require.Greater(t, len(payloads), 0, "Relay should be actively delivering payloads in builder-playground environment")
		t.Logf("Relay has delivered %d payloads", len(payloads))

		// For blob transaction testing, check if recent blocks contain blob transactions
		if testingTxType == "blobs" {
			t.Logf("🔍 Checking for blob transactions in recent blocks...")

			currentSlot, err := beaconClient.GetCurrentSlot(ctx)
			require.NoError(t, err, "Should be able to get current slot")

			blobTxFound := false
			totalBlobGasUsed := uint64(0)

			// Check the last 5 blocks for blob transactions
			for i := 0; i < 15; i++ {
				blockNumber := fmt.Sprintf("0x%x", uint64(currentSlot)-uint64(i))

				// Get block details from execution layer
				blockResp, err := http.Post(ExecutionURL, "application/json",
					strings.NewReader(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["%s",true],"id":1}`, blockNumber)))
				if err != nil {
					t.Log("err encountered while tryong to look for block")
					fmt.Println("err", err)
					continue
				}
				defer blockResp.Body.Close()

				var blockResult struct {
					Result struct {
						BlobGasUsed   string                   `json:"blobGasUsed"`
						ExcessBlobGas string                   `json:"excessBlobGas"`
						Transactions  []map[string]interface{} `json:"transactions"`
					} `json:"result"`
				}

				if err := json.NewDecoder(blockResp.Body).Decode(&blockResult); err != nil {
					t.Log("err encountered while decoding block response")
					fmt.Println("err", err)
					continue
				}

				if blockResult.Result.BlobGasUsed != "" && blockResult.Result.BlobGasUsed != "0x0" {
					blobGasUsed, _ := strconv.ParseUint(strings.TrimPrefix(blockResult.Result.BlobGasUsed, "0x"), 16, 64)
					if blobGasUsed > 0 {
						blobTxFound = true
						totalBlobGasUsed += blobGasUsed
						t.Logf("✅ Found blob transactions in block %s: %d blob gas used", blockNumber, blobGasUsed)

						// Count blob transactions in this block
						blobTxCount := 0
						for _, tx := range blockResult.Result.Transactions {
							if txType, exists := tx["type"]; exists && txType == "0x3" { // EIP-4844 blob tx type
								blobTxCount++
							}
						}
						t.Logf("   Block contains %d blob transactions", blobTxCount)
					}
				} else {
					t.Logf("no blobs for found for blockNumber %d", blockNumber)
				}
			}

			if blobTxFound {
				t.Logf("✅ Successfully detected blob transaction activity (total blob gas: %d)", totalBlobGasUsed)
			} else {
				t.Logf("⚠️  No blob transactions found in recent blocks (may need more time for contender to generate blobs)")
			}
		}
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
			t.Logf("deliveries %d: length", len(deliveries))
			require.Greater(t, len(deliveries), 0, "Slot %d should have been built via MEV-boost (builder-playground should provide bids)", slotToCheck)

			mevBoostBlocks++
			t.Logf("Slot %d: Block was built via MEV-boost as expected", slotToCheck)
		}

		require.Greater(t, totalBlocks, 0, "Should have at least one block in recent slots")
		require.Equal(t, mevBoostBlocks, totalBlocks, "All blocks should be built via MEV-boost in builder-playground environment (no local fallback expected)")
	})

	t.Run("status check", func(t *testing.T) {
		resp, err := relayClient.Get(MEVBoostURL + params.PathStatus)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("request validation on invalid pub key", func(t *testing.T) {
		resp, err := relayClient.Get(MEVBoostURL + "/eth/v1/builder/header/1/0x0000000000000000000000000000000000000000000000000000000000000000/invalid_pubkey")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MEV-boost bid selection", func(t *testing.T) {
		t.Logf("Testing MEV-boost bid selection with real validator and parent hash...")

		currentSlot, err := beaconClient.GetCurrentSlot(ctx)
		require.NoError(t, err, "Should be able to get current slot")

		currentSlot = currentSlot - 1
		// Get the actual parent hash from the current block
		currentHeader, err := beaconClient.GetBlockHeader(ctx, currentSlot)
		require.NoError(t, err, "Should be able to get current block header")

		parentHash := fmt.Sprintf("0x%x", currentHeader.ParentRoot)
		t.Logf("Using real parent hash: %s", parentHash)

		// // Get the scheduled validator for the next slot
		// futureSlot := currentSlot + 1
		scheduledValidator, err := getScheduledValidatorForSlot(ctx, beaconClient, currentSlot)
		require.NoError(t, err, "Should be able to get scheduled validator for slot %d", currentSlot)

		t.Logf("Scheduled validator for slot %d: %s", currentSlot, scheduledValidator)

		// Test bid retrieval with real validator and parent hash
		url := fmt.Sprintf("%s/eth/v1/builder/header/%d/%s/%s",
			MEVBoostURL, currentSlot, parentHash, scheduledValidator)

		t.Logf("Requesting bid: slot=%d, parent=%s, validator=%s", currentSlot, parentHash, scheduledValidator)

		resp, err := relayClient.Get(url)
		require.NoError(t, err, "Should handle header requests")
		defer resp.Body.Close()

		// Response can be 204 (no bid) or 200 (bid available)
		require.True(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK,
			"Should return either 204 (no bid) or 200 (bid available), got %d", resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			t.Logf("✅ MEV-boost returned bid for slot %d with real validator", currentSlot)

			// Parse and validate the bid response
			var bidResponse map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&bidResponse)
			require.NoError(t, err, "Should be able to parse bid response")

			// Validate bid structure
			require.Contains(t, bidResponse, "data", "Bid response should contain data field")

			data, ok := bidResponse["data"].(map[string]interface{})
			require.True(t, ok, "Data field should be an object")

			// Check for key bid fields
			if message, exists := data["message"]; exists {
				messageObj, ok := message.(map[string]interface{})
				require.True(t, ok, "Message field should be an object")

				// Validate bid contains expected fields
				require.Contains(t, messageObj, "header", "Bid should contain header")
				require.Contains(t, messageObj, "value", "Bid should contain value")
				require.Contains(t, messageObj, "pubkey", "Bid should contain pubkey")

				// Log bid value for debugging
				if value, exists := messageObj["value"]; exists {
					t.Logf("Bid value: %v", value)
				}
			}

			t.Logf("✅ Bid response structure validated")
		} else {
			t.Logf("✅ MEV-boost correctly returned no-bid (204) for slot %d", currentSlot)
			t.Logf("This is expected if no builders have submitted bids for this slot/parent combination")
		}

		// Test with current slot too (should typically have no bid since it's already being built)
		url = fmt.Sprintf("%s/eth/v1/builder/header/%d/%s/%s",
			MEVBoostURL, currentSlot, parentHash, scheduledValidator)

		resp, err = relayClient.Get(url)
		require.NoError(t, err, "Should handle header requests for current slot")
		defer resp.Body.Close()

		// Current slot should typically return 204 (no bid) since it's being built already
		t.Logf("Current slot %d bid status: %d", currentSlot, resp.StatusCode)
	})

	// t.Run("MEV-boost payload delivery", func(t *testing.T) {
	// 	t.Logf("Testing MEV-boost payload delivery mechanism...")

	// 	// Test the payload delivery endpoint structure
	// 	// Note: We're not actually submitting a blind block, just testing the endpoint exists
	// 	url := MEVBoostURL + "/eth/v1/builder/blinded_blocks"

	// 	// Make a HEAD request to check if endpoint exists without actually submitting
	// 	req, err := http.NewRequest("HEAD", url, nil)
	// 	require.NoError(t, err, "Should be able to create HEAD request")

	// 	resp, err := relayClient.Do(req)
	// 	require.NoError(t, err, "Should be able to reach payload delivery endpoint")
	// 	defer resp.Body.Close()

	// 	// Endpoint should exist (even if it returns error for empty request)
	// 	require.True(t, resp.StatusCode != http.StatusNotFound, "Payload delivery endpoint should exist")
	// 	t.Logf("✅ MEV-boost payload delivery endpoint is accessible")
	// })

	t.Run("MEV-boost performance", func(t *testing.T) {
		// testing concurrent calls
		concurrentRequests := 5
		var wg sync.WaitGroup
		errors := make(chan error, concurrentRequests)

		for i := 0; i < concurrentRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := relayClient.Get(MEVBoostURL + "/eth/v1/builder/status")
				if err != nil {
					errors <- err
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					errors <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				}
			}()
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			errorCount++
			t.Logf("Concurrent request error: %v", err)
		}

		require.Equal(t, 0, errorCount)
	})
}
