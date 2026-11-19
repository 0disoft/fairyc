#Requires -Version 5.1
[CmdletBinding()]
param([string]$ZipPath, [string]$OutputDirectory = (Join-Path $HOME 'Downloads'))

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$tag = 'v0.1.0-preview.2'
$revision = '485592aa611626f427d248de659fa622e917d45f'
$archiveName = 'fairyc-windows-amd64-preview-candidate.zip'
$archiveHash = '03d1c345b21519bdb586357b0a1b8179334b0ded3954f28d794e1e6b4284b9ef'
$baseUrl = "https://github.com/0disoft/fairyc/releases/download/$tag"
$files = @('fairyc.exe','fairyc-lsp.exe','README.txt','LICENSE','LICENSING.md','THIRD_PARTY_NOTICES.md','GO-LICENSE.txt')
$stepNames = @('compiler-version','lsp-version','lsp-protocol','new','check','check-locked','semantic-test')
foreach ($profile in 'development','release') {
  $stepNames += @("native-test-$profile","build-$profile","image-$profile","run-$profile")
}
$output = (Resolve-Path -LiteralPath $OutputDirectory).Path
$root = Join-Path $output ('fairyc-preview2-' + [Guid]::NewGuid().ToString('N').Substring(0,12))
$null = New-Item -ItemType Directory -Path $root
$logs = Join-Path $root 'logs'
$null = New-Item -ItemType Directory -Path $logs
$utf8 = [Text.UTF8Encoding]::new($false, $true)
$steps = [Collections.Generic.List[object]]::new()
$images = [Collections.Generic.List[object]]::new()
$result = [ordered]@{
  schema = 'fairyc.windows-public-preview-verification/v1'
  tag = $tag; sourceRevision = $revision; host = [Environment]::MachineName
  windowsVersion = [Environment]::OSVersion.Version.ToString()
  powershellVersion = $PSVersionTable.PSVersion.ToString()
  architecture = $env:PROCESSOR_ARCHITECTURE
  startedUtc = [DateTime]::UtcNow.ToString('o'); passed = $false
  acquisition = $(if ($ZipPath) { 'local-copy' } else { 'public-download' })
  expectedZipSHA256 = $archiveHash; actualZipSHA256 = $null
  checksums = [ordered]@{}; failedStage = $null; error = $null
}

function Get-Sha256([string]$Path) {
  (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Assert-Amd64Image([string]$Path) {
  $stream = [IO.File]::OpenRead($Path)
  $reader = [IO.BinaryReader]::new($stream)
  try {
    if ($reader.ReadUInt16() -ne 0x5a4d) { throw "Missing DOS signature: $Path" }
    $stream.Position = 0x3c
    $stream.Position = $reader.ReadUInt32()
    if ($reader.ReadUInt32() -ne 0x4550 -or $reader.ReadUInt16() -ne 0x8664) {
      throw "Expected a Windows AMD64 PE image: $Path"
    }
  } finally { $reader.Dispose() }
}

function Invoke-Probe([string]$Name, [string]$File, [string]$Directory,
    [string[]]$Arguments = @(), [string]$Expected = '\A\z', [string]$Stdin = '') {
  # All arguments are fixed whitespace-free tokens. Never go through a shell.
  $start = [Diagnostics.ProcessStartInfo]::new($File, ($Arguments -join ' '))
  $start.WorkingDirectory = $Directory
  $start.UseShellExecute = $false
  $start.CreateNoWindow = $true
  $start.RedirectStandardInput = $true
  $start.RedirectStandardOutput = $true
  $start.RedirectStandardError = $true
  $start.StandardOutputEncoding = $utf8
  $start.StandardErrorEncoding = $utf8
  foreach ($key in @($start.EnvironmentVariables.Keys)) {
    if ($key -notmatch '^(SystemRoot|SystemDrive|windir|ComSpec|ProgramFiles(\(x86\))?|ProgramW6432|ProgramData|ALLUSERSPROFILE|USERPROFILE|HOMEDRIVE|HOMEPATH|LOCALAPPDATA|APPDATA|PATHEXT|OS|NUMBER_OF_PROCESSORS|PROCESSOR_ARCHITECTURE)$') {
      $start.EnvironmentVariables.Remove($key)
    }
  }
  $start.EnvironmentVariables['PATH'] = "$(Join-Path $env:SystemRoot 'System32');$env:SystemRoot"
  $start.EnvironmentVariables['TEMP'] = $root
  $start.EnvironmentVariables['TMP'] = $root
  $step = [ordered]@{
    name = $Name; executable = $File; arguments = $Arguments; cwd = $Directory
    status = 'failed'; exitCode = $null; error = $null
    stdout = "logs/$Name.stdout.log"; stderr = "logs/$Name.stderr.log"
  }
  $steps.Add($step)
  $process = [Diagnostics.Process]::new()
  $process.StartInfo = $start
  $started = $false
  try {
    $originalInputEncoding = $null
    try {
      if ($Stdin) {
        if ($start.PSObject.Properties['StandardInputEncoding']) {
          $start.StandardInputEncoding = $utf8
        } elseif ([Console]::InputEncoding.GetPreamble().Length -ne 0) {
          # .NET Framework initializes the stdin writer from the console encoding.
          # Its initial flush can write a BOM before the raw LSP frame.
          $originalInputEncoding = [Console]::InputEncoding
          [Console]::InputEncoding = $utf8
        }
      }
      $started = $process.Start()
    } finally {
      if ($null -ne $originalInputEncoding) { [Console]::InputEncoding = $originalInputEncoding }
    }
    if (!$started) { throw 'Process did not start' }
    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()
    # The LSP probe sends only ASCII JSON and framing bytes.
    $bytes = [Text.Encoding]::ASCII.GetBytes($Stdin)
    $process.StandardInput.BaseStream.Write($bytes, 0, $bytes.Length)
    $process.StandardInput.Close()
    if (!$process.WaitForExit(90000)) { throw 'Process timed out after 90 seconds' }
    $step.exitCode = $process.ExitCode
    if (![Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($stdoutTask,$stderrTask), 5000)) {
      throw 'Process output did not close after exit'
    }
    $stdout = $stdoutTask.GetAwaiter().GetResult()
    $stderr = $stderrTask.GetAwaiter().GetResult()
    [IO.File]::WriteAllText((Join-Path $root $step.stdout), $stdout, $utf8)
    [IO.File]::WriteAllText((Join-Path $root $step.stderr), $stderr, $utf8)
    if ($step.exitCode -ne 0 -or $stderr -cne '' -or $stdout -cnotmatch $Expected) {
      throw "Unexpected process result (exit $($step.exitCode)); see $($step.stdout) and $($step.stderr)"
    }
    $step.status = 'passed'
    return $stdout
  } catch {
    $step.error = $_.Exception.Message
    throw "${Name}: $($step.error)"
  } finally {
    try {
      if ($started -and !$process.HasExited) {
        $process.Kill()
        $null = $process.WaitForExit(5000)
      }
    } finally { $process.Dispose() }
  }
}

function Test-Lsp([string]$Server, [string]$Directory) {
  $requests = @(
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"processId":null,"rootUri":null,"capabilities":{}}}',
    '{"jsonrpc":"2.0","method":"initialized","params":{}}',
    '{"jsonrpc":"2.0","id":2,"method":"shutdown","params":null}',
    '{"jsonrpc":"2.0","method":"exit"}'
  )
  $wire = ($requests | ForEach-Object { "Content-Length: $($_.Length)`r`n`r`n$_" }) -join ''
  $response = Invoke-Probe 'lsp-protocol' $Server $Directory -Expected '\AContent-Length:' -Stdin $wire
  $messages = @()
  $bytes = $utf8.GetBytes($response)
  $offset = 0
  while ($offset -lt $bytes.Length) {
    $header = [Text.Encoding]::ASCII.GetString($bytes, $offset, [Math]::Min(64, $bytes.Length - $offset))
    if ($header -cnotmatch '\AContent-Length: ([0-9]{1,8})\r\n\r\n') { throw 'Invalid LSP response header' }
    $length = [int]$Matches[1]
    $offset += $Matches[0].Length
    if ($length -gt $bytes.Length - $offset) { throw 'Truncated LSP response body' }
    $messages += $utf8.GetString($bytes, $offset, $length) | ConvertFrom-Json
    $offset += $length
  }
  if ($messages.Count -ne 2 -or $messages[0].id -ne 1 -or $messages[1].id -ne 2 -or
      @($messages | Where-Object { $_.jsonrpc -cne '2.0' -or $_.PSObject.Properties['error'] }).Count -ne 0 -or
      $messages[0].result.serverInfo.name -cne 'fairyc-lsp' -or
      $messages[0].result.capabilities.positionEncoding -cne 'utf-16' -or
      !$messages[0].result.capabilities.definitionProvider -or
      !$messages[0].result.capabilities.PSObject.Properties['completionProvider'] -or
      $null -ne $messages[1].result) { throw 'Unexpected LSP initialize/shutdown response' }
}

$stage = 'host'
try {
  if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT -or
      ![Environment]::Is64BitProcess -or $env:PROCESSOR_ARCHITECTURE -cne 'AMD64') {
    throw 'Run this verifier in 64-bit PowerShell on Windows AMD64'
  }
  $stage = 'acquisition'
  $zip = Join-Path $root $archiveName
  if ($ZipPath) {
    Copy-Item -LiteralPath $ZipPath -Destination $zip
    Copy-Item -LiteralPath "$ZipPath.sha256" -Destination "$zip.sha256"
  } else {
    foreach ($name in $archiveName,"$archiveName.sha256") {
      $null = Invoke-WebRequest -Uri "$baseUrl/$name" -UseBasicParsing -TimeoutSec 90 -OutFile (Join-Path $root $name)
    }
  }
  $stage = 'archive-integrity'
  $result.actualZipSHA256 = Get-Sha256 $zip
  $sidecar = [IO.File]::ReadAllText("$zip.sha256").TrimEnd("`r", "`n")
  if ($result.actualZipSHA256 -cne $archiveHash -or $sidecar -cne "$archiveHash  $archiveName" -or
      (Get-Item -LiteralPath $zip).Length -ne 10104881) {
    throw "ZIP or sidecar differs. Expected $archiveHash; actual $($result.actualZipSHA256)"
  }

  $stage = 'extracted-integrity'
  $package = Join-Path $root 'package'
  Expand-Archive -LiteralPath $zip -DestinationPath $package
  $entries = @(Get-ChildItem -LiteralPath $package -Force)
  if (@(Compare-Object ($entries.Name | Sort-Object) (($files + 'SHA256SUMS.txt') | Sort-Object) -CaseSensitive).Count -ne 0 -or
      @($entries | Where-Object { $_.PSIsContainer -or ($_.Attributes -band [IO.FileAttributes]::ReparsePoint) }).Count -ne 0) {
    throw 'Unexpected ZIP contents'
  }
  foreach ($line in Get-Content -LiteralPath (Join-Path $package 'SHA256SUMS.txt')) {
    if ($line -cnotmatch '\A([0-9a-f]{64})  ([A-Za-z0-9_.-]+)\z' -or
        $files -cnotcontains $Matches[2] -or $result.checksums.Contains($Matches[2])) { throw 'Invalid checksum entry' }
    $expected = $Matches[1]
    $name = $Matches[2]
    $actual = Get-Sha256 (Join-Path $package $name)
    if ($actual -cne $expected) { throw "Hash mismatch for ${name}: expected $expected; actual $actual" }
    $result.checksums[$name] = $actual
  }
  if ($result.checksums.Count -ne $files.Count) { throw 'Missing internal checksum' }
  $compiler = Join-Path $package 'fairyc.exe'
  $server = Join-Path $package 'fairyc-lsp.exe'
  Assert-Amd64Image $compiler
  Assert-Amd64Image $server

  $stage = 'versions-and-lsp'
  $scratch = Join-Path $root 'scratch'
  $null = New-Item -ItemType Directory -Path $scratch
  $null = Invoke-Probe 'compiler-version' $compiler $scratch @('--version') "(?m)^revision: $revision\r?\nmodified: false\r?$"
  $null = Invoke-Probe 'lsp-version' $server $scratch @('--version') '\Afairyc-lsp 0\.1\.0\r?\n\z'
  try { Test-Lsp $server $scratch }
  catch {
    $step = $steps[$steps.Count - 1]
    $step.status = 'failed'
    $step.error = $_.Exception.Message
    throw
  }

  $stage = 'starter'
  $null = Invoke-Probe 'new' $compiler $scratch @('new','hello-fairy','--uid','example.hello-fairy') 'created Fairy C package'
  $project = Join-Path $scratch 'hello-fairy'
  $null = Invoke-Probe 'check' $compiler $project @('check') 'lock updated\r?\n\z'
  $starter = [ordered]@{}
  foreach ($name in 'fairy.toml','fairy.lock','src/main.fy','tests/main.fy','README.md','.gitignore') {
    $starter[$name] = Get-Sha256 (Join-Path $project $name)
  }
  $null = Invoke-Probe 'check-locked' $compiler $project @('check','--locked') 'lock verified\r?\n\z'
  $summary = 'test result: ok\. 1 passed; 0 failed; lock verified\r?\n\z'
  $null = Invoke-Probe 'semantic-test' $compiler $project @('test','--locked') $summary

  foreach ($profile in 'development','release') {
    $stage = "native-$profile"
    $flags = @('--locked')
    if ($profile -eq 'release') { $flags = @('--release') + $flags }
    $null = Invoke-Probe "native-test-$profile" $compiler $project (@('test','--native') + $flags) $summary
    $null = Invoke-Probe "build-$profile" $compiler $project (@('build') + $flags) "\Abuilt: build/hello-fairy\.exe \(x86_64-windows-msvc, $profile, lock verified\)\r?\n\z"
    $image = Join-Path $project 'build/hello-fairy.exe'
    Assert-Amd64Image $image
    $imageCopy = Join-Path $root "hello-fairy-$profile.exe"
    Copy-Item -LiteralPath $image -Destination $imageCopy
    $imageHash = Get-Sha256 $imageCopy
    $null = Invoke-Probe "image-$profile" $imageCopy $project
    $images.Add([ordered]@{ profile = $profile; file = "hello-fairy-$profile.exe"; sha256 = $imageHash })
    $null = Invoke-Probe "run-$profile" $compiler $project (@('run') + $flags)
  }
  $stage = 'post-run-integrity'
  foreach ($name in $starter.Keys) {
    if ((Get-Sha256 (Join-Path $project $name)) -cne $starter[$name]) { throw "Starter changed: $name" }
  }
  foreach ($name in $files) {
    if ((Get-Sha256 (Join-Path $package $name)) -cne $result.checksums[$name]) { throw "Package file changed: $name" }
  }
  $result.passed = $true
} catch {
  $result.failedStage = $stage
  $result.error = $_.Exception.Message
} finally {
  $result.steps = $steps.ToArray()
  $result.skipped = @($stepNames | Where-Object { $_ -cnotin @($steps | ForEach-Object { $_.name }) })
  $result.images = $images.ToArray()
  $result.finishedUtc = [DateTime]::UtcNow.ToString('o')
  $receipt = Join-Path $root 'results.json'
  [IO.File]::WriteAllText($receipt, ($result | ConvertTo-Json -Depth 8), $utf8)
  Write-Host "Results: $receipt"
}
if (!$result.passed) {
  Write-Host "FAIL [$($result.failedStage)] $($result.error)"
  exit 1
}
Write-Host "PASS: $tag ZIP SHA-256 $($result.actualZipSHA256)"
Write-Host "PASS: $($steps.Count)/$($stepNames.Count) process checks"
exit 0
