package core

import (
    "context"
    "log"
    "os"
    "path"
    "path/filepath"
    "time"

    "github.com/chromedp/chromedp"
)

// RequestWithChrome Do request with real browser
func RequestWithChrome(url string, contentID string, timeout int) string {
    // prepare the chrome options
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", true),
        chromedp.Flag("ignore-certificate-errors", true),
        chromedp.Flag("disable-gpu", true),
        chromedp.Flag("enable-automation", true),
        chromedp.Flag("disable-extensions", true),
        chromedp.Flag("disable-setuid-sandbox", true),
        chromedp.Flag("disable-web-security", true),
        chromedp.Flag("no-first-run", true),
        chromedp.Flag("no-default-browser-check", true),
    )

    allocCtx, bcancel := chromedp.NewExecAllocator(context.Background(), opts...)
    defer bcancel()

    ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
    ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
    defer cancel()

    // run task list
    var data string
    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.OuterHTML(contentID, &data, chromedp.NodeVisible, chromedp.ByID),
    )
    DebugF(data)

    // clean chromedp-runner folder
    cleanUp()

    if err != nil {
        InforF("[ERRR] %v", err)
        return ""
    }
    return data
}

func cleanUp() {
    tmpFolder := path.Join(os.TempDir(), "chromedp-runner*")
    if _, err := os.Stat("/tmp/"); !os.IsNotExist(err) {
        tmpFolder = path.Join("/tmp/", "chromedp-runner*")
    }
    junks, err := filepath.Glob(tmpFolder)
    if err != nil {
        return
    }
    for _, junk := range junks {
        os.RemoveAll(junk)
    }
}
