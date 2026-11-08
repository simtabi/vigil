# Deprecated location, kept so the published one-liner keeps working:
#
#   irm https://raw.githubusercontent.com/simtabi/vigil/main/scripts/uninstall.ps1 | iex
#
# The script moved to src/scripts/uninstall.ps1 when the Go module moved into
# src/, and installers from 0.1.0 to 0.7.0 printed this URL. This file only
# forwards there, with its parameters, run locally or through iex. It can go in
# the first release after binary releases resume and the README has named the
# new URL for at least one release.

param([switch]$Purge)

$ErrorActionPreference = "Stop"
$local = if ($PSScriptRoot) { Join-Path $PSScriptRoot "..\src\scripts\uninstall.ps1" } else { $null }
if ($local -and (Test-Path $local)) {
    & $local @PSBoundParameters
    return
}
$script = Invoke-RestMethod "https://raw.githubusercontent.com/simtabi/vigil/main/src/scripts/uninstall.ps1"
& ([scriptblock]::Create($script)) @PSBoundParameters
