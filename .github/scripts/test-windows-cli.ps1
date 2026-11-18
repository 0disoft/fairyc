param(
    [int]$Shard,
    [ValidateRange(1, 16)]
    [int]$ShardCount
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
$assigned = @(for ($index = $Shard; $index -lt $entries.Count; $index += $ShardCount) { $entries[$index] })
if ($assigned.Count -eq 0) { throw 'No tests assigned to the Windows CLI shard' }

$selection = '^(' + (($assigned | ForEach-Object { [regex]::Escape($_) }) -join '|') + ')$'
Write-Host "Windows Go CLI shard $Shard/$ShardCount runs $($assigned.Count) of $($entries.Count) entries"
go test -mod=readonly -count=1 -timeout 25m -v -run $selection ./internal/cli
exit $LASTEXITCODE
