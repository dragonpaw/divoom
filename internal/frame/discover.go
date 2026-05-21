// Package frame talks to a Divoom Times Frame: LAN discovery + the local
// HTTP API exposed by the device on port 9000.
package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DiscoveryURL is Divoom's cloud-hosted LAN discovery endpoint. It returns
// every Divoom device that has phoned home from the same WAN IP as the caller,
// which works as a LAN-membership proxy. GET and POST both work.
const DiscoveryURL = "https://app.divoom-gz.com/Device/ReturnSameLANDevice"

// Device is one entry in the discovery response.
type Device struct {
	DeviceName      string `json:"DeviceName"`
	DeviceID        int64  `json:"DeviceId"`
	DevicePrivateIP string `json:"DevicePrivateIP"`
	DeviceMac       string `json:"DeviceMac"`
	Hardware        int    `json:"Hardware"`
}

// IsTimesFrame reports whether this entry looks like a Times Frame. Divoom
// returns the name "Timesframe" (one word) for these.
func (d Device) IsTimesFrame() bool {
	return d.DeviceName == "Timesframe"
}

type discoveryResponse struct {
	ReturnCode    int      `json:"ReturnCode"`
	ReturnMessage string   `json:"ReturnMessage"`
	DeviceList    []Device `json:"DeviceList"`
}

// Discover returns all Divoom devices visible to this LAN via the cloud
// discovery endpoint. The cloud appears to dedupe rapid repeat calls from the
// same source IP — callers should cache the result rather than poll.
func Discover(ctx context.Context) ([]Device, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DiscoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build discovery request: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discovery request: %w", err)
	}
	defer resp.Body.Close()

	var body discoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode discovery response: %w", err)
	}
	if body.ReturnCode != 0 {
		return nil, fmt.Errorf("discovery error %d: %s", body.ReturnCode, body.ReturnMessage)
	}
	return body.DeviceList, nil
}

// FindTimesFrame returns the first Times Frame on the LAN. The optional
// preferredMAC argument disambiguates when more than one Times Frame is
// present; pass an empty string to take the first one.
func FindTimesFrame(ctx context.Context, preferredMAC string) (Device, error) {
	devices, err := Discover(ctx)
	if err != nil {
		return Device{}, err
	}
	var firstFrame *Device
	for i := range devices {
		d := &devices[i]
		if !d.IsTimesFrame() {
			continue
		}
		if preferredMAC != "" && d.DeviceMac == preferredMAC {
			return *d, nil
		}
		if firstFrame == nil {
			firstFrame = d
		}
	}
	if preferredMAC != "" {
		return Device{}, fmt.Errorf("no Times Frame with MAC %q in discovery response", preferredMAC)
	}
	if firstFrame == nil {
		return Device{}, fmt.Errorf("no Times Frame found on LAN (got %d devices)", len(devices))
	}
	return *firstFrame, nil
}
