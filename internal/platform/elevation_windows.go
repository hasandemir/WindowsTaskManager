//go:build windows

package platform

import (
	"os"
	"strings"

	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
	"golang.org/x/sys/windows"
)

// IsAdmin reports whether the current process is running with administrator privileges.
func IsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

// RequestElevation re-launches the current binary with the "runas" verb.
func RequestElevation() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := strings.Join(os.Args[1:], " ")
	if err := winapi.ShellExecute("runas", exe, args, "", winapi.SW_SHOWNORMAL); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
