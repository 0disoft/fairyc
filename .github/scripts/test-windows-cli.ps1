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

# Estimates use the slowest completed top-level result in Frontend runs
# 44349085023, 44517150809 and 44525960360, rounded up to 15s for tests
# taking at least 45s. Use earlier samples for tests missing from timed-out shards.
$weights = @{
    TestCandidateAggregateCommandOrdinarySDK = 105
    TestCandidateAtlasAggregateCommand = 120
    TestCandidateAtlasAggregateInstances = 90
    TestCandidateAtlasAggregateProduction = 195
    TestCandidateAtlasAllocationSites = 195
    TestCandidateAtlasBorrowObservations = 90
    TestCandidateAtlasBorrowSites = 90
    TestCandidateAtlasCommand = 330
    TestCandidateAtlasCopiedClosureCommand = 150
    TestCandidateAtlasCopiedClosureProduction = 90
    TestCandidateAtlasDropSites = 210
    TestCandidateAtlasEnumOwnerCommand = 90
    TestCandidateAtlasEnumOwnerProduction = 195
    TestCandidateAtlasGenericCommand = 105
    TestCandidateAtlasGenericFunctionSchema = 60
    TestCandidateAtlasGenericProduction = 105
    TestCandidateAtlasMoveSites = 315
    TestCandidateAtlasOwnerQualifiedCommand = 300
    TestCandidateAtlasOwnerQualifiedNamespaceProduction = 90
    TestCandidateAtlasOwnerQualifiedProduction = 150
    TestCandidateAtlasProduction = 75
    TestCandidateAtlasReturnErrorSites = 225
    TestCandidateAtlasSDKCommand = 330
    TestCandidateAtlasSDKProduction = 165
    TestCandidateAtlasSDKUserAggregateCommand = 345
    TestCandidateAtlasSDKUserAggregateProduction = 750
    TestCandidateAtlasSDKVectorCommand = 405
    TestCandidateAtlasSDKVectorProduction = 510
    TestCandidateAtlasSchema = 75
    TestCandidateAtlasSourceVersionPart1 = 645
    TestCandidateAtlasSourceVersionPart2 = 390
    TestCandidateAtlasSourceVersionPart3 = 765
    TestCandidateAtlasSourceVersionPart4 = 555
    TestCandidateAtlasStaticOperationCommand = 105
    TestCandidateAtlasStaticOperationProduction = 105
    TestCandidateAtlasTraps = 315
    TestCandidateAtlasTypes = 60
    TestCandidateAtlasUnsafeSites = 360
    TestCandidateCleanupRecipeFields = 75
    TestCandidateCleanupRecipeIndex = 60
    TestCandidateClosureCleanupLowerNative = 75
    TestCandidateClosureMixedCacheNative = 210
    TestCandidateClosureProgramArtifactNative = 60
    TestCandidateClosureProjectCacheNative = 225
    TestCandidateClosureProjectLinkNative = 75
    TestCandidateClosureSourceRecipesNative = 90
    TestCandidateClosureUnitArtifactNative = 75
    TestCandidateClosureUnitBodyNative = 60
    TestCandidateClosureUnitCNative = 60
    TestCandidateClosureUnitCacheNative = 180
    TestCandidateClosureUnitRestoreNative = 75
    TestCandidateCopiedClosureCalls = 60
    TestCandidateEditAggregateCommand = 105
    TestCandidateEditCommand = 270
    TestCandidateEditCopiedClosureCommand = 90
    TestCandidateEditOwnerQualifiedCommand = 90
    TestCandidateEditPreview = 150
    TestCandidateEditRecheck = 135
    TestCandidateEditSDKCommand = 240
    TestCandidateEditSDKUserAggregateCommand = 300
    TestCandidateEditSDKVectorCommand = 240
    TestCandidateEditStaticOperationCommand = 90
    TestCandidateGenericAggregateArtifactRestore = 105
    TestCandidateGenericAggregateArtifactTransport = 90
    TestCandidateGenericAggregateBodyMIR = 60
    TestCandidateGenericAggregateC = 75
    TestCandidateGenericAggregateDiskCache = 225
    TestCandidateGenericAggregateIdentities = 75
    TestCandidateGenericAggregatePartitionC = 120
    TestCandidateGenericAggregateProjectC = 135
    TestCandidateGenericAggregatePublicInterfaces = 75
    TestCandidateGenericAggregateUnitCache = 225
    TestCandidateImplicitReceiverExecution = 75
    TestCandidateImplicitReceiverPersistence = 315
    TestCandidateMCPAggregateCommand = 90
    TestCandidateMCPCommand = 105
    TestCandidateMCPCopiedClosureCommand = 75
    TestCandidateMCPOwnerQualifiedCommand = 120
    TestCandidateMCPSDKCommand = 210
    TestCandidateMCPSDKUserAggregateCommand = 225
    TestCandidateMCPSDKVectorCommand = 120
    TestCandidateMCPStaticOperationCommand = 90
    TestCandidateOrdinaryASTReuse = 90
    TestCandidateOrdinaryProjectLink = 60
    TestCandidateOwnerQualifiedComplete = 60
    TestCandidatePackAround = 285
    TestCandidatePackShow = 75
    TestCandidateProjectMetadataMap = 75
    TestCandidateRegionBytesExecution = 240
    TestCandidateStateCommandConsoleCancellation = 60
    TestCandidateStateGraphCacheKey = 75
    TestCandidateStateGraphCacheLookup = 60
    TestCandidateStateGraphRequest = 60
    TestCandidateStateGraphResponse = 120
    TestCandidateStateGraphWindowsCancellationBoundary = 60
    TestLinuxAtlasCallsCommand = 180
    TestLinuxAtlasEditCommand = 90
    TestLinuxAtlasImpactCommand = 90
    TestLinuxAtlasMCPCommand = 75
    TestLinuxAtlasPackCommand = 150
    TestLinuxAtlasSymbolsCommand = 270
}
$estimatedSeconds = [int[]]::new($ShardCount)
$ordered = $entries | Sort-Object -Stable -Descending -Property { $weights[$_] ?? 3 }
$assigned = @(foreach ($entry in $ordered) {
    $lightest = 0
    for ($index = 1; $index -lt $ShardCount; $index++) {
        if ($estimatedSeconds[$index] -lt $estimatedSeconds[$lightest]) { $lightest = $index }
    }
    $estimatedSeconds[$lightest] += $weights[$entry] ?? 3
    if ($lightest -eq $Shard) { $entry }
})
if ($assigned.Count -eq 0) { throw 'No tests assigned to the Windows CLI shard' }

$selection = '^(' + (($assigned | ForEach-Object { [regex]::Escape($_) }) -join '|') + ')$'
Write-Host "Windows Go CLI shard $Shard/$ShardCount runs $($assigned.Count) of $($entries.Count) entries (estimated $($estimatedSeconds[$Shard])s)"
go test -mod=readonly -count=1 -timeout 45m -v -run $selection ./internal/cli
exit $LASTEXITCODE
