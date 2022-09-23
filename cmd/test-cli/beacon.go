package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flashbots/mev-boost/server"
)

// Beacon - beacon node interface
type Beacon interface {
	onGetHeader() error
	getCurrentBeaconBlock() (beaconBlockData, error)
}

type beaconBlockData struct {
	Slot      uint64
	BlockHash common.Hash
}

func createBeacon(isMergemock bool, beaconEndpoint, engineEndpoint string) Beacon {
	if isMergemock {
		return &MergemockBeacon{engineEndpoint}
	}
	return &BeaconNode{beaconEndpoint}
}

// BeaconNode - beacon node wrapper
type BeaconNode struct {
	beaconEndpoint string
}

func (b *BeaconNode) onGetHeader() error { return nil }

func (b *BeaconNode) getCurrentBeaconBlock() (beaconBlockData, error) {
	return getCurrentBeaconBlock(b.beaconEndpoint)
}

type partialSignedBeaconBlock struct {
	Data struct {
		Message struct {
			Slot string `json:"slot"`
			Body struct {
				ExecutionPayload struct {
					BlockHash common.Hash `json:"block_hash"`
				} `json:"execution_payload"`
			} `json:"body"`
		} `json:"message"`
	} `json:"data"`
}

func getCurrentBeaconBlock(beaconEndpoint string) (beaconBlockData, error) {
	var blockResp partialSignedBeaconBlock
	_, err := server.SendHTTPRequest(context.TODO(), *http.DefaultClient, http.MethodGet, beaconEndpoint+"/eth/v2/beacon/blocks/head", "test-cli/beacon", nil, &blockResp)
	if err != nil {
		return beaconBlockData{}, err
	}

	slot, err := strconv.ParseUint(blockResp.Data.Message.Slot, 10, 64)
	if err != nil {
		return beaconBlockData{}, err
	}

	return beaconBlockData{Slot: slot, BlockHash: blockResp.Data.Message.Body.ExecutionPayload.BlockHash}, err
}

// MergemockBeacon - fake beacon for use with mergemock relay's engine
type MergemockBeacon struct {
	mergemockEngineEndpoint string
}

func (m *MergemockBeacon) onGetHeader() error {
	block, err := getLatestEngineBlock(m.mergemockEngineEndpoint)
	if err != nil {
		return err
	}
	err = sendForkchoiceUpdate(m.mergemockEngineEndpoint, block)
	if err != nil {
		return err
	}

	return nil
}

func (m *MergemockBeacon) getCurrentBeaconBlock() (beaconBlockData, error) {
	block, err := getLatestEngineBlock(m.mergemockEngineEndpoint)
	if err != nil {
		log.WithError(err).Info("could not get latest block")
		return beaconBlockData{}, err
	}

	return beaconBlockData{Slot: 50, BlockHash: block.Hash}, nil
}
