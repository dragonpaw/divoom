package frame

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a thin client for the Times Frame's local HTTP API. Divoom's
// convention is GET requests carrying a JSON body whose `Command` field
// selects the operation. We honor that quirk faithfully.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New builds a Client for a Times Frame at the given LAN IP. Port is fixed at
// 9000 per the device's published spec.
func New(ip string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:9000/divoom_api", ip),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Call sends a command and decodes the response into out. If out is nil, the
// response body is discarded (after checking ReturnCode). The command struct
// must contain a `Command` field tagged for JSON; helpers in commands.go
// define the typed shapes.
func (c *Client) Call(ctx context.Context, cmd any, out any) error {
	body, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, raw)
	}

	// Every Times Frame response has at least ReturnCode + ReturnMessage.
	var envelope struct {
		ReturnCode    int    `json:"ReturnCode"`
		ReturnMessage string `json:"ReturnMessage"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode envelope: %w (body: %s)", err, raw)
	}
	if envelope.ReturnCode != 0 {
		return fmt.Errorf("device error %d: %s", envelope.ReturnCode, envelope.ReturnMessage)
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode response: %w (body: %s)", err, raw)
		}
	}
	return nil
}
