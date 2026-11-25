package browser

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func TestPageState(t *testing.T) {
	if os.Getenv("KAMUI_BROWSER_LIVE_TEST") != "1" {
		t.Skip("live provider test; set KAMUI_BROWSER_LIVE_TEST=1 to run")
	}
	l := launcher.New().Headless(true).Set("no-sandbox").Set("disable-gpu")
	u, err := l.Launch()
	if err != nil {
		t.Fatal(err)
	}
	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		t.Fatal(err)
	}
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
