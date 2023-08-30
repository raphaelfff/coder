package provisionerdserver

import (
	"sync"
	"time"
)

func NewDebouncer(dur time.Duration) *Debouncer {
	return &Debouncer{dur: dur, now: time.Now}
}

type Debouncer struct {
	lastAcquire      time.Time
	lastAcquireMutex sync.RWMutex
	dur              time.Duration

	// for testing
	now func() time.Time
}

// Debounce returns true if the request should be denied due to a recent Signal()
func (d *Debouncer) Debounce() bool {
	d.lastAcquireMutex.RLock()
	defer d.lastAcquireMutex.RUnlock()
	if !d.lastAcquire.IsZero() && d.now().Sub(d.lastAcquire) < d.dur {
		return true
	}
	return false
}

// Signal the debouncer of an event that should then debounce other requests for dur
func (d *Debouncer) Signal() {
	d.lastAcquireMutex.Lock()
	defer d.lastAcquireMutex.Unlock()
	d.lastAcquire = d.now()
}
