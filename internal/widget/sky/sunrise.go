package sky

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// Sunrise fetches today's sunrise/sunset times for our fixed location from
// Open-Meteo and emits the three pipe-separated fields the "sunrise" scene
// splits across separate Text elements.
//
// Output:  "06:42 AM|07:58 PM|13h 16m"
//
// Coordinates are Richmond, CA (the same default the weather widget used);
// the API returns local-clock ISO times when timezone=auto, so we parse and
// reformat without timezone arithmetic on our side.
type Sunrise struct {
	client *http.Client
}

// Hard-coded location — Richmond, CA centroid. Same coords the older
// weather widget shipped with; revisit when the device moves.
const sunriseURL = "https://api.open-meteo.com/v1/forecast" +
	"?latitude=37.9358&longitude=-122.3477" +
	"&daily=sunrise,sunset&timezone=auto"

func NewSunrise() *Sunrise {
	return &Sunrise{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Sunrise) Name() string { return "sky/sunrise" }

// openMeteoResponse is the slice of the Open-Meteo /forecast JSON we care
// about: just the daily sunrise/sunset arrays. Times come back in the
// device-local clock when timezone=auto, formatted as "2006-01-02T15:04".
type openMeteoResponse struct {
	Daily struct {
		Sunrise []string `json:"sunrise"`
		Sunset  []string `json:"sunset"`
	} `json:"daily"`
}

func (s *Sunrise) Fetch(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sunriseURL, nil)
	if err != nil {
		return "", fmt.Errorf("sunrise: build request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sunrise: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sunrise: http %d", resp.StatusCode)
	}
	var body openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("sunrise: decode: %w", err)
	}
	if len(body.Daily.Sunrise) == 0 || len(body.Daily.Sunset) == 0 {
		return "", fmt.Errorf("sunrise: empty daily payload")
	}

	const apiLayout = "2006-01-02T15:04"
	rise, err := time.Parse(apiLayout, body.Daily.Sunrise[0])
	if err != nil {
		return "", fmt.Errorf("sunrise: parse sunrise %q: %w", body.Daily.Sunrise[0], err)
	}
	set, err := time.Parse(apiLayout, body.Daily.Sunset[0])
	if err != nil {
		return "", fmt.Errorf("sunrise: parse sunset %q: %w", body.Daily.Sunset[0], err)
	}

	daylight := set.Sub(rise)
	h := int(daylight.Hours())
	m := int(daylight.Minutes()) - h*60
	return fmt.Sprintf("%s|%s|%dh %dm",
		rise.Format("3:04 PM"),
		set.Format("3:04 PM"),
		h, m,
	), nil
}

var _ widget.Widget = (*Sunrise)(nil)
