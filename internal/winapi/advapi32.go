//go:build windows

package winapi

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// HKEY values.
const (
	HKEY_LOCAL_MACHINE = 0x80000002
	KEY_READ           = 0x20019
	REG_SZ             = 1
	REG_DWORD          = 4
	REG_QWORD          = 11
)

// RegReadString reads a string registry value (REG_SZ).
func RegReadString(rootKey uintptr, subKey, value string) (string, error) {
	subPtr, err := windows.UTF16PtrFromString(subKey)
	if err != nil {
		return "", err
	}
	var hKey uintptr
	r1, _, e := procRegOpenKeyExW.Call(
		rootKey,
		uintptr(unsafe.Pointer(subPtr)),
		0,
		KEY_READ,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r1 != 0 {
		return "", e
	}
	defer procRegCloseKey.Call(hKey)

	valPtr, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return "", err
	}

	var bufType uint32
	var bufSize uint32 = 1024
	buf := make([]uint16, bufSize/2)
	r1, _, e = procRegQueryValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valPtr)),
		0,
		uintptr(unsafe.Pointer(&bufType)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bufSize)),
	)
	if r1 != 0 {
		return "", e
	}
	return windows.UTF16ToString(buf), nil
}

// RegReadDWORD reads a DWORD registry value.
func RegReadDWORD(rootKey uintptr, subKey, value string) (uint32, error) {
	subPtr, err := windows.UTF16PtrFromString(subKey)
	if err != nil {
		return 0, err
	}
	var hKey uintptr
	r1, _, e := procRegOpenKeyExW.Call(
		rootKey,
		uintptr(unsafe.Pointer(subPtr)),
		0,
		KEY_READ,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r1 != 0 {
		return 0, e
	}
	defer procRegCloseKey.Call(hKey)

	valPtr, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return 0, err
	}

	var bufType uint32
	var data uint32
	var bufSize uint32 = 4
	r1, _, e = procRegQueryValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valPtr)),
		0,
		uintptr(unsafe.Pointer(&bufType)),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&bufSize)),
	)
	if r1 != 0 {
		return 0, e
	}
	return data, nil
}

// RegReadQWORD reads a QWORD registry value.
func RegReadQWORD(rootKey uintptr, subKey, value string) (uint64, error) {
	subPtr, err := windows.UTF16PtrFromString(subKey)
	if err != nil {
		return 0, err
	}
	var hKey uintptr
	r1, _, e := procRegOpenKeyExW.Call(
		rootKey,
		uintptr(unsafe.Pointer(subPtr)),
		0,
		KEY_READ,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r1 != 0 {
		return 0, e
	}
	defer procRegCloseKey.Call(hKey)

	valPtr, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return 0, err
	}

	var bufType uint32
	var data uint64
	var bufSize uint32 = 8
	r1, _, e = procRegQueryValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valPtr)),
		0,
		uintptr(unsafe.Pointer(&bufType)),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&bufSize)),
	)
	if r1 != 0 {
		return 0, e
	}
	return data, nil
}
