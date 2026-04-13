//go:build windows

package platform

import (
	"errors"

	"golang.org/x/sys/windows"
)

var ErrAlreadyRunning = errors.New("another Windows Task Manager instance is already running")

// AcquireSingleInstance reserves a named system-wide mutex for the lifetime
// of the current process. Call the returned release func during shutdown.
func AcquireSingleInstance(name string) (func(), error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	h, err := windows.CreateMutex(nil, false, namePtr)
	if err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			if h != 0 {
				_ = windows.CloseHandle(h)
			}
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}
	if errors.Is(windows.GetLastError(), windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(h)
		return nil, ErrAlreadyRunning
	}

	return func() {
		_ = windows.CloseHandle(h)
	}, nil
}
