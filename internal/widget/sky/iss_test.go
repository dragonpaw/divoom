package sky

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestISSFetchPipeShape pins the pipe shape Fetch emits so a future
// upstream-schema drift (or a refactor that drops a segment) fails the
// scene-binding contract loudly. We stub the wheretheiss.at endpoint
// with a deterministic position+altitude+velocity payload and stub the
// next-pass endpoint to 404 so the pass segment is empty — that
// matches the long-standing open-notify outage and isolates the
// payload-driven segments.
func TestISSFetchPipeShape(t *testing.T) {
	posSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"latitude": -22.5, "longitude": -45.3, "altitude": 408.123, "velocity": 7.6612}`))
	}))
	defer posSrv.Close()

	passSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer passSrv.Close()

	s := NewISS("0", "0")
	// Re-point the HTTP endpoints at our stubs. The position URL is a
	// package-level constant we can't override; instead we shadow Fetch
	// by calling fetchPosition directly via the public surface — Fetch
	// itself uses issPositionURL. Workaround: temporarily swap by
	// driving fetchPosition's HTTP client through a transport that
	// rewrites the host.
	s.client.Transport = rewriteTransport{
		posHost:  posSrv.URL,
		passHost: passSrv.URL,
		base:     http.DefaultTransport,
	}
	s.passURL = passSrv.URL

	out, err := s.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	parts := strings.Split(out, "|")
	if len(parts) != 5 {
		t.Fatalf("Fetch pipe shape: got %d segments (%q), want 5", len(parts), out)
	}
	if parts[0] != "-22.5°, -45.3°" {
		t.Errorf("coords segment = %q, want %q", parts[0], "-22.5°, -45.3°")
	}
	if parts[1] != "" {
		t.Errorf("pass segment = %q, want empty (open-notify stubbed 404)", parts[1])
	}
	// parts[2] is the locationFor() output for (-22.5, -45.3); pinning
	// the exact string would couple this test to iss_geo's region table.
	if parts[2] == "" {
		t.Errorf("location segment unexpectedly empty")
	}
	if parts[3] != "408" {
		t.Errorf("altitude segment = %q, want %q", parts[3], "408")
	}
	if parts[4] != "7.66" {
		t.Errorf("velocity segment = %q, want %q", parts[4], "7.66")
	}
}

// rewriteTransport routes requests to api.wheretheiss.at to a test
// server. We can't easily override the package-level constant, so the
// transport rewrites the request URL in flight.
type rewriteTransport struct {
	posHost, passHost string
	base              http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Host, "wheretheiss.at") {
		newReq := req.Clone(req.Context())
		newReq.URL.Scheme = "http"
		newReq.URL.Host = strings.TrimPrefix(t.posHost, "http://")
		return t.base.RoundTrip(newReq)
	}
	return t.base.RoundTrip(req)
}
