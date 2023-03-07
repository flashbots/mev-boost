package rcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/flashbots/mev-boost/config/relay"
)

// JSONAPI fetches the config using JSON API RCP.
type JSONAPI struct {
	providerURL string
	client      *http.Client
}

// NewJSONAPI creates a new instance of JSONAPI.
//
// It takes an HTTP Client and the providerURL.
// If the client is not specified, http.DefaultClient will be used.
func NewJSONAPI(client *http.Client, providerURL string) *JSONAPI {
	if client == nil {
		client = http.DefaultClient
	}

	return &JSONAPI{providerURL: providerURL, client: client}
}

// FetchConfig fetches the relay configuration from JSON API RCP.
//
// It returns *relay.Config on success.
// It returns an error if the RCP providerURL is malformed.
// It returns an error if the RCP returns a non http.StatusOK.
// It returns an error if it cannot execute the HTTP request.
// It returns an error if it cannot unmarshal the response body.
func (p *JSONAPI) FetchConfig() (*relay.Config, error) {
	resp, err := p.doRequest(p.providerURL)
	if err != nil {
		return nil, WrapErr(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := decodeResponseBody(resp.Body, &apiErr); err != nil {
			return nil, WrapErr(err)
		}

		return nil, apiErr
	}

	var payload *relay.Config
	if err := decodeResponseBody(resp.Body, &payload); err != nil {
		return nil, WrapErr(err)
	}

	return payload, nil
}

func (p *JSONAPI) doRequest(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedProviderURL, err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Add("Accept", `application/json`)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHTTPRequestFailed, err)
	}

	return resp, nil
}

func decodeResponseBody(body io.Reader, target any) error {
	if err := json.NewDecoder(body).Decode(target); err != nil {
		return fmt.Errorf("%w: cannot decode response: %w", ErrMalformedResponseBody, err)
	}

	return nil
}
