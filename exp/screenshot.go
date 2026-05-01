//go:build windows

package exp

import (
	"fmt"
	"image"
	"strings"
	"syscall"
	"unsafe"

	"github.com/kbinani/screenshot"
	"github.com/winlabs/gowin32"
	"golang.org/x/sys/windows"
)

var (
	user32            = windows.NewLazyDLL("user32.dll")
	procEnumWindows   = user32.NewProc("EnumWindows")
	procGetWindowRect = user32.NewProc("GetWindowRect")
)

func newWindowCapturer(title string) *windowCapturer {
	wh := &windowCapturer{}
	wh.title = title
	wh.findWindowCallback = windows.NewCallback(func(h windows.Handle, _ uintptr) uintptr {
		wh.windowHwnd = 0
		text, err := gowin32.GetWindowText(syscall.Handle(h))
		if err != nil {
			return 1
		}

		if strings.TrimSpace(text) == wh.title {
			wh.windowHwnd = h
			return 0
		}

		return 1
	})

	return wh
}

type windowCapturer struct {
	title              string
	findWindowCallback uintptr
	windowHwnd         windows.Handle
}

func (wh *windowCapturer) findWindow() (windows.Handle, error) {
	_, _, _ = procEnumWindows.Call(wh.findWindowCallback, 0)
	if wh.windowHwnd == 0 {
		return 0, fmt.Errorf("failed to find %q window", wh.title)
	}
	return wh.windowHwnd, nil
}

func (wh *windowCapturer) capture() (*image.RGBA, error) {
	r := struct{ l, t, r, b int32 }{}

	hwnd, err := wh.findWindow()
	if err != nil {
		return nil, err
	}

	ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))
	if ret == 0 {
		return nil, fmt.Errorf("failed to get %q window rect", wh.title)
	}

	img, err := screenshot.CaptureRect(image.Rect(int(r.l), int(r.t), int(r.r), int(r.b)))
	if err != nil {
		return nil, err
	}

	return img, nil
}
