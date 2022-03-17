package lib

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/sirupsen/logrus"
)

// NewRouter creates a json rpc router that handles all methods
func NewRouter(relayURLs []string, store Store, log *logrus.Entry) (*mux.Router, error) {
	relay, err := newRelayService(relayURLs, store, log)
	if err != nil {
		return nil, err
	}

	rpcServer := rpc.NewServer()

	rpcServer.RegisterCodec(json.NewCodec(), "application/json")
	rpcServer.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	if err := rpcServer.RegisterService(relay, "engine"); err != nil {
		return nil, err
	}
	if err := rpcServer.RegisterService(relay, "builder"); err != nil {
		return nil, err
	}
	if err := rpcServer.RegisterService(relay, "relay"); err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	router.Handle("/", rpcServer)

	return router, nil
}
