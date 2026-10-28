<#
.SYNOPSIS
Verify the transferred Fairy C ZIP and VSIX using Windows PowerShell and VS Code.
#>
param(
    [Parameter(Mandatory = $true)][string]$CodeRoot,
    [string]$ResultsDirectory
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
Import-Module (Join-Path $PSHOME 'Modules/Microsoft.PowerShell.Utility/Microsoft.PowerShell.Utility.psd1') -ErrorAction Stop
if (-not $ResultsDirectory) {
    $ResultsDirectory = Join-Path (Split-Path $PSScriptRoot -Parent) ('fairyc-editor-results-' + [guid]::NewGuid().ToString('N'))
}
if (-not [System.IO.Path]::IsPathRooted($ResultsDirectory)) { throw 'ResultsDirectory must be an absolute path' }
$results = [System.IO.Path]::GetFullPath($ResultsDirectory)
New-Item -ItemType Directory -Path $results -ErrorAction Stop | Out-Null
$utf8 = [System.Text.UTF8Encoding]::new($false)
$receipt = @{ schema = 'fairyc.windows-editor-verification/v1'; passed = $false;
    cleanMachineEvidence = $false; publicationApproved = $false; results = $results;
    computer = $env:COMPUTERNAME; powershell = $PSVersionTable.PSVersion.ToString();
    processArchitecture = $env:PROCESSOR_ARCHITECTURE; toolPaths = @{} }
function Hash([string]$Path) { (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant() }
try {
    if ($env:OS -ne 'Windows_NT' -or $env:PROCESSOR_ARCHITECTURE -ne 'AMD64') { throw 'Requires Windows AMD64 PowerShell' }
    $osInfo = Get-CimInstance Win32_OperatingSystem
    $receipt.os = @{ caption = $osInfo.Caption; version = $osInfo.Version; build = $osInfo.BuildNumber; architecture = $osInfo.OSArchitecture }
    foreach ($name in @('go.exe', 'node.exe', 'bun.exe')) {
        $found = Get-Command $name -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
        $receipt.toolPaths[$name] = $(if ($found) { $found.Source } else { $null })
    }
    $manifestPath = Join-Path $PSScriptRoot 'manifest.json'
    $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
    if ($manifest.schema -cne 'fairyc.windows-editor-bundle/v1') { throw 'Unsupported editor bundle manifest' }
    $receipt.manifestSHA256 = Hash $manifestPath
    $seen = @{}
    foreach ($entry in $manifest.files) {
        if ($entry.name -cnotmatch '^[A-Za-z0-9_.-]+$' -or $seen.ContainsKey($entry.name)) { throw 'Invalid or duplicate bundle filename' }
        $file = Get-Item -LiteralPath (Join-Path $PSScriptRoot $entry.name)
        if ($file.PSIsContainer -or ($file.Attributes -band [IO.FileAttributes]::ReparsePoint) -or $file.Length -ne $entry.bytes -or (Hash $file.FullName) -cne $entry.sha256) { throw "Bundle integrity mismatch: $($entry.name)" }
        $seen[$entry.name] = $true
    }
    $actual = @(Get-ChildItem -LiteralPath $PSScriptRoot -Force | Where-Object { $_.Name -ne 'manifest.json' })
    if ($actual.Count -ne $seen.Count -or @($actual | Where-Object { -not $seen.ContainsKey($_.Name) }).Count) { throw 'Unexpected bundle contents' }
    foreach ($required in @('verify.ps1', 'test-vscode-install.ps1', 'test-vscode-installed.cjs', 'extension.json', 'LICENSE', 'LICENSING.md', 'THIRD_PARTY_NOTICES.md', 'GO-LICENSE.txt', 'README.txt', $manifest.preview.name, ($manifest.preview.name + '.sha256'), $manifest.vsix.name, ($manifest.vsix.name + '.sha256'))) {
        if (-not $seen.ContainsKey($required)) { throw "Missing bundle file: $required" }
    }
    $archive = Join-Path $PSScriptRoot $manifest.preview.name
    $vsix = Join-Path $PSScriptRoot $manifest.vsix.name
    if ((Hash $archive) -cne $manifest.preview.sha256 -or (Hash $vsix) -cne $manifest.vsix.sha256) { throw 'Artifact identity differs from bundle manifest' }
    foreach ($artifact in @($archive, $vsix)) {
        $expected = (Hash $artifact) + '  ' + [IO.Path]::GetFileName($artifact)
        if ([IO.File]::ReadAllText($artifact + '.sha256').Trim() -cne $expected) { throw 'Artifact checksum sidecar differs' }
    }
    $receipt.sourceRevision = $manifest.sourceRevision
    $receipt.preview = $manifest.preview
    $receipt.vsix = $manifest.vsix
    $receipt.verifiedBundleFiles = $seen.Count
    $artifacts = Join-Path $results 'artifacts'
    New-Item -ItemType Directory -Path $artifacts | Out-Null
    Expand-Archive -LiteralPath $archive -DestinationPath (Join-Path $artifacts 'preview')
    $server = Join-Path $artifacts 'preview/fairyc-lsp.exe'
    if ((Hash $server) -cne $manifest.preview.lspSHA256) { throw 'Extracted LSP hash differs from bundle manifest' }
    foreach ($name in @('extension.json', $manifest.vsix.name, ($manifest.vsix.name + '.sha256'))) {
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination (Join-Path $artifacts $name)
    }
    & (Join-Path $PSScriptRoot 'test-vscode-install.ps1') -CodeRoot $CodeRoot -ArtifactsRoot $artifacts
    $install = @(Get-ChildItem -LiteralPath $artifacts -Directory -Filter 'install-*')
    if ($install.Count -ne 1) { throw 'Expected one isolated installation' }
    $installReceipt = Get-Content -LiteralPath (Join-Path $install[0].FullName 'receipt.json') -Raw | ConvertFrom-Json
    if (-not $installReceipt.host.passed -or $installReceipt.sha256 -cne $manifest.vsix.sha256 -or $installReceipt.server.sha256 -cne $manifest.preview.lspSHA256) { throw 'Installed artifact identities differ' }
    foreach ($entry in $manifest.files) {
        if ((Hash (Join-Path $PSScriptRoot $entry.name)) -cne $entry.sha256) { throw "Bundle changed during verification: $($entry.name)" }
    }
    $receipt.installation = $installReceipt
    $receipt.passed = $true
} catch {
    $receipt.error = $_.Exception.Message
    throw
} finally {
    [IO.File]::WriteAllText((Join-Path $results 'receipt.json'), ($receipt | ConvertTo-Json -Depth 12), $utf8)
    Write-Output "Editor verification receipt: $(Join-Path $results 'receipt.json')"
}
