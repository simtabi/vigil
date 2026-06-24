# Build mta from source and install it on Windows.
#
#   ./scripts/install.ps1                 # install to %LOCALAPPDATA%\Programs\mta
#   ./scripts/install.ps1 -Prefix C:\Tools

param(
    [string]$Prefix = "$env:LOCALAPPDATA\Programs\mta"
)

$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go is required (https://go.dev/dl). Install Go >= 1.23 and retry."
}

New-Item -ItemType Directory -Force -Path $Prefix | Out-Null

Write-Host "Building mta..."
go build -trimpath -o (Join-Path $Prefix "mta.exe") .

Write-Host "Installed: $(Join-Path $Prefix 'mta.exe')"

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$Prefix*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$Prefix", "User")
    Write-Host "Added $Prefix to your user PATH (restart your terminal to pick it up)."
}

Write-Host ""
Write-Host "Next steps:"
Write-Host "  mta config init      # write the default config"
Write-Host "  mta doctor           # check capabilities"
Write-Host "  mta install          # install + start the logon task / service"
