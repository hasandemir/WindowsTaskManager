//go:build windows

package platform

import "testing"

func TestAcquireSingleInstance(t *testing.T) {
	release, err := AcquireSingleInstance(`Local\WTM-Test-SingleInstance`)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer release()

	secondRelease, err := AcquireSingleInstance(`Local\WTM-Test-SingleInstance`)
	if err != ErrAlreadyRunning {
		if secondRelease != nil {
			secondRelease()
		}
		t.Fatalf("second acquire err=%v want %v", err, ErrAlreadyRunning)
	}
}
