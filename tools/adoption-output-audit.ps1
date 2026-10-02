# Shared by the opt-in ledger/log/pack adoption harnesses; dot-sourcing starts no process.
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
function Invoke-AdoptionOrdinalAudit([string]$ReceiptPath, [ValidateSet('ledger','log','pack')][string]$Kind, [string]$Fairyc, [string]$HarnessPath) {
    $repository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
    $receiptPath = (Resolve-Path -LiteralPath $ReceiptPath).Path
    $cacheRoot = [IO.Path]::GetFullPath((Join-Path $repository ".cache/$Kind-compare-adoption")) + [IO.Path]::DirectorySeparatorChar
    if (!$receiptPath.StartsWith($cacheRoot, [StringComparison]::OrdinalIgnoreCase) -or [IO.Path]::GetFileName($receiptPath) -cne 'receipt.json') { throw 'Audit only this checkout matching adoption receipt' }
    $runRoot = Split-Path $receiptPath
    $receiptHash = (Get-FileHash -LiteralPath $receiptPath).Hash.ToLowerInvariant()
    $receipt = Get-Content -LiteralPath $receiptPath -Raw | ConvertFrom-Json
    $expectedCount = @{ledger=68; log=104; pack=128}[$Kind]
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
    $helperHash = (Get-FileHash -LiteralPath $PSCommandPath).Hash.ToLowerInvariant()
    if ($receipt.auditHelperSha256 -and $receipt.auditHelperSha256 -ne $helperHash) { throw 'Executed output helper changed' }
    $rawFiles = [Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    $paths = if ($Kind -eq 'pack') { @('legacy/evidence/raw-executions.jsonl','sdk/evidence/raw-executions.jsonl') } else { @('raw-executions.jsonl') }
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
    for ($caseIndex=0; $caseIndex -lt $receipt.cases.Count; $caseIndex++) {
        $entry = $receipt.cases[$caseIndex]
        $key = $entry.variant + '/' + $entry.profile
        if (!$imageIndex.ContainsKey($key) -or !$seen.Add($key+'/'+$entry.name)) { throw 'Unexpected or duplicate case' }
        $imageCounts[$key]++
        $raw = $programRecords[$caseIndex]
        if ($Kind -eq 'pack') {
            $pointerPath = [IO.Path]::GetFullPath($entry.rawEvidence.path)
            if (!$rawFiles.ContainsKey($pointerPath) -or $entry.rawEvidence.line -lt 1 -or $entry.rawEvidence.line -gt $rawFiles[$pointerPath].Count) { throw 'Unexpected raw evidence pointer' }
            $pointed = $rawFiles[$pointerPath][$entry.rawEvidence.line-1]
            if ($pointed.kind -cne 'program' -or ![StringComparer]::Ordinal.Equals(($pointed | ConvertTo-Json -Depth 8 -Compress), ($raw | ConvertTo-Json -Depth 8 -Compress))) { throw 'Raw pointer/order mismatch' }
            if (![StringComparer]::Ordinal.Equals($raw.cwd, $entry.cwd) -or $raw.argv.Count -ne $entry.argv.Count) { throw 'Raw cwd/argv mismatch' }
            for ($i=0; $i -lt $raw.argv.Count; $i++) { if (![StringComparer]::Ordinal.Equals($raw.argv[$i], $entry.argv[$i])) { throw 'Raw argv mismatch' } }
            foreach ($output in $entry.outputs) {
                if ((Get-FileHash -LiteralPath $output.path).Hash.ToLowerInvariant() -ne $output.actualSha256 -or $output.actualSha256 -ne $output.expectedSha256) { throw 'Retained oracle output changed' }
            }
        }
        if ($raw.kind -cne 'program' -or ![StringComparer]::Ordinal.Equals($raw.executable, $imageIndex[$key].executable) -or
            ![StringComparer]::Ordinal.Equals($raw.cwd, (Join-Path $runRoot $entry.variant))) { throw 'Raw executable/workspace mismatch' }
        foreach ($argument in $raw.argv) { if ($argument -isnot [string]) { throw 'Malformed raw argv' } }
        $rawActual = if ($Kind -eq 'pack') { $raw } else { $raw.actual }
        Assert-AdoptionObservation $rawActual $entry.actual "raw/$caseIndex"
        if ($Kind -ne 'pack' -and $entry.inputSha256Before) {
            $input = [IO.Path]::GetFullPath((Join-Path $raw.cwd $raw.argv[0]))
            if (!$input.StartsWith($raw.cwd+[IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase) -or
                (Get-FileHash -LiteralPath $input).Hash.ToLowerInvariant() -ne $entry.inputSha256Before -or $entry.inputSha256Before -ne $entry.inputSha256After) { throw 'Retained input changed' }
        }
    }
    foreach ($key in $imageIndex.Keys) { if ($imageCounts[$key] -ne $receipt.casesPerImage) { throw 'Per-image inventory mismatch' } }
    if ((Get-FileHash -LiteralPath $receiptPath).Hash.ToLowerInvariant() -ne $receiptHash) { throw 'Receipt changed during audit' }
    $rawHashes = @{}
    foreach ($path in $rawFiles.Keys) { $rawHashes[$path]=(Get-FileHash -LiteralPath $path).Hash.ToLowerInvariant() }
    $audit = @{schema='fairyc.adoption-ordinal-audit/v1'; kind=$Kind; receiptSha256=$receiptHash; executedHarnessSha256=$receipt.harnessSha256; auditHarnessSha256=(Get-FileHash -LiteralPath $HarnessPath).Hash.ToLowerInvariant(); auditHelperSha256=$helperHash; rawFiles=$rawHashes; ordinalCases=$expectedCount; nativeExecutions=0}
    [IO.File]::WriteAllText((Join-Path $runRoot 'ordinal-audit.json'), ($audit | ConvertTo-Json -Depth 5) + "`n", [Text.UTF8Encoding]::new($false))
    Write-Output "PASS $Kind $expectedCount ordinal output/raw/input-or-output/EXE/original checks; no native executions"
}
