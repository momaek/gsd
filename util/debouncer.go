package util

import (
	"sync"
	"time"
)

// Debouncer returns a debounced function that takes another functions as its argument.
// This function will be called when the debounced function stops being called
// for the given duration.
// The debounced function can be invoked with different functions, if needed,
// the last one will win.
func Debouncer(after time.Duration) func(f func()) {

	d := &debouncer{
		after: after,
	}

	return func(f func()) {
		d.add(f)
	}
}

// debouncer provides a debouncer func. The most typical use case would be the user
// typing a text into a form; the UI needs an update, but let's wait for a break.
type debouncer struct {
	mu    sync.Mutex
	after time.Duration
	timer *time.Timer
}

// add callback func
func (d *debouncer) add(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.after, f)
}
