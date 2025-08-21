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
	"github.com/stretchr/testify/require"
)

const (
	MEVBoostURL   = "http://localhost:18550"
	BeaconNodeURL = "http://localhost:3500"
	RelayURL      = "http://localhost:5555"
	ExecutionURL  = "http://localhost:8545"

	RelaySecretKey = "0x5eae315483f028b5cdd5d1090ff0c7618b18737ea9bf3c35047189db22835c48"
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

func getScheduledValidatorForSlot(client *BeaconNodeClient, slot phase0.Slot) (string, error) {
	epoch := slot / 32

	// retrieve valdiator duties
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

	// check for the required slot
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
	// Skip this test if we're not running integration te sts
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	waitForMEVBoost(t, 10*time.Second)

	beaconClient := NewBeaconNodeClient(BeaconNodeURL)
	httpClient := &http.Client{Timeout: 10 * time.Second}

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

	// Test 1: check mev-boost status
	t.Run("mev-boost status check", func(t *testing.T) {
		//check mev-boost status
		resp, err := httpClient.Get(MEVBoostURL + "/eth/v1/builder/status")
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test 2: validate chain activity
	t.Run("Validate blockchain activity and transaction types", func(t *testing.T) {
		// should be delivering payloads since its via mev-boost we can directly check the relay api
		resp, err := httpClient.Get(RelayURL + "/relay/v1/data/bidtraces/proposer_payload_delivered")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var payloads []map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payloads))
		// since payloads are being requested via mev-boost their length being greater shows its working
		require.Greater(t, len(payloads), 0)

		// for blob transaction testing, check if recent blocks contain blob transactions
		if testingTxType == "blobs" {
			blobTxFound := false
			totalBlobGasUsed := uint64(0)

			latestBlockResp, err := http.Post(ExecutionURL, "application/json",
				strings.NewReader(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`))
			require.NoError(t, err)
			defer latestBlockResp.Body.Close()

			var latestBlockResult struct {
				Result string `json:"result"`
			}
			require.NoError(t, json.NewDecoder(latestBlockResp.Body).Decode(&latestBlockResult))

			latestBlockNum, err := strconv.ParseUint(strings.TrimPrefix(latestBlockResult.Result, "0x"), 16, 64)
			require.NoError(t, err)

			// check the last 15 blocks for blob transactions
			for i := uint64(0); i < 15; i++ {
				if latestBlockNum < i {
					continue
				}

				blockNumber := fmt.Sprintf("0x%x", latestBlockNum-i)
				blockResp, err := http.Post(ExecutionURL, "application/json",
					strings.NewReader(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["%s",true],"id":1}`, blockNumber)))
				if err != nil {
					fmt.Println("err", err)
					continue
				}
				defer blockResp.Body.Close()

				var blockResult struct {
					Result struct {
						Number        string                   `json:"number"`
						BlobGasUsed   string                   `json:"blobGasUsed"`
						ExcessBlobGas string                   `json:"excessBlobGas"`
						Transactions  []map[string]interface{} `json:"transactions"`
					} `json:"result"`
				}

				if err := json.NewDecoder(blockResp.Body).Decode(&blockResult); err != nil {
					fmt.Println("err", err)
					continue
				}

				// we could also have check for txtypes but that would have been traversering through
				// alot of them so for simplicity we can make sure of BlobGasUsed to check for block txs
				if blockResult.Result.BlobGasUsed != "" && blockResult.Result.BlobGasUsed != "0x0" {
					blobGasUsed, _ := strconv.ParseUint(strings.TrimPrefix(blockResult.Result.BlobGasUsed, "0x"), 16, 64)
					if blobGasUsed > 0 {
						blobTxFound = true
						totalBlobGasUsed += blobGasUsed
						t.Logf("✅ Found blob transactions in block %s: %d blob gas used", blockResult.Result.Number, blobGasUsed)

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
					t.Logf("no blobs found for blockNumber %s", blockResult.Result.Number)
				}
			}

			if blobTxFound {
				t.Logf("✅ Successfully detected blob transaction activity (total blob gas: %d)", totalBlobGasUsed)
			} else {
				t.Logf("⚠️  No blob transactions found in recent blocks (may need more time for contender to generate blobs)")
			}
		}
	})

	// Test 3: Validate MEV-boost is consistently building all blocks
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
				continue
			}

			totalBlocks++

			// payload for this block must have been delivered by the relay
			resp, err := httpClient.Get(fmt.Sprintf("%s/relay/v1/data/bidtraces/proposer_payload_delivered?slot=%d", RelayURL, slotToCheck))
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			var deliveries []map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&deliveries))
			t.Logf("deliveries %d: length", len(deliveries))
			require.Greater(t, len(deliveries), 0)

			mevBoostBlocks++
		}

		require.Greater(t, totalBlocks, 0)
		require.Equal(t, mevBoostBlocks, totalBlocks)
	})

	t.Run("request validation on invalid pub key", func(t *testing.T) {
		resp, err := httpClient.Get(MEVBoostURL + "/eth/v1/builder/header/1/0x0000000000000000000000000000000000000000000000000000000000000000/0x000000")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MEV-boost bid selection", func(t *testing.T) {
		t.Logf("Testing MEV-boost bid selection with real validator and parent hash...")

		currentSlot, err := beaconClient.GetCurrentSlot(ctx)
		require.NoError(t, err, "Should be able to get current slot")

		previousSlot := currentSlot - 1
		// Get the actual parent hash from the current block
		previousHeader, err := beaconClient.GetBlockHeader(ctx, previousSlot)
		require.NoError(t, err, "Should be able to get current block header")

		parentHash := fmt.Sprintf("0x%x", previousHeader.ParentRoot)
		t.Logf("Using real parent hash: %s", parentHash)

		// get the scheduled validator for the previous slot
		scheduledValidator, err := getScheduledValidatorForSlot(beaconClient, previousSlot)
		require.NoError(t, err, "Should be able to get scheduled validator for slot %d", previousSlot)

		t.Logf("Scheduled validator for slot %d: %s", previousSlot, scheduledValidator)

		// Test bid retrieval with real validator and parent hash
		url := fmt.Sprintf("%s/eth/v1/builder/header/%d/%s/%s",
			MEVBoostURL, previousSlot, parentHash, scheduledValidator)

		t.Logf("Requesting bid: slot=%d, parent=%s, validator=%s", previousSlot, parentHash, scheduledValidator)

		resp, err := httpClient.Get(url)
		require.NoError(t, err, "Should handle header requests")
		defer resp.Body.Close()

		// Response can be 204 (no bid) or 200 (bid available)
		require.True(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK,
			"Should return either 204 (no bid) or 200 (bid available), got %d", resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			t.Logf("✅ MEV-boost returned bid for slot %d with real validator", previousSlot)

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
			t.Logf("✅ MEV-boost correctly returned no-bid (204) for slot %d", previousSlot)
			t.Logf("This is expected if no builders have submitted bids for this slot/parent combination")
		}
	})

	t.Run("MEV-boost performance", func(t *testing.T) {
		// testing concurrent calls
		concurrentRequests := 5
		var wg sync.WaitGroup
		errors := make(chan error, concurrentRequests)

		for i := 0; i < concurrentRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := httpClient.Get(MEVBoostURL + "/eth/v1/builder/status")
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
