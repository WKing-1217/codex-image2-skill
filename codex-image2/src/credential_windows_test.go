//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestWindowsCredentialRoundTrip(t *testing.T) {
	target := fmt.Sprintf("codex-image2-test/%d", time.Now().UnixNano())
	const key = "test-key-not-a-real-secret"
	t.Cleanup(func() {
		_ = deleteWindowsCredential(target)
	})
	if err := writeWindowsCredential(target, key); err != nil {
		t.Fatal(err)
	}
	got, err := readWindowsCredential(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != key {
		t.Fatalf("readWindowsCredential() = %q, want test key", got)
	}
	if err := deleteWindowsCredential(target); err != nil {
		t.Fatal(err)
	}
	if _, err := readWindowsCredential(target); err != errCredentialNotFound {
		t.Fatalf("read after delete error = %v, want errCredentialNotFound", err)
	}
}

func TestSetupDialogPowerShellParses(t *testing.T) {
	command := exec.Command("powershell.exe", "-NoProfile", "-Command", `$tokens=$null; $parseErrors=$null; [System.Management.Automation.Language.Parser]::ParseInput($env:CODEX_IMAGE2_TEST_SCRIPT, [ref]$tokens, [ref]$parseErrors) | Out-Null; if ($parseErrors.Count -gt 0) { $parseErrors | ForEach-Object { Write-Error $_.Message }; exit 1 }`)
	command.Env = append(os.Environ(), "CODEX_IMAGE2_TEST_SCRIPT="+setupDialogPowerShell)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("PowerShell setup dialog script did not parse: %v\n%s", err, output)
	}
}

func TestSetupDialogDoesNotPrefillAPIURL(t *testing.T) {
	for _, forbidden := range []string{"$urlBox.Text =", "CODEX_IMAGE2_SETUP_"} {
		if strings.Contains(strings.ToLower(setupDialogPowerShell), strings.ToLower(forbidden)) {
			t.Fatalf("setup dialog contains forbidden API URL prefill marker %q", forbidden)
		}
	}
}

func TestUserVisibleDesktopDetection(t *testing.T) {
	for _, test := range []struct {
		station string
		desktop string
		want    bool
	}{
		{"WinSta0", "Default", true},
		{"winsta0", "default", true},
		{"CodexSandbox", "Private-1", false},
		{"WinSta0", "Private-1", false},
		{"Service-0x0-3e7$", "Default", false},
	} {
		if got := isUserVisibleDesktop(test.station, test.desktop); got != test.want {
			t.Fatalf("isUserVisibleDesktop(%q, %q) = %v, want %v", test.station, test.desktop, got, test.want)
		}
	}
}

func TestCurrentTestProcessUsesVisibleDesktop(t *testing.T) {
	stationHandle, _, _ := getProcessWindowStation.Call()
	threadID, _, _ := getCurrentThreadID.Call()
	desktopHandle, _, _ := getThreadDesktop.Call(threadID)
	station, stationErr := windowsUserObjectName(stationHandle)
	desktop, desktopErr := windowsUserObjectName(desktopHandle)
	if stationErr != nil || desktopErr != nil {
		t.Skipf("desktop identity unavailable: station=%v desktop=%v", stationErr, desktopErr)
	}
	if !isUserVisibleDesktop(station, desktop) {
		t.Skipf("tests are running on a non-interactive desktop %q/%q", station, desktop)
	}
	if err := ensureVisibleSetupDesktop(); err != nil {
		t.Fatalf("visible desktop was rejected: %v", err)
	}
}
