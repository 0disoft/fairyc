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
	index, code := prepareAtlasQueryIndex(sourceRoot, options.packageUID, options.locked, stderr)
	if code != ExitSuccess {
		return code
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

func prepareAtlasQueryIndex(sourceRoot, packageUID string, locked bool, stderr io.Writer) (*atlasmap.QueryIndex, int) {
	workspace, _, err := loadWorkspaceWithLock(sourceRoot, locked)
	if err != nil {
		return nil, packageFailure(stderr, err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return nil, packageFailure(stderr, err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		return nil, internalError(stderr)
	}
	if len(result.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, result.Semantic.Diagnostics); err != nil {
			return nil, ExitInternalFailure
		}
		return nil, ExitSourceDiagnostics
	}
	if len(result.Semantic.Unsupported) != 0 {
		if err := writeUnsupportedSemantics(stderr, result.Semantic.Unsupported); err != nil {
			return nil, ExitInternalFailure
		}
		return nil, ExitSourceDiagnostics
	}
	if !result.Verified {
		return nil, internalError(stderr)
	}
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		return nil, internalError(stderr)
	}
	_, partitions, err := atlasmap.PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		return nil, internalError(stderr)
	}
	expectations, err := atlasmap.CurrentWorkspaceExpectations(workspace, result, compilerBuild)
	if err != nil {
		return nil, internalError(stderr)
	}
	var selected atlasmap.Partition
	found := false
	for _, partition := range partitions {
		if partition.Header().PackageUID == packageUID {
			selected, found = partition, true
			break
		}
	}
	if !found {
		return nil, invocationError(stderr, fmt.Sprintf("workspace has no package UID %q", packageUID))
	}
	expectation, found := expectations[selected.Header().PackageInstance]
	if !found {
		return nil, internalError(stderr)
	}
	index, err := atlasmap.NewQueryIndex(selected, expectation)
	if err != nil {
		return nil, internalError(stderr)
	}
	return index, ExitSuccess
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
