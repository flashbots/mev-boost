package server

import (
	"github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
	"strconv"
	"sync"
	"testing"
)

func TestSaveValidatorPreferences(t *testing.T) {
	testCases := []struct {
		name        string
		goroutineNb int

		expectedSize int
	}{
		{
			name:        "Should add 1 items",
			goroutineNb: 10,

			expectedSize: 1,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			vp := ValidatorPreferences{preferences: make(map[string]types.SignedValidatorRegistration)}

			for i := 0; i < tt.goroutineNb; i++ {
				wg.Add(1)

				go func() {
					defer wg.Done()

					vp.Save(types.SignedValidatorRegistration{
						Message: &types.RegisterValidatorRequestMessage{
							Pubkey: _HexToPubkey(
								"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"),
						},
					})
				}()
			}

			wg.Wait()

			require.Equal(t, tt.expectedSize, len(vp.preferences))
		})
	}
}

func TestPrepareValidatorPreferencesPayload(t *testing.T) {
	testCases := []struct {
		name         string
		expectedSize int
	}{
		{
			name:         "Should return an empty array",
			expectedSize: 0,
		},
		{
			name:         "Should return an array with 5 elements",
			expectedSize: 5,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			vp := ValidatorPreferences{preferences: make(map[string]types.SignedValidatorRegistration)}

			for i := 0; i < tt.expectedSize; i++ {
				vp.preferences[strconv.Itoa(i)] = types.SignedValidatorRegistration{}
			}

			payload := vp.PreparePayload()
			require.Equal(t, tt.expectedSize, len(payload))
		})
	}
}
