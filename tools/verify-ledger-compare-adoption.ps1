#requires -Version 7.0
param(
    [string]$Original = 'C:\Users\cherr\Downloads\fairy_c_test_projects\round4\01-ledger-reconciler',
    [string]$Fairyc = (Join-Path $PSScriptRoot '../.cache/ledger-compare-toolchain/fairyc.exe'),
    [string]$AuditReceipt = ''
)
$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
. (Join-Path $PSScriptRoot 'adoption-output-audit.ps1')
if ($AuditReceipt) { Invoke-AdoptionOrdinalAudit $AuditReceipt 'ledger' $Fairyc $PSCommandPath; return }
$Original = (Resolve-Path -LiteralPath $Original).Path
$Fairyc = (Resolve-Path -LiteralPath $Fairyc).Path
$utf8 = [Text.UTF8Encoding]::new($false, $true)
$run = Join-Path $repo ('.cache/ledger-compare-adoption/' + [Guid]::NewGuid().ToString('N'))
[void][IO.Directory]::CreateDirectory($run)
function Hash([string]$Path) { (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant() }
function Execute([string]$Exe, [string[]]$Argv, [string]$Cwd, [string]$Kind) {
    $start = [Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $Exe; $start.WorkingDirectory = $Cwd
    $start.UseShellExecute = $false; $start.CreateNoWindow = $true
    $start.RedirectStandardOutput = $true; $start.RedirectStandardError = $true
    $start.StandardOutputEncoding = $utf8; $start.StandardErrorEncoding = $utf8
    foreach ($argument in $Argv) { $start.ArgumentList.Add($argument) }
    $process = [Diagnostics.Process]::new(); $process.StartInfo = $start
    try {
        [void]$process.Start()
        $stdout = $process.StandardOutput.ReadToEndAsync(); $stderr = $process.StandardError.ReadToEndAsync()
        if (!$process.WaitForExit(60000)) { $process.Kill($true); $process.WaitForExit(); throw 'Owned child exceeded 60 seconds' }
        $result = [pscustomobject]@{exitCode=$process.ExitCode; stdout=$stdout.GetAwaiter().GetResult(); stderr=$stderr.GetAwaiter().GetResult()}
        $raw = @{kind=$Kind; executable=$Exe; argv=@($Argv); cwd=$Cwd; actual=$result}
        [IO.File]::AppendAllText((Join-Path $run 'raw-executions.jsonl'), ($raw | ConvertTo-Json -Depth 6 -Compress) + "`n", $utf8)
        return $result
    } finally { $process.Dispose() }
}
$pinned = Join-Path $Original '../_toolchain/fairyc-c34718bd/fairyc.exe'
$pinnedHash = Hash $pinned
if ($pinnedHash -ne '7d58dd9c8860f01723f55f2f289a609d0941fa3cd6b06699954ca46336eeb917') { throw 'Pinned compiler identity changed' }
$compilerHash = Hash $Fairyc
if ($compilerHash -eq $pinnedHash) { throw 'Use a separately built stage0.78 compiler' }
$version = Execute $Fairyc @('--version') $repo 'identity'
if ($version.exitCode -ne 0 -or $version.stdout -notmatch 'sdk: fairyc-sdk-stdlib/0 0\.0\.0-stage0\.78') { throw 'Expected SDK stage0.78' }
$ledgerPath = Join-Path $Original 'src/ledger.fy'
if ((Hash $ledgerPath) -ne 'e9b5afccb0b0da265f04b13e6070490f0d8103c02ff3f340b5f3707895f95488') { throw 'Reviewed ledger source changed' }
$legacy = @'
    let usize a = starts[left]; let usize b = starts[right];
    var usize i = 0;
    while (i < lengths[left] && i < lengths[right]) {
        if (data[a + i] != data[b + i]) { return data[a + i] < data[b + i]; }
        i += 1;
    }
    return lengths[left] < lengths[right];
'@
$replacement = @'
    let usize a = starts[left]; let usize b = starts[right];
    return bytes::compare(data[a..a + lengths[left]], data[b..b + lengths[right]]) < 0;
'@
$ledger = [IO.File]::ReadAllText($ledgerPath, $utf8).Replace("`r`n", "`n")
if ($ledger.Split(@($legacy), [StringSplitOptions]::None).Count -ne 2) { throw 'Comparator must match exactly once' }
$historical = Get-Content -LiteralPath (Join-Path $Original 'verification-results.json') -Raw | ConvertFrom-Json
$selected = @('empty', '한글 공백', 'bom-crlf-final-no-newline', 'i64-max-income', 'i64-max-expense', 'income-overflow', 'expense-overflow', 'amount-overflow', 'late-blank-row', 'invalid-utf8', 'name-256', 'accounts-128', 'accounts-129', 'help', 'missing-file', 'no-argument')
$cases = @($historical.cases | Where-Object { $_.profile -eq 'development' -and $_.name -in $selected })
if ($cases.Count -ne $selected.Count -or @($cases.name | Select-Object -Unique).Count -ne $selected.Count) { throw 'Historical case inventory mismatch' }
foreach ($case in $cases) {
    if ($case.inputPath -and $case.inputPath -cne ('samples/' + $case.name + '.tsv')) { throw 'Historical fixture path must stay in samples' }
}
$protected = @{}
$paths = @(Get-ChildItem -LiteralPath (Join-Path $Original 'src') -File | ForEach-Object FullName)
$paths += @(Join-Path $Original 'fairy.toml'; Join-Path $Original 'REPORT.ko.md'; Join-Path $Original 'verification-results.json'; Join-Path $Original 'verify.ps1'; $pinned)
foreach ($case in $cases) { if ($case.inputPath) { $paths += Join-Path $Original $case.inputPath } }
foreach ($path in $paths | Select-Object -Unique) { $protected[$path] = Hash $path }
# Independent fixture oracle: BigInteger aggregation and UTF-8 hex ordinal keys.
$names = @('한', 'é', ('e' + [char]0x301), 'a', 'AA', 'A', ' A', 'A ', '2', '10')
if ($names.Count -ne 10 -or $names[2].Length -ne 2) { throw 'Unicode fixture inventory mismatch' }
$fixture = ''; $expected = ''; $ordinal = [Collections.Generic.SortedDictionary[string,object]]::new([StringComparer]::Ordinal)
foreach ($name in $names) {
    $fixture += "$name`tincome`t2147483648`n$name`texpense`t2147483649`n$name`tincome`t2147483648`n"
    $ordinal.Add([Convert]::ToHexString($utf8.GetBytes($name)), $name)
}
$income = [Numerics.BigInteger]::Parse('2147483648') * 2
$expense = [Numerics.BigInteger]::Parse('2147483649')
foreach ($name in $ordinal.Values) { $expected += "$name`tincome=$income`texpense=$expense`tbalance=$($income - $expense)`n" }
$cases += [pscustomobject]@{name='byte-order-and-grouping'; arguments=@('samples/adoption.tsv'); inputPath='samples/adoption.tsv'; expected=[pscustomobject]@{exitCode=0; stdout=$expected; stderr=''}}
$results = [Collections.Generic.List[object]]::new(); $images = [Collections.Generic.List[object]]::new()
$digests = [Collections.Generic.List[object]]::new()
foreach ($variant in 'legacy', 'sdk') {
    $workspace = Join-Path $run $variant
    [void][IO.Directory]::CreateDirectory((Join-Path $workspace 'src'))
    [void][IO.Directory]::CreateDirectory((Join-Path $workspace 'samples'))
    [IO.File]::Copy((Join-Path $Original 'fairy.toml'), (Join-Path $workspace 'fairy.toml'))
    foreach ($source in Get-ChildItem -LiteralPath (Join-Path $Original 'src') -File -Filter '*.fy') {
        [IO.File]::Copy($source.FullName, (Join-Path $workspace ('src/' + $source.Name)))
    }
    if ($variant -eq 'sdk') { [IO.File]::WriteAllText((Join-Path $workspace 'src/ledger.fy'), $ledger.Replace($legacy, $replacement), $utf8) }
    foreach ($case in $cases) {
        if ($case.inputPath -and $case.name -ne 'byte-order-and-grouping') {
            if ((Hash (Join-Path $Original $case.inputPath)) -ne $case.inputSha256Before -or $case.inputSha256Before -ne $case.inputSha256After) { throw 'Historical input hash mismatch' }
            [IO.File]::Copy((Join-Path $Original $case.inputPath), (Join-Path $workspace $case.inputPath))
        }
    }
    [IO.File]::WriteAllText((Join-Path $workspace 'samples/adoption.tsv'), $fixture, $utf8)
    $check = Execute $Fairyc @('check') $workspace 'check'
    if ($check.exitCode -ne 0) { throw "Check $variant failed: $($check.stderr)" }
    $lock = Get-Content -LiteralPath (Join-Path $workspace 'fairy.lock') -Raw | ConvertFrom-Json
    $sdk = $lock.packages | Where-Object sourceKind -eq sdk
    $digests.Add(@{variant=$variant; sourceDigest=($lock.packages | Where-Object role -eq root).sourceDigest; sdkVersion=$sdk.version; sdkSourceDigest=$sdk.sourceDigest})
    foreach ($profile in 'development', 'release') {
        $buildArgs = if ($profile -eq 'release') { @('build', '--release', '--locked') } else { @('build', '--locked') }
        $build = Execute $Fairyc $buildArgs $workspace 'build'
        if ($build.exitCode -ne 0) { throw "Build $variant/$profile failed: $($build.stderr)" }
        $exe = Join-Path $workspace "build/ledger-reconciler-$profile.exe"
        [IO.File]::Copy((Join-Path $workspace 'build/ledger-reconciler.exe'), $exe)
        $images.Add(@{variant=$variant; profile=$profile; executable=$exe; sha256=(Hash $exe)})
        foreach ($case in $cases) {
            $inputPath = if ($case.inputPath) { Join-Path $workspace $case.inputPath } else { $null }
            $before = if ($inputPath) { Hash $inputPath } else { $null }
            $actual = Execute $exe @($case.arguments) $workspace 'program'
            $after = if ($inputPath) { Hash $inputPath } else { $null }
            $entry = @{variant=$variant; profile=$profile; name=$case.name; expected=$case.expected; actual=$actual; inputSha256Before=$before; inputSha256After=$after}
            $results.Add($entry)
            Assert-AdoptionOutputs @($entry)
            if ($inputPath -and $after -ne $before) { throw 'Copied input changed' }
        }
    }
}
foreach ($path in $protected.Keys) { if ((Hash $path) -ne $protected[$path]) { throw "Protected original changed: $path" } }
$receipt = @{schema='fairyc.ledger-compare-adoption/v1'; harnessSha256=(Hash $PSCommandPath); compilerSha256=$compilerHash; compilerVersion=$version.stdout; pinnedCompilerSha256=$pinnedHash; variants=@('legacy','sdk'); profiles=@('development','release'); casesPerImage=$cases.Count; passed=$results.Count; failed=0; skipped=0; protectedFileCount=$protected.Count; protectedFiles=$protected; digests=$digests; images=$images; cases=$results; oracle='Historical independent expectations plus BigInteger/UTF-8 hex ordinal fixture'; scope='Same current compiler for both variants; not a historical-compiler rerun or full round4 revalidation'}
$receipt.auditHelperSha256 = Hash (Join-Path $PSScriptRoot 'adoption-output-audit.ps1')
[IO.File]::WriteAllText((Join-Path $run 'receipt.json'), ($receipt | ConvertTo-Json -Depth 10) + "`n", $utf8)
Write-Output "PASS $($results.Count) exact-output executions; protected originals unchanged; receipt: $(Join-Path $run 'receipt.json')"
