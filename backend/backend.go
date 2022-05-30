package backend

import "net/http"

// Router paths
var (
	PathStatus            = "/eth/v1/builder/status"
	PathRegisterValidator = "/eth/v1/builder/validators"
	PathGetHeader         = "/eth/v1/builder/header/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}"
	PathGetPayload        = "/eth/v1/builder/blinded_blocks"
)

// BoostBackend defines the interface any boost backend, used both by mev-boost and the tests, must implement
type BoostBackend interface {
	handleRoot(w http.ResponseWriter, req *http.Request)
	handleStatus(w http.ResponseWriter, req *http.Request)
	handleRegisterValidator(w http.ResponseWriter, req *http.Request)
	handleGetHeader(w http.ResponseWriter, req *http.Request)
	handleGetPayload(w http.ResponseWriter, req *http.Request)
}
