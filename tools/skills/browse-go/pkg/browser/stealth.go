// Package browser provides the browser automation core backed by chromedp.
package browser

import (
	"context"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// WebdriverMaskScript removes the navigator.webdriver automation flag.
const WebdriverMaskScript = `Object.defineProperty(navigator, 'webdriver', { get: () => false });`

// StealthLaunchArgs are flags passed to chromium to suppress automation
// signals at the protocol layer.
var StealthLaunchArgs = []string{
	"--disable-blink-features=AutomationControlled",
}

// ApplyStealth injects the webdriver mask script into every new document
// in the context. Call once after creating a BrowserContext.
func ApplyStealth(ctx context.Context) error {
	return chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(WebdriverMaskScript).Do(c)
		return err
	}))
}
