package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// ExtractStream launches a headless Chromium browser, navigates to the given
// embed page (e.g. VidSrc, 2Embed), runs the page's JavaScript, and attempts
// to intercept the real stream URL (m3u8 or mp4) observed in network requests.
//
// Streaming sites protect their streams behind JavaScript/WASM encryption,
// so no pure Go HTTP client can reverse that. A real browser is required.
//
// The function returns the first stream URL it intercepts, or an error if
// the page does not produce a playable URL within the given timeout.
func ExtractStream(ctx context.Context, pageURL string, timeout time.Duration) (string, error) {
	// Launch Chromium (cached binary ~150 MB, downloaded once).
	l := launcher.New().Headless(true).Set("no-sandbox").Set("disable-gpu")
	u, err := l.Launch()
	if err != nil {
		return "", fmt.Errorf("launch browser: %w", err)
	}
	defer l.Cleanup()

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return "", fmt.Errorf("connect browser: %w", err)
	}
	defer browser.Close()

	page := browser.MustPage(pageURL)
	defer page.Close()

	page.MustWaitLoad()

	// Allow JavaScript to initialise (load CONFIG, vs_src.php, etc.).
	// We then intercept network requests looking for stream URLs.
	streamCh := make(chan string, 1)

	// Collect any m3u8/mp4 URL that appears in network requests.
	go func() {
		page.EachEvent(func(e *proto.NetworkRequestWillBeSent) bool {
			u := e.Request.URL
			if strings.Contains(u, ".m3u8") || strings.Contains(u, ".mp4") {
				select {
				case streamCh <- u:
				default:
				}
			}
			return false
		})
	}()

	// Wait for the page to run its setup and produce a stream URL.
	select {
	case url := <-streamCh:
		return url, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("no stream URL intercepted within %s", timeout)
	case <-ctx.Done():
		return "", ctx.Err()
	}
}