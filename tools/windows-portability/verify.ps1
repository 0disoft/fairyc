param([Parameter(Mandatory=$true)][string]$Receipt)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
function Get-FixtureHash([string]$Path) {
    $stream = [IO.File]::OpenRead($Path)
    $digest = [Security.Cryptography.SHA256]::Create()
    try { return [BitConverter]::ToString($digest.ComputeHash($stream)).Replace('-','').ToLowerInvariant() }
    finally { $digest.Dispose(); $stream.Dispose() }
}
if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT -or ![Environment]::Is64BitProcess -or $env:PROCESSOR_ARCHITECTURE -ne 'AMD64') {
    throw 'This fixture requires a 64-bit Windows PowerShell process'
}
$receiptPath = [IO.Path]::GetFullPath($Receipt)
if (Test-Path -LiteralPath $receiptPath) { throw 'Receipt already exists' }
$bundle = (Resolve-Path -LiteralPath $PSScriptRoot).Path
$manifestPath = Join-Path $bundle 'manifest.json'
$manifest = Get-Content -LiteralPath $manifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ($manifest.schema -ne 'fairyc.windows-portability-bundle/v1' -or $manifest.platform -ne 'windows/amd64') {
    throw 'Unexpected portability manifest'
}
if (($manifest.inputBytes -join ',') -ne '0,1,127,128,255,10,42' -or $manifest.nativeStdout -cne "Fairy C portability`n") {
    throw 'Unexpected native fixture expectations'
}
$expectedFiles = @('fairyc.exe','native-development.exe','native-release.exe','verify.ps1','README.txt',
    'LICENSE','LICENSING.md','THIRD_PARTY_NOTICES.md','GO-LICENSE.txt')
$actualFiles = @($manifest.files | ForEach-Object { $_.name })
if ($actualFiles.Count -ne $expectedFiles.Count -or @(Compare-Object ($actualFiles | Sort-Object) ($expectedFiles | Sort-Object)).Count -ne 0) {
    throw 'Manifest file set differs from the portability fixture'
}
$entries = @(Get-ChildItem -LiteralPath $bundle | ForEach-Object { $_.Name })
if (@(Compare-Object ($entries | Sort-Object) (@($expectedFiles) + 'manifest.json' | Sort-Object)).Count -ne 0) {
    throw 'Bundle contains missing or unlisted files'
}
foreach ($file in $manifest.files) {
    $path = Join-Path $bundle $file.name
    $info = Get-Item -LiteralPath $path
    if (($info.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0 -or $info.Length -ne $file.bytes -or
        (Get-FixtureHash $path) -ne $file.sha256) {
        throw "Bundle file differs: $($file.name)"
    }
}
$temporaryParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$runName = 'fairyc-portability-' + [Guid]::NewGuid().ToString('N')
$runRoot = Join-Path $temporaryParent $runName
New-Item -ItemType Directory -Path $runRoot | Out-Null
$runRoot = (Resolve-Path -LiteralPath $runRoot).Path
$utf8 = New-Object Text.UTF8Encoding($false, $true)
$executions = New-Object 'Collections.Generic.List[object]'
$result = [ordered]@{
    schema='fairyc.windows-portability-execution/v1'; passed=$false
    manifestSHA256=(Get-FixtureHash $manifestPath)
    compilerRevision=$manifest.compilerRevision; windowsVersion=[Environment]::OSVersion.Version.ToString()
    powershellVersion=$PSVersionTable.PSVersion.ToString(); process64Bit=[Environment]::Is64BitProcess
    processorArchitecture=$env:PROCESSOR_ARCHITECTURE
    visualStudioDirectoryPresent=((Test-Path -LiteralPath (Join-Path $env:ProgramFiles 'Microsoft Visual Studio')) -or
        (Test-Path -LiteralPath (Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio')))
    windowsSDKDirectoryPresent=(Test-Path -LiteralPath (Join-Path ${env:ProgramFiles(x86)} 'Windows Kits'))
    childPath=''; nativeCompilationAttempted=$false; cleanMachineEvidence=$false
    scratchRemoved=$false; executions=@(); error=$null
}
function Invoke-Fixture {
    param([string]$Name, [string]$Image, [string[]]$Arguments, [string]$Directory, [AllowNull()][object]$Expected=$null)
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = Join-Path $bundle $Image
    $start.WorkingDirectory = $Directory
    # All fixture arguments are fixed whitespace-free tokens, including on .NET Framework.
    foreach ($argument in $Arguments) {
        if ($argument -notmatch '^[A-Za-z0-9_.-]+$') { throw 'Unexpected fixture argument' }
    }
    $start.Arguments = [string]::Join(' ', $Arguments)
    $start.UseShellExecute = $false; $start.CreateNoWindow = $true
    $start.RedirectStandardOutput = $true; $start.RedirectStandardError = $true; $start.RedirectStandardInput = $true
    $start.StandardOutputEncoding = $utf8; $start.StandardErrorEncoding = $utf8
    foreach ($key in @($start.EnvironmentVariables.Keys)) {
        if ($key -match '^(PATH|INCLUDE|LIB|LIBPATH|CC|CXX|CL|_CL_|GO.*|VS.*|VC.*|WindowsSdk.*|FAIRYC.*)$') {
            $start.EnvironmentVariables.Remove($key)
        }
    }
    $start.EnvironmentVariables['PATH'] = ''
    $start.EnvironmentVariables['TEMP'] = $runRoot; $start.EnvironmentVariables['TMP'] = $runRoot
    $child = New-Object Diagnostics.Process
    $child.StartInfo = $start
    $started = $false
    try {
        $started = $child.Start()
        if (!$started) { throw 'Fixture process did not start' }
        $child.StandardInput.Close()
        $stdoutTask = $child.StandardOutput.ReadToEndAsync(); $stderrTask = $child.StandardError.ReadToEndAsync()
        if (!$child.WaitForExit(45000)) { throw "Fixture timed out: $Name" }
        if (![Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($stdoutTask,$stderrTask), 10000)) { throw 'Output drain timed out' }
        $stdout = $stdoutTask.GetAwaiter().GetResult(); $stderr = $stderrTask.GetAwaiter().GetResult()
        $executions.Add([ordered]@{name=$Name; image=$Image; arguments=@($Arguments); exit=$child.ExitCode; stdout=$stdout; stderr=$stderr})
        if ($child.ExitCode -ne 0 -or $stderr -ne '' -or ($null -ne $Expected -and $stdout -cne $Expected)) {
            throw "Fixture failed: $Name (exit $($child.ExitCode)): $stderr"
        }
        return $stdout
    } finally {
        try {
            if ($started -and !$child.HasExited) {
                $child.Kill()
                if (!$child.WaitForExit(10000)) { throw 'Fixture process did not exit after termination' }
            }
        } finally { $child.Dispose() }
    }
}
try {
    $null = Invoke-Fixture -Name version -Image fairyc.exe -Arguments @('--version') -Directory $runRoot -Expected $manifest.compilerVersion
    $created = Invoke-Fixture -Name new -Image fairyc.exe -Arguments @('new','hello','--uid','example.portability.starter') -Directory $runRoot
    if ($created -notmatch 'created Fairy C package') { throw 'Starter was not created' }
    $project = Join-Path $runRoot 'hello'
    $checked = Invoke-Fixture -Name check -Image fairyc.exe -Arguments @('check') -Directory $project
    if ($checked -notmatch 'lock updated') { throw 'Initial check did not create the lock' }
    $checked = Invoke-Fixture -Name check-locked -Image fairyc.exe -Arguments @('check','--locked') -Directory $project
    if ($checked -notmatch 'lock verified') { throw 'Locked check did not verify the lock' }
    $tested = Invoke-Fixture -Name semantic-test -Image fairyc.exe -Arguments @('test','--locked') -Directory $project
    if ($tested -notmatch '1 passed; 0 failed') { throw 'Starter semantic test failed' }
    foreach ($profile in @('development','release')) {
        $directory = Join-Path $runRoot $profile
        New-Item -ItemType Directory -Path $directory | Out-Null
        $inputPath = Join-Path $directory 'input.bin'
        [IO.File]::WriteAllBytes($inputPath, [byte[]]@(0,1,127,128,255,10,42))
        $null = Invoke-Fixture -Name "native-$profile" -Image "native-$profile.exe" -Arguments @() -Directory $directory -Expected "Fairy C portability`n"
        $outputPath = Join-Path $directory 'output.bin'
        if ((Get-FixtureHash $inputPath) -ne (Get-FixtureHash $outputPath)) {
            throw "Binary file round trip failed: $profile"
        }
    }
    $result.passed = $true
} catch {
    $result.error = $_.Exception.Message
} finally {
    # Remove only the exact fresh directory created by this invocation.
    $resolved = (Resolve-Path -LiteralPath $runRoot).Path
    $expected = [IO.Path]::GetFullPath((Join-Path $temporaryParent $runName))
    if (![string]::Equals($resolved, $expected, [StringComparison]::OrdinalIgnoreCase)) { throw 'Scratch directory escaped its owned path' }
    Remove-Item -LiteralPath $resolved -Recurse -Force
    $result.scratchRemoved = !(Test-Path -LiteralPath $resolved)
    $result.executions = $executions.ToArray()
    $wire = $utf8.GetBytes(($result | ConvertTo-Json -Depth 8) + "`n")
    $stream = [IO.File]::Open($receiptPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write)
    try { $stream.Write($wire, 0, $wire.Length) } finally { $stream.Dispose() }
}
if (!$result.passed) { throw $result.error }
Write-Output "Passed $($executions.Count) portability executions; clean-machine evidence remains separate."
