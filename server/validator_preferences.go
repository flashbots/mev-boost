package server

import "github.com/flashbots/go-boost-utils/types"

// sharedRegistrationsRequests is used by handleRegisterValidator to send incoming requests along with a unique
// channel for registerValidatorAtInterval to send back the number of successful registered validator preferences.
type sharedRegistrationsRequests struct {
	preferences               []types.SignedValidatorRegistration
	numSuccessRequestsToRelay chan uint64
}
