param(
    [int]$Shard,
    [ValidateRange(1, 16)]
    [int]$ShardCount,
    [switch]$PlanOnly
)

if ($Shard -lt 0 -or $Shard -ge $ShardCount) { throw 'Invalid Windows CLI shard' }

$skip = @(
    '^TestWindowsFairy(Lexer|Parser[A-Z])',
    '^TestTokenIndexNominalTotalMSVC$',
    '^TestTokenIndexNominalCostMSVC$',
    '^TestTokenIndexStructCostMSVC$',
    '^TestStableGenericWorkspaceGrowthMSVC$',
    '^TestResultBranchReviewIsOkNativeSizes$',
    '^TestScannerFieldShorthandNativeSizes$',
    '^TestArgumentLabelsParserNativeSizes$',
    '^TestByteViewRealLexerWorkloadComparison$'
) -join '|'

$listed = @(go test -mod=readonly -list '^(Test|Example|Fuzz)' ./internal/cli)
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
[string[]]$entries = @($listed | Where-Object { $_ -cmatch '^(Test|Example|Fuzz)[\p{L}\p{N}_]*$' -and $_ -cnotmatch $skip })
[Array]::Sort($entries, [StringComparer]::Ordinal)

$timings = Get-Content -Raw -LiteralPath (Join-Path $PSScriptRoot 'windows-cli-timings.json') |
    ConvertFrom-Json -AsHashtable
$weights = $timings.elapsedCentiseconds
$estimatedCentiseconds = [long[]]::new($ShardCount)
$observed = 0
$ordered = $entries | Sort-Object -Stable -Descending -Property { [Math]::Max(1, ($weights[$_] ?? 300)) }
$placements = @(foreach ($entry in $ordered) {
    $lightest = 0
    for ($index = 1; $index -lt $ShardCount; $index++) {
        if ($estimatedCentiseconds[$index] -lt $estimatedCentiseconds[$lightest]) { $lightest = $index }
    }
    $weight = $weights[$entry]
    if ($null -ne $weight) { $observed++ }
    $estimatedCentiseconds[$lightest] += [Math]::Max(1, ($weight ?? 300))
    [pscustomobject]@{ entry = $entry; shard = $lightest }
})
if ($PlanOnly) {
    [pscustomobject]@{
        entries = $entries
        placements = $placements
        estimatedCentiseconds = $estimatedCentiseconds
        observed = $observed
    } | ConvertTo-Json -Depth 4
    return
}

$assigned = @($placements | Where-Object shard -EQ $Shard | Select-Object -ExpandProperty entry)
if ($assigned.Count -eq 0) { throw 'No tests assigned to the Windows CLI shard' }

$selection = '^(' + (($assigned | ForEach-Object { [regex]::Escape($_) }) -join '|') + ')$'
Write-Host "Windows Go CLI shard $Shard/$ShardCount runs $($assigned.Count) of $($entries.Count) entries (estimated $($estimatedCentiseconds[$Shard] / 100)s, timing run $($timings.runId))"
go test -mod=readonly -count=1 -timeout 45m -v -run $selection ./internal/cli
exit $LASTEXITCODE
