package sky

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// ISS emits the International Space Station's current sub-satellite point
// and (when available) the wall-clock time until its next visible pass over
// our location. Output is pipe-separated for the "iss" scene's three Text
// elements:
//
//	"<lat>°, <lon>°|next pass in 1h 23m|over <location>"
//
// The position is sourced from wheretheiss.at (HTTPS, no-auth, stable);
// the next-pass segment is sourced from open-notify.org's iss-pass
// endpoint, which has historically been flaky. When open-notify fails or
// returns an empty payload, the second segment is left blank — the
// scene's mounts mark it AllowEmpty so the row simply doesn't render.
//
// "over <location>" is computed locally — see locationFor in iss_geo.go.
type ISS struct {
	client  *http.Client
	lat     string
	lon     string
	passURL string
}

func NewISS(lat, lon string) *ISS {
	q := url.Values{}
	q.Set("lat", lat)
	q.Set("lon", lon)
	return &ISS{
		client:  &http.Client{Timeout: 15 * time.Second},
		lat:     lat,
		lon:     lon,
		passURL: "http://api.open-notify.org/iss-pass.json?" + q.Encode(),
	}
}

func (s *ISS) Name() string { return "sky/iss" }

const issPositionURL = "https://api.wheretheiss.at/v1/satellites/25544"

type issPosition struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type issPassResponse struct {
	Response []struct {
		Risetime int64 `json:"risetime"`
	} `json:"response"`
}

func (s *ISS) Fetch(ctx context.Context) (string, error) {
	pos, err := s.fetchPosition(ctx)
	if err != nil {
		return "", err
	}
	// Pass lookup is best-effort. Treat any failure as "no upcoming pass
	// known" and leave the segment empty.
	passSeg := ""
	if when, ok := s.fetchNextPass(ctx); ok {
		passSeg = formatNextPass(when, time.Now())
	}
	loc := locationFor(pos.Latitude, pos.Longitude)
	return fmt.Sprintf("%s|%s|%s",
		formatCoords(pos.Latitude, pos.Longitude),
		passSeg,
		loc,
	), nil
}

func (s *ISS) fetchPosition(ctx context.Context) (issPosition, error) {
	var pos issPosition
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issPositionURL, nil)
	if err != nil {
		return pos, fmt.Errorf("iss: build position request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return pos, fmt.Errorf("iss: position http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return pos, fmt.Errorf("iss: position http %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&pos); err != nil {
		return pos, fmt.Errorf("iss: position decode: %w", err)
	}
	return pos, nil
}

// fetchNextPass returns the unix time of the next pass, or ok=false on
// any failure. open-notify's iss-pass endpoint has been intermittently
// 404ing for a long time — the widget tolerates this so the scene still
// renders the position even when the pass lookup is dead.
func (s *ISS) fetchNextPass(ctx context.Context) (time.Time, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.passURL, nil)
	if err != nil {
		return time.Time{}, false
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return time.Time{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return time.Time{}, false
	}
	var body issPassResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return time.Time{}, false
	}
	if len(body.Response) == 0 {
		return time.Time{}, false
	}
	return time.Unix(body.Response[0].Risetime, 0), true
}

// formatCoords renders a lat/lon pair as "<lat>°, <lon>°" with one
// decimal — same precision the scene displays at FontSize 80.
func formatCoords(lat, lon float64) string {
	return strconv.FormatFloat(lat, 'f', 1, 64) + "°, " +
		strconv.FormatFloat(lon, 'f', 1, 64) + "°"
}

// formatNextPass returns "next pass in 1h 23m" / "next pass in 47m"
// for a future risetime. Past or zero times yield "" so the row stays
// blank rather than showing a misleading negative duration.
func formatNextPass(when, now time.Time) string {
	d := when.Sub(now)
	if d <= 0 {
		return ""
	}
	h := int(d.Hours())
	m := int(d.Minutes()) - h*60
	if h > 0 {
		return fmt.Sprintf("next pass in %dh %dm", h, m)
	}
	return fmt.Sprintf("next pass in %dm", m)
}

var _ widget.Widget = (*ISS)(nil)

