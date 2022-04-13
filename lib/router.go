package lib

import (
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/sirupsen/logrus"
)

// RouterOptions contains the router configuration for NewRouter
type RouterOptions struct {
	RelayURLs []string
	Store     Store
	Log       *logrus.Entry

	GetHeaderTimeout time.Duration
}

// NewRouter creates a json rpc router that handles all methods
func NewRouter(opts RouterOptions) (*mux.Router, error) {
	boost, err := newBoostService(opts.RelayURLs, opts.Store, opts.Log)
	if err != nil {
		return nil, err
	}

	// Set custom GetHeader timeout
	if opts.GetHeaderTimeout > 0 {
		boost.getHeaderClient.Timeout = opts.GetHeaderTimeout
	}

	rpcServer := rpc.NewServer()
	rpcServer.RegisterCodec(json.NewCodec(), "application/json")
	rpcServer.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	if err := rpcServer.RegisterService(boost, "builder"); err != nil {
		return nil, err
	}
	rpcServer.RegisterMethodNotFoundFunc(boost.methodNotFound)

	router := mux.NewRouter()
	router.Handle("/", rpcServer)

	return router, nil
}
