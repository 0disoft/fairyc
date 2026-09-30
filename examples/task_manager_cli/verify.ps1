param(
    [string]$Fairyc=(Join-Path $PSScriptRoot '../_toolchain/fairyc-0bafe327/fairyc.exe'),
    [switch]$ReuseDevelopment,
    [switch]$ReuseBothImages
)
$ErrorActionPreference='Stop'
$Fairyc=(Resolve-Path -LiteralPath $Fairyc).Path
$compilerHash=(Get-FileHash -LiteralPath $Fairyc).Hash.ToLowerInvariant()
if ($compilerHash -ne 'd0ebc6dc9f807fc2118f816214b906bc22fa7f26759891c5a6adfcb5d7850648') { throw 'Compiler checksum mismatch' }
if ($ReuseDevelopment -and $ReuseBothImages) { throw 'Choose only one image reuse option' }
$previousEvidence=$null; $previousEvidenceSha256=$null; $imageProvenance=@()
if ($ReuseBothImages) {
    $previousEvidencePath=Join-Path $PSScriptRoot 'verification-results.json'
    $previousEvidenceSha256=(Get-FileHash -LiteralPath $previousEvidencePath).Hash.ToLowerInvariant()
    $previousEvidence=Get-Content -LiteralPath $previousEvidencePath -Raw|ConvertFrom-Json
}
$utf8=[Text.UTF8Encoding]::new($false)
function Local([string]$Relative) {
    $path=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot $Relative))
    if (!$path.StartsWith($PSScriptRoot+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)) { throw 'Fixture path outside project' }
    return $path
}
function Write-Bytes([string]$Relative,[byte[]]$Data) { [IO.File]::WriteAllBytes((Local $Relative),$Data) }
function Write-Text([string]$Relative,[string]$Data) { Write-Bytes $Relative ($utf8.GetBytes($Data)) }
function Byte-Hash([byte[]]$Data) { return [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($Data)).ToLowerInvariant() }
function Snapshot([string]$Relative) {
    $path=Local $Relative
    if (![IO.File]::Exists($path)) { return @{ exists=$false } }
    $data=[IO.File]::ReadAllBytes($path)
    return @{ exists=$true; bytes=$data.Length; sha256=(Byte-Hash $data) }
}
function Run-Program([string]$Exe,[string[]]$Arguments) {
    $start=[Diagnostics.ProcessStartInfo]::new(); $start.FileName=$Exe; $start.WorkingDirectory=$PSScriptRoot
    $start.UseShellExecute=$false; $start.CreateNoWindow=$true; $start.RedirectStandardOutput=$true; $start.RedirectStandardError=$true
    $start.StandardOutputEncoding=$utf8; $start.StandardErrorEncoding=$utf8
    foreach ($arg in $Arguments) { $start.ArgumentList.Add($arg) }
    $process=[Diagnostics.Process]::new(); $process.StartInfo=$start
    try {
        [void]$process.Start(); $outTask=$process.StandardOutput.ReadToEndAsync(); $errTask=$process.StandardError.ReadToEndAsync()
        if (!$process.WaitForExit(15000)) { $process.Kill($true); $process.WaitForExit(); throw '15 second timeout' }
        return [pscustomobject]@{exitCode=$process.ExitCode;stdout=$outTask.GetAwaiter().GetResult();stderr=$errTask.GetAwaiter().GetResult()}
    } finally { $process.Dispose() }
}
function Assert-ObservedPaths([string]$Store,[string[]]$Paths) {
    $allowed=if ($Store -eq '') { @() } else { @($Store,($Store+'.tmp'),($Store+'.bak'),($Store+'.lock')) }
    foreach ($path in $Paths) {
        if ($path -notin $allowed) { throw "Unexpected watch/preserve path [$path] for store [$Store]" }
        [void](Local $path)
    }
}
function Assert-SnapshotKeys([hashtable]$Observed,[string[]]$Paths) {
    if ($Observed.Count -ne $Paths.Count) { throw 'Snapshot path count mismatch' }
    foreach ($path in $Paths) { if (!$Observed.ContainsKey($path)) { throw "Snapshot missing intended path [$path]" } }
}
function Assert-Case([string]$Name,[string[]]$Arguments,[int]$Code,[string]$Stdout,[string]$Stderr,[string]$Store='',
    [byte[]]$ExpectedFile=$null,[switch]$Absent,[string[]]$Preserve=@(),[switch]$Pending,[switch]$LockedHandle,
    [hashtable]$ExpectedSidecars=@{}) {
    $before=@{}; $after=@{}
    $watch=@($Preserve)
    if ($Store -ne '' -and !$LockedHandle) { $watch+=@($Store,($Store+'.tmp'),($Store+'.bak'),($Store+'.lock')) }
    $watch=@($watch|Select-Object -Unique)
    Assert-ObservedPaths $Store $Preserve
    Assert-ObservedPaths $Store $watch
    Assert-ObservedPaths $Store @($ExpectedSidecars.Keys)
    foreach ($path in $ExpectedSidecars.Keys) {
        if ($path -eq $Store -or $path -notin $watch) { throw 'Expected sidecar must have its own watched path' }
    }
    foreach ($path in $watch) { $before[$path]=Snapshot $path }
    Assert-SnapshotKeys $before $watch
    $actual=Run-Program $script:currentExe $Arguments
    if ($actual.exitCode -ne $Code -or $actual.stdout -cne $Stdout -or $actual.stderr -cne $Stderr) {
        throw "$script:profile/$Name expected $Code stdout [$Stdout] stderr [$Stderr]; got $($actual|ConvertTo-Json -Compress)"
    }
    foreach ($path in $watch) { $after[$path]=Snapshot $path }
    Assert-SnapshotKeys $after $watch
    foreach ($path in $Preserve) {
        if ($before[$path].exists -ne $after[$path].exists -or $before[$path].sha256 -ne $after[$path].sha256) { throw "$Name changed $path" }
    }
    if ($Absent -and [IO.File]::Exists((Local $Store))) { throw "$Name unexpectedly created store" }
    $expectedStore=$null
    if ($null -ne $ExpectedFile) {
        $got=[IO.File]::ReadAllBytes((Local $Store))
        if ($got.Length -ne $ExpectedFile.Length -or [Convert]::ToHexString($got) -cne [Convert]::ToHexString($ExpectedFile)) { throw "$Name store bytes mismatch" }
        $expectedStore=@{bytes=$ExpectedFile.Length;sha256=(Byte-Hash $ExpectedFile);utf8Text=$utf8.GetString($ExpectedFile)}
    }
    $expectedSidecarStates=@{}; $sidecarComparisons=@{}
    foreach ($path in $ExpectedSidecars.Keys) {
        $wanted=$ExpectedSidecars[$path]
        if (!$after[$path].exists) { throw "$Name expected sidecar missing: $path" }
        $got=[IO.File]::ReadAllBytes((Local $path))
        if ($got.Length -ne $wanted.Length -or [Convert]::ToHexString($got) -cne [Convert]::ToHexString($wanted)) { throw "$Name sidecar bytes mismatch: $path" }
        $expectedSidecarStates[$path]=@{exists=$true;bytes=$wanted.Length;sha256=(Byte-Hash $wanted);utf8Text=$utf8.GetString($wanted)}
        $sidecarComparisons[$path]=@{fullBytesEqual=$true;actualBytes=$got.Length;actualSha256=(Byte-Hash $got)}
    }
    if ($Store -ne '' -and !$Pending) {
        foreach ($suffix in '.lock','.tmp','.bak') { if ([IO.File]::Exists((Local ($Store+$suffix)))) { throw "$Name unexpected sidecar $suffix" } }
    }
    Write-Host "PASS $script:profile/$Name exit=$($actual.exitCode)"
    $item=@{profile=$script:profile;name=$Name;arguments=$Arguments;storePath=$Store;watchedPaths=$watch;watchPathsCheckedBeforeAfter=$true;expected=@{exitCode=$Code;stdout=$Stdout;stderr=$Stderr;store=$expectedStore;storeAbsent=[bool]$Absent;sidecars=$expectedSidecarStates};actual=$actual;before=$before;after=$after;preservedPaths=$Preserve;sidecarComparisons=$sidecarComparisons}
    $script:results.Add($item)
}
[void](New-Item -ItemType Directory -Path (Local 'samples') -Force)
[void](New-Item -ItemType Directory -Path (Local 'build') -Force)
$run='build/verification-'+[Guid]::NewGuid().ToString('N')
[void](New-Item -ItemType Directory -Path (Local $run))
$header="FYTASK1`t1`n"
$valid="FYTASK1`t3`n1`t0`t한글 작업`n2`t1`tsecond task`n"
Write-Text 'samples/tasks.tsv' $valid
$badStores=@(
    @{name='empty-file';data=$utf8.GetBytes('');message='invalid task store header'},
    @{name='header';data=$utf8.GetBytes("OTHER`t1`n");message='invalid task store header'},
    @{name='bom';data=([byte[]](239,187,191)+$utf8.GetBytes($header));message='invalid task store header'},
    @{name='next-zero';data=$utf8.GetBytes("FYTASK1`t0`n");message='invalid next task ID'},
    @{name='next-over';data=$utf8.GetBytes("FYTASK1`t2147483648`n");message='invalid next task ID'},
    @{name='missing-lf';data=$utf8.GetBytes("FYTASK1`t2`n1`t0`ttask");message='task record must end with LF'},
    @{name='status';data=$utf8.GetBytes("FYTASK1`t2`n1`t2`ttask`n");message='invalid task record fields'},
    @{name='id-leading-zero';data=$utf8.GetBytes("FYTASK1`t2`n01`t0`ttask`n");message='invalid task ID in store'},
    @{name='id-at-next';data=$utf8.GetBytes("FYTASK1`t1`n1`t0`ttask`n");message='task IDs must increase and be below next ID'},
    @{name='duplicate-id';data=$utf8.GetBytes("FYTASK1`t2`n1`t0`ta`n1`t1`tb`n");message='task IDs must increase and be below next ID'},
    @{name='unsorted-id';data=$utf8.GetBytes("FYTASK1`t3`n2`t0`tb`n1`t0`ta`n");message='task IDs must increase and be below next ID'},
    @{name='blank-title';data=$utf8.GetBytes("FYTASK1`t2`n1`t0`t   `n");message='invalid task title in store'},
    @{name='tab-title';data=$utf8.GetBytes("FYTASK1`t2`n1`t0`ta`tbad`n");message='invalid task title in store'},
    @{name='invalid-utf8';data=($utf8.GetBytes("FYTASK1`t2`n1`t0`t")+[byte[]](255,10));message='invalid task title in store'},
    @{name='over-file';data=[byte[]]::new(262145);message='store exceeds 262144 bytes'}
)
foreach ($fixture in $badStores) { Write-Bytes ('samples/'+$fixture.name+'.tsv') $fixture.data }
$full="FYTASK1`t257`n"; for ($i=1;$i -le 256;$i++) { $full+="$i`t0`tx`n" }
Write-Text 'samples/full.tsv' $full
Write-Text 'samples/exhausted.tsv' "FYTASK1`t2147483647`n"
$help="Usage: task-manager-cli <store-path> list`n       task-manager-cli <store-path> add `"title`"`n       task-manager-cli <store-path> done <id>`n       task-manager-cli <store-path> delete <id>`n       task-manager-cli --help`nIDs: canonical decimal, assigned 1..2147483646; never reused after deletion`nList: ascending ID, tab-separated ID/open|done/title; empty prints empty`nStore: UTF-8 no BOM, LF TSV, FYTASK1<TAB>next<LF> then id<TAB>0|1<TAB>title<LF>`nTitles: spaces/Korean preserved; nonblank UTF-8, no ASCII controls`nSave: exclusive .lock, checked write/sync/close of .tmp, then replace_file; no backup gap`nPending .lock/.tmp/.bak needs manual inspection/recovery; no automatic deletion`nLimits: 256 tasks, title 512 bytes, store 262144 bytes, path 4000 UTF-8 bytes`nExit: 0 success, 1 store/data/allocation/cleanup error, 2 CLI usage, 3 stdout failure`n"
$script:results=[Collections.Generic.List[object]]::new(); $images=@()
Push-Location $PSScriptRoot
try {
    & $Fairyc check --locked; if ($LASTEXITCODE -ne 0) { throw 'check --locked failed' }
    $lock=Get-Content fairy.lock -Raw|ConvertFrom-Json; $sourceDigest=($lock.packages|Where-Object role -eq root).sourceDigest
    $sdk=$lock.packages|Where-Object sourceKind -eq sdk
    if ($ReuseBothImages) {
        if ($previousEvidence.compilerSha256 -ne $compilerHash -or $previousEvidence.sourceDigest -ne $sourceDigest -or $previousEvidence.sdkSourceDigest -ne $sdk.sourceDigest) { throw 'Previous image provenance does not match compiler/source/SDK' }
        foreach ($candidateProfile in 'development','release') {
            $candidatePath="build/task-manager-cli-$candidateProfile.exe"
            $receipt=@($previousEvidence.images|Where-Object profile -eq $candidateProfile)
            $actualHash=(Get-FileHash -LiteralPath (Local $candidatePath)).Hash.ToLowerInvariant()
            $actualBytes=(Get-Item -LiteralPath (Local $candidatePath)).Length
            if ($receipt.Count -ne 1 -or $receipt[0].path -ne $candidatePath -or $receipt[0].sha256 -ne $actualHash -or $receipt[0].bytes -ne $actualBytes) { throw "$candidateProfile image provenance mismatch" }
            $imageProvenance+=@{profile=$candidateProfile;receipt='previous verification-results.json';compilerSha256=$compilerHash;sourceDigest=$sourceDigest;sdkSourceDigest=$sdk.sourceDigest;sha256=$actualHash;bytes=$actualBytes}
        }
        $developmentReceipt=Get-Content build/development-receipt.json -Raw|ConvertFrom-Json
        if ($developmentReceipt.compilerSha256 -ne $compilerHash -or $developmentReceipt.sourceDigest -ne $sourceDigest -or $developmentReceipt.sha256 -ne ($imageProvenance|Where-Object profile -eq development).sha256) { throw 'Development build receipt mismatch' }
    }
    foreach ($script:profile in 'development','release') {
        $script:currentExe=Local "build/task-manager-cli-$script:profile.exe"
        if ($ReuseBothImages) {
            Write-Host "Reusing $script:profile image; compiler/source/SDK/image provenance verified; no build"
        } elseif ($script:profile -eq 'development' -and $ReuseDevelopment) {
            $receipt=Get-Content build/development-receipt.json -Raw|ConvertFrom-Json
            if ($receipt.compilerSha256 -ne $compilerHash -or $receipt.sourceDigest -ne $sourceDigest -or $receipt.sha256 -ne (Get-FileHash -LiteralPath $script:currentExe).Hash.ToLowerInvariant()) { throw 'Development image provenance mismatch' }
        } else {
            if ($script:profile -eq 'release') { & $Fairyc build --release --locked } else { & $Fairyc build --locked }
            if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
            Copy-Item -LiteralPath build/task-manager-cli.exe -Destination $script:currentExe
            if ($script:profile -eq 'development') { @{compilerSha256=$compilerHash;sourceDigest=$sourceDigest;sha256=(Get-FileHash -LiteralPath $script:currentExe).Hash.ToLowerInvariant()}|ConvertTo-Json|Set-Content build/development-receipt.json -Encoding utf8 }
        }
        $store="$run/$script:profile-한글 공백.tsv"
        Assert-Case 'missing-list' @($store,'list') 0 "empty`n" '' $store -Absent
        $state="FYTASK1`t2`n1`t0`t한글 작업 내용`n"
        Assert-Case 'add-korean' @($store,'add','한글 작업 내용') 0 "added id=1`n" '' $store ($utf8.GetBytes($state))
        $state="FYTASK1`t3`n1`t0`t한글 작업 내용`n2`t0`t  second task  `n"
        Assert-Case 'add-spaces' @($store,'add','  second task  ') 0 "added id=2`n" '' $store ($utf8.GetBytes($state))
        Assert-Case 'restart-list' @($store,'list') 0 "1`topen`t한글 작업 내용`n2`topen`t  second task  `n" '' $store ($utf8.GetBytes($state)) -Preserve $store
        $state="FYTASK1`t3`n1`t1`t한글 작업 내용`n2`t0`t  second task  `n"
        Assert-Case 'done' @($store,'done','1') 0 "completed id=1`n" '' $store ($utf8.GetBytes($state))
        Assert-Case 'done-again' @($store,'done','1') 0 "completed id=1`n" '' $store ($utf8.GetBytes($state)) -Preserve $store
        Assert-Case 'restart-done-list' @($store,'list') 0 "1`tdone`t한글 작업 내용`n2`topen`t  second task  `n" '' $store ($utf8.GetBytes($state)) -Preserve $store
        $state="FYTASK1`t3`n2`t0`t  second task  `n"
        Assert-Case 'delete' @($store,'delete','1') 0 "deleted id=1`n" '' $store ($utf8.GetBytes($state))
        Assert-Case 'restart-delete-list' @($store,'list') 0 "2`topen`t  second task  `n" '' $store ($utf8.GetBytes($state)) -Preserve $store
        $state="FYTASK1`t4`n2`t0`t  second task  `n3`t0`tthird`n"
        Assert-Case 'id-not-reused' @($store,'add','third') 0 "added id=3`n" '' $store ($utf8.GetBytes($state))
        foreach ($command in 'done','delete') { Assert-Case ('missing-id-'+$command) @($store,$command,'1') 1 '' "${store}: error: task ID does not exist`n" $store -Preserve $store }
        $state="FYTASK1`t4`n3`t0`tthird`n"
        Assert-Case 'delete-second' @($store,'delete','2') 0 "deleted id=2`n" '' $store ($utf8.GetBytes($state))
        $state="FYTASK1`t4`n"
        Assert-Case 'delete-all' @($store,'delete','3') 0 "deleted id=3`n" '' $store ($utf8.GetBytes($state))
        Assert-Case 'restart-empty' @($store,'list') 0 "empty`n" '' $store ($utf8.GetBytes($state)) -Preserve $store
        $state="FYTASK1`t5`n4`t0`tnew after empty`n"
        Assert-Case 'id-after-empty' @($store,'add','new after empty') 0 "added id=4`n" '' $store ($utf8.GetBytes($state))
        foreach ($fixture in $badStores) {
            $bad='samples/'+$fixture.name+'.tsv'
            Assert-Case ('corrupt-'+$fixture.name) @($bad,'add','valid request') 1 '' "${bad}: error: $($fixture.message)`n" $bad -Preserve $bad
        }
        Assert-Case 'count-limit' @('samples/full.tsv','add','extra') 1 '' "samples/full.tsv: error: task count exceeds 256`n" 'samples/full.tsv' -Preserve 'samples/full.tsv'
        Assert-Case 'id-exhausted' @('samples/exhausted.tsv','add','extra') 1 '' "samples/exhausted.tsv: error: task ID space exhausted`n" 'samples/exhausted.tsv' -Preserve 'samples/exhausted.tsv'
        $usageCases=@(
            @{name='empty-title';args=@($store,'add','');message='title must be valid UTF-8, 1..512 bytes, nonblank, without ASCII controls'},
            @{name='blank-title';args=@($store,'add','   ');message='title must be valid UTF-8, 1..512 bytes, nonblank, without ASCII controls'},
            @{name='tab-title';args=@($store,'add',"a`tb");message='title must be valid UTF-8, 1..512 bytes, nonblank, without ASCII controls'},
            @{name='long-title';args=@($store,'add',('a'*513));message='title must be valid UTF-8, 1..512 bytes, nonblank, without ASCII controls'},
            @{name='argument-overflow';args=@($store,'add',('a'*514));message='cannot read title/ID (maximum title 512 UTF-8 bytes)'},
            @{name='id-zero';args=@($store,'done','0');message='ID must be canonical decimal 1..2147483647'},
            @{name='id-leading-zero';args=@($store,'delete','01');message='ID must be canonical decimal 1..2147483647'},
            @{name='id-negative';args=@($store,'done','-1');message='ID must be canonical decimal 1..2147483647'},
            @{name='id-overflow';args=@($store,'done','2147483648');message='ID must be canonical decimal 1..2147483647'},
            @{name='id-junk';args=@($store,'done','1x');message='ID must be canonical decimal 1..2147483647'},
            @{name='unknown-command';args=@($store,'bogus');message='unknown command; use list, add, done or delete'},
            @{name='extra-list-arg';args=@($store,'list','extra');message='wrong command argument count; use --help'},
            @{name='missing-add-title';args=@($store,'add');message='wrong command argument count; use --help'},
            @{name='no-args';args=@();message='expected store path and command; use --help'},
            @{name='empty-path';args=@('','list');message='store path must contain 1..4000 UTF-8 bytes'}
        )
        foreach ($case in $usageCases) { Assert-Case $case.name $case.args 2 '' "task-manager-cli: error: $($case.message)`n" $store -Preserve $store }
        Assert-Case 'help' @('--help') 0 $help ''
        $missing="$run/$script:profile-missing.tsv"
        Assert-Case 'missing-id-empty-store' @($missing,'done','1') 1 '' "${missing}: error: task ID does not exist`n" $missing -Absent
        $noParent="$run/no-parent/$script:profile.tsv"
        Assert-Case 'missing-parent' @($noParent,'add','title') 1 '' "${noParent}.lock: error: create lock failed; another run or stale lock (status=1 os_code=3)`n" $noParent -Absent
        foreach ($suffix in '.tmp','.bak','.lock') {
            Write-Text ($store+$suffix) 'do not change'
            $message=if ($suffix -eq '.lock') { "${store}.lock: error: create lock failed; another run or stale lock (status=1 os_code=80)`n" } else { "${store}${suffix}: error: recovery sidecar exists; inspect .tmp/.bak before retry`n" }
            Assert-Case ('pending-'+$suffix) @($store,'add','must not save') 1 '' $message $store -Preserve @($store,($store+$suffix)) -Pending
            # Remove only this fixture file, using the checked project-contained absolute path.
            Remove-Item -LiteralPath (Local ($store+$suffix))
        }
        $wanted=$utf8.GetBytes("FYTASK1`t6`n4`t0`tnew after empty`n5`t0`tblocked save`n")
        $held=[IO.File]::Open((Local $store),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::Read)
        try {
            Assert-Case 'replacement-denied' @($store,'add','blocked save') 1 '' "${store}: error: replace failed; live store unchanged; temporary file preserved (status=1 os_code=5)`n" $store -Preserve $store -Pending -ExpectedSidecars @{($store+'.tmp')=$wanted}
        } finally { $held.Dispose() }
        Assert-Case 'retry-refuses-recovery-file' @($store,'list') 1 '' "${store}.tmp: error: recovery sidecar exists; inspect .tmp/.bak before retry`n" $store -Preserve @($store,($store+'.tmp')) -Pending
        Remove-Item -LiteralPath (Local ($store+'.tmp'))
        $held=[IO.File]::Open((Local $store),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
        try { Assert-Case 'read-denied' @($store,'list') 1 '' "${store}: error: open store failed (status=1 os_code=32)`n" $store -LockedHandle } finally { $held.Dispose() }
        Assert-Case 'original-after-failed-save' @($store,'list') 0 "4`topen`tnew after empty`n" '' $store ($utf8.GetBytes($state)) -Preserve $store
        $boundary="$run/$script:profile-title-boundary.tsv"; $title='a'*512
        Assert-Case 'title-512' @($boundary,'add',$title) 0 "added id=1`n" '' $boundary ($utf8.GetBytes("FYTASK1`t2`n1`t0`t$title`n"))
        $images+=@{profile=$script:profile;path="build/task-manager-cli-$script:profile.exe";sha256=(Get-FileHash -LiteralPath $script:currentExe).Hash.ToLowerInvariant();bytes=(Get-Item -LiteralPath $script:currentExe).Length}
    }
    $sdk=$lock.packages|Where-Object sourceKind -eq sdk
    $evidence=[ordered]@{verifiedAt=[DateTimeOffset]::Now.ToString('o');compilerSha256=$compilerHash;sourceDigest=$sourceDigest;sdkVersion=$sdk.version;sdkSourceDigest=$sdk.sourceDigest;passed=$script:results.Count;failed=0;skipped=0;oracle='Fixed command/state table; independently constructed UTF-8 bytes and SHA256; separate process per command';runDirectory=$run;reusedBothImages=[bool]$ReuseBothImages;buildsThisRun=$(if ($ReuseBothImages) { 0 } elseif ($ReuseDevelopment) { 1 } else { 2 });previousEvidenceSha256=$previousEvidenceSha256;imageProvenance=$imageProvenance;watchPathsValidated=$true;unverified=@('Disk full, forced short writes, sync/close/stdout/allocation failure','Physical power loss, arbitrary interruption inside OS move, lock cleanup failure','Real concurrent process race and different path aliases','Other OS/CPU, clean-PC deployment, performance');images=$images;cases=$script:results}
    $evidence|ConvertTo-Json -Depth 10|Set-Content verification-results.json -Encoding utf8
    $script:results|ConvertTo-Json -Depth 10|Set-Content (Join-Path $run 'raw-executions.json') -Encoding utf8
    Write-Host "Verified $($script:results.Count) exact-output cases across development/release."
} finally { Pop-Location }
