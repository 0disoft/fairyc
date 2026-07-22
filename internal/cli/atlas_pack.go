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
}

func runAtlasPack(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		return invocationError(stderr, `experimental command "atlas-pack" requires a query kind`)
	}
	if arguments[0] != "overview" {
		return invocationError(stderr, fmt.Sprintf("experimental command \"atlas-pack\" does not support query %q", arguments[0]))
	}
	options, err := parseAtlasPackOverviewOptions(arguments[1:])
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	workspace, _, err := loadWorkspaceWithLock(sourceRoot, options.locked)
	if err != nil {
		return packageFailure(stderr, err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return packageFailure(stderr, err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		return internalError(stderr)
	}
	if len(result.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, result.Semantic.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if len(result.Semantic.Unsupported) != 0 {
		if err := writeUnsupportedSemantics(stderr, result.Semantic.Unsupported); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if !result.Verified {
		return internalError(stderr)
	}
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		return internalError(stderr)
	}
	_, partitions, err := atlasmap.PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		return internalError(stderr)
	}
	expectations, err := atlasmap.CurrentWorkspaceExpectations(workspace, result, compilerBuild)
	if err != nil {
		return internalError(stderr)
	}
	var selected atlasmap.Partition
	found := false
	for _, partition := range partitions {
		if partition.Header().PackageUID == options.packageUID {
			selected, found = partition, true
			break
		}
	}
	if !found {
		return invocationError(stderr, fmt.Sprintf("workspace has no package UID %q", options.packageUID))
	}
	expectation, found := expectations[selected.Header().PackageInstance]
	if !found {
		return internalError(stderr)
	}
	index, err := atlasmap.NewQueryIndex(selected, expectation)
	if err != nil {
		return internalError(stderr)
	}
	encoded, err := contextpack.EncodeOverview(index, atlasmap.QueryBudget{
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
		case "--package-uid", "--max-records", "--max-selection-bytes", "--max-output-bytes":
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
		case "--package-uid":
			if err := validateAtlasPackageUID(value); err != nil {
				return atlasPackOverviewOptions{}, err
			}
			options.packageUID = value
		case "--max-records":
			parsed, err := parseAtlasPackLimit(flag, value, maxAtlasPackRecords)
			if err != nil {
				return atlasPackOverviewOptions{}, err
			}
			options.maxRecords = int(parsed)
		case "--max-selection-bytes":
			parsed, err := parseAtlasPackLimit(flag, value, maxAtlasPackBytes)
			if err != nil {
				return atlasPackOverviewOptions{}, err
			}
			options.maxSelectionBytes = parsed
		case "--max-output-bytes":
			parsed, err := parseAtlasPackLimit(flag, value, maxAtlasPackBytes)
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

func parseAtlasPackLimit(flag, value string, maximum uint64) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 || parsed > maximum || value != strconv.FormatUint(parsed, 10) {
		return 0, fmt.Errorf("experimental command \"atlas-pack overview\" requires %s as a canonical decimal from 1 through %d", flag, maximum)
	}
	return parsed, nil
}

func validateAtlasPackageUID(value string) error {
	if len(value) > 256 || !projectmanifest.ValidPackageUID(value) {
		return fmt.Errorf("experimental command \"atlas-pack overview\" requires --package-uid as at most 256 bytes in manifest-v1 lowercase dotted form")
	}
	return nil
}
