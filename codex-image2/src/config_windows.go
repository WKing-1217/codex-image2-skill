//go:build windows

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
	errorNotFound           = syscall.Errno(1168)
)

type windowsCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

var (
	advapi32DLL             = syscall.NewLazyDLL("Advapi32.dll")
	user32DLL               = syscall.NewLazyDLL("User32.dll")
	kernel32DLL             = syscall.NewLazyDLL("Kernel32.dll")
	credWriteW              = advapi32DLL.NewProc("CredWriteW")
	credReadW               = advapi32DLL.NewProc("CredReadW")
	credDeleteW             = advapi32DLL.NewProc("CredDeleteW")
	credFree                = advapi32DLL.NewProc("CredFree")
	getProcessWindowStation = user32DLL.NewProc("GetProcessWindowStation")
	getThreadDesktop        = user32DLL.NewProc("GetThreadDesktop")
	getUserObjectInfoW      = user32DLL.NewProc("GetUserObjectInformationW")
	getCurrentThreadID      = kernel32DLL.NewProc("GetCurrentThreadId")
)

const userObjectName = 2

func writeStoredAPIKey(key string) error {
	return writeWindowsCredential(credentialTarget, key)
}

func writeWindowsCredential(targetName, key string) error {
	target, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return errors.New("could not prepare the Windows credential target")
	}
	username, _ := syscall.UTF16PtrFromString("codex-image2")
	blob := []byte(key)
	if len(blob) == 0 || len(blob) > 2560 {
		return errors.New("API key has an unsupported length")
	}
	credential := windowsCredential{
		Type:               credTypeGeneric,
		TargetName:         target,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     &blob[0],
		Persist:            credPersistLocalMachine,
		UserName:           username,
	}
	result, _, callErr := credWriteW.Call(uintptr(unsafe.Pointer(&credential)), 0)
	if result == 0 {
		return fmt.Errorf("could not save the API key in Windows Credential Manager: %v", callErr)
	}
	return nil
}

func readStoredAPIKey() (string, error) {
	return readWindowsCredential(credentialTarget)
}

func readWindowsCredential(targetName string) (string, error) {
	target, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return "", errors.New("could not prepare the Windows credential target")
	}
	var credential *windowsCredential
	result, _, callErr := credReadW.Call(
		uintptr(unsafe.Pointer(target)),
		credTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&credential)),
	)
	if result == 0 {
		if errors.Is(callErr, errorNotFound) {
			return "", errCredentialNotFound
		}
		return "", fmt.Errorf("could not read Windows Credential Manager: %v", callErr)
	}
	defer credFree.Call(uintptr(unsafe.Pointer(credential)))
	if credential == nil || credential.CredentialBlob == nil || credential.CredentialBlobSize == 0 {
		return "", errCredentialNotFound
	}
	blob := unsafe.Slice(credential.CredentialBlob, int(credential.CredentialBlobSize))
	copyOfBlob := append([]byte(nil), blob...)
	return strings.TrimSpace(string(copyOfBlob)), nil
}

func deleteStoredAPIKey() error {
	return deleteWindowsCredential(credentialTarget)
}

func deleteWindowsCredential(targetName string) error {
	target, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return errors.New("could not prepare the Windows credential target")
	}
	result, _, callErr := credDeleteW.Call(uintptr(unsafe.Pointer(target)), credTypeGeneric, 0)
	if result == 0 {
		if errors.Is(callErr, errorNotFound) {
			return errCredentialNotFound
		}
		return fmt.Errorf("could not remove the API key from Windows Credential Manager: %v", callErr)
	}
	return nil
}

func runSetupDialog() (setupInput, error) {
	if err := ensureVisibleSetupDesktop(); err != nil {
		return setupInput{}, err
	}
	command := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-STA",
		"-ExecutionPolicy", "Bypass",
		"-WindowStyle", "Hidden",
		"-Command", setupDialogPowerShell,
	)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 10 {
			return setupInput{}, errors.New("setup was cancelled")
		}
		if errors.Is(err, exec.ErrNotFound) {
			return setupInput{}, errors.New("could not open the setup window because Windows PowerShell is unavailable or blocked")
		}
		if errors.As(err, &exitErr) {
			return setupInput{}, fmt.Errorf("could not open the setup window (PowerShell exit code %d); Windows PowerShell, System.Windows.Forms, or security policy may be blocking it", exitErr.ExitCode())
		}
		return setupInput{}, errors.New("could not open the setup window; Windows PowerShell or security policy may be blocking it")
	}
	encoded := strings.TrimSpace(string(output))
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return setupInput{}, errors.New("the setup window returned invalid data")
	}
	var input setupInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return setupInput{}, errors.New("the setup window returned invalid data")
	}
	return input, nil
}

func ensureVisibleSetupDesktop() error {
	stationHandle, _, _ := getProcessWindowStation.Call()
	threadID, _, _ := getCurrentThreadID.Call()
	desktopHandle, _, _ := getThreadDesktop.Call(threadID)
	station, stationErr := windowsUserObjectName(stationHandle)
	desktop, desktopErr := windowsUserObjectName(desktopHandle)
	// Older Windows environments may not expose names. Let PowerShell try rather
	// than rejecting a setup that could still work.
	if stationErr != nil || desktopErr != nil {
		return nil
	}
	if !isUserVisibleDesktop(station, desktop) {
		return errors.New("the setup window cannot be shown from the Windows sandbox/private desktop; select Full access for this current task, then run setup again")
	}
	return nil
}

func windowsUserObjectName(handle uintptr) (string, error) {
	if handle == 0 {
		return "", errors.New("Windows desktop handle is unavailable")
	}
	var required uint32
	_, _, _ = getUserObjectInfoW.Call(handle, userObjectName, 0, 0, uintptr(unsafe.Pointer(&required)))
	if required < 2 {
		return "", errors.New("Windows desktop name is unavailable")
	}
	buffer := make([]uint16, (required+1)/2)
	result, _, callErr := getUserObjectInfoW.Call(
		handle,
		userObjectName,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(required),
		uintptr(unsafe.Pointer(&required)),
	)
	if result == 0 {
		return "", fmt.Errorf("could not inspect the Windows desktop: %v", callErr)
	}
	return syscall.UTF16ToString(buffer), nil
}

func isUserVisibleDesktop(windowStation, desktop string) bool {
	return strings.EqualFold(strings.TrimSpace(windowStation), "WinSta0") &&
		strings.EqualFold(strings.TrimSpace(desktop), "Default")
}

const setupDialogPowerShell = `
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$form = New-Object System.Windows.Forms.Form
$form.Text = 'Codex Image2 安全配置'
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedDialog'
$form.MaximizeBox = $false
$form.MinimizeBox = $false
$form.TopMost = $true
$form.ClientSize = New-Object System.Drawing.Size(560, 270)
$form.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 9)

$title = New-Object System.Windows.Forms.Label
$title.Text = '配置图片生成 API'
$title.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 14, [System.Drawing.FontStyle]::Bold)
$title.AutoSize = $true
$title.Location = New-Object System.Drawing.Point(24, 18)
$form.Controls.Add($title)

$note = New-Object System.Windows.Forms.Label
$note.Text = '密钥只会保存在 Windows 凭据管理器中，不会发送到 Codex 聊天或写入 Skill。'
$note.AutoSize = $true
$note.ForeColor = [System.Drawing.Color]::DimGray
$note.Location = New-Object System.Drawing.Point(27, 53)
$form.Controls.Add($note)

$urlLabel = New-Object System.Windows.Forms.Label
$urlLabel.Text = 'API 地址（请向服务商获取）'
$urlLabel.AutoSize = $true
$urlLabel.Location = New-Object System.Drawing.Point(27, 86)
$form.Controls.Add($urlLabel)

$urlBox = New-Object System.Windows.Forms.TextBox
$urlBox.Location = New-Object System.Drawing.Point(30, 107)
$urlBox.Size = New-Object System.Drawing.Size(500, 25)
$form.Controls.Add($urlBox)

$keyLabel = New-Object System.Windows.Forms.Label
$keyLabel.Text = 'API 密钥'
$keyLabel.AutoSize = $true
$keyLabel.Location = New-Object System.Drawing.Point(27, 145)
$form.Controls.Add($keyLabel)

$keyBox = New-Object System.Windows.Forms.TextBox
$keyBox.Location = New-Object System.Drawing.Point(30, 166)
$keyBox.Size = New-Object System.Drawing.Size(500, 25)
$keyBox.UseSystemPasswordChar = $true
$form.Controls.Add($keyBox)

$ok = New-Object System.Windows.Forms.Button
$ok.Text = '保存并测试'
$ok.Location = New-Object System.Drawing.Point(344, 215)
$ok.Size = New-Object System.Drawing.Size(95, 32)
$form.Controls.Add($ok)

$cancel = New-Object System.Windows.Forms.Button
$cancel.Text = '取消'
$cancel.Location = New-Object System.Drawing.Point(451, 215)
$cancel.Size = New-Object System.Drawing.Size(79, 32)
$cancel.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
$form.Controls.Add($cancel)

$form.AcceptButton = $ok
$form.CancelButton = $cancel

$ok.Add_Click({
    if ([string]::IsNullOrWhiteSpace($urlBox.Text) -or [string]::IsNullOrWhiteSpace($keyBox.Text)) {
        [System.Windows.Forms.MessageBox]::Show('请填写 API 地址和 API 密钥。', '信息不完整', 'OK', 'Warning') | Out-Null
        return
    }
    $form.DialogResult = [System.Windows.Forms.DialogResult]::OK
    $form.Close()
})

$urlBox.Select()
$result = $form.ShowDialog()
if ($result -ne [System.Windows.Forms.DialogResult]::OK) { exit 10 }

$payload = @{ api_url = $urlBox.Text.Trim(); api_key = $keyBox.Text.Trim() } | ConvertTo-Json -Compress
$bytes = [System.Text.Encoding]::UTF8.GetBytes($payload)
[Console]::Out.Write([Convert]::ToBase64String($bytes))
`
