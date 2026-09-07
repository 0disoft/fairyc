package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contextpack"
	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	maxAtlasPackRecords = 65_536
	maxAtlasPackBytes   = uint64(1 << 30)
)

type atlasPackOverviewOptions struct {
	packageUID        string
	maxRecords        int
	maxSelectionBytes uint64
	maxOutputBytes    uint64
	locked            bool
	packV2            bool
}

func runAtlasPack(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		return invocationError(stderr, `experimental command "atlas-pack" requires a query kind`)
	}
	switch arguments[0] {
	case "overview":
		return runAtlasPackOverview(arguments[1:], sourceRoot, stdout, stderr)
	case "show", "around":
		return runAtlasPackRelations(arguments[0], arguments[1:], sourceRoot, stdout, stderr)
	default:
		return invocationError(stderr, fmt.Sprintf("experimental command \"atlas-pack\" does not support query %q", arguments[0]))
	}
}

func runAtlasPackOverview(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseAtlasPackOverviewOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	prepared, code := prepareAtlasQueryVersion(sourceRoot, options.packageUID, options.locked, stderr, options.packV2)
	if code != ExitSuccess {
		return code
	}
	encode := contextpack.EncodeOverview
	if options.packV2 {
		encode = contextpack.EncodeOverviewV2
	}
	encoded, err := encode(prepared.index, atlasmap.QueryBudget{
		MaxRecords: options.maxRecords, MaxRelations: 0, MaxDepth: 0,
		MaxSelectionBytes: options.maxSelectionBytes,
	}, options.maxOutputBytes)
	if err != nil {
		if errors.Is(err, atlasmap.ErrQueryBudgetTooSmall) || errors.Is(err, contextpack.ErrPackTooLarge) {
			return invocationError(stderr, err.Error())
		}
		return internalError(stderr)
	}
	if _, err := stdout.Write(encoded); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func prepareAtlasQueryIndex(sourceRoot, packageUID string, locked bool, stderr io.Writer) (*atlasmap.QueryIndex, int) {
	prepared, code := prepareAtlasQuery(sourceRoot, packageUID, locked, stderr)
	return prepared.index, code
}

type preparedAtlasQuery struct {
	workspace workspaceloader.Workspace
	index     *atlasmap.QueryIndex
	snapshot  semanticsnapshot.Snapshot
}

type preparedAtlasWorkspace struct {
	workspace    workspaceloader.Workspace
	snapshot     semanticsnapshot.Snapshot
	partitions   []atlasmap.Partition
	expectations map[string]atlasmap.LoadExpectation
}

func prepareAtlasQuery(
	sourceRoot, packageUID string,
	locked bool,
	stderr io.Writer,
) (preparedAtlasQuery, int) {
	return prepareAtlasQueryVersion(sourceRoot, packageUID, locked, stderr, false)
}

func prepareAtlasQueryVersion(sourceRoot, packageUID string, locked bool, stderr io.Writer, generic bool) (preparedAtlasQuery, int) {
	prepared, code := prepareAtlasWorkspaceVersion(sourceRoot, locked, stderr, generic)
	if code != ExitSuccess {
		return preparedAtlasQuery{}, code
	}
	selected, found := selectAtlasPartition(prepared.partitions, packageUID)
	if !found {
		return preparedAtlasQuery{}, invocationError(stderr, fmt.Sprintf("workspace has no package UID %q", packageUID))
	}
	expectation, found := prepared.expectations[selected.Header().PackageInstance]
	if !found {
		return preparedAtlasQuery{}, internalError(stderr)
	}
	query := atlasmap.NewQueryIndex
	if generic {
		query = atlasmap.NewQueryIndexV2
	}
	index, err := query(selected, expectation)
	if err != nil {
		return preparedAtlasQuery{}, internalError(stderr)
	}
	return preparedAtlasQuery{workspace: prepared.workspace, index: index, snapshot: prepared.snapshot}, ExitSuccess
}

func prepareAtlasWorkspace(
	sourceRoot string,
	locked bool,
	stderr io.Writer,
) (preparedAtlasWorkspace, int) {
	return prepareAtlasWorkspaceVersion(sourceRoot, locked, stderr, false)
}

func prepareAtlasWorkspaceVersion(sourceRoot string, locked bool, stderr io.Writer, generic bool) (preparedAtlasWorkspace, int) {
	workspace, _, err := loadWorkspaceWithLock(sourceRoot, locked)
	if err != nil {
		return preparedAtlasWorkspace{}, packageFailure(stderr, err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return preparedAtlasWorkspace{}, packageFailure(stderr, err)
	}
	var result compiler.Result
	if generic {
		result, err = workspacecompiler.Check(workspace)
	} else {
		result, err = compiler.CheckPackages(input)
	}
	if err != nil {
		return preparedAtlasWorkspace{}, internalError(stderr)
	}
	if len(result.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, result.Semantic.Diagnostics); err != nil {
			return preparedAtlasWorkspace{}, ExitInternalFailure
		}
		return preparedAtlasWorkspace{}, ExitSourceDiagnostics
	}
	if len(result.Semantic.Unsupported) != 0 {
		if err := writeUnsupportedSemantics(stderr, result.Semantic.Unsupported); err != nil {
			return preparedAtlasWorkspace{}, ExitInternalFailure
		}
		return preparedAtlasWorkspace{}, ExitSourceDiagnostics
	}
	if !result.Verified {
		return preparedAtlasWorkspace{}, internalError(stderr)
	}
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		return preparedAtlasWorkspace{}, internalError(stderr)
	}
	prepare, current := atlasmap.PrepareWorkspace, atlasmap.CurrentWorkspaceExpectations
	if generic {
		prepare, current = atlasmap.PrepareWorkspaceV2, atlasmap.CurrentWorkspaceExpectationsV2
	}
	snapshot, partitions, err := prepare(workspace, result, compilerBuild)
	if err != nil {
		return preparedAtlasWorkspace{}, internalError(stderr)
	}
	expectations, err := current(workspace, result, compilerBuild)
	if err != nil {
		return preparedAtlasWorkspace{}, internalError(stderr)
	}
	return preparedAtlasWorkspace{
		workspace: workspace, snapshot: snapshot, partitions: partitions, expectations: expectations,
	}, ExitSuccess
}

func selectAtlasPartition(partitions []atlasmap.Partition, packageUID string) (atlasmap.Partition, bool) {
	for _, partition := range partitions {
		if partition.Header().PackageUID == packageUID {
			return partition, true
		}
	}
	return atlasmap.Partition{}, false
}

func parseAtlasPackOverviewOptions(arguments []string) (atlasPackOverviewOptions, error) {
	options := atlasPackOverviewOptions{}
	seen := make(map[string]bool, 5)
	for index := 0; index < len(arguments); {
		flag := arguments[index]
		if flag == "--locked" {
			if seen[flag] {
				return atlasPackOverviewOptions{}, fmt.Errorf("experimental command \"atlas-pack overview\" received %s more than once", flag)
			}
			seen[flag], options.locked = true, true
			index++
			continue
		}
		switch flag {
		case "--pack-version", "--package-uid", "--max-records", "--max-selection-bytes", "--max-output-bytes":
		default:
			return atlasPackOverviewOptions{}, fmt.Errorf("experimental command \"atlas-pack overview\" does not accept flag %q", flag)
		}
		if seen[flag] {
			return atlasPackOverviewOptions{}, fmt.Errorf("experimental command \"atlas-pack overview\" received %s more than once", flag)
		}
		if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
			return atlasPackOverviewOptions{}, fmt.Errorf("experimental command \"atlas-pack overview\" requires a value for %s", flag)
		}
		seen[flag] = true
		value := arguments[index+1]
		switch flag {
		case "--pack-version":
			if value != "1" && value != "2" {
				return atlasPackOverviewOptions{}, fmt.Errorf("experimental command \"atlas-pack overview\" requires --pack-version as 1 or 2")
			}
			options.packV2 = value == "2"
		case "--package-uid":
			if err := validateAtlasPackageUID("atlas-pack overview", value); err != nil {
				return atlasPackOverviewOptions{}, err
			}
			options.packageUID = value
		case "--max-records":
			parsed, err := parseAtlasPositiveLimit("atlas-pack overview", flag, value, maxAtlasPackRecords)
			if err != nil {
				return atlasPackOverviewOptions{}, err
			}
			options.maxRecords = int(parsed)
		case "--max-selection-bytes":
			parsed, err := parseAtlasPositiveLimit("atlas-pack overview", flag, value, maxAtlasPackBytes)
			if err != nil {
				return atlasPackOverviewOptions{}, err
			}
			options.maxSelectionBytes = parsed
		case "--max-output-bytes":
			parsed, err := parseAtlasPositiveLimit("atlas-pack overview", flag, value, maxAtlasPackBytes)
			if err != nil {
				return atlasPackOverviewOptions{}, err
			}
			options.maxOutputBytes = parsed
		}
		index += 2
	}
	for _, flag := range []string{"--package-uid", "--max-records", "--max-selection-bytes", "--max-output-bytes"} {
		if !seen[flag] {
			return atlasPackOverviewOptions{}, fmt.Errorf("experimental command \"atlas-pack overview\" requires %s", flag)
		}
	}
	return options, nil
}

func parseAtlasPositiveLimit(command, flag, value string, maximum uint64) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 || parsed > maximum || value != strconv.FormatUint(parsed, 10) {
		return 0, fmt.Errorf("experimental command %q requires %s as a canonical decimal from 1 through %d", command, flag, maximum)
	}
	return parsed, nil
}

func parseAtlasNonnegativeLimit(command, flag, value string, maximum uint64) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed > maximum || value != strconv.FormatUint(parsed, 10) {
		return 0, fmt.Errorf("experimental command %q requires %s as a canonical decimal from 0 through %d", command, flag, maximum)
	}
	return parsed, nil
}

func validateAtlasPackageUID(command, value string) error {
	if len(value) > 256 || !projectmanifest.ValidPackageUID(value) {
		return fmt.Errorf("experimental command %q requires --package-uid as at most 256 bytes in manifest-v1 lowercase dotted form", command)
	}
	return nil
}
