package dashboard

import (
	"context"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"golang.org/x/xerrors"
	"net/url"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

// this is a default set of selectors for the dashboard
var defaultSelectors = selectors(
	"workspaces_list", `a[href="/workspaces"]:not(.active)`,
	"templates_list", `a[href="/templates"]:not(.active)`,
	"users_list", `a[href^="/users"]:not(.active)`,
	"deployment_status", `a[href="/deployment/general"]:not(.active)`,
	"starter_templates", `a[href="/starter-templates"]`,
	"table_element", `tr[role="button"]`,
	"templates_table_header", `a[href^="/templates/"]:not([aria-current])`,
)

type selector string
type label string

func selectors(kvs ...string) map[label]selector {
	m := make(map[label]selector)
	for i := 0; i < len(kvs); i += 2 {
		m[label(kvs[i])] = selector(kvs[i+1])
	}
	return m
}

func clickRandElement(ctx context.Context, sels map[label]selector) (label, error) {
	// iterate through selectors at random
	// and pick one that matches at random
	var matched selector
	var matchedLabel label
	var found bool
	var err error
	for l, s := range sels {
		matched, found, err = randMatch(ctx, s)
		if err != nil {
			return "", xerrors.Errorf("find matches for %q: %w", s, err)
		}
		if !found {
			continue
		}
		matchedLabel = l
		break
	}
	if !found {
		return "", xerrors.Errorf("no matches found")
	}

	// click it
	if err := click(ctx, matched); err != nil {
		return "", xerrors.Errorf("click %q: %w", matched, err)
	}
	return matchedLabel, nil
}

// randMatch returns a random match for the given selector.
// The returned selector is the full XPath of the matched node.
// If no matches are found, an error is returned.
// If multiple matches are found, one is chosen at random.
func randMatch(ctx context.Context, s selector) (selector, bool, error) {
	var nodes []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes(s, &nodes, chromedp.NodeVisible, chromedp.AtLeast(0)))
	if err != nil {
		return "", false, xerrors.Errorf("get nodes for selector %q: %w", s, err)
	}
	if nodes == nil || len(nodes) == 0 {
		return "", false, nil
	}
	nodeMap := make(map[*cdp.Node]struct{})
	for _, n := range nodes {
		nodeMap[n] = struct{}{}
	}
	// Take the first element from the map; this will be a random
	// element from the slice of nodes.
	for n := range nodeMap {
		return selector(n.FullXPath()), true, nil
	}
	return "", false, xerrors.Errorf("unreachable")
}

func click(ctx context.Context, s selector) error {
	return chromedp.Run(ctx, chromedp.Click(s, chromedp.NodeVisible))
}

func initChromeDPCtx(ctx context.Context, u *url.URL, sessionToken string, headless bool) (context.Context, context.CancelFunc, error) {
	dir, err := os.MkdirTemp("", "scaletest-dashboard")
	if err != nil {
		return nil, nil, err
	}

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(dir),
		chromedp.DisableGPU,
	)

	if !headless { // headless is the default
		allocOpts = append(allocOpts, chromedp.Flag("headless", false))
	}

	allocCtx, allocCtxCancel := chromedp.NewExecAllocator(ctx, allocOpts...)
	cdpCtx, cdpCancel := chromedp.NewContext(allocCtx)
	cancelFunc := func() {
		cdpCancel()
		allocCtxCancel()
		_ = os.RemoveAll(dir)
	}

	// set cookies
	if err := setSessionTokenCookie(cdpCtx, sessionToken, u.Host); err != nil {
		cancelFunc()
		return nil, nil, xerrors.Errorf("set session token cookie: %w", err)
	}

	// visit main page
	if err := visitMainPage(cdpCtx, u); err != nil {
		cancelFunc()
		return nil, nil, xerrors.Errorf("visit main page: %w", err)
	}

	return cdpCtx, cancelFunc, nil
}

func setSessionTokenCookie(ctx context.Context, token, domain string) error {
	exp := cdp.TimeSinceEpoch(time.Now().Add(30 * 24 * time.Hour))
	err := chromedp.Run(ctx, network.SetCookie("coder_session_token", token).
		WithExpires(&exp).
		WithDomain(domain).
		WithHTTPOnly(false))
	if err != nil {
		return xerrors.Errorf("set coder_session_token cookie: %w", err)
	}
	return nil
}

func visitMainPage(ctx context.Context, u *url.URL) error {
	return chromedp.Run(ctx, chromedp.Navigate(u.String()))
}
