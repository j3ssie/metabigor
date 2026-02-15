package httpclient

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/j3ssie/metabigor/internal/output"
)

// ChromeGet navigates to url via headless Chrome and returns the rendered HTML.
func ChromeGet(targetURL string, timeoutSec int) (string, error) {
	output.Debug("Chrome GET %s", targetURL)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("ignore-certificate-errors", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.WaitReady("body"),
		chromedp.OuterHTML("html", &body),
	)
	if err != nil {
		output.Debug("Chrome GET %s failed: %v", targetURL, err)
		return "", err
	}
	output.Debug("Chrome GET %s -> %d bytes", targetURL, len(body))
	return body, nil
}
