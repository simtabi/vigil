# Install vigil on Windows by downloading the latest prebuilt release and
# verifying its SHA-256. Falls back to `go install` if Go is present.
#
#   irm https://raw.githubusercontent.com/simtabi/vigil/main/scripts/install.ps1 | iex
#   ./scripts/install.ps1 -Prefix C:\Tools
#   ./scripts/install.ps1 -WithService   # also configure + install + start the daemon

param(
    [string]$Prefix = "$env:LOCALAPPDATA\Programs\vigil",
    [switch]$WithService
)

$ErrorActionPreference = "Stop"
$repo = "simtabi/vigil"
$base = "https://github.com/$repo/releases/latest/download"
$asset = "vigil_windows_amd64.zip"

New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
$tmp = New-Item -ItemType Directory -Force -Path (Join-Path $env:TEMP ("vigil-" + [guid]::NewGuid()))

try {
    Write-Host "Downloading $asset..."
    Invoke-WebRequest "$base/$asset" -OutFile "$tmp\$asset"
    Invoke-WebRequest "$base/checksums.txt" -OutFile "$tmp\checksums.txt"

    $want = (Select-String -Path "$tmp\checksums.txt" -Pattern ([regex]::Escape($asset)) |
        Select-Object -First 1).Line.Split(" ")[0]
    $got = (Get-FileHash "$tmp\$asset" -Algorithm SHA256).Hash.ToLower()
    if ($want -ne $got) { throw "checksum mismatch for $asset (want $want got $got)" }

    Write-Host "Verifying checksum... OK"
    Expand-Archive -Path "$tmp\$asset" -DestinationPath $tmp -Force
    # The zip contains a flat-named binary (vigil_windows_<arch>.exe); install it as vigil.exe.
    $inner = $asset -replace '\.zip$', '.exe'
    Copy-Item "$tmp\$inner" (Join-Path $Prefix "vigil.exe") -Force
    Write-Host "Installed: $(Join-Path $Prefix 'vigil.exe')"
}
catch {
    Write-Warning "Download failed: $_"
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "Building from source..."
        $env:GOBIN = $Prefix
        go install "github.com/$repo/cmd/vigil@latest"
    }
    else { throw "Go not found and download failed." }
}
finally { Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue }

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$Prefix*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$Prefix", "User")
    Write-Host "Added $Prefix to your user PATH (restart your terminal)."
}

$vigil = Join-Path $Prefix "vigil.exe"
if ($WithService) {
    Write-Host "Setting up the background service..."
    & $vigil install --init
    Write-Host "Done. Manage it with: vigil status / vigil restart / vigil stop"
}
else {
    Write-Host ""
    Write-Host "Next steps:"
    Write-Host "  vigil config wizard    # guided setup (or: vigil config init)"
    Write-Host "  vigil doctor           # check capabilities"
    Write-Host "  vigil install          # install + start the logon task / service"
    Write-Host "                       # (or re-run this installer with -WithService)"
    Write-Host ""
    Write-Host "Uninstall later:  irm https://raw.githubusercontent.com/$repo/main/scripts/uninstall.ps1 | iex"
}
