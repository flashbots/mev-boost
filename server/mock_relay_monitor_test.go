package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestMockRelayMonitor(t *testing.T) {
	t.Run("bad payload", func(t *testing.T) {
		relay := newMockRelayMonitor(t)
		req, err := http.NewRequest(http.MethodPost, pathAuctionTranscript, bytes.NewReader([]byte("123")))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		relay.getRouter().ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("can override response", func(t *testing.T) {
		relay := newMockRelayMonitor(t)
		faultsRecord := make(analysis.FaultRecord)
		relay.GetFaultsResponse = &api.FaultsResponse{
			Span: api.Span{
				Start: 0,
				End:   100,
			},
			FaultRecord: faultsRecord,
		}
		req, err := http.NewRequest(http.MethodGet, pathFault, bytes.NewReader([]byte("123")))
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		relay.getRouter().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, `{"span":{"start_epoch":"0","end_epoch":"100"},"data":{}}`+"\n", rr.Body.String())
	})
}
