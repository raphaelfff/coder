package dashboard

import (
	"cdr.dev/slog"
	"context"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"net/url"
	"time"

	"github.com/coder/coder/v2/codersdk"
)

// DefaultActions is a table of actions to perform.
// D&D nerds will feel right at home here :-)
// Note that the order of the table is important!
// Entries must be in ascending order.
var DefaultActions RollTable = []RollTableEntry{
	{0, visitHomepage, "visit home page"},
	{1, visitFirstWorkspace, "visit first workspace"},
	{2, visitWorkspaceBuildLog, "visit workspace build log"},
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
	URL          *url.URL
	SessionToken string
	// me is the currently authenticated user. Lots of actions require this.
	me  codersdk.User
	log slog.Logger
}

func visitHomepage(ctx context.Context, p *Params) error {
	l := withLog(p.log)
	userAvatarSelector := fmt.Sprintf(`div[title=%q]`, p.me.Username)
	return chromedp.Run(ctx, chromedp.Tasks{
		l(setSessionTokenCookie(p.SessionToken, p.URL.Host), "set cookie"),
		l(chromedp.Navigate(p.URL.String()), "visit homepage"),
		l(chromedp.WaitVisible(userAvatarSelector), "wait for user avatar"),
	})
}

func visitFirstWorkspace(ctx context.Context, p *Params) error {
	l := withLog(p.log)
	workspaceRowSelector := `table tr[tabindex]`
	return chromedp.Run(ctx, chromedp.Tasks{
		l(setSessionTokenCookie(p.SessionToken, p.URL.Host), "set cookie"),
		l(chromedp.Navigate(p.URL.String()), "visit homepage"),
		l(chromedp.Click(workspaceRowSelector, chromedp.NodeVisible), "click workspace row"),
	})
}

func visitWorkspaceBuildLog(ctx context.Context, p *Params) error {
	l := withLog(p.log)
	workspaceRowSelector := `table tr[tabindex]`
	workspaceBuildRowSelector := `table[data-testid="builds-table"] tr[role="button"]`
	return chromedp.Run(ctx, chromedp.Tasks{
		l(setSessionTokenCookie(p.SessionToken, p.URL.Host), "set cookie"),
		l(chromedp.Navigate(p.URL.String()), "visit homepage"),
		l(chromedp.Click(workspaceRowSelector, chromedp.NodeVisible), "click workspace row"),
		l(chromedp.Click(workspaceBuildRowSelector, chromedp.NodeVisible), "click workspace build row"),
	})
}

func setSessionTokenCookie(token, domain string) chromedp.Action {
	exp := cdp.TimeSinceEpoch(time.Now().Add(30 * 24 * time.Hour))
	return network.SetCookie("coder_session_token", token).
		WithExpires(&exp).
		WithDomain(domain).
		WithHTTPOnly(false)
}

func withLog(log slog.Logger) func(act chromedp.Action, name string) chromedp.Action {
	return func(act chromedp.Action, name string) chromedp.Action {
		return chromedp.ActionFunc(func(ctx context.Context) error {
			log.Debug(ctx, "start action", slog.F("name", name))
			start := time.Now()
			err := act.Do(ctx)
			elapsed := time.Since(start)
			if err != nil {
				log.Error(ctx, "do action", slog.F("name", name), slog.Error(err), slog.F("elapsed", elapsed))
				return err
			}
			log.Debug(ctx, "completed action", slog.F("name", name), slog.F("elapsed", elapsed))
			return nil
		})
	}
}
