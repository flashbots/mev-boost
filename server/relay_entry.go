package server

import (
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/flashbots/go-boost-utils/types"
	prysmTypes "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/v2/config/params"
)

const (
	PayloadWithdrawn int = -1
	NotSelected      int = 0
	PayloadReturned  int = 1
)

// TODO: hardcoded
const numOfAveragedSamples = 5

// RelayEntry represents a relay that mev-boost connects to.
// Address will be schema://hostname:port
// PublicKey holds the relay's BLS public key used to verify message signatures.
type RelayEntry struct {
	Address   string
	PublicKey types.PublicKey
	URL       *url.URL

	// For each slot, stores:
	// -1 if the relay withdrawn the payload
	// -0 if the relay wasn't selected
	// +1 if the relay successfully sent the payload
	// Just n slots are stored as a moving window
	// This field is used to calculate the Reputation (see GetRelayReputation)
	ResponseStatus []int

	IsBlackListed     bool
	CommittedToHeader bool
	PayloadSent       bool
}

func (r *RelayEntry) String() string {
	return r.URL.String()
}

// NewRelayEntry creates a new instance based on an input string
// relayURL can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string) (entry RelayEntry, err error) {
	// Add protocol scheme prefix if it does not exist.
	if !strings.HasPrefix(relayURL, "http") {
		relayURL = "http://" + relayURL
	}

	// Parse the provided relay's URL and save the parsed URL in the RelayEntry.
	entry.URL, err = url.Parse(relayURL)
	if err != nil {
		return entry, err
	}

	// Build the relay's address.
	entry.Address = entry.URL.Scheme + "://" + entry.URL.Host

	// TODO: numOfAveragedSamples is hardcoded to 5
	entry.ResponseStatus = make([]int, numOfAveragedSamples)

	// Extract the relay's public key from the parsed URL.
	// TODO: Remove the if condition, as it is mandatory to verify relay's message signature.
	if entry.URL.User.Username() != "" {
		err = entry.PublicKey.UnmarshalText([]byte(entry.URL.User.Username()))
	}

	// TODO init with all 0s

	return entry, err
}

// TODO: Assumption that this function is called every slot to simplify
func (r *RelayEntry) SetResponseStatus(state int) {
	r.ResponseStatus = append(r.ResponseStatus[1:], state)
}

func (r *RelayEntry) GetRelayReputation() float64 {
	if numOfAveragedSamples != 5 {
		// TODO: Error. 5 is hardcoded
	}

	// TODO: Hardcoded for numOfAveragedSamples=5
	// x has to meet that x + x^1 + ... x^5 = 1
	x := float64(0.50866)

	reputation := float64(0)
	for i, response := range r.ResponseStatus {
		reputation += math.Pow(x, float64(numOfAveragedSamples-i)) * float64(response)
	}
	return reputation
}

func GetSlotFromTime(slotTime time.Time) uint64 {
	// Hardcoded genesis time for mainnet
	// TODO: Fetch programmatically from: eth/v1/beacon/genesis
	genesisTimeSec := int64(1606824023)
	genesis := int64(genesisTimeSec)
	if slotTime.Unix() < genesis {
		return 0
	}
	return uint64(prysmTypes.Slot(uint64(slotTime.Unix()-genesis) / params.BeaconConfig().SecondsPerSlot))
}
