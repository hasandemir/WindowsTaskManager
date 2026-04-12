//go:build windows

package winapi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	pdhFmtDouble = 0x00000200
	pdhMoreData  = 0x800007D2
)

type pdhFmtCounterValueDouble struct {
	CStatus uint32
	_       uint32
	Value   float64
}

type pdhFmtCounterValueItemDouble struct {
	Name  *uint16
	Value pdhFmtCounterValueDouble
}

func pdhError(op string, status uintptr) error {
	if status == 0 {
		return nil
	}
	return fmt.Errorf("%s: 0x%08X", op, uint32(status))
}

// PdhQuery is an opaque PDH query handle.
type PdhQuery uintptr

// PdhCounter is an opaque PDH counter handle.
type PdhCounter uintptr

// OpenPdhQuery opens a PDH query for subsequent counter registration.
func OpenPdhQuery() (PdhQuery, error) {
	var query PdhQuery
	r1, _, _ := procPdhOpenQueryW.Call(0, 0, uintptr(unsafe.Pointer(&query)))
	if err := pdhError("PdhOpenQueryW", r1); err != nil {
		return 0, err
	}
	return query, nil
}

// Close closes the underlying PDH query handle.
func (q PdhQuery) Close() {
	if q == 0 {
		return
	}
	procPdhCloseQuery.Call(uintptr(q))
}

// AddEnglishCounter registers a wildcard or concrete counter path with a query.
func AddEnglishCounter(query PdhQuery, path string) (PdhCounter, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var counter PdhCounter
	r1, _, _ := procPdhAddEnglishCounterW.Call(
		uintptr(query),
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&counter)),
	)
	if err := pdhError("PdhAddEnglishCounterW", r1); err != nil {
		return 0, err
	}
	return counter, nil
}

// CollectQueryData samples all counters attached to the PDH query.
func CollectQueryData(query PdhQuery) error {
	r1, _, _ := procPdhCollectQueryData.Call(uintptr(query))
	return pdhError("PdhCollectQueryData", r1)
}

// GetFormattedCounterArrayDouble returns the cooked values for a wildcard counter.
func GetFormattedCounterArrayDouble(counter PdhCounter) (map[string]float64, error) {
	var bufSize uint32
	var itemCount uint32
	r1, _, _ := procPdhGetFormattedCounterArrayW.Call(
		uintptr(counter),
		uintptr(pdhFmtDouble),
		uintptr(unsafe.Pointer(&bufSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		0,
	)
	if r1 != pdhMoreData && r1 != 0 {
		return nil, pdhError("PdhGetFormattedCounterArrayW(size)", r1)
	}
	if bufSize == 0 || itemCount == 0 {
		return map[string]float64{}, nil
	}

	buf := make([]byte, bufSize)
	r1, _, _ = procPdhGetFormattedCounterArrayW.Call(
		uintptr(counter),
		uintptr(pdhFmtDouble),
		uintptr(unsafe.Pointer(&bufSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if err := pdhError("PdhGetFormattedCounterArrayW(data)", r1); err != nil {
		return nil, err
	}

	items := unsafe.Slice((*pdhFmtCounterValueItemDouble)(unsafe.Pointer(&buf[0])), itemCount)
	values := make(map[string]float64, itemCount)
	for _, item := range items {
		if item.Name == nil || item.Value.CStatus != 0 {
			continue
		}
		values[windows.UTF16PtrToString(item.Name)] = item.Value.Value
	}
	return values, nil
}
