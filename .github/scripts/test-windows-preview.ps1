#Requires -Version 5.1
[CmdletBinding()]
param([Parameter(Mandatory = $true)][string]$OutputDirectory)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$release = 'v0.1.0-preview.4'
$verifier = Join-Path $PSScriptRoot '../../tools/windows-preview/verify-preview2.ps1'
$engine = Join-Path $PSHOME 'powershell.exe'
$root = Join-Path $OutputDirectory ('fairyc-preview-ci-' + [Guid]::NewGuid().ToString('N'))
$public = (New-Item -ItemType Directory -Path (Join-Path $root 'public') -Force).FullName
$negative = (New-Item -ItemType Directory -Path (Join-Path $root 'negative')).FullName

function Get-VerificationDirectory([string]$Parent) {
  $runs = @(Get-ChildItem -LiteralPath $Parent -Directory)
  if ($runs.Count -ne 1) { throw "Expected exactly one verifier run under $Parent" }
  return $runs[0].FullName
}

& $engine -NoProfile -NonInteractive -File $verifier -Release $release -OutputDirectory $public
if ($LASTEXITCODE -ne 0) { throw "Public ZIP verification failed (exit $LASTEXITCODE)" }
$verified = Get-VerificationDirectory $public
$receipt = Get-Content -LiteralPath (Join-Path $verified 'results.json') -Raw | ConvertFrom-Json
if ($receipt.tag -cne $release -or $receipt.passed -ne $true -or $receipt.acquisition -cne 'public-download' -or
    @($receipt.steps).Count -ne 15 -or @($receipt.steps | Where-Object status -cne 'passed').Count -ne 0) {
  throw 'Expected 15 successful process checks for the public ZIP'
}

$archives = @(Get-ChildItem -LiteralPath $verified -File -Filter '*.zip')
if ($archives.Count -ne 1) { throw 'Expected exactly one downloaded ZIP' }
$badZip = Join-Path $negative $archives[0].Name
Copy-Item -LiteralPath $archives[0].FullName -Destination $badZip
Copy-Item -LiteralPath "$($archives[0].FullName).sha256" -Destination "$badZip.sha256"
$stream = [IO.File]::OpenWrite($badZip)
try {
  $stream.Position = 0
  $stream.WriteByte(0)
} finally {
  $stream.Dispose()
}

& $engine -NoProfile -NonInteractive -File $verifier -Release $release -ZipPath $badZip -OutputDirectory $negative
if ($LASTEXITCODE -ne 1) { throw "Expected a rejected ZIP (exit 1), got exit $LASTEXITCODE" }
$rejected = Get-VerificationDirectory $negative
$receipt = Get-Content -LiteralPath (Join-Path $rejected 'results.json') -Raw | ConvertFrom-Json
if ($receipt.tag -cne $release -or $receipt.passed -ne $false -or $receipt.failedStage -cne 'archive-integrity' -or
    $receipt.actualZipSHA256 -ceq $receipt.expectedZipSHA256 -or @($receipt.steps).Count -ne 0 -or
    (Test-Path -LiteralPath (Join-Path $rejected 'package'))) {
  throw 'Corrupt ZIP must fail integrity verification before extraction or execution'
}
Write-Host 'PASS: corrupt ZIP rejected before extraction or execution'
exit 0
