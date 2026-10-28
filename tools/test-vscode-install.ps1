<#
.SYNOPSIS
Verify a packaged VSIX in an isolated Windows VS Code profile.
.EXAMPLE
./tools/test-vscode-install.ps1 -CodeRoot 'C:\Program Files\Microsoft VS Code'
Requires the packaged VSIX and .cache/editor-release/fairyc-lsp.exe.
Use -PreviewReceipt and -PreviewRevision together to verify and extract a retained preview ZIP instead.
Use -VSIXPath to select a newly packaged candidate without replacing retained artifacts.
#>
param(
    [Parameter(Mandatory = $true)][string]$CodeRoot,
    [string]$PreviewReceipt,
    [string]$PreviewRevision,
    [string]$ArtifactsRoot,
    [string]$VSIXPath
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$repoRoot = Split-Path $PSScriptRoot -Parent
$releaseRoot = Join-Path $repoRoot '.cache/editor-release'
$manifestPath = Join-Path $repoRoot 'editors/vscode/package.json'
$licensePath = Join-Path $repoRoot 'LICENSE'
if ($ArtifactsRoot) {
    if ($VSIXPath) { throw 'VSIXPath cannot be combined with ArtifactsRoot' }
    if ($PreviewReceipt -or $PreviewRevision) { throw 'ArtifactsRoot cannot be combined with preview receipt arguments' }
    $releaseRoot = (Resolve-Path -LiteralPath $ArtifactsRoot).Path
    $manifestPath = Join-Path $releaseRoot 'extension.json'
    $licensePath = Join-Path $releaseRoot 'preview/LICENSE'
}
$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
$vsix = Join-Path $releaseRoot ($manifest.name + '-' + $manifest.version + '.vsix')
if ($VSIXPath) { $vsix = (Resolve-Path -LiteralPath $VSIXPath).Path }
$server = Join-Path $releaseRoot 'fairyc-lsp.exe'
if ($ArtifactsRoot) { $server = Join-Path $releaseRoot 'preview/fairyc-lsp.exe' }
$preview = $null
if ([bool]$PreviewReceipt -ne [bool]$PreviewRevision) { throw 'PreviewReceipt and PreviewRevision must be provided together' }
if ($PreviewReceipt) {
    $PreviewReceipt = (Resolve-Path -LiteralPath $PreviewReceipt).Path
    $recheck = @(& node (Join-Path $PSScriptRoot 'windows-preview/verify.mjs') --receipt $PreviewReceipt --revision $PreviewRevision 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "Preview verification failed: $($recheck -join "`n")" }
    $preview = ($recheck -join "`n") | ConvertFrom-Json
    if (-not $preview.passed -or -not $preview.PSObject.Properties['lspSHA256']) { throw 'Preview receipt must describe a verified compiler and LSP ZIP' }
}
$codeCLI = Join-Path $CodeRoot 'bin/code.cmd'
$codeEXE = Join-Path $CodeRoot 'Code.exe'
foreach ($required in @($vsix, "$vsix.sha256", $codeCLI, $codeEXE)) {
    if (-not (Test-Path -LiteralPath $required -PathType Leaf)) { throw "Required file missing: $required" }
}
$digest = (Get-FileHash -LiteralPath $vsix -Algorithm SHA256).Hash.ToLowerInvariant()
$expected = ((Get-Content -LiteralPath "$vsix.sha256" -Raw).Trim() -split '\s+')[0]
if ($digest -ne $expected) { throw 'VSIX checksum mismatch' }
$build = $null
$buildPath = "$vsix.build.json"
if (Test-Path -LiteralPath $buildPath -PathType Leaf) {
    $buildDigest = (Get-FileHash -LiteralPath $buildPath -Algorithm SHA256).Hash.ToLowerInvariant()
    $build = Get-Content -LiteralPath $buildPath -Raw | ConvertFrom-Json
    if ($build.schema -cne 'fairyc.vsix-build/v1' -or
        $build.artifact.sha256 -cne $digest -or $build.artifact.bytes -ne (Get-Item -LiteralPath $vsix).Length -or
        $build.artifact.name -cne (Split-Path $vsix -Leaf) -or
        $build.extension.id -cne ($manifest.publisher + '.' + $manifest.name) -or
        $build.extension.version -cne $manifest.version) { throw 'VSIX build receipt does not match package' }
}

Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [System.IO.Compression.ZipFile]::OpenRead($vsix)
try {
    $names = @($archive.Entries | ForEach-Object FullName)
    foreach ($required in @('extension/package.json', 'extension/extension.js', 'extension/readme.md', 'extension/LICENSE.txt',
        'extension/syntaxes/fairyc.tmLanguage.json', 'extension/snippets/fairyc.json',
        'extension/node_modules/vscode-languageclient/lib/node/main.js',
        'extension/node_modules/vscode-jsonrpc/lib/node/main.js')) {
        if ($names -cnotcontains $required) { throw "VSIX missing: $required" }
    }
    if ($names -match '^extension/node_modules/(?:@vscode/vsce|vscode-textmate|vscode-oniguruma)/') {
        throw 'Development dependencies leaked into VSIX'
    }
    $licenseStream = [System.IO.StreamReader]::new($archive.GetEntry('extension/LICENSE.txt').Open())
    try { $licenseText = $licenseStream.ReadToEnd() } finally { $licenseStream.Dispose() }
    if ($licenseText -cne [System.IO.File]::ReadAllText($licensePath)) { throw 'Packaged Apache license differs from supplied license' }
    $dependencyLicenses = @($names | Where-Object { $_ -match '^extension/node_modules/.+/(?:LICENSE|LICENCE)(?:\.[^/]+)?$' })
    $dependencyRoots = @($names | Where-Object { $_ -match '^extension/node_modules/(?:@[^/]+/)?[^/]+/package.json$' } | ForEach-Object { $_ -replace 'package.json$', '' })
    foreach ($dependency in $dependencyRoots) {
        if (-not ($dependencyLicenses | Where-Object { $_ -match ('^' + [regex]::Escape($dependency) + '(?:LICENSE|LICENCE)(?:\.[^/]+)?$') })) {
            throw "Dependency license missing: $dependency"
        }
    }
} finally { $archive.Dispose() }

$runRoot = Join-Path (Split-Path $vsix -Parent) ('install-' + [guid]::NewGuid().ToString('N'))
$userData = Join-Path $runRoot 'user-data'
$extensions = Join-Path $runRoot 'extensions'
$workspace = Join-Path $runRoot 'workspace'
$harness = Join-Path $runRoot 'harness'
foreach ($directory in @((Join-Path $userData 'User'), $extensions, $workspace, $harness)) {
    New-Item -ItemType Directory -Path $directory -Force | Out-Null
}
if ($preview) {
    $previewDirectory = Join-Path $runRoot 'preview'
    Expand-Archive -LiteralPath $preview.archive -DestinationPath $previewDirectory
    $server = Join-Path $previewDirectory 'fairyc-lsp.exe'
}
$server = (Resolve-Path -LiteralPath $server).Path
$serverDigest = (Get-FileHash -LiteralPath $server -Algorithm SHA256).Hash.ToLowerInvariant()
if ($preview -and $serverDigest -cne $preview.lspSHA256) { throw 'Extracted LSP hash differs from the verified preview' }
$serverBytes = (Get-Item -LiteralPath $server).Length
$utf8 = [System.Text.UTF8Encoding]::new($false)
$settings = @{ 'fairyc.server.path' = $server; 'telemetry.telemetryLevel' = 'off'; 'update.mode' = 'none'; 'extensions.autoUpdate' = $false }
[System.IO.File]::WriteAllText((Join-Path $userData 'User/settings.json'), ($settings | ConvertTo-Json), $utf8)
[System.IO.File]::WriteAllText((Join-Path $workspace 'main.fy'), "module main;`nfn i32 helper() { return 7; }`nfn i32 main() { return helper(); }`n", $utf8)
[System.IO.File]::WriteAllText((Join-Path $harness 'package.json'), '{"name":"fairyc-install-smoke","publisher":"fairyc-local-test","version":"0.0.0","engines":{"vscode":"^1.90.0"}}', $utf8)

function Invoke-CodeCLI([string[]]$Arguments) {
    # Windows PowerShell wraps native stderr as ErrorRecords even on exit 0.
    # Retain warnings in the receipt and judge the CLI by its exit code.
    $previousPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $output = @(& $codeCLI @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally { $ErrorActionPreference = $previousPreference }
    if ($exitCode -ne 0) { throw ($output -join "`n") }
    return $output
}
$installation = @(Invoke-CodeCLI -Arguments @('--user-data-dir', $userData, '--extensions-dir', $extensions, '--install-extension', $vsix, '--force'))
$installed = @(Invoke-CodeCLI -Arguments @('--user-data-dir', $userData, '--extensions-dir', $extensions, '--list-extensions', '--show-versions'))
if ($installed -notcontains ('0disoft.fairyc@' + $manifest.version)) { throw "Installed version not found: $installed" }
$arguments = @('--user-data-dir', $userData, '--extensions-dir', $extensions,
    '--extensionDevelopmentPath', $harness, '--extensionTestsPath', (Join-Path $PSScriptRoot 'test-vscode-installed.cjs'),
    '--disable-workspace-trust', '--skip-welcome', '--skip-release-notes', $workspace)
$quotedArguments = @($arguments | ForEach-Object { '"' + $_ + '"' })
$hostProcess = Start-Process -FilePath $codeEXE -ArgumentList $quotedArguments -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $runRoot 'host.stdout.log') -RedirectStandardError (Join-Path $runRoot 'host.stderr.log')
# Retain a handle before waiting; Windows PowerShell 5.1 can otherwise lose ExitCode.
$null = $hostProcess.Handle
if (-not $hostProcess.WaitForExit(60000)) {
    Stop-Process -Id $hostProcess.Id -Force
    throw "VS Code host timed out; see $runRoot"
}
$hostProcess.Refresh()
if ($hostProcess.ExitCode -ne 0) { throw "VS Code host exited $($hostProcess.ExitCode); see $runRoot" }
$hostReceiptPath = Join-Path $runRoot 'host-receipt.json'
if (-not (Test-Path -LiteralPath $hostReceiptPath)) { throw "Extension host produced no receipt; see $runRoot" }
$hostReceipt = Get-Content -LiteralPath $hostReceiptPath -Raw | ConvertFrom-Json
if (-not $hostReceipt.passed) { throw $hostReceipt.error }
if ($hostReceipt.serverPath -cne $server -or $hostReceipt.serverSHA256 -cne $serverDigest) { throw 'Extension host used a different configured LSP' }
if ((Get-FileHash -LiteralPath $server -Algorithm SHA256).Hash.ToLowerInvariant() -cne $serverDigest) { throw 'LSP changed during verification' }
if ((Get-FileHash -LiteralPath $vsix -Algorithm SHA256).Hash.ToLowerInvariant() -cne $digest) { throw 'VSIX changed during verification' }
if ($build -and (Get-FileHash -LiteralPath $buildPath -Algorithm SHA256).Hash.ToLowerInvariant() -cne $buildDigest) { throw 'VSIX build receipt changed during verification' }
if ($preview -and (Get-FileHash -LiteralPath $preview.archive -Algorithm SHA256).Hash.ToLowerInvariant() -cne $preview.archiveSHA256) { throw 'Preview ZIP changed during verification' }
$receipt = @{ schema = 'fairyc.vscode-install/v2'; vsix = $vsix; sha256 = $digest;
    build = $(if ($build) { @{ receipt = $buildPath; sha256 = $buildDigest; sourceRevision = $build.source.revision; modified = $build.source.modified } } else { $null });
    server = @{ path = $server; sha256 = $serverDigest; bytes = $serverBytes };
    preview = $(if ($preview) { @{ receipt = $PreviewReceipt; revision = $preview.sourceRevision; archive = $preview.archive; sha256 = $preview.archiveSHA256 } } else { $null });
    cleanMachineEvidence = $false; publicationApproved = $false;
    archiveFiles = $names.Count; dependencyLicenseFiles = $dependencyLicenses.Count;
    installationOutput = @($installation | ForEach-Object { "$_" }); installed = @($installed | ForEach-Object { "$_" }); host = $hostReceipt }
$receiptPath = Join-Path $runRoot 'receipt.json'
[System.IO.File]::WriteAllText($receiptPath, ($receipt | ConvertTo-Json -Depth 6), $utf8)
Write-Output "Installed VSIX and extension host verification passed: $receiptPath"
