// Package widget defines the contract every dashboard widget implements.
// A widget fetches a single short text string on demand; the daemon
// decides how often to call it and patches the result onto a `Text`
// element via `Device/UpdateDisplayItems`.
package widget

import "context"

// Widget fetches one short text string at a time. Cadence is the caller's
// concern, not the widget's — see `cmd/divoom/serve.go` for where the
// dashboard sets per-widget intervals.
type Widget interface {
	// Name is used for logging; should be short and stable.
	Name() string
	// Fetch returns the current text for the widget, or an error. Errors
	// are logged by the caller; the previously-rendered text stays on
	// screen rather than being replaced with an error message.
	Fetch(ctx context.Context) (string, error)
}
