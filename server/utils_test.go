package server

import (
	"encoding/hex"
	"log"

	"github.com/OffchainLabs/prysm/v6/runtime/interop"
)

// initValidatorsWithBLS initializes validator keys using BLS for integration testing
// This is only available in test builds
func initValidatorsWithBLS() {
	// Create validator key map
	globalValidatorKeyMap = NewValidatorKeyMap()

	// Generate the same 100 validator keys as builder playground
	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0, 100)
	if err != nil {
		log.Printf("Warning: Failed to generate keys: %v", err)
		return
	}

	// Store keys in the map
	for i, privKey := range privKeys {
		pubKey := pubKeys[i]

		// Convert public key to hex string
		pubKeyHex := "0x" + hex.EncodeToString(pubKey.Marshal())

		// Store private key in the map
		globalValidatorKeyMap.keys[pubKeyHex] = privKey.Marshal()

		if i < 5 { // Only log first 5 to avoid spam
			log.Printf("Stored validator %d: %s\n", i, pubKeyHex)
		}
	}
	log.Printf("Initialized %d validator keys with BLS", globalValidatorKeyMap.Count())
}
