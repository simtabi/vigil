# Install mta on Windows by downloading the latest prebuilt release and
# verifying its SHA-256. Falls back to `go install` if Go is present.
#
#   irm https://raw.githubusercontent.com/simtabi/ms-teams-activity/main/scripts/install.ps1 | iex
#   ./scripts/install.ps1 -Prefix C:\Tools

param([string]$Prefix = "$env:LOCALAPPDATA\Programs\mta")

$ErrorActionPreference = "Stop"
$repo = "simtabi/ms-teams-activity"
$base = "https://github.com/$repo/releases/latest/download"
$asset = "mta_windows_amd64.zip"

New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
$tmp = New-Item -ItemType Directory -Force -Path (Join-Path $env:TEMP ("mta-" + [guid]::NewGuid()))

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
    # The zip contains a flat-named binary (mta_windows_<arch>.exe); install it as mta.exe.
    $inner = $asset -replace '\.zip$', '.exe'
    Copy-Item "$tmp\$inner" (Join-Path $Prefix "mta.exe") -Force
    Write-Host "Installed: $(Join-Path $Prefix 'mta.exe')"
}
catch {
    Write-Warning "Download failed: $_"
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "Building from source..."
        $env:GOBIN = $Prefix
        go install "github.com/$repo/cmd/mta@latest"
    }
    else { throw "Go not found and download failed." }
}
finally { Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue }

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$Prefix*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$Prefix", "User")
    Write-Host "Added $Prefix to your user PATH (restart your terminal)."
}

Write-Host ""
Write-Host "Next steps:"
Write-Host "  mta config wizard    # guided setup (or: mta config init)"
Write-Host "  mta doctor           # check capabilities"
Write-Host "  mta install          # install + start the logon task / service"
