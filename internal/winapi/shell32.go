//go:build windows

package winapi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ShellNotifyIcon adds, modifies, or removes a notification area icon.
func ShellNotifyIcon(message uint32, nid *NOTIFYICONDATAW) error {
	nid.Size = uint32(unsafe.Sizeof(*nid))
	r1, _, e := procShellNotifyIconW.Call(uintptr(message), uintptr(unsafe.Pointer(nid))) // #nosec G103 -- Audited Win32 unsafe interop.
	if r1 == 0 {
		return fmt.Errorf("Shell_NotifyIconW: %w", e)
	}
	return nil
}

// ShellExecute opens or launches a file or URL.
func ShellExecute(verb, file, params, dir string, show int32) error {
	verbPtr, _ := utf16PtrOrNil(verb)
	filePtr, err := windows.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	paramsPtr, _ := utf16PtrOrNil(params)
	dirPtr, _ := utf16PtrOrNil(dir)

	r1, _, e := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),   // #nosec G103 -- Audited Win32 unsafe interop.
		uintptr(unsafe.Pointer(filePtr)),   // #nosec G103 -- Audited Win32 unsafe interop.
		uintptr(unsafe.Pointer(paramsPtr)), // #nosec G103 -- Audited Win32 unsafe interop.
		uintptr(unsafe.Pointer(dirPtr)),    // #nosec G103 -- Audited Win32 unsafe interop.
		int32Param(show),
	)
	if r1 <= 32 {
		return fmt.Errorf("ShellExecuteW: code %d (%w)", r1, e)
	}
	return nil
}

func utf16PtrOrNil(s string) (*uint16, error) {
	if s == "" {
		return nil, nil
	}
	return windows.UTF16PtrFromString(s)
}

// SetTipString writes text into a fixed-size uint16 array, NUL-terminating.
func SetTipString(dst []uint16, s string) {
	src := windows.StringToUTF16(s)
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	for i := 0; i < n; i++ {
		dst[i] = src[i]
	}
	if n < len(dst) {
		dst[n-1] = 0
	}
}
