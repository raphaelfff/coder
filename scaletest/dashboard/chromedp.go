package dashboard

import (
	"context"
	"os"

	"github.com/chromedp/chromedp"
)

func initChromeDPCtx(ctx context.Context, headless bool) (context.Context, context.CancelFunc, error) {
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

	return cdpCtx, cancelFunc, nil
}
