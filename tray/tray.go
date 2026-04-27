//go:build windows

package tray

import (
	"os/exec"

	"github.com/getlantern/systray"
	"github.com/s5i/tassist/assets"
)

func Run(url string) {
	systray.Run(func() { onReady(url) }, func() {})
}

func onReady(url string) {
	systray.SetIcon(assets.Favicon)
	systray.SetTooltip("Tibiantis Assistant")

	mOpen := systray.AddMenuItem("Open", "Open in browser")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit the application")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				OpenBrowser(url)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func OpenBrowser(url string) {
	exec.Command("explorer", url).Start()
}
