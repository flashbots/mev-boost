package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func makePostRequest(ctx context.Context, client http.Client, url string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return resp, fmt.Errorf("HTTP error status code: %d", resp.StatusCode)
	}

	return resp, nil
}

type httpResponseContainer struct {
	url  string
	err  error
	resp *http.Response
}
