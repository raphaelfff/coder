package dashboard

import (
	"cdr.dev/slog"
	"time"

	"golang.org/x/xerrors"
)

type Config struct {
	// MinWait is the minimum interval between fetches.
	MinWait time.Duration `json:"duration_min"`
	// MaxWait is the maximum interval between fetches.
	MaxWait time.Duration `json:"duration_max"`
	// Trace is whether to trace the requests.
	Trace bool `json:"trace"`
	// Logger is the logger to use.
	Logger slog.Logger `json:"-"`
	// Headless controls headless mode for chromedp.
	Headless bool `json:"no_headless"`
}

func (c Config) Validate() error {
	if c.MinWait <= 0 {
		return xerrors.Errorf("validate duration_min: must be greater than zero")
	}

	if c.MaxWait <= 0 {
		return xerrors.Errorf("validate duration_max: must be greater than zero")
	}

	if c.MinWait > c.MaxWait {
		return xerrors.Errorf("validate duration_min: must be less than duration_max")
	}

	return nil
}
