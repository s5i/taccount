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

func windowHandle(title string) (windows.Handle, error) {
	var hwnd windows.Handle
	_, _, _ = procEnumWindows.Call(windows.NewCallback(func(h windows.Handle, _ uintptr) uintptr {
		text, err := gowin32.GetWindowText(syscall.Handle(h))
		if err != nil {
			return 1
		}

		if strings.TrimSpace(text) == title {
			hwnd = h
			return 0
		}

		return 1
	}), 0)

	if hwnd == 0 {
		return 0, fmt.Errorf("window not found")
	}

	return hwnd, nil
}

func captureWindow(title string) (*image.RGBA, error) {
	hwnd, err := windowHandle(title)
	if err != nil {
		return nil, err
	}

	r := struct{ l, t, r, b int32 }{}
	ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))
	if ret == 0 {
		return nil, fmt.Errorf("GetWindowRect failed")
	}

	img, err := screenshot.CaptureRect(image.Rect(int(r.l), int(r.t), int(r.r), int(r.b)))
	if err != nil {
		return nil, err
	}
	return img, nil
}
