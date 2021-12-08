package lib

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

// NewRouter creates a json rpc router that handles all methods
func NewRouter(executionURL string, relayURL string) (*mux.Router, error) {
	mev, err := newMevService(executionURL, relayURL)
	if err != nil {
		return nil, err
	}
	relay, err := newRelayService(executionURL, relayURL)
	if err != nil {
		return nil, err
	}

	rpcServer := rpc.NewServer()

	rpcServer.RegisterCodec(json.NewCodec(), "application/json")
	rpcServer.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	rpcServer.RegisterService(mev, "engine")
	rpcServer.RegisterService(relay, "builder")
	rpcServer.RegisterMethodNotFoundFunc(mev.methodNotFound)

	router := mux.NewRouter()
	router.Handle("/", rpcServer)

	return router, nil
}
