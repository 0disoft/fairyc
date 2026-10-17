package cli

import (
	"errors"
	"fmt"
	"io"

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
	catalogVersion string
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
	schema := atlasmap.LogicalSchema
	if options.catalogVersion == "1" {
		schema = atlasmap.LogicalSchemaV2
	}
	if options.catalogVersion == "2" {
		schema = atlasmap.LogicalSchemaV3
	}
	prepared, code := prepareAtlasWorkspaceSchema(sourceRoot, options.locked, stderr, schema)
	if code != ExitSuccess {
		return code
	}
	selected, found := selectAtlasPartition(prepared.partitions, options.packageUID)
	if !found {
		return invocationError(stderr, fmt.Sprintf("workspace has no package UID %q", options.packageUID))
	}
	join := atlasmap.NewWorkspaceCallIndex
	encode := workspacecallcatalog.Encode
	if options.catalogVersion == "1" {
		join, encode = atlasmap.NewWorkspaceCallIndexV2, workspacecallcatalog.EncodeV1
	} else if options.catalogVersion == "2" {
		join, encode = atlasmap.NewWorkspaceCallIndexV3, workspacecallcatalog.EncodeV2
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
	_, err := parseCommandFlags(arguments, flagSchema{Command: "atlas-calls list", Experimental: true, Flags: []flagDefinition{
		{Name: "--catalog-version", Kind: flagScalar},
		{Name: "--package-uid", Kind: flagScalar},
		{Name: "--symbol", Kind: flagScalar},
		{Name: "--direction", Kind: flagScalar},
		{Name: "--max-packages", Kind: flagScalar},
		{Name: "--max-calls", Kind: flagScalar},
		{Name: "--max-output-bytes", Kind: flagScalar},
		{Name: "--locked", Kind: flagSwitch},
	}, Required: []string{"--package-uid", "--symbol", "--direction", "--max-packages", "--max-calls", "--max-output-bytes"}}, func(flag, value string) error {
		switch flag {
		case "--locked":
			options.locked = true
		case "--catalog-version":
			if value != "0" && value != "1" && value != "2" {
				return fmt.Errorf("experimental command \"atlas-calls list\" requires --catalog-version as 0, 1 or 2")
			}
			options.catalogVersion = value
		case "--package-uid":
			if err := validateAtlasPackageUID("atlas-calls list", value); err != nil {
				return err
			}
			options.packageUID = value
		case "--symbol":
			if _, err := symbolidentity.ParseVisibleID(value); err != nil {
				if _, fullErr := symbolidentity.ParseFullDigest(value); fullErr != nil {
					return fmt.Errorf("experimental command \"atlas-calls list\" requires --symbol as a canonical visible or full SymbolId")
				}
			}
			options.symbol = value
		case "--direction":
			options.direction = atlasmap.QueryDirection(value)
			if options.direction != atlasmap.DirectionOutgoing && options.direction != atlasmap.DirectionIncoming && options.direction != atlasmap.DirectionBoth {
				return fmt.Errorf("experimental command \"atlas-calls list\" does not support direction %q", value)
			}
		case "--max-packages":
			parsed, err := parseAtlasPositiveLimit("atlas-calls list", flag, value, maxAtlasPackRecords)
			if err != nil {
				return err
			}
			options.maxPackages = int(parsed)
		case "--max-calls":
			parsed, err := parseAtlasPositiveLimit("atlas-calls list", flag, value, maxAtlasPackRecords)
			if err != nil {
				return err
			}
			options.maxCalls = int(parsed)
		case "--max-output-bytes":
			parsed, err := parseAtlasPositiveLimit("atlas-calls list", flag, value, maxAtlasPackBytes)
			if err != nil {
				return err
			}
			options.maxOutputBytes = parsed
		}
		return nil
	})
	if err != nil {
		return atlasCallsOptions{}, err
	}

	return options, nil
}
