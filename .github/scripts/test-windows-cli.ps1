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

# Estimates use the slower completed top-level result in Frontend runs
# 44349085023 and 44517150809, rounded up to 15s for tests taking at least 45s.
# Run 44517150809 timed out in shard 4; use the earlier timing for unreported tests.
$weights = @{
    TestCandidateAggregateCommandOrdinarySDK = 105
    TestCandidateAtlasAggregateCommand = 120
    TestCandidateAtlasAggregateInstances = 90
    TestCandidateAtlasAggregateProduction = 195
    TestCandidateAtlasAllocationSites = 195
    TestCandidateAtlasBorrowObservations = 75
    TestCandidateAtlasBorrowSites = 60
    TestCandidateAtlasCommand = 315
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
    TestCandidateAtlasSDKUserAggregateProduction = 480
    TestCandidateAtlasSDKVectorCommand = 405
    TestCandidateAtlasSDKVectorProduction = 510
    TestCandidateAtlasSchema = 75
    TestCandidateAtlasSourceVersionPart1 = 645
    TestCandidateAtlasSourceVersionPart2 = 390
    TestCandidateAtlasSourceVersionPart3 = 375
    TestCandidateAtlasSourceVersionPart4 = 555
    TestCandidateAtlasStaticOperationCommand = 90
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
    TestCandidateClosureUnitCacheNative = 180
    TestCandidateClosureUnitRestoreNative = 75
    TestCandidateCopiedClosureCalls = 60
    TestCandidateEditAggregateCommand = 105
    TestCandidateEditCommand = 150
    TestCandidateEditCopiedClosureCommand = 90
    TestCandidateEditOwnerQualifiedCommand = 90
    TestCandidateEditPreview = 90
    TestCandidateEditRecheck = 135
    TestCandidateEditSDKCommand = 240
    TestCandidateEditSDKUserAggregateCommand = 300
    TestCandidateEditSDKVectorCommand = 240
    TestCandidateEditStaticOperationCommand = 75
    TestCandidateGenericAggregateArtifactRestore = 105
    TestCandidateGenericAggregateArtifactTransport = 75
    TestCandidateGenericAggregateBodyMIR = 60
    TestCandidateGenericAggregateC = 75
    TestCandidateGenericAggregateDiskCache = 225
    TestCandidateGenericAggregateIdentities = 75
    TestCandidateGenericAggregatePartitionC = 120
    TestCandidateGenericAggregateProjectC = 135
    TestCandidateGenericAggregatePublicInterfaces = 75
    TestCandidateGenericAggregateUnitCache = 210
    TestCandidateImplicitReceiverExecution = 75
    TestCandidateImplicitReceiverPersistence = 315
    TestCandidateMCPAggregateCommand = 75
    TestCandidateMCPCommand = 105
    TestCandidateMCPCopiedClosureCommand = 75
    TestCandidateMCPOwnerQualifiedCommand = 120
    TestCandidateMCPSDKCommand = 210
    TestCandidateMCPSDKUserAggregateCommand = 225
    TestCandidateMCPSDKVectorCommand = 120
    TestCandidateMCPStaticOperationCommand = 75
    TestCandidateOrdinaryASTReuse = 90
    TestCandidateOrdinaryProjectLink = 60
    TestCandidateOwnerQualifiedComplete = 60
    TestCandidatePackAround = 285
    TestCandidatePackShow = 75
    TestCandidateProjectMetadataMap = 60
    TestCandidateRegionBytesExecution = 240
    TestCandidateStateCommandConsoleCancellation = 60
    TestCandidateStateGraphCacheKey = 75
    TestCandidateStateGraphCacheLookup = 60
    TestCandidateStateGraphRequest = 60
    TestCandidateStateGraphResponse = 105
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
