package lib

import (
	"context"
	"fmt"
	glog "log"
	"net"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/node"
	gethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

// Error that can be returned by the RPC server with an error code.
type Error struct {
	Err error
	ID  int
}

// ErrorCode returns the ID of the error.
func (e *Error) ErrorCode() int { return e.ID }

// Error returns the string of the error.
func (e *Error) Error() string { return e.Err.Error() }

// Timeout is used to configure the RPC server timeouts for user requests
type Timeout struct {
	Read       time.Duration // Timeout for body reads. None if 0.
	ReadHeader time.Duration // Timeout for header reads. None if 0.
	Write      time.Duration // Timeout for writes. None if 0.
	Idle       time.Duration // Timeout to disconnect idle client connections. None if 0.
}

// BoostRPCServerOptions contains the router configuration for NewRouter
type BoostRPCServerOptions struct {
	ListenAddr string
	RelayURLs  []string
	Cors       []string
	Log        *logrus.Entry

	GetHeaderTimeout   time.Duration // maximum wait time for a relay response to getHeaderV1
	UserRequestTimeout Timeout
}

// NewBoostRPCServer creates a new boost server
func NewBoostRPCServer(opts BoostRPCServerOptions) (*http.Server, error) {
	boost, err := newBoostService(opts.RelayURLs, opts.Log, opts.GetHeaderTimeout)
	if err != nil {
		return nil, err
	}

	srv, err := NewRPCServer("builder", boost, true)
	if err != nil {
		return nil, err
	}
	return NewHTTPServer(context.Background(), opts.Log, srv, opts.ListenAddr, opts.UserRequestTimeout, opts.Cors), nil
}

// NewRPCServer creates a new RPC server
func NewRPCServer(namespace string, backend interface{}, authenticated bool) (*gethRpc.Server, error) {
	srv := gethRpc.NewServer()
	srv.RegisterName(namespace, backend)
	apis := []gethRpc.API{
		{
			Namespace: namespace,
			Version:   "1.0",
			Service:   backend,
			Public:    true,
		},
	}
	if err := node.RegisterApis(apis, []string{namespace}, srv, false); err != nil {
		return nil, fmt.Errorf("could not register api: %s", err)
	}
	return srv, nil
}

// NewHTTPServer creates a new HTTP server interface for a RPC server
func NewHTTPServer(ctx context.Context, log logrus.Ext1FieldLogger, rpcSrv *gethRpc.Server, addr string, timeout Timeout, cors []string) *http.Server {
	httpRPCHandler := node.NewHTTPHandlerStack(rpcSrv, cors, nil)
	mux := http.NewServeMux()
	mux.Handle("/", httpRPCHandler)
	logHTTP := log.WithField("type", "http")
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       timeout.Read,
		ReadHeaderTimeout: timeout.ReadHeader,
		WriteTimeout:      timeout.Write,
		IdleTimeout:       timeout.Idle,
		ConnState: func(conn net.Conn, state http.ConnState) {
			e := logHTTP.WithField("addr", conn.RemoteAddr().String())
			e.WithField("state", state.String())
			e.Debug("client changed connection state")
		},
		ErrorLog: glog.New(logHTTP.Writer(), "", 0),
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}
}
