# Shared by all four opt-in adoption harnesses; dot-sourcing starts no process.
function Assert-AdoptionObservation($Actual, $Expected, [string]$Label) {
    foreach ($observation in @($Actual, $Expected)) {
        if ($null -eq $observation -or $observation.stdout -isnot [string] -or $observation.stderr -isnot [string] -or
            ($observation.exitCode -isnot [int] -and $observation.exitCode -isnot [long])) { throw "Malformed output observation: $Label" }
    }
    if ($Actual.exitCode -ne $Expected.exitCode -or
        ![StringComparer]::Ordinal.Equals($Actual.stdout, $Expected.stdout) -or
        ![StringComparer]::Ordinal.Equals($Actual.stderr, $Expected.stderr)) { throw "Ordinal output mismatch: $Label" }
}
function Assert-AdoptionOutputs($Entries) {
    foreach ($entry in $Entries) { Assert-AdoptionObservation $entry.actual $entry.expected "$($entry.variant)/$($entry.profile)/$($entry.name)" }
}
function Invoke-AdoptionOrdinalAudit([string]$ReceiptPath, [ValidateSet('ledger','log','pack','kv')][string]$Kind, [string]$Fairyc, [string]$HarnessPath, [string]$OutputPath = '', [switch]$PassThru) {
    $repository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
    $receiptPath = (Resolve-Path -LiteralPath $ReceiptPath).Path
    $cacheRoot = [IO.Path]::GetFullPath((Join-Path $repository ".cache/$Kind-compare-adoption")) + [IO.Path]::DirectorySeparatorChar
    if (!$receiptPath.StartsWith($cacheRoot, [StringComparison]::OrdinalIgnoreCase) -or [IO.Path]::GetFileName($receiptPath) -cne 'receipt.json') { throw 'Audit only this checkout matching adoption receipt' }
    $runRoot = Split-Path $receiptPath
    $receiptHash = (Get-FileHash -LiteralPath $receiptPath).Hash.ToLowerInvariant()
    $receipt = Get-Content -LiteralPath $receiptPath -Raw | ConvertFrom-Json
    $expectedCount = @{ledger=68; log=104; pack=128; kv=328}[$Kind]
    $flat = $Kind -in @('pack','kv')
    if ($receipt.schema -cne "fairyc.$Kind-compare-adoption/v1" -or $receipt.passed -ne $expectedCount -or
        $receipt.cases.Count -ne $expectedCount -or $receipt.images.Count -ne 4 -or $receipt.failed -ne 0 -or $receipt.skipped -ne 0) { throw 'Unexpected adoption receipt inventory' }
    $imageIndex = [Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    foreach ($image in $receipt.images) {
        $key = $image.variant + '/' + $image.profile
        if ($image.variant -cnotin 'legacy','sdk' -or $image.profile -cnotin 'development','release' -or $imageIndex.ContainsKey($key)) { throw 'Unexpected or duplicate image' }
        $imageIndex.Add($key, $image)
        if ((Get-FileHash -LiteralPath $image.executable).Hash.ToLowerInvariant() -ne $image.sha256) { throw 'Retained EXE changed' }
    }
    if ((Get-FileHash -LiteralPath $Fairyc).Hash.ToLowerInvariant() -ne $receipt.compilerSha256) { throw 'Retained compiler changed' }
    foreach ($property in $receipt.protectedFiles.PSObject.Properties) {
        if ((Get-FileHash -LiteralPath $property.Name).Hash.ToLowerInvariant() -ne $property.Value) { throw 'Protected original changed' }
    }
    $executedHarness = Join-Path $runRoot 'harness-executed.ps1'
    if (!(Test-Path -LiteralPath $executedHarness)) { $executedHarness=$HarnessPath }
    if ((Get-FileHash -LiteralPath $executedHarness).Hash.ToLowerInvariant() -ne $receipt.harnessSha256) { throw 'Executed harness hash mismatch' }
    $helperHash = (Get-FileHash -LiteralPath (Join-Path $PSScriptRoot 'adoption-output-audit.ps1')).Hash.ToLowerInvariant()
    foreach ($helper in @(@{property='auditHelperSha256'; file='adoption-output-audit'}, @{property='processHelperSha256'; file='adoption-process'})) {
        $recordedHash = $receipt.($helper.property)
        if ($recordedHash) {
            $executedHelper = Join-Path $runRoot ($helper.file+'-executed.ps1')
            if (!(Test-Path -LiteralPath $executedHelper)) { $executedHelper=Join-Path $PSScriptRoot ($helper.file+'.ps1') }
            if ((Get-FileHash -LiteralPath $executedHelper).Hash.ToLowerInvariant() -ne $recordedHash) { throw 'Executed helper hash mismatch' }
        }
    }
    $rawFiles = [Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    $paths = if ($flat) { @('legacy/evidence/raw-executions.jsonl','sdk/evidence/raw-executions.jsonl') } else { @('raw-executions.jsonl') }
    $programRecords = [Collections.Generic.List[object]]::new()
    foreach ($path in $paths) {
        $fullPath = [IO.Path]::GetFullPath((Join-Path $runRoot $path))
        $records = @(Get-Content -LiteralPath $fullPath | ForEach-Object { $_ | ConvertFrom-Json })
        $rawFiles.Add($fullPath, $records)
        foreach ($record in $records) { if ($record.kind -ceq 'program') { $programRecords.Add($record) } }
    }
    if ($programRecords.Count -ne $expectedCount) { throw 'Raw program inventory mismatch' }
    $seen = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    $imageCounts = @{}
    Assert-AdoptionOutputs $receipt.cases
    $normalizationMismatchRejected = $false
    try { Assert-AdoptionObservation @{exitCode=0;stdout='é';stderr=''} @{exitCode=0;stdout=('e'+[char]0x301);stderr=''} 'unicode-guard' }
    catch { if (!$_.Exception.Message.StartsWith('Ordinal output mismatch:', [StringComparison]::Ordinal)) { throw }; $normalizationMismatchRejected=$true }
    if (!$normalizationMismatchRejected) { throw 'Ordinal mismatch guard failed' }
    for ($caseIndex=0; $caseIndex -lt $receipt.cases.Count; $caseIndex++) {
        $entry = $receipt.cases[$caseIndex]
        $key = $entry.variant + '/' + $entry.profile
        if (!$imageIndex.ContainsKey($key) -or !$seen.Add($key+'/'+$entry.name)) { throw 'Unexpected or duplicate case' }
        $imageCounts[$key]++
        $raw = $programRecords[$caseIndex]
        if ($flat) {
            $pointerPath = [IO.Path]::GetFullPath($entry.rawEvidence.path)
            if (!$rawFiles.ContainsKey($pointerPath) -or $entry.rawEvidence.line -lt 1 -or $entry.rawEvidence.line -gt $rawFiles[$pointerPath].Count) { throw 'Unexpected raw evidence pointer' }
            $pointed = $rawFiles[$pointerPath][$entry.rawEvidence.line-1]
            if ($pointed.kind -cne 'program' -or ![StringComparer]::Ordinal.Equals(($pointed | ConvertTo-Json -Depth 8 -Compress), ($raw | ConvertTo-Json -Depth 8 -Compress))) { throw 'Raw pointer/order mismatch' }
            if (![StringComparer]::Ordinal.Equals($raw.cwd, $entry.cwd) -or $raw.argv.Count -ne $entry.argv.Count) { throw 'Raw cwd/argv mismatch' }
            for ($i=0; $i -lt $raw.argv.Count; $i++) { if (![StringComparer]::Ordinal.Equals($raw.argv[$i], $entry.argv[$i])) { throw 'Raw argv mismatch' } }
            foreach ($output in $entry.outputs) {
                if ((Get-FileHash -LiteralPath $output.path).Hash.ToLowerInvariant() -ne $output.actualSha256 -or $output.actualSha256 -ne $output.expectedSha256) { throw 'Retained oracle output changed' }
            }
            if ($Kind -eq 'kv') {
                if ($entry.evidence.kind -eq 'checkpoint-kill') {
                    $checkpoint = $raw.checkpoint
                    if ($checkpoint.startedPid -ne $checkpoint.checkpointPid -or $checkpoint.startedPid -ne $checkpoint.killedPid -or !$checkpoint.exited -or
                        $checkpoint.startedPid -ne $entry.evidence.checkpoint.startedPid -or $entry.evidence.powerLossClaim) { throw 'Raw owned-child evidence mismatch' }
                }
                if ($entry.evidence.kind -eq 'actual-sdk-os-publication-failure' -and
                    (!$raw.checkpoint.lockAcquiredAfterCheckpoint -or !$entry.evidence.isActualSdkFailure)) { throw 'Actual SDK failure evidence mismatch' }
            }
        }
        if ($raw.kind -cne 'program' -or ![StringComparer]::Ordinal.Equals($raw.executable, $imageIndex[$key].executable) -or
            ![StringComparer]::Ordinal.Equals($raw.cwd, (Join-Path $runRoot $entry.variant))) { throw 'Raw executable/workspace mismatch' }
        foreach ($argument in $raw.argv) { if ($argument -isnot [string]) { throw 'Malformed raw argv' } }
        $rawActual = if ($flat) { $raw } else { $raw.actual }
        Assert-AdoptionObservation $rawActual $entry.actual "raw/$caseIndex"
        if (!$flat -and $entry.inputSha256Before) {
            $input = [IO.Path]::GetFullPath((Join-Path $raw.cwd $raw.argv[0]))
            if (!$input.StartsWith($raw.cwd+[IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase) -or
                (Get-FileHash -LiteralPath $input).Hash.ToLowerInvariant() -ne $entry.inputSha256Before -or $entry.inputSha256Before -ne $entry.inputSha256After) { throw 'Retained input changed' }
        }
    }
    foreach ($key in $imageIndex.Keys) { if ($imageCounts[$key] -ne $receipt.casesPerImage) { throw 'Per-image inventory mismatch' } }
    if ((Get-FileHash -LiteralPath $receiptPath).Hash.ToLowerInvariant() -ne $receiptHash) { throw 'Receipt changed during audit' }
    $rawHashes = @{}
    foreach ($path in $rawFiles.Keys) { $rawHashes[$path]=(Get-FileHash -LiteralPath $path).Hash.ToLowerInvariant() }
    $audit = @{schema='fairyc.adoption-ordinal-audit/v1'; kind=$Kind; receiptPath=$receiptPath; receiptSha256=$receiptHash;
        executedHarnessSha256=$receipt.harnessSha256; auditHarnessSha256=(Get-FileHash -LiteralPath $HarnessPath).Hash.ToLowerInvariant();
        executedAuditHelperSha256=$receipt.auditHelperSha256; executedProcessHelperSha256=$receipt.processHelperSha256;
        auditHelperSha256=$helperHash; processHelperSha256=(Get-FileHash -LiteralPath (Join-Path $PSScriptRoot 'adoption-process.ps1')).Hash.ToLowerInvariant();
        rawFiles=$rawHashes; ordinalCases=$expectedCount; nativeExecutions=0; normalizationMismatchRejected=$normalizationMismatchRejected}
    if (!$OutputPath) { $OutputPath=Join-Path $runRoot ('ordinal-audit-'+[Guid]::NewGuid().ToString('N')+'.json') }
    # Append a new result: never overwrite an earlier audit or native receipt.
    $bytes = [Text.UTF8Encoding]::new($false).GetBytes(($audit | ConvertTo-Json -Depth 5) + "`n")
    $stream = [IO.File]::Open($OutputPath, [IO.FileMode]::CreateNew)
    try { $stream.Write($bytes, 0, $bytes.Length) } finally { $stream.Dispose() }
    if ($PassThru) { return [pscustomobject]$audit }
    Write-Output "PASS $Kind $expectedCount ordinal output/raw/input-or-output/EXE/original checks; no native executions; audit: $OutputPath"
}
