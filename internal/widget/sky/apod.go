package sky

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// APOD fetches NASA's Astronomy Picture of the Day metadata and emits a
// pipe-separated "<image_url>|<title>|<date>" string. The "nasa" scene
// splits these three fields across an Image element (URL) and a Text
// element (title); the date is currently unused by the scene but kept in
// the payload so future layouts can surface it without a widget change.
//
// Quota: NASA's DEMO_KEY allows ~30 requests / hour per IP, plenty for one
// scene refreshing every ~3 minutes. NASA_API_KEY env var overrides the
// default when present.
type APOD struct {
	client *http.Client
	apiKey string
}

const apodEndpoint = "https://api.nasa.gov/planetary/apod"

func NewAPOD() *APOD {
	key := os.Getenv("NASA_API_KEY")
	if key == "" {
		key = "DEMO_KEY"
	}
	return &APOD{
		// api.nasa.gov occasionally takes >15s to respond, especially with
		// DEMO_KEY under shared rate limits — 45s leaves headroom while
		// staying well under the scene driver's 60s rotation interval.
		client: &http.Client{Timeout: 45 * time.Second},
		apiKey: key,
	}
}

func (a *APOD) Name() string { return "sky/apod" }

// apodResponse is the slice of the NASA APOD JSON we care about. The full
// response has more fields (explanation, copyright, hdurl, service_version)
// but the scene only renders title + image, so we ignore the rest.
type apodResponse struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Date      string `json:"date"`
	MediaType string `json:"media_type"`
}

func (a *APOD) Fetch(ctx context.Context) (string, error) {
	url := apodEndpoint + "?api_key=" + a.apiKey
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("apod: build request: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("apod: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("apod: http %d", resp.StatusCode)
	}
	var body apodResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("apod: decode: %w", err)
	}
	// Video days happen ~1-2x a month. The scene driver keeps the prior
	// cached value when Fetch errors, so returning an error here is the
	// right move: yesterday's picture stays on the wall until tomorrow's
	// APOD is an image again.
	if body.MediaType != "image" {
		return "", fmt.Errorf("apod: media_type %q is not an image", body.MediaType)
	}
	if body.URL == "" || body.Title == "" {
		return "", fmt.Errorf("apod: empty url or title in response")
	}
	return fmt.Sprintf("%s|%s|%s", body.URL, body.Title, body.Date), nil
}
