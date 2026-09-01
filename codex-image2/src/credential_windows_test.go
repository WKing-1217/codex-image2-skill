//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
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
