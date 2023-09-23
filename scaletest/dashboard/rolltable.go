package dashboard

import (
	"cdr.dev/slog"
	"context"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"golang.org/x/xerrors"
	"os"
	"time"

	"github.com/coder/coder/v2/codersdk"
)

// DefaultActions is a table of actions to perform.
// D&D nerds will feel right at home here :-)
// Note that the order of the table is important!
// Entries must be in ascending order.
var DefaultActions RollTable = []RollTableEntry{
	{0, loadMainPage, "load main page"},
}

// RollTable is a slice of rollTableEntry.
type RollTable []RollTableEntry

// RollTableEntry is an entry in the roll table.
type RollTableEntry struct {
	// Roll is the minimum number required to perform the action.
	Roll int
	// Fn is the function to call.
	Fn func(ctx context.Context, p *Params) error
	// Label is used for logging.
	Label string
}

// choose returns the first entry in the table that is greater than or equal to n.
func (r RollTable) choose(n int) RollTableEntry {
	for _, entry := range r {
		if entry.Roll >= n {
			return entry
		}
	}
	return RollTableEntry{}
}

// max returns the maximum roll in the table.
// Important: this assumes that the table is sorted in ascending order.
func (r RollTable) max() int {
	return r[len(r)-1].Roll
}

// Params is a set of parameters to pass to the actions in a rollTable.
type Params struct {
	// client is the client to use for performing the action.
	client *codersdk.Client
	// me is the currently authenticated user. Lots of actions require this.
	me       codersdk.User
	log      slog.Logger
	headless bool
}

func logAdapter(ctx context.Context, log func(ctx context.Context, msg string, fields ...any)) func(string, ...interface{}) {
	return func(msg string, args ...interface{}) {
		log(ctx, msg, slog.F("args", fmt.Sprintf("%+v", args...)))
	}
}

func loadMainPage(ctx context.Context, p *Params) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	dir, err := os.MkdirTemp("", "scaletest-dashboard")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			p.log.Error(ctx, "remove temp dir", slog.Error(err))
		}
	}()

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(dir),
		chromedp.DisableGPU,
	)

	if !p.headless { // headless is the default
		allocOpts = append(allocOpts, chromedp.Flag("headless", false))
	}

	allocCtx, allocCtxCancel := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer allocCtxCancel()

	cdpCtx, cdpCancel := chromedp.NewContext(allocCtx, chromedp.WithDebugf(logAdapter(ctx, p.log.Debug)))
	defer cdpCancel()
	err = chromedp.Run(cdpCtx, chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			exp := cdp.TimeSinceEpoch(time.Now().Add(time.Hour))
			if err := network.SetCookie("coder_session_token", p.client.SessionToken()).
				WithExpires(&exp).
				WithDomain(p.client.URL.Host).
				WithHTTPOnly(false).
				Do(ctx); err != nil {
				return xerrors.Errorf("set cookie: %w", err)
			}
			return nil
		}),
		chromedp.Navigate(p.client.URL.String()),
		chromedp.WaitVisible(fmt.Sprintf(`div[title=%q]`, p.me.Username)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}),
	})
	if err != nil {
		return xerrors.Errorf("run chromedp: %w", err)
	}
	return nil
}
