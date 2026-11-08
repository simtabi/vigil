# Remove vigil on Windows: stops + removes the logon task / service and deletes the
# binary (delegates to `vigil self uninstall`).
#
#   irm https://raw.githubusercontent.com/simtabi/vigil/main/scripts/uninstall.ps1 | iex
#   ./scripts/uninstall.ps1 -Purge    # also delete config and runtime data

param([switch]$Purge)

$ErrorActionPreference = "Stop"

$cmd = Get-Command vigil -ErrorAction SilentlyContinue
$vigil = if ($cmd) { $cmd.Source } else { $null }
if (-not $vigil) {
    $cand = Join-Path "$env:LOCALAPPDATA\Programs\vigil" "vigil.exe"
    if (Test-Path $cand) { $vigil = $cand }
}
if (-not $vigil) { throw "vigil not found on PATH or in the usual install dir." }

Write-Host "Removing vigil (service + binary)..."
if ($Purge) { & $vigil self uninstall --yes --purge }
else { & $vigil self uninstall --yes }
Write-Host "Done."
