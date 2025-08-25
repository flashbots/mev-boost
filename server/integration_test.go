package server

import (
	"context"
	"encoding/json"
	"errors"
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

var (
	MEVBoostURL   = os.Getenv("MEV_BOOST_URL")
	BeaconNodeURL = os.Getenv("BEACON_NODE_URL")
	RelayURL      = os.Getenv("RELAY_URL")
	ExecutionURL  = os.Getenv("EXECUTION_URL")

	RelaySecretKey = "0x5eae315483f028b5cdd5d1090ff0c7618b18737ea9bf3c35047189db22835c48"
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

func (c *BeaconNodeClient) GetCurrentSlot() (phase0.Slot, error) {
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

func (c *BeaconNodeClient) GetBlockHeader(slot phase0.Slot) (*phase0.BeaconBlockHeader, error) {
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
		return "", errors.New("failed to request proposer duties") //nolint:err113
	}

	var result struct {
		Data []struct {
			Pubkey string `json:"pubkey"`
			Slot   string `json:"slot"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", errors.New("failed to decode proposer duties") //nolint:err113
	}

	// check for the required slot
	targetSlotStr := fmt.Sprintf("%d", slot)
	for _, duty := range result.Data {
		if duty.Slot == targetSlotStr {
			return duty.Pubkey, nil
		}
	}

	return "", fmt.Errorf("no proposer found for slot %d", slot) //nolint:err113
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
		return fmt.Errorf("status check failed with status: %d", resp.StatusCode) //nolint:err113
	}

	return nil
}

// waitForMEVBoost waits for MEV-boost to be available
func waitForMEVBoost(t *testing.T, timeout time.Duration) {
	t.Helper()

	client := NewMEVBoostClient(MEVBoostURL)
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
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

func TestMEVBoostIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
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

	t.Logf("Testing Fork: %s", testingFork)
	t.Logf("services: Beacon (%s), MEV-boost (%s), Relay (%s)", BeaconNodeURL, MEVBoostURL, RelayURL)

	// check mev-boost status
	t.Run("mev-boost status check", func(t *testing.T) {
		resp, err := httpClient.Get(MEVBoostURL + "/eth/v1/builder/status")
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// validate chain activity
	t.Run("Validate blockchain activity and transaction types", func(t *testing.T) {
		// should be delivering payloads since its via mev-boost we can directly check the relay api
		resp, err := httpClient.Get(RelayURL + "/relay/v1/data/bidtraces/proposer_payload_delivered")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var payloads []map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payloads))
		// since payloads are being requested via mev-boost their length being greater shows its working
		require.NotEmpty(t, payloads)

		// for blob transaction testing, check if recent blocks contain blob transactions
		if testingTxType == "blobs" { //nolint:nestif
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
					continue
				}

				// we could also have check for txtypes but that would have been traversering through
				// alot of them so for simplicity we can make sure of BlobGasUsed to check for block txs
				if blockResult.Result.BlobGasUsed != "" && blockResult.Result.BlobGasUsed != "0x0" {
					blobGasUsed, _ := strconv.ParseUint(strings.TrimPrefix(blockResult.Result.BlobGasUsed, "0x"), 16, 64)
					if blobGasUsed > 0 {
						blobTxFound = true
						totalBlobGasUsed += blobGasUsed
						t.Logf("found blob transactions in block %s: %d blob gas used", blockResult.Result.Number, blobGasUsed)

						// Count blob transactions in this block
						blobTxCount := 0
						for _, tx := range blockResult.Result.Transactions {
							if txType, exists := tx["type"]; exists && txType == "0x3" {
								blobTxCount++
							}
						}
						require.Positive(t, blobTxCount)
					}
				}
			}

			require.True(t, blobTxFound)

			if blobTxFound {
				t.Logf("successfully detected blob transactions (total blob gas: %d)", totalBlobGasUsed)
			} else {
				t.Logf("no blob transactions found")
			}
		}
	})

	// validate MEV-boost is consistently building all blocks
	t.Run("MEV-boost consistent block building", func(t *testing.T) {
		t.Logf("Validating that MEV-boost is building all blocks (builder-playground environment)...")

		currentSlot, err := beaconClient.GetCurrentSlot()
		require.NoError(t, err, "Should be able to get current slot")

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
			require.NotEmpty(t, deliveries)

			mevBoostBlocks++
		}

		require.Positive(t, totalBlocks)
		require.Equal(t, mevBoostBlocks, totalBlocks)
	})

	t.Run("request validation on invalid pub key", func(t *testing.T) {
		resp, err := httpClient.Get(MEVBoostURL + "/eth/v1/builder/header/1/0x0000000000000000000000000000000000000000000000000000000000000000/0x000000")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("bid retrieval", func(t *testing.T) {
		currentSlot, err := beaconClient.GetCurrentSlot()
		require.NoError(t, err)

		previousSlot := currentSlot - 1

		previousHeader, err := beaconClient.GetBlockHeader(previousSlot)
		require.NoError(t, err)

		parentHash := fmt.Sprintf("0x%x", previousHeader.ParentRoot)

		// get the scheduled validator for the previous slot
		scheduledValidator, err := getScheduledValidatorForSlot(beaconClient, previousSlot)
		require.NoError(t, err)

		t.Logf("scheduled validator for slot %d: %s", previousSlot, scheduledValidator)

		url := fmt.Sprintf("%s/eth/v1/builder/header/%d/%s/%s",
			MEVBoostURL, previousSlot, parentHash, scheduledValidator)

		t.Logf("requesting bid: slot=%d, parent=%s, validator=%s", previousSlot, parentHash, scheduledValidator)

		resp, err := httpClient.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.True(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK)
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
					errors <- fmt.Errorf("unexpected status code: %d", resp.StatusCode) //nolint:err113
				}
			}()
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			errorCount++
			t.Logf("concurrent request error: %v", err)
		}

		require.Equal(t, 0, errorCount)
	})
}
