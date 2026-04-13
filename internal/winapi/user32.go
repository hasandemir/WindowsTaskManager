//go:build windows

package winapi

import (
	"fmt"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DefWindowProc calls the default window procedure.
func DefWindowProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	r1, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r1
}

// RegisterClassEx registers a window class.
func RegisterClassEx(c *WNDCLASSEXW) (uint16, error) {
	c.Size = uint32(unsafe.Sizeof(*c))
	r1, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(c))) // #nosec G103 -- Audited Win32 unsafe interop.
	if r1 == 0 {
		return 0, fmt.Errorf("RegisterClassExW: %w", e)
	}
	if uint64(r1) > math.MaxUint16 {
		return 0, fmt.Errorf("RegisterClassExW: class id %d overflows uint16", r1)
	}
	return uint16(r1), nil
}

// CreateWindowEx creates a window.
func CreateWindowEx(exStyle uint32, className, windowName *uint16, style uint32,
	x, y, width, height int32, parent uintptr, menu, instance uintptr,
	param unsafe.Pointer) (uintptr, error) {
	r1, _, e := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(className)),  // #nosec G103 -- Audited Win32 unsafe interop.
		uintptr(unsafe.Pointer(windowName)), // #nosec G103 -- Audited Win32 unsafe interop.
		uintptr(style),
		int32Param(x),
		int32Param(y),
		int32Param(width),
		int32Param(height),
		parent,
		menu,
		instance,
		uintptr(param),
	)
	if r1 == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %w", e)
	}
	return r1, nil
}

// DestroyWindow destroys a window.
func DestroyWindow(hwnd uintptr) {
	r1, _, _ := procDestroyWindow.Call(hwnd)
	_ = r1
}

// GetMessage retrieves a message from the calling thread's queue.
func GetMessage(msg *MSG, hwnd uintptr) int32 {
	r1, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(msg)), hwnd, 0, 0) // #nosec G103 -- Audited Win32 unsafe interop.
	switch r1 {
	case 0:
		return 0
	case ^uintptr(0):
		return -1
	default:
		return 1
	}
}

// TranslateMessage translates virtual-key messages.
func TranslateMessage(msg *MSG) {
	r1, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(msg))) // #nosec G103 -- Audited Win32 unsafe interop.
	_ = r1
}

// DispatchMessage dispatches a message to a window procedure.
func DispatchMessage(msg *MSG) uintptr {
	r1, _, _ := procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg))) // #nosec G103 -- Audited Win32 unsafe interop.
	return r1
}

// PostQuitMessage posts a WM_QUIT message.
func PostQuitMessage(exitCode int32) {
	r1, _, _ := procPostQuitMessage.Call(int32Param(exitCode))
	_ = r1
}

// PostMessage posts a message into the message queue of the specified window.
func PostMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) {
	r1, _, _ := procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	_ = r1
}

// CreatePopupMenu creates a popup menu.
func CreatePopupMenu() uintptr {
	r1, _, _ := procCreatePopupMenu.Call()
	return r1
}

// AppendMenu adds an item to a menu.
func AppendMenu(menu uintptr, flags, idNewItem uintptr, text string) error {
	textPtr, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return err
	}
	r1, _, e := procAppendMenuW.Call(menu, flags, idNewItem, uintptr(unsafe.Pointer(textPtr))) // #nosec G103 -- Audited Win32 unsafe interop.
	if r1 == 0 {
		return fmt.Errorf("AppendMenuW: %w", e)
	}
	return nil
}

// DestroyMenu destroys a menu.
func DestroyMenu(menu uintptr) {
	r1, _, _ := procDestroyMenu.Call(menu)
	_ = r1
}

// TrackPopupMenu displays a popup menu.
func TrackPopupMenu(menu uintptr, flags uint32, x, y int32, hwnd uintptr) uintptr {
	r1, _, _ := procTrackPopupMenu.Call(
		menu, uintptr(flags), int32Param(x), int32Param(y), 0, hwnd, 0,
	)
	return r1
}

// GetCursorPos retrieves the cursor position.
func GetCursorPos() (POINT, error) {
	var pt POINT
	r1, _, e := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))) // #nosec G103 -- Audited Win32 unsafe interop.
	if r1 == 0 {
		return pt, fmt.Errorf("GetCursorPos: %w", e)
	}
	return pt, nil
}

// SetForegroundWindow brings the specified window to the foreground.
func SetForegroundWindow(hwnd uintptr) {
	r1, _, _ := procSetForegroundWindow.Call(hwnd)
	_ = r1
}

// LoadIcon loads a stock icon (e.g., IDI_APPLICATION).
func LoadIcon(name uintptr) uintptr {
	r1, _, _ := procLoadIconW.Call(0, name)
	return r1
}

// LoadCursor loads a stock cursor (e.g., IDC_ARROW).
func LoadCursor(name uintptr) uintptr {
	r1, _, _ := procLoadCursorW.Call(0, name)
	return r1
}

// Stock icons / cursors.
const (
	IDI_APPLICATION uintptr = 32512
	IDC_ARROW       uintptr = 32512
)

func int32Param(v int32) uintptr {
	// #nosec G115 -- Win32 APIs consume signed 32-bit values in these fields.
	return uintptr(uint32(v))
}

// TPM constants.
const (
	TPM_LEFTALIGN   = 0x0000
	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100
)
