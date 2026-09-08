[CmdletBinding()]
param(
    [switch]$ValidateOnly
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
    throw "This installer supports Windows only."
}

$sourceSkill = Join-Path $PSScriptRoot "codex-image2"
$checksumFile = Join-Path $PSScriptRoot "SHA256SUMS.txt"
if (-not (Test-Path -LiteralPath (Join-Path $sourceSkill "SKILL.md") -PathType Leaf)) {
    throw "The repository package is incomplete: codex-image2/SKILL.md is missing."
}
if (-not (Test-Path -LiteralPath $checksumFile -PathType Leaf)) {
    throw "The repository package is incomplete: SHA256SUMS.txt is missing."
}

$checksums = Get-Content -LiteralPath $checksumFile
if ($checksums.Count -ne 4) {
    throw "SHA256SUMS.txt must contain exactly four binary checksums."
}
$expectedBinaries = @(
    "codex-image2/bin/codex-image2-darwin-amd64",
    "codex-image2/bin/codex-image2-darwin-arm64",
    "codex-image2/bin/codex-image2-windows-amd64.exe",
    "codex-image2/bin/codex-image2-windows-arm64.exe"
)
$verifiedBinaries = @{}
foreach ($line in $checksums) {
    if ($line -notmatch '^([0-9a-f]{64})  (codex-image2/bin/[A-Za-z0-9.-]+)$') {
        throw "SHA256SUMS.txt contains an invalid entry."
    }
    $expected = $Matches[1]
    $manifestPath = $Matches[2]
    if ($manifestPath -notin $expectedBinaries -or $verifiedBinaries.ContainsKey($manifestPath)) {
        throw "SHA256SUMS.txt contains an unexpected or duplicate binary entry."
    }
    $verifiedBinaries[$manifestPath] = $true
    $relativePath = $manifestPath -replace '/', [IO.Path]::DirectorySeparatorChar
    $binaryPath = Join-Path $PSScriptRoot $relativePath
    if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
        throw "The repository package is incomplete: $relativePath is missing."
    }
    $actual = (Get-FileHash -LiteralPath $binaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "Binary checksum validation failed for $relativePath."
    }
}

if ($ValidateOnly) {
    Write-Output "Package validation passed."
    exit 0
}

$userProfile = [Environment]::GetFolderPath([Environment+SpecialFolder]::UserProfile)
$codexHome = if ([string]::IsNullOrWhiteSpace($env:CODEX_HOME)) {
    Join-Path $userProfile ".codex"
} else {
    [IO.Path]::GetFullPath($env:CODEX_HOME)
}
$skillsRoot = Join-Path $codexHome "skills"
$targetSkill = Join-Path $skillsRoot "codex-image2"
$stagingSkill = Join-Path $skillsRoot (".codex-image2.install-" + [Guid]::NewGuid().ToString("N"))
$backupSkill = $null

New-Item -ItemType Directory -Path $skillsRoot -Force | Out-Null
Copy-Item -LiteralPath $sourceSkill -Destination $stagingSkill -Recurse

try {
    if (Test-Path -LiteralPath $targetSkill) {
        $backupRoot = Join-Path $codexHome "skill-backups"
        New-Item -ItemType Directory -Path $backupRoot -Force | Out-Null
        $backupSkill = Join-Path $backupRoot ("codex-image2-" + (Get-Date -Format "yyyyMMdd-HHmmssfff") + "-" + [Guid]::NewGuid().ToString("N").Substring(0, 8))
        Move-Item -LiteralPath $targetSkill -Destination $backupSkill
    }
    Move-Item -LiteralPath $stagingSkill -Destination $targetSkill
}
catch {
    if (-not (Test-Path -LiteralPath $targetSkill) -and $backupSkill -and (Test-Path -LiteralPath $backupSkill)) {
        Move-Item -LiteralPath $backupSkill -Destination $targetSkill
    }
    throw
}
finally {
    if (Test-Path -LiteralPath $stagingSkill) {
        Remove-Item -LiteralPath $stagingSkill -Recurse -Force
    }
}

$architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
$executableName = switch ($architecture) {
    "x64" { "codex-image2-windows-amd64.exe" }
    "arm64" { "codex-image2-windows-arm64.exe" }
    default { throw "Unsupported Windows architecture: $architecture" }
}
$executable = Join-Path $targetSkill (Join-Path "bin" $executableName)
$testImageRoot = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "CodexImage2\setup-tests"
New-Item -ItemType Directory -Path $testImageRoot -Force | Out-Null
$testImage = Join-Path $testImageRoot ("setup-test-" + (Get-Date -Format "yyyyMMdd-HHmmssfff") + "-" + [Guid]::NewGuid().ToString("N").Substring(0, 8) + ".png")

Write-Output "Skill installed or updated. Opening the local secure setup window..."
& $executable setup --out $testImage
if ($LASTEXITCODE -ne 0) {
    throw "Initialization did not complete. Follow the error above, then run this installer again."
}
if (-not (Test-Path -LiteralPath $testImage -PathType Leaf)) {
    throw "Initialization returned success but the test image is missing."
}
Write-Output "Installation and initialization completed. Test image: $testImage"
