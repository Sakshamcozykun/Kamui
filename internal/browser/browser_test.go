package browser

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func TestPageState(t *testing.T) {
	l := launcher.New().Headless(true).Set("no-sandbox").Set("disable-gpu")
	u, _ := l.Launch()
	browser := rod.New().ControlURL(u)
	browser.Connect()
	defer browser.Close()

	page := browser.MustPage("https://vidsrc.to/embed/movie/tt1979376/")
	page.MustWaitLoad()
	time.Sleep(5 * time.Second)

	// Check window.CFG
	cfg := page.MustEval("() => window.CFG").String()
	fmt.Println("CFG:", cfg[:min(200, len(cfg))])

	// Check if there's a way to trigger stream loading
	result := page.MustEval("() => { try { if (typeof start === 'function') { start({data:{}}); return 'called start'; } return 'no start func'; } catch(e) { return 'error: ' + e.message; } }").String()
	fmt.Println("result:", result)

	time.Sleep(10 * time.Second)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}