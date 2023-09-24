package dashboard_test

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/scaletest/dashboard"
	"github.com/coder/coder/v2/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Run(t *testing.T) {
	t.Parallel()
	if testutil.RaceEnabled() {
		t.Skip("skipping timing-sensitive test because of race detector")
	}
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	client := coderdtest.New(t, nil)
	_ = coderdtest.CreateFirstUser(t, client)

	log := slogtest.Make(t, &slogtest.Options{
		IgnoreErrors: true,
	})
	m := &testMetrics{}
	cfg := dashboard.Config{
		MinWait: time.Millisecond,
		MaxWait: 10 * time.Millisecond,
		Logger:  log,
	}
	r := dashboard.NewRunner(client, m, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
	t.Cleanup(cancel)
	done := make(chan error)
	go func() {
		defer close(done)
		done <- r.Run(ctx, "", nil)
	}()
	err, ok := <-done
	assert.True(t, ok)
	require.NoError(t, err)

	if assert.NotEmpty(t, m.ObservedDurations["succeeds"]) {
		assert.NotZero(t, m.ObservedDurations["succeeds"][0])
	}

	if assert.NotEmpty(t, m.ObservedDurations["fails"]) {
		assert.NotZero(t, m.ObservedDurations["fails"][0])
	}

	if assert.NotEmpty(t, m.ObservedDurations["hangs"]) {
		assert.GreaterOrEqual(t, m.ObservedDurations["hangs"][0], cfg.MaxWait.Seconds())
	}
	assert.Zero(t, m.Errors["succeeds"])
	assert.NotZero(t, m.Errors["fails"])
	assert.NotZero(t, m.Errors["hangs"])
	assert.NotEmpty(t, m.Statuses["succeeds"])
	assert.NotEmpty(t, m.Statuses["fails"])
	assert.NotEmpty(t, m.Statuses["hangs"])
}

type testMetrics struct {
	sync.RWMutex
	ObservedDurations map[string][]float64
	Errors            map[string]int
	Statuses          map[string]map[string]int
}

func (m *testMetrics) ObserveDuration(action string, d time.Duration) {
	m.Lock()
	defer m.Unlock()
	if m.ObservedDurations == nil {
		m.ObservedDurations = make(map[string][]float64)
	}
	m.ObservedDurations[action] = append(m.ObservedDurations[action], d.Seconds())
}

func (m *testMetrics) IncErrors(action string) {
	m.Lock()
	defer m.Unlock()
	if m.Errors == nil {
		m.Errors = make(map[string]int)
	}
	m.Errors[action]++
}

func (m *testMetrics) IncStatuses(action string, code string) {
	m.Lock()
	defer m.Unlock()
	if m.Statuses == nil {
		m.Statuses = make(map[string]map[string]int)
	}
	if m.Statuses[action] == nil {
		m.Statuses[action] = make(map[string]int)
	}
	m.Statuses[action][code]++
}
