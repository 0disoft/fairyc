#requires -Version 7.0
param(
    [string]$Original = 'C:\Users\cherr\Downloads\fairy_c_test_projects\round4\05-resource-pack-cli',
    [string]$Fairyc = (Join-Path $PSScriptRoot '../.cache/ledger-compare-toolchain/fairyc.exe'),
    [string]$AuditReceipt = ''
)
$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
. (Join-Path $PSScriptRoot 'adoption-output-audit.ps1')
if ($AuditReceipt) { Invoke-AdoptionOrdinalAudit $AuditReceipt 'pack' $Fairyc $PSCommandPath; return }
$Original = (Resolve-Path -LiteralPath $Original).Path
$Fairyc = (Resolve-Path -LiteralPath $Fairyc).Path
$utf8 = [Text.UTF8Encoding]::new($false, $true)
$evidenceRoot = Join-Path $repo ('.cache/pack-compare-adoption/' + [Guid]::NewGuid().ToString('N'))
[void][IO.Directory]::CreateDirectory($evidenceRoot)
$run = $evidenceRoot
$workspace = $repo
function Hash([string]$Path) { (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant() }
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
function Diagnostic([string]$Path, [int]$Line, [string]$Message) {
    $where = if ($Line -eq 0) { $Path } else { "${Path}:line=$Line" }
    return "${where}: error: $Message`n"
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
    if ((Hash (Join-Path $Original $source.path)) -ne $source.sha256) { throw 'Reviewed pack source changed' }
}
$helpers = @{'verify-support.ps1'='7fb3f6839bb06bf085b30b80d0703b47ddbac2e6c96dac2cc92f9d4e7a709b8f'; 'verify-cases.ps1'='3628fd3465959e3ae437e56f13999525fa1a5195f89c84b9dcf5505032fcc095'}
foreach ($name in $helpers.Keys) {
    if ((Hash (Join-Path $Original $name)) -ne $helpers[$name]) { throw 'Reviewed independent oracle changed' }
}
$codecPath = Join-Path $Original 'src/codec.fy'
if ((Hash $codecPath) -ne 'afb09ec8c09d47f896079d30f60ade4129e9f3a72d6dfa8c0e5092841660a5c8') { throw 'Reviewed comparator changed' }
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
$paths += @(foreach ($name in @('fairy.toml','fairy.lock','README.md','REPORT.ko.md','verification-results.json','verify.ps1','verify-support.ps1','verify-cases.ps1','compiler-identity.json','check-result.json','artifact-audit.json','build/resource-pack-cli-development.exe','build/resource-pack-cli-release.exe')) { Join-Path $Original $name })
$paths += $pinned
foreach ($path in $paths | Select-Object -Unique) { $protected[$path] = Hash $path }
# Reuse only reviewed comparison consumers and file-preservation checks, not all 109 cases.
$Phase = 'existing'
$Cases = @('set-empty','set-one','set-multi','fixed-pack','pack-duplicate','pack-unsorted','pack-late-invalid-list','pack-late-invalid-extract','create-duplicate','create-existing','create-source-alias','create-tsv-alias','create-hardlink','extract-sentinel','extract-hardlink','extract-missing-name')
$historicalNames = @('empty/create','empty/list','one/create','one/list','one/extract','multi/create','multi/list','multi/extract','multi/extract-A','multi/extract-../outside.bin','multi/extract-C:stream','multi/create-reversed','fixed-pack/list','fixed-pack/extract','pack-duplicate','pack-unsorted','pack-late-invalid-list','pack-late-invalid-extract','create-duplicate','create-existing','create-source-alias','create-tsv-alias','create-hardlink','extract-sentinel','extract-hardlink','extract-missing-name')
$newNames = @('byte-order/create','byte-order/list','byte-order/extract-0','byte-order/extract-1','byte-order/extract-2','byte-order/create-reversed')
$allResults = [Collections.Generic.List[object]]::new()
$images = [Collections.Generic.List[object]]::new(); $digests = [Collections.Generic.List[object]]::new()
foreach ($variant in 'legacy','sdk') {
    $workspace = Join-Path $evidenceRoot $variant
    [void][IO.Directory]::CreateDirectory((Join-Path $workspace 'src'))
    [void][IO.Directory]::CreateDirectory((Join-Path $workspace 'samples'))
    [IO.File]::Copy((Join-Path $Original 'fairy.toml'), (Join-Path $workspace 'fairy.toml'))
    foreach ($source in Get-ChildItem -LiteralPath (Join-Path $Original 'src') -File -Filter '*.fy') {
        [IO.File]::Copy($source.FullName, (Join-Path $workspace ('src/' + $source.Name)))
    }
    if ($variant -eq 'sdk') { [IO.File]::WriteAllText((Join-Path $workspace 'src/codec.fy'), $codec.Replace($legacy, $replacement), $utf8) }
    foreach ($name in $helpers.Keys) { [IO.File]::Copy((Join-Path $Original $name), (Join-Path $workspace $name)) }
    # The copied helpers resolve their PSScriptRoot inside this unique workspace.
    . (Join-Path $workspace 'verify-support.ps1')
    . (Join-Path $workspace 'verify-cases.ps1')
    $run = Join-Path $workspace 'evidence'
    [void][IO.Directory]::CreateDirectory($run)
    $check = Execute $Fairyc @('check') 'check'
    if ($check.exitCode -ne 0) { throw "Check $variant failed: $($check.stderr)" }
    $lock = Get-Content -LiteralPath (Join-Path $workspace 'fairy.lock') -Raw | ConvertFrom-Json
    $sdk = $lock.packages | Where-Object sourceKind -eq sdk
    $sourceDigest = ($lock.packages | Where-Object role -eq root).sourceDigest
    if ($variant -eq 'legacy' -and $sourceDigest -ne $historical.sourceDigest) { throw 'Baseline source digest differs from preserved pack' }
    $digests.Add(@{variant=$variant; sourceDigest=$sourceDigest; sdkVersion=$sdk.version; sdkSourceDigest=$sdk.sourceDigest})
    foreach ($profile in 'development','release') {
        $buildArgs = if ($profile -eq 'release') { @('build','--release','--locked') } else { @('build','--locked') }
        $build = Execute $Fairyc $buildArgs 'build'
        if ($build.exitCode -ne 0) { throw "Build $variant/$profile failed: $($build.stderr)" }
        $exe = Join-Path $workspace "build/resource-pack-cli-$profile.exe"
        [IO.File]::Copy((Join-Path $workspace 'build/resource-pack-cli.exe'), $exe)
        $images.Add(@{variant=$variant; profile=$profile; executable=$exe; sha256=(Hash $exe)})
        $results = [Collections.Generic.List[object]]::new()
        Run-Cases $exe $profile
        if ($results.Count -ne $historicalNames.Count -or @($results.name | Where-Object { $_ -notin $historicalNames }).Count -ne 0 -or @($results.name | Select-Object -Unique).Count -ne $historicalNames.Count) { throw 'Selected historical case inventory mismatch' }
        # Name ordering oracle: UTF-8 hex ordinal/BinaryWriter, including distinct Unicode spellings.
        $names = @('/한','é',('e' + [char]0x301),'a','aa','A',' A','a ','2','10')
        if ($names.Count -ne 10 -or $names[2].Length -ne 2) { throw 'Unicode fixture inventory mismatch' }
        $entries = @(for ($i=0; $i -lt $names.Count; $i++) { Entry $names[$i] ([byte[]](0,255,128,$i)) })
        $scope = Root $profile 'byte-order'; Fixture-Entries $scope $entries
        $pack = $scope + '/pack.frpk'; $expected = Pack $entries
        Run 'byte-order/create' $profile $exe @('create',($scope+'/entries.tsv'),$pack) 0 '' $scope @{'pack.frpk'=$expected} (ProtectedTree $scope)
        Run 'byte-order/list' $profile $exe @('list',$pack) 0 '' $scope @{} (ProtectedTree $scope) '' (Listing $entries)
        for ($i=0; $i -lt 3; $i++) {
            Run "byte-order/extract-$i" $profile $exe @('extract',$pack,$names[$i],($scope+"/output-$i.bin")) 0 '' $scope @{("output-$i.bin")=$entries[$i].data} (ProtectedTree $scope)
        }
        $reverse = @($entries); [Array]::Reverse($reverse)
        Fixture-Entries $scope $reverse
        Run 'byte-order/create-reversed' $profile $exe @('create',($scope+'/entries.tsv'),($scope+'/reverse.frpk')) 0 '' $scope @{'reverse.frpk'=$expected} (ProtectedTree $scope)
        if ($results.Count -ne $historicalNames.Count + $newNames.Count) { throw 'Per-image case count mismatch' }
        Assert-AdoptionOutputs $results
        foreach ($entry in $results) { $entry.variant=$variant; $allResults.Add($entry) }
    }
}
foreach ($path in $protected.Keys) { if ((Hash $path) -ne $protected[$path]) { throw "Protected original changed: $path" } }
if ((Hash $Fairyc) -ne $compilerHash) { throw 'Compiler changed during verification' }
$receipt = @{schema='fairyc.pack-compare-adoption/v1'; harnessSha256=(Hash $PSCommandPath); compilerSha256=$compilerHash; compilerVersion=$version.stdout; pinnedCompilerSha256=$pinnedHash; helpers=$helpers; variants=@('legacy','sdk'); profiles=@('development','release'); casesPerImage=($historicalNames.Count+$newNames.Count); passed=$allResults.Count; failed=0; skipped=0; protectedFileCount=$protected.Count; protectedFiles=$protected; digests=$digests; images=$images; cases=$allResults; oracle='Reviewed independent BinaryWriter/BigInteger/UTF-8 hex ordinal helpers plus handwritten fixed pack; whole-byte, file-tree and exact output comparisons'; scope='Same current compiler for both variants; not full round4 revalidation, historical-compiler rerun or fault injection'}
$receipt.auditHelperSha256 = Hash (Join-Path $PSScriptRoot 'adoption-output-audit.ps1')
[IO.File]::WriteAllText((Join-Path $evidenceRoot 'receipt.json'), ($receipt | ConvertTo-Json -Depth 14) + "`n", $utf8)
Write-Output "PASS $($allResults.Count) executions and whole-byte oracles; protected originals unchanged; receipt: $(Join-Path $evidenceRoot 'receipt.json')"
