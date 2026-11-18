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

# Estimates in seconds, rounded up from Frontend run 44296743202 for tests
# taking at least 45s. New Atlas source version parts use the same estimate
# until their individual timings are available.
$weights = @{
    TestCandidateAggregateCommandOrdinarySDK = 60
    TestCandidateAtlasAggregateCommand = 60
    TestCandidateAtlasAggregateInstances = 75
    TestCandidateAtlasAggregateProduction = 75
    TestCandidateAtlasAllocationSites = 285
    TestCandidateAtlasBorrowObservations = 60
    TestCandidateAtlasCommand = 105
    TestCandidateAtlasCopiedClosureCommand = 150
    TestCandidateAtlasCopiedClosureProduction = 75
    TestCandidateAtlasDropSites = 225
    TestCandidateAtlasEnumOwnerProduction = 75
    TestCandidateAtlasGenericCommand = 135
    TestCandidateAtlasGenericFunctionSchema = 60
    TestCandidateAtlasGenericProduction = 105
    TestCandidateAtlasMoveSites = 315
    TestCandidateAtlasOwnerQualifiedCommand = 300
    TestCandidateAtlasOwnerQualifiedProduction = 60
    TestCandidateAtlasProduction = 60
    TestCandidateAtlasReturnErrorSites = 75
    TestCandidateAtlasSchema = 60
    TestCandidateAtlasSDKCommand = 330
    TestCandidateAtlasSDKProduction = 75
    TestCandidateAtlasSDKUserAggregateCommand = 165
    TestCandidateAtlasSDKUserAggregateProduction = 390
    TestCandidateAtlasSDKVectorCommand = 405
    TestCandidateAtlasSDKVectorProduction = 270
    TestCandidateAtlasSourceVersionPart1 = 720
    TestCandidateAtlasSourceVersionPart2 = 720
    TestCandidateAtlasSourceVersionPart3 = 720
    TestCandidateAtlasSourceVersionPart4 = 720
    TestCandidateAtlasStaticOperationCommand = 60
    TestCandidateAtlasStaticOperationProduction = 60
    TestCandidateAtlasTraps = 165
    TestCandidateAtlasUnsafeSites = 375
    TestCandidateCleanupRecipeIndex = 75
    TestCandidateClosureCleanupLowerNative = 60
    TestCandidateClosureMixedCacheNative = 105
    TestCandidateClosureProgramArtifactNative = 90
    TestCandidateClosureProjectCacheNative = 120
    TestCandidateClosureProjectLinkNative = 75
    TestCandidateClosureSourceRecipesNative = 75
    TestCandidateClosureUnitArtifactNative = 90
    TestCandidateClosureUnitCacheNative = 90
    TestCandidateClosureUnitRestoreNative = 60
    TestCandidateCopiedClosureCalls = 150
    TestCandidateEditAggregateCommand = 60
    TestCandidateEditCommand = 150
    TestCandidateEditOwnerQualifiedCommand = 75
    TestCandidateEditPreview = 90
    TestCandidateEditSDKCommand = 90
    TestCandidateEditSDKUserAggregateCommand = 105
    TestCandidateEditSDKVectorCommand = 135
    TestCandidateEditStaticOperationCommand = 105
    TestCandidateGenericAggregateArtifactTransport = 60
    TestCandidateGenericAggregateBodyMIR = 60
    TestCandidateGenericAggregateDiskCache = 135
    TestCandidateGenericAggregateProjectC = 90
    TestCandidateGenericAggregatePublicInterfaces = 60
    TestCandidateGenericAggregateUnitCache = 150
    TestCandidateImplicitReceiverPersistence = 210
    TestCandidateMCPAggregateCommand = 60
    TestCandidateMCPOwnerQualifiedCommand = 90
    TestCandidateMCPSDKCommand = 75
    TestCandidateMCPSDKUserAggregateCommand = 90
    TestCandidateMCPSDKVectorCommand = 120
    TestCandidateRegionBytesExecution = 90
    TestCandidateStateGraphResponse = 60
    TestLinuxAtlasEditCommand = 60
    TestLinuxAtlasImpactCommand = 60
    TestLinuxAtlasMCPCommand = 75
    TestLinuxAtlasPackCommand = 90
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
