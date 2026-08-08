package httpclient

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/j3ssie/metabigor/internal/output"
)

// ChromeGet navigates to url via headless Chrome and returns the rendered HTML.
func ChromeGet(targetURL string, timeoutSec int) (string, error) {
	output.Debug("Chrome GET %s", targetURL)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), chromeOpts("")...)
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

func chromeOpts(proxy string) []chromedp.ExecAllocatorOption {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("ignore-certificate-errors", true),
	)
	if proxy != "" {
		opts = append(opts, chromedp.ProxyServer(proxy))
		output.Debug("Chrome using proxy: %s", proxy)
	}
	return opts
}

// pollInterval is how often GetJSON re-checks the DOM while waiting for a body.
const pollInterval = 300 * time.Millisecond

// jsonBodyExpr reads the raw text of the document Chrome renders for a JSON
// response. Chrome wraps non-HTML bodies in a single <pre>.
const jsonBodyExpr = `(() => { const p = document.querySelector('pre'); return p ? p.textContent : ''; })()`

// ChromeSession is a long-lived headless Chrome instance reused across requests.
// Booting Chrome costs a second or two, so paginated scrapes share one session
// rather than paying that per page.
//
// Requests are serialized: the session drives a single tab.
type ChromeSession struct {
	mu          sync.Mutex
	ctx         context.Context
	cancelCtx   context.CancelFunc
	cancelAlloc context.CancelFunc
	timeout     time.Duration
}

// NewChromeSession launches headless Chrome and returns a session ready to use.
// Callers must Close it.
func NewChromeSession(timeoutSec int, proxy string) (*ChromeSession, error) {
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), chromeOpts(proxy)...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)

	// Running with no actions boots the browser, so a missing Chrome surfaces
	// here instead of on the first real request.
	if err := chromedp.Run(ctx); err != nil {
		cancelCtx()
		cancelAlloc()
		return nil, fmt.Errorf("%w: %v", ErrNoChrome, err)
	}

	return &ChromeSession{
		ctx:         ctx,
		cancelCtx:   cancelCtx,
		cancelAlloc: cancelAlloc,
		timeout:     time.Duration(timeoutSec) * time.Second,
	}, nil
}

// Close shuts down the browser.
func (s *ChromeSession) Close() {
	s.cancelCtx()
	s.cancelAlloc()
}

// GetJSON navigates to targetURL and returns the JSON body once it appears.
//
// Sites behind an interstitial bot challenge (grep.app sits behind Vercel's
// checkpoint) answer the first request with an HTML page and only swap in the
// real body after their script clears, so this polls the DOM instead of trusting
// whatever loaded first.
func (s *ChromeSession) GetJSON(targetURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	output.Debug("Chrome GET %s", targetURL)

	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	// A challenge page reloads itself, which can abort the initial navigation.
	// The poll below decides success, so just hold on to the error for context.
	navErr := chromedp.Run(ctx, chromedp.Navigate(targetURL))

	for {
		var body string
		// Evaluate errors while the page is mid-navigation; keep polling.
		if err := chromedp.Run(ctx, chromedp.Evaluate(jsonBodyExpr, &body)); err == nil {
			if trimmed := strings.TrimSpace(body); isJSONDoc(trimmed) {
				output.Debug("Chrome GET %s -> %d bytes", targetURL, len(trimmed))
				return trimmed, nil
			}
		}

		select {
		case <-ctx.Done():
			if navErr != nil {
				return "", fmt.Errorf("navigate %s: %w", targetURL, navErr)
			}
			return "", fmt.Errorf("timed out after %s waiting for a JSON body from %s (bot challenge may not have cleared)", s.timeout, targetURL)
		case <-time.After(pollInterval):
		}
	}
}

func isJSONDoc(s string) bool {
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}
