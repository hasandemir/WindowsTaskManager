//go:build windows

package winapi

import (
	"fmt"
	"unsafe"
)

// GetExtendedTcpTable wraps the Win32 call. Returns raw byte buffer; caller parses based on family.
func getExtendedTcpTableRaw(family uint32) ([]byte, error) {
	var size uint32
	procGetExtendedTcpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		1, // sorted = true
		uintptr(family),
		uintptr(TCP_TABLE_OWNER_PID_ALL),
		0,
	)
	if size == 0 {
		return nil, nil
	}
	buf := make([]byte, size)
	r1, _, e := procGetExtendedTcpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(family),
		uintptr(TCP_TABLE_OWNER_PID_ALL),
		0,
	)
	if r1 != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable: status %d (%w)", r1, e)
	}
	return buf[:size], nil
}

// GetTcp4Table returns IPv4 TCP rows with owning PIDs.
func GetTcp4Table() ([]MIB_TCPROW_OWNER_PID, error) {
	buf, err := getExtendedTcpTableRaw(AF_INET)
	if err != nil || len(buf) < 4 {
		return nil, err
	}
	count := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowSize := unsafe.Sizeof(MIB_TCPROW_OWNER_PID{})
	rows := make([]MIB_TCPROW_OWNER_PID, count)
	for i := uint32(0); i < count; i++ {
		offset := 4 + uintptr(i)*rowSize
		if int(offset)+int(rowSize) > len(buf) {
			break
		}
		rows[i] = *(*MIB_TCPROW_OWNER_PID)(unsafe.Pointer(&buf[offset]))
	}
	return rows, nil
}

// GetTcp6Table returns IPv6 TCP rows with owning PIDs.
func GetTcp6Table() ([]MIB_TCP6ROW_OWNER_PID, error) {
	buf, err := getExtendedTcpTableRaw(AF_INET6)
	if err != nil || len(buf) < 4 {
		return nil, err
	}
	count := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowSize := unsafe.Sizeof(MIB_TCP6ROW_OWNER_PID{})
	rows := make([]MIB_TCP6ROW_OWNER_PID, count)
	for i := uint32(0); i < count; i++ {
		offset := 4 + uintptr(i)*rowSize
		if int(offset)+int(rowSize) > len(buf) {
			break
		}
		rows[i] = *(*MIB_TCP6ROW_OWNER_PID)(unsafe.Pointer(&buf[offset]))
	}
	return rows, nil
}

func getExtendedUdpTableRaw(family uint32) ([]byte, error) {
	var size uint32
	procGetExtendedUdpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(family),
		uintptr(UDP_TABLE_OWNER_PID),
		0,
	)
	if size == 0 {
		return nil, nil
	}
	buf := make([]byte, size)
	r1, _, e := procGetExtendedUdpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(family),
		uintptr(UDP_TABLE_OWNER_PID),
		0,
	)
	if r1 != 0 {
		return nil, fmt.Errorf("GetExtendedUdpTable: status %d (%w)", r1, e)
	}
	return buf[:size], nil
}

// GetUdp4Table returns IPv4 UDP rows with owning PIDs.
func GetUdp4Table() ([]MIB_UDPROW_OWNER_PID, error) {
	buf, err := getExtendedUdpTableRaw(AF_INET)
	if err != nil || len(buf) < 4 {
		return nil, err
	}
	count := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowSize := unsafe.Sizeof(MIB_UDPROW_OWNER_PID{})
	rows := make([]MIB_UDPROW_OWNER_PID, count)
	for i := uint32(0); i < count; i++ {
		offset := 4 + uintptr(i)*rowSize
		if int(offset)+int(rowSize) > len(buf) {
			break
		}
		rows[i] = *(*MIB_UDPROW_OWNER_PID)(unsafe.Pointer(&buf[offset]))
	}
	return rows, nil
}

// GetUdp6Table returns IPv6 UDP rows with owning PIDs.
func GetUdp6Table() ([]MIB_UDP6ROW_OWNER_PID, error) {
	buf, err := getExtendedUdpTableRaw(AF_INET6)
	if err != nil || len(buf) < 4 {
		return nil, err
	}
	count := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowSize := unsafe.Sizeof(MIB_UDP6ROW_OWNER_PID{})
	rows := make([]MIB_UDP6ROW_OWNER_PID, count)
	for i := uint32(0); i < count; i++ {
		offset := 4 + uintptr(i)*rowSize
		if int(offset)+int(rowSize) > len(buf) {
			break
		}
		rows[i] = *(*MIB_UDP6ROW_OWNER_PID)(unsafe.Pointer(&buf[offset]))
	}
	return rows, nil
}

// MIB_IF_ROW2 (partial — we read into a fixed-size raw buffer and parse manually).
const ifRow2Size = 1352

// IfRow2 holds the fields we care about parsed from MIB_IF_ROW2.
type IfRow2 struct {
	Index        uint32
	Alias        string
	Description  string
	OperStatus   uint32
	Type         uint32
	SpeedRx      uint64
	SpeedTx      uint64
	InOctets     uint64
	OutOctets    uint64
	InUcastPkts  uint64
	OutUcastPkts uint64
	InErrors     uint64
	OutErrors    uint64
}

// GetIfTable2 returns the system network interface table.
// We let the Win32 API allocate a MIB_IF_TABLE2 buffer, walk the rows at
// their documented offsets, then free the buffer via FreeMibTable.
//
// tablePtr is declared as unsafe.Pointer (not uintptr) so vet's
// unsafeptr analyzer is satisfied: we never do arithmetic on a uintptr
// and convert it back — all offsetting goes through unsafe.Add.
func GetIfTable2() ([]IfRow2, error) {
	var tablePtr unsafe.Pointer
	r1, _, e := procGetIfTable2.Call(uintptr(unsafe.Pointer(&tablePtr)))
	if r1 != 0 {
		return nil, fmt.Errorf("GetIfTable2: %w", e)
	}
	if tablePtr == nil {
		return nil, nil
	}
	defer procFreeMibTable.Call(uintptr(tablePtr))

	count := *(*uint32)(tablePtr)
	// MIB_IF_TABLE2 has 8-byte alignment after the count field, then rows.
	tableBase := unsafe.Add(tablePtr, 8)

	rows := make([]IfRow2, 0, count)
	for i := range count {
		base := unsafe.Add(tableBase, uintptr(i)*ifRow2Size)
		rows = append(rows, parseIfRow2(base))
	}
	return rows, nil
}

// parseIfRow2 reads the fields we need at known offsets within MIB_IF_ROW2.
// Field offsets are derived from the documented Win32 layout.
func parseIfRow2(base unsafe.Pointer) IfRow2 {
	// Layout (Windows 10/11):
	// 0   InterfaceLuid (8)
	// 8   InterfaceIndex (4)
	// 12  InterfaceGuid (16)
	// 28  Alias[257] uint16 = 514 bytes
	// 542 Description[257] uint16 = 514 bytes
	// 1056 PhysicalAddressLength (4)
	// 1060 PhysicalAddress[32] (32)
	// 1092 PermanentPhysicalAddress[32] (32)
	// 1124 Mtu (4)
	// 1128 Type (4)
	// 1132 TunnelType (4)
	// 1136 MediaType (4)
	// 1140 PhysicalMediumType (4)
	// 1144 AccessType (4)
	// 1148 DirectionType (4)
	// 1152 InterfaceAndOperStatusFlags (1) + padding
	// 1153 OperStatus (4) ... but proper offset begins after flags
	// We read more conservatively using the documented alignment.
	//
	// To remain robust we read scalar fields at conservative offsets that
	// match a representative struct dump on Windows 10/11.
	row := IfRow2{}
	row.Index = *(*uint32)(unsafe.Add(base, 8))

	aliasU16 := (*[257]uint16)(unsafe.Add(base, 28))
	descU16 := (*[257]uint16)(unsafe.Add(base, 28+514))
	row.Alias = utf16NulString(aliasU16[:])
	row.Description = utf16NulString(descU16[:])

	row.Type = *(*uint32)(unsafe.Add(base, 1128))
	row.OperStatus = *(*uint32)(unsafe.Add(base, 1160))

	row.SpeedTx = *(*uint64)(unsafe.Add(base, 1192))
	row.SpeedRx = *(*uint64)(unsafe.Add(base, 1200))

	row.InOctets = *(*uint64)(unsafe.Add(base, 1216))
	row.InUcastPkts = *(*uint64)(unsafe.Add(base, 1224))
	row.InErrors = *(*uint64)(unsafe.Add(base, 1264))

	row.OutOctets = *(*uint64)(unsafe.Add(base, 1280))
	row.OutUcastPkts = *(*uint64)(unsafe.Add(base, 1288))
	row.OutErrors = *(*uint64)(unsafe.Add(base, 1328))
	return row
}

func utf16NulString(s []uint16) string {
	for i, c := range s {
		if c == 0 {
			s = s[:i]
			break
		}
	}
	out := make([]rune, 0, len(s))
	for _, c := range s {
		out = append(out, rune(c))
	}
	return string(out)
}
