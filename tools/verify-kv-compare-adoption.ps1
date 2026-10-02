#requires -Version 7.0
param(
    [string]$Original = 'C:\Users\cherr\Downloads\fairy_c_test_projects\round4\09-persistent-kv-store',
    [string]$Fairyc = (Join-Path $PSScriptRoot '../.cache/ledger-compare-toolchain/fairyc.exe'),
    [string]$AuditReceipt = ''
)
$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
function Assert-OrdinalOutputs($Entries) {
    foreach ($entry in $Entries) {
        if ($entry.actual.exitCode -ne $entry.expected.exitCode -or
            ![StringComparer]::Ordinal.Equals($entry.actual.stdout, $entry.expected.stdout) -or
            ![StringComparer]::Ordinal.Equals($entry.actual.stderr, $entry.expected.stderr)) {
            throw "Ordinal output mismatch: $($entry.variant)/$($entry.profile)/$($entry.name)"
        }
    }
}
if ($AuditReceipt) {
    $receiptPath = (Resolve-Path -LiteralPath $AuditReceipt).Path
    $cacheRoot = [IO.Path]::GetFullPath((Join-Path $repo '.cache/kv-compare-adoption')) + [IO.Path]::DirectorySeparatorChar
    if (!$receiptPath.StartsWith($cacheRoot, [StringComparison]::OrdinalIgnoreCase) -or [IO.Path]::GetFileName($receiptPath) -cne 'receipt.json') { throw 'Audit only this checkout KV adoption receipt' }
    $receipt = Get-Content -LiteralPath $receiptPath -Raw | ConvertFrom-Json
    if ($receipt.schema -cne 'fairyc.kv-compare-adoption/v1' -or $receipt.passed -ne 328 -or $receipt.cases.Count -ne 328) { throw 'Unexpected KV adoption receipt' }
    # Prove that the stricter gate rejects Unicode spellings the cultural operator merges.
    $caught = $false
    try { Assert-OrdinalOutputs @(@{actual=@{exitCode=0; stdout='é'; stderr=''}; expected=@{exitCode=0; stdout=('e'+[char]0x301); stderr=''}}) }
    catch { if (!$_.Exception.Message.StartsWith('Ordinal output mismatch:', [StringComparison]::Ordinal)) { throw }; $caught=$true }
    if (!$caught) { throw 'Ordinal mismatch guard failed' }
    Assert-OrdinalOutputs $receipt.cases
    foreach ($entry in $receipt.cases) {
        $raw = [IO.File]::ReadAllLines($entry.rawEvidence.path)[$entry.rawEvidence.line-1] | ConvertFrom-Json
        if ($raw.kind -cne 'program' -or $raw.exitCode -ne $entry.actual.exitCode -or
            ![StringComparer]::Ordinal.Equals($raw.cwd, $entry.cwd) -or
            ![StringComparer]::Ordinal.Equals($raw.stdout, $entry.actual.stdout) -or
            ![StringComparer]::Ordinal.Equals($raw.stderr, $entry.actual.stderr) -or $raw.argv.Count -ne $entry.argv.Count) { throw 'Raw record mismatch' }
        for ($i=0; $i -lt $raw.argv.Count; $i++) { if (![StringComparer]::Ordinal.Equals($raw.argv[$i], $entry.argv[$i])) { throw 'Raw argv mismatch' } }
        if ($entry.evidence.kind -eq 'checkpoint-kill') {
            $checkpoint = $raw.checkpoint
            if ($checkpoint.startedPid -ne $checkpoint.checkpointPid -or $checkpoint.startedPid -ne $checkpoint.killedPid -or !$checkpoint.exited -or $checkpoint.startedPid -ne $entry.evidence.checkpoint.startedPid -or $entry.evidence.powerLossClaim) { throw 'Raw owned-child evidence mismatch' }
        }
        if ($entry.evidence.kind -eq 'actual-sdk-os-publication-failure' -and (!$raw.checkpoint.lockAcquiredAfterCheckpoint -or !$entry.evidence.isActualSdkFailure)) { throw 'Actual SDK failure evidence mismatch' }
        foreach ($output in $entry.outputs) { if ((Get-FileHash -LiteralPath $output.path).Hash.ToLowerInvariant() -ne $output.actualSha256 -or $output.actualSha256 -ne $output.expectedSha256) { throw 'Retained snapshot bytes changed' } }
    }
    foreach ($image in $receipt.images) { if ((Get-FileHash -LiteralPath $image.executable).Hash.ToLowerInvariant() -ne $image.sha256) { throw 'Retained EXE changed' } }
    if ((Get-FileHash -LiteralPath $Fairyc).Hash.ToLowerInvariant() -ne $receipt.compilerSha256) { throw 'Retained compiler changed' }
    foreach ($property in $receipt.protectedFiles.PSObject.Properties) { if ((Get-FileHash -LiteralPath $property.Name).Hash.ToLowerInvariant() -ne $property.Value) { throw 'Protected original changed' } }
    $executedHarness = Join-Path (Split-Path $receiptPath) 'harness-executed.ps1'
    if (!(Test-Path -LiteralPath $executedHarness)) { $executedHarness=$PSCommandPath }
    if ((Get-FileHash -LiteralPath $executedHarness).Hash.ToLowerInvariant() -ne $receipt.harnessSha256) { throw 'Executed harness hash mismatch' }
    $audit = @{receiptSha256=(Get-FileHash -LiteralPath $receiptPath).Hash.ToLowerInvariant(); executedHarnessSha256=$receipt.harnessSha256; auditHarnessSha256=(Get-FileHash -LiteralPath $PSCommandPath).Hash.ToLowerInvariant(); ordinalCases=328; normalizationMismatchRejected=$caught; nativeExecutions=0}
    [IO.File]::WriteAllText((Join-Path (Split-Path $receiptPath) 'ordinal-audit.json'), ($audit | ConvertTo-Json) + "`n", [Text.UTF8Encoding]::new($false))
    Write-Output 'PASS 328 ordinal output/raw/state/EXE/original checks; Unicode mismatch guard rejected; no native executions'
    return
}
$Original = (Resolve-Path -LiteralPath $Original).Path
$Fairyc = (Resolve-Path -LiteralPath $Fairyc).Path
$utf8 = [Text.UTF8Encoding]::new($false, $true)
$evidenceRoot = Join-Path $repo ('.cache/kv-compare-adoption/' + [Guid]::NewGuid().ToString('N'))
[void][IO.Directory]::CreateDirectory($evidenceRoot)
$run = $evidenceRoot; $workspace = $repo
function Hash([string]$Path) { (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant() }
function Full([string]$Path) {
    if ([IO.Path]::IsPathFullyQualified($Path)) { return [IO.Path]::GetFullPath($Path) }
    return [IO.Path]::GetFullPath((Join-Path $workspace $Path))
}
function Execute([string]$Exe, [object[]]$Arguments, [string]$Kind) {
    $start = [Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $Exe; $start.WorkingDirectory = $workspace
    $start.UseShellExecute = $false; $start.CreateNoWindow = $true
    $start.RedirectStandardOutput = $true; $start.RedirectStandardError = $true
    $start.StandardOutputEncoding = $utf8; $start.StandardErrorEncoding = $utf8
    foreach ($argument in $Arguments) {
        if ($null -eq $argument -or $argument -isnot [string]) { throw 'Arguments must be non-null strings' }
        $start.ArgumentList.Add($argument)
    }
    $process = [Diagnostics.Process]::new(); $process.StartInfo = $start
    try {
        [void]$process.Start()
        $stdout = $process.StandardOutput.ReadToEndAsync(); $stderr = $process.StandardError.ReadToEndAsync()
        if (!$process.WaitForExit(60000)) { $process.Kill($true); $process.WaitForExit(); throw 'Owned child exceeded 60 seconds' }
        $actual = [pscustomobject]@{exitCode=$process.ExitCode; stdout=$stdout.GetAwaiter().GetResult(); stderr=$stderr.GetAwaiter().GetResult()}
        $raw = @{kind=$Kind; executable=$Exe; argv=@($Arguments); cwd=$workspace; exitCode=$actual.exitCode; stdout=$actual.stdout; stderr=$actual.stderr}
        [IO.File]::AppendAllText((Join-Path $run 'raw-executions.jsonl'), ($raw | ConvertTo-Json -Depth 6 -Compress) + "`n", $utf8)
        return $actual
    } finally { $process.Dispose() }
}
$pinned = Join-Path $Original '../_toolchain/fairyc-c34718bd/fairyc.exe'
$pinnedHash = Hash $pinned
if ($pinnedHash -ne '7d58dd9c8860f01723f55f2f289a609d0941fa3cd6b06699954ca46336eeb917') { throw 'Pinned compiler identity changed' }
$compilerHash = Hash $Fairyc
if ($compilerHash -eq $pinnedHash) { throw 'Use a separately built stage0.78 compiler' }
$version = Execute $Fairyc @('--version') 'identity'
if ($version.exitCode -ne 0 -or $version.stdout -notmatch 'sdk: fairyc-sdk-stdlib/0 0\.0\.0-stage0\.78') { throw 'Expected SDK stage0.78' }
$historical = Get-Content -LiteralPath (Join-Path $Original 'verification-results.json') -Raw | ConvertFrom-Json
foreach ($source in $historical.sourceFiles) {
    if ($source.path -notmatch '^(src/[a-z]+\.fy|fairy\.toml)$') { throw 'Unexpected historical source path' }
    if ((Hash (Join-Path $Original $source.path)) -ne $source.sha256) { throw 'Reviewed KV source changed' }
}
$helpers = @{
    'verify-support.ps1'='a5316c64dd7d96c69859243ad9daadb5b49b5145b761251196b8cbd91b91d372'
    'verify-oracle.ps1'='f87c3ae265bb2328a5a4906e3a4ba4cee9a7ecbf03542f013f962b80dac7e635'
    'verify-state.ps1'='999cbc5ec789e5f344cd3713798a9bf33d1c3578484d2339c1097350bb64080e'
    'verify-controlled.ps1'='53a5d3850f7db19aff156d11ff4cb0dff40d6f76a37c9f377f30a3ff1d4e298f'
    'verify-normal.ps1'='d6e336a2afc66fa4b245697ba010cba2dd2d122357378a2d55978fac2c14085f'
    'verify-faults.ps1'='91f3f6081ef49036b4e76f7d66d1c4760c49198cbec4fa80d91199ff111cf6cd'
    'verify-cases.ps1'='1e8124c978c8c63cee9a1d2ace9f21aee1fcfcdeeb4acc93954d59c6804b74ab'
}
foreach ($name in $helpers.Keys) {
    if ((Hash (Join-Path $Original $name)) -ne $helpers[$name]) { throw "Reviewed oracle/control helper changed: $name" }
}
$codecPath = Join-Path $Original 'src/codec.fy'
if ((Hash $codecPath) -ne '0cd44acb5c24b04ee8eb0d211c0d631912137ae4b2c038b6f7ca9ea75f9270d7') { throw 'Reviewed comparator changed' }
$legacy = @'
pub fn i32 compare(ref u8[] a, ref u8[] b) {
    var usize i = 0;
    while (i < a.len && i < b.len) { if (a[i] < b[i]) { return -1; } if (a[i] > b[i]) { return 1; } i += 1; }
    if (a.len < b.len) { return -1; } if (a.len > b.len) { return 1; } return 0;
}
'@
$replacement = 'pub fn i32 compare(ref u8[] a, ref u8[] b) { return bytes::compare(a, b); }'
$codec = [IO.File]::ReadAllText($codecPath, $utf8).Replace("`r`n", "`n")
if ($codec.Split(@($legacy), [StringSplitOptions]::None).Count -ne 2) { throw 'Comparator must match exactly once' }
$protected = @{}
$paths = @(Get-ChildItem -LiteralPath (Join-Path $Original 'src') -File | ForEach-Object FullName)
$paths += @(foreach ($name in @('fairy.toml','fairy.lock','README.md','REPORT.ko.md','verification-results.json','verify.ps1','check-result.json','build-results.json','compiler-identity.json','artifact-audit.json','fault-evidence.json','build/persistent-kv-store-development.exe','build/persistent-kv-store-release.exe') + @($helpers.Keys)) { Join-Path $Original $name })
$paths += @($pinned, (Join-Path $Original '../_toolchain/fairyc-c34718bd/receipt.json'))
foreach ($path in $paths | Select-Object -Unique) { $protected[$path] = Hash $path }
$selectedGroups = @('first-create','update-empty','insert-escape-sort','delete','delete-last','entries-128','entry-129','generation-max','pending-collision','pending-hardlink','missing-live-valid-orphan','fixed-empty-snapshot','corrupt-duplicate','corrupt-unsorted','corrupt-key-invalid-utf8','corrupt-checksum','corrupt-trailing','corrupt-short-header','app-fault-write','app-fault-sync','app-fault-publish','app-fault-stdout','kill-partial','kill-before-sync','kill-before-publish','kill-after-publish','sdk-os-publication-lock','first-create-kill-before-publish')
$historicalNames = @($historical.cases | Where-Object { $_.profile -ceq 'development' -and $_.name.Split('/')[0] -cin $selectedGroups } | ForEach-Object name)
if ($historicalNames.Count -ne 67 -or @($historicalNames | Select-Object -Unique).Count -ne 67) { throw 'Historical case inventory mismatch' }
$newGroups = @('byte-order-insert-first','byte-order-insert-middle','byte-order-insert-last','byte-order-update-composed','byte-order-delete-decomposed')
$Cases = @($selectedGroups) + @($newGroups)
$allResults = [Collections.Generic.List[object]]::new()
$images = [Collections.Generic.List[object]]::new(); $digests = [Collections.Generic.List[object]]::new()
foreach ($variant in 'legacy','sdk') {
    $workspace = Join-Path $evidenceRoot $variant
    [void][IO.Directory]::CreateDirectory((Join-Path $workspace 'src'))
    [IO.File]::Copy((Join-Path $Original 'fairy.toml'), (Join-Path $workspace 'fairy.toml'))
    foreach ($source in Get-ChildItem -LiteralPath (Join-Path $Original 'src') -File -Filter '*.fy') {
        [IO.File]::Copy($source.FullName, (Join-Path $workspace ('src/' + $source.Name)))
    }
    if ($variant -eq 'sdk') { [IO.File]::WriteAllText((Join-Path $workspace 'src/codec.fy'), $codec.Replace($legacy, $replacement), $utf8) }
    foreach ($name in $helpers.Keys) { [IO.File]::Copy((Join-Path $Original $name), (Join-Path $workspace $name)) }
    foreach ($name in @('verify-support.ps1','verify-oracle.ps1','verify-state.ps1','verify-controlled.ps1','verify-normal.ps1','verify-faults.ps1','verify-cases.ps1')) { . (Join-Path $workspace $name) }
    $run = Join-Path $workspace 'evidence'; [void][IO.Directory]::CreateDirectory($run)
    $completedKeys = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    $check = Execute $Fairyc @('check') 'check'
    if ($check.exitCode -ne 0) { throw "Check $variant failed: $($check.stderr)" }
    $lock = Get-Content -LiteralPath (Join-Path $workspace 'fairy.lock') -Raw | ConvertFrom-Json
    $sdk = $lock.packages | Where-Object sourceKind -eq sdk
    $sourceDigest = ($lock.packages | Where-Object role -eq root).sourceDigest
    if ($variant -eq 'legacy' -and $sourceDigest -ne $historical.sourceDigest) { throw 'Baseline source digest differs from preserved KV' }
    $digests.Add(@{variant=$variant; sourceDigest=$sourceDigest; sdkVersion=$sdk.version; sdkSourceDigest=$sdk.sourceDigest})
    foreach ($profile in 'development','release') {
        $buildArgs = if ($profile -eq 'release') { @('build','--release','--locked') } else { @('build','--locked') }
        $build = Execute $Fairyc $buildArgs 'build'
        if ($build.exitCode -ne 0) { throw "Build $variant/$profile failed: $($build.stderr)" }
        $exe = Join-Path $workspace "build/persistent-kv-store-$profile.exe"
        [IO.File]::Copy((Join-Path $workspace 'build/persistent-kv-store.exe'), $exe)
        $images.Add(@{variant=$variant; profile=$profile; executable=$exe; sha256=(Hash $exe)})
        $results = [Collections.Generic.List[object]]::new()
        Run-Cases $exe $profile
        if ($results.Count -ne $historicalNames.Count -or @($results.name | Where-Object { $_ -cnotin $historicalNames }).Count -ne 0 -or @($results.name | Select-Object -Unique).Count -ne $historicalNames.Count) { throw 'Selected case inventory mismatch' }
        # Distinct values and spellings; ordinary source preserves its generation/checksum/save policy.
        $names = @('한','é',('e' + [char]0x301),'a','aa','A',' A','a ','2','10')
        if ($names.Count -ne 10 -or $names[2].Length -ne 2) { throw 'Unicode fixture inventory mismatch' }
        $entries = @(for ($i=0; $i -lt $names.Count; $i++) { Entry $names[$i] ("value-$i"+[char]9+[char]10+[char]13+'\end') })
        foreach ($spec in @(@{group='byte-order-insert-first';key=' A'},@{group='byte-order-insert-middle';key='aa'},@{group='byte-order-insert-last';key='한'})) {
            $oldEntries = @($entries | Where-Object { ![StringComparer]::Ordinal.Equals($_.key, $spec.key) })
            $insert = @($entries | Where-Object { [StringComparer]::Ordinal.Equals($_.key, $spec.key) })[0]
            Mutation $spec.group $profile $exe 'set' $insert.key $insert.value 2147483648 $oldEntries $entries $insert.key $insert.value
        }
        $updated = @($entries | ForEach-Object { if ([StringComparer]::Ordinal.Equals($_.key, 'é')) { Entry 'é' '' } else { $_ } })
        Mutation 'byte-order-update-composed' $profile $exe 'set' 'é' '' 2147483648 $entries $updated 'é' ''
        $remaining = @($entries | Where-Object { ![StringComparer]::Ordinal.Equals($_.key, $names[2]) })
        $composed = @($entries | Where-Object { [StringComparer]::Ordinal.Equals($_.key, 'é') })[0]
        Mutation 'byte-order-delete-decomposed' $profile $exe 'delete' $names[2] '' 2147483648 $entries $remaining 'é' $composed.value
        if ($results.Count -ne 82) { throw 'Per-image case count mismatch' }
        Assert-OrdinalOutputs $results
        foreach ($entry in $results) {
            if ($entry.evidence.kind -eq 'checkpoint-kill') {
                $checkpoint = $entry.evidence.checkpoint
                if ($checkpoint.startedPid -ne $checkpoint.checkpointPid -or $checkpoint.startedPid -ne $checkpoint.killedPid -or !$checkpoint.exited -or $entry.evidence.powerLossClaim) { throw 'Invalid owned-child termination evidence' }
            }
            $entry.variant=$variant; $allResults.Add($entry)
        }
    }
}
foreach ($path in $protected.Keys) { if ((Hash $path) -ne $protected[$path]) { throw "Protected original changed: $path" } }
if ((Hash $Fairyc) -ne $compilerHash) { throw 'Compiler changed during verification' }
$receipt = @{schema='fairyc.kv-compare-adoption/v1'; harnessSha256=(Hash $PSCommandPath); compilerSha256=$compilerHash; compilerVersion=$version.stdout; pinnedCompilerSha256=$pinnedHash; helpers=$helpers; variants=@('legacy','sdk'); profiles=@('development','release'); casesPerImage=82; passed=$allResults.Count; failed=0; skipped=0; protectedFileCount=$protected.Count; protectedFiles=$protected; digests=$digests; images=$images; cases=$allResults; selectedGroups=$Cases; oracle='Reviewed independent BinaryWriter/BigInteger/UTF-8 ordinal snapshot and checksum oracle, whole-byte state trees, fresh processes and checkpoint-controlled faults'; scope='Same current compiler for both variants; selected adoption cases, not full round4 revalidation or physical power-loss evidence'}
[IO.File]::WriteAllText((Join-Path $evidenceRoot 'receipt.json'), ($receipt | ConvertTo-Json -Depth 16) + "`n", $utf8)
Write-Output "PASS $($allResults.Count) executions and whole-state oracles; protected originals unchanged; receipt: $(Join-Path $evidenceRoot 'receipt.json')"
