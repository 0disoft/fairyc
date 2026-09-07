package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/symbolidentity"
	"github.com/0disoft/fairyc/internal/workspacecallcatalog"
)

type atlasCallsOptions struct {
	packageUID     string
	symbol         string
	direction      atlasmap.QueryDirection
	maxPackages    int
	maxCalls       int
	maxOutputBytes uint64
	locked         bool
	catalogV1      bool
}

func runAtlasCalls(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		return invocationError(stderr, `experimental command "atlas-calls" requires "list"`)
	}
	if arguments[0] != "list" {
		return invocationError(stderr, fmt.Sprintf("experimental command \"atlas-calls\" does not support action %q", arguments[0]))
	}
	options, err := parseAtlasCallsOptions(arguments[1:])
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	prepared, code := prepareAtlasWorkspaceVersion(sourceRoot, options.locked, stderr, options.catalogV1)
	if code != ExitSuccess {
		return code
	}
	selected, found := selectAtlasPartition(prepared.partitions, options.packageUID)
	if !found {
		return invocationError(stderr, fmt.Sprintf("workspace has no package UID %q", options.packageUID))
	}
	join := atlasmap.NewWorkspaceCallIndex
	encode := workspacecallcatalog.Encode
	if options.catalogV1 {
		join, encode = atlasmap.NewWorkspaceCallIndexV2, workspacecallcatalog.EncodeV1
	}
	index, err := join(
		prepared.partitions,
		prepared.expectations,
		atlasmap.WorkspaceCallBudget{MaxPackages: options.maxPackages, MaxCalls: options.maxCalls},
	)
	if err != nil {
		if errors.Is(err, atlasmap.ErrWorkspaceJoinBudget) {
			return invocationError(stderr, err.Error())
		}
		return internalError(stderr)
	}
	encoded, err := encode(
		index, selected.Header().PackageInstance, options.symbol, options.direction, options.maxOutputBytes,
	)
	if err != nil {
		if errors.Is(err, atlasmap.ErrSymbolNotFound) || errors.Is(err, workspacecallcatalog.ErrCatalogTooLarge) {
			return invocationError(stderr, err.Error())
		}
		return internalError(stderr)
	}
	if _, err := stdout.Write(encoded); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseAtlasCallsOptions(arguments []string) (atlasCallsOptions, error) {
	options := atlasCallsOptions{}
	seen := make(map[string]bool, 7)
	for position := 0; position < len(arguments); {
		flag := arguments[position]
		if flag == "--locked" {
			if seen[flag] {
				return atlasCallsOptions{}, fmt.Errorf("experimental command \"atlas-calls list\" received %s more than once", flag)
			}
			seen[flag], options.locked = true, true
			position++
			continue
		}
		switch flag {
		case "--catalog-version", "--package-uid", "--symbol", "--direction", "--max-packages", "--max-calls", "--max-output-bytes":
		default:
			return atlasCallsOptions{}, fmt.Errorf("experimental command \"atlas-calls list\" does not accept flag %q", flag)
		}
		if seen[flag] {
			return atlasCallsOptions{}, fmt.Errorf("experimental command \"atlas-calls list\" received %s more than once", flag)
		}
		if position+1 >= len(arguments) || strings.HasPrefix(arguments[position+1], "--") {
			return atlasCallsOptions{}, fmt.Errorf("experimental command \"atlas-calls list\" requires a value for %s", flag)
		}
		seen[flag] = true
		value := arguments[position+1]
		switch flag {
		case "--catalog-version":
			if value != "0" && value != "1" {
				return atlasCallsOptions{}, fmt.Errorf("experimental command \"atlas-calls list\" requires --catalog-version as 0 or 1")
			}
			options.catalogV1 = value == "1"
		case "--package-uid":
			if err := validateAtlasPackageUID("atlas-calls list", value); err != nil {
				return atlasCallsOptions{}, err
			}
			options.packageUID = value
		case "--symbol":
			if _, err := symbolidentity.ParseVisibleID(value); err != nil {
				if _, fullErr := symbolidentity.ParseFullDigest(value); fullErr != nil {
					return atlasCallsOptions{}, fmt.Errorf("experimental command \"atlas-calls list\" requires --symbol as a canonical visible or full SymbolId")
				}
			}
			options.symbol = value
		case "--direction":
			options.direction = atlasmap.QueryDirection(value)
			if options.direction != atlasmap.DirectionOutgoing && options.direction != atlasmap.DirectionIncoming && options.direction != atlasmap.DirectionBoth {
				return atlasCallsOptions{}, fmt.Errorf("experimental command \"atlas-calls list\" does not support direction %q", value)
			}
		case "--max-packages":
			parsed, err := parseAtlasPositiveLimit("atlas-calls list", flag, value, maxAtlasPackRecords)
			if err != nil {
				return atlasCallsOptions{}, err
			}
			options.maxPackages = int(parsed)
		case "--max-calls":
			parsed, err := parseAtlasPositiveLimit("atlas-calls list", flag, value, maxAtlasPackRecords)
			if err != nil {
				return atlasCallsOptions{}, err
			}
			options.maxCalls = int(parsed)
		case "--max-output-bytes":
			parsed, err := parseAtlasPositiveLimit("atlas-calls list", flag, value, maxAtlasPackBytes)
			if err != nil {
				return atlasCallsOptions{}, err
			}
			options.maxOutputBytes = parsed
		}
		position += 2
	}
	for _, flag := range []string{"--package-uid", "--symbol", "--direction", "--max-packages", "--max-calls", "--max-output-bytes"} {
		if !seen[flag] {
			return atlasCallsOptions{}, fmt.Errorf("experimental command \"atlas-calls list\" requires %s", flag)
		}
	}
	return options, nil
}
