package rcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/flashbots/mev-boost/config/relay"
)

const (
	relaysByValidatorPublicKey = "/proposer-configs"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type JSONAPI struct {
	providerURL string
	client      HTTPClient
}

func NewJSONAPI(client HTTPClient, providerURL string) *JSONAPI {
	if client == nil {
		client = http.DefaultClient
	}

	return &JSONAPI{providerURL: providerURL, client: client}
}

func (p *JSONAPI) FetchConfig() (*relay.Config, error) {
	endpoint := p.providerURL + relaysByValidatorPublicKey

	resp, err := p.doRequest(endpoint)
	if err != nil {
		return nil, p.wrapConfigProviderErr(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr *APIError
		if err := decodeResponseBody(resp.Body, &apiErr); err != nil {
			return nil, p.wrapConfigProviderErr(err)
		}

		return nil, apiErr
	}

	var payload *relay.Config
	if err := decodeResponseBody(resp.Body, &payload); err != nil {
		return nil, p.wrapConfigProviderErr(err)
	}

	return payload, nil
}

func (p *JSONAPI) doRequest(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedProviderURL, err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Add("Accept", `application/json`)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHTTPRequestFailed, err)
	}

	return resp, nil
}

func decodeResponseBody(body io.Reader, target any) error {
	if err := json.NewDecoder(body).Decode(target); err != nil {
		return fmt.Errorf("%w: cannot decode response: %v", ErrMalformedResponseBody, err)
	}

	return nil
}

func (p *JSONAPI) wrapConfigProviderErr(err error) Error {
	return Error{
		Cause:   err,
		Message: fmt.Sprintf("%v", ErrCannotFetchRelays),
	}
}
