#requires -Version 7.0
param(
    [string]$LedgerReceipt = (Join-Path $PSScriptRoot '../.cache/ledger-compare-adoption/2b8950a036a746b59de93bf02d842969/receipt.json'),
    [string]$LogReceipt = (Join-Path $PSScriptRoot '../.cache/log-compare-adoption/fda5f6e7f8d44122938d893b870e0416/receipt.json'),
    [string]$PackReceipt = (Join-Path $PSScriptRoot '../.cache/pack-compare-adoption/59910b3e53ea41e494f4b4a667255393/receipt.json'),
    [string]$KvReceipt = (Join-Path $PSScriptRoot '../.cache/kv-compare-adoption/f4fbe4e3d2694913905bbcbcc726171e/receipt.json'),
    [string]$Fairyc = (Join-Path $PSScriptRoot '../.cache/ledger-compare-toolchain/fairyc.exe')
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'adoption-output-audit.ps1')
& (Join-Path $PSScriptRoot 'test-adoption-output-audit.ps1')
& (Join-Path $PSScriptRoot 'test-adoption-process.ps1')
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot ('../.cache/adoption-audit/'+[Guid]::NewGuid().ToString('N'))))
[void][IO.Directory]::CreateDirectory($root)
$receipts = [ordered]@{ledger=$LedgerReceipt; log=$LogReceipt; pack=$PackReceipt; kv=$KvReceipt}
$audits = [Collections.Generic.List[object]]::new()
$priorAudits = @{}
foreach ($kind in $receipts.Keys) {
    $prior=Join-Path (Split-Path $receipts[$kind]) 'ordinal-audit.json'
    if (Test-Path -LiteralPath $prior) { $priorAudits[$prior]=(Get-FileHash -LiteralPath $prior).Hash.ToLowerInvariant() }
    $harness=Join-Path $PSScriptRoot "verify-$kind-compare-adoption.ps1"
    $audit=Invoke-AdoptionOrdinalAudit $receipts[$kind] $kind $Fairyc $harness -OutputPath (Join-Path $root ($kind+'.json')) -PassThru
    $audits.Add($audit)
    Write-Output "PASS $kind $($audit.ordinalCases) retained cases"
}
foreach ($path in $priorAudits.Keys) {
    if ((Get-FileHash -LiteralPath $path).Hash.ToLowerInvariant() -ne $priorAudits[$path]) { throw 'Previous audit changed' }
}
$total=($audits | Measure-Object -Property ordinalCases -Sum).Sum
if ($total -ne 628 -or @($audits | Where-Object nativeExecutions -ne 0).Count) { throw 'Consolidated inventory mismatch' }
$scripts=@{}
foreach ($name in @('audit-adoption-evidence','adoption-output-audit','adoption-process','test-adoption-output-audit','test-adoption-process')) {
    $scripts[$name]=(Get-FileHash -LiteralPath (Join-Path $PSScriptRoot ($name+'.ps1'))).Hash.ToLowerInvariant()
}
$summary=@{schema='fairyc.adoption-evidence-audit/v1'; ordinalCases=[int]$total; nativeExecutions=0; helperHostExecutions=3;
    outputGateCases=10; processBoundaryCases=5; scripts=$scripts; preservedPreviousAudits=$priorAudits; audits=$audits}
[IO.File]::WriteAllText((Join-Path $root 'summary.json'), ($summary | ConvertTo-Json -Depth 8)+"`n", [Text.UTF8Encoding]::new($false))
Write-Output "PASS 628 retained cases; zero Fairy/compiler executions; summary: $(Join-Path $root 'summary.json')"
