package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBeacon(t *testing.T) {

	t.Run("create beacon: BeaconNode", func(t *testing.T) {
		isMergemock := false
		beaconEndpoint := "http://localhost:5052"
		engineEndpoint := "http://localhost:5053"
		beacon := createBeacon(isMergemock, beaconEndpoint, engineEndpoint)
		require.Equal(t, &BeaconNode{beaconEndpoint}, beacon)
	})

	t.Run("create beacon: MergemockBeacon", func(t *testing.T) {
		isMergemock := true
		beaconEndpoint := "http://localhost:5052"
		engineEndpoint := "http://localhost:5053"
		beacon := createBeacon(isMergemock, beaconEndpoint, engineEndpoint)
		require.Equal(t, &MergemockBeacon{engineEndpoint}, beacon)
	})
}
