package cli

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/contextpack"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

const maxAtlasPackDepth = 64

type atlasPackRelationOptions struct {
	packageUID        string
	symbol            string
	direction         atlasmap.QueryDirection
	relations         []atlasmap.RelationKind
	maxRecords        int
	maxRelations      int
	maxDepth          int
	maxSelectionBytes uint64
	maxOutputBytes    uint64
	locked            bool
	packVersion       string
}

func runAtlasPackRelations(query string, arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseAtlasPackRelationOptions(query, arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	prepared, code := prepareAtlasQuerySchema(sourceRoot, options.packageUID, options.locked, stderr, atlasPackSchema(options.packVersion))
	if code != ExitSuccess {
		return code
	}
	request := atlasmap.RelationRequest{
		Direction: options.direction, Relations: options.relations,
		Budget: atlasmap.QueryBudget{
			MaxRecords: options.maxRecords, MaxRelations: options.maxRelations,
			MaxDepth: options.maxDepth, MaxSelectionBytes: options.maxSelectionBytes,
		},
	}
	var encoded []byte
	show, around := contextpack.EncodeShow, contextpack.EncodeAround
	if options.packVersion == "2" {
		show, around = contextpack.EncodeShowV2, contextpack.EncodeAroundV2
	} else if options.packVersion == "3" {
		show, around = contextpack.EncodeShowV3, contextpack.EncodeAroundV3
	}
	if query == "show" {
		encoded, err = show(prepared.index, options.symbol, request, options.maxOutputBytes)
	} else {
		encoded, err = around(prepared.index, options.symbol, request, options.maxOutputBytes)
	}
	if err != nil {
		if errors.Is(err, atlasmap.ErrSymbolNotFound) || errors.Is(err, atlasmap.ErrQueryBudgetTooSmall) ||
			errors.Is(err, contextpack.ErrPackTooLarge) {
			return invocationError(stderr, err.Error())
		}
		return internalError(stderr)
	}
	if _, err := stdout.Write(encoded); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseAtlasPackRelationOptions(query string, arguments []string) (atlasPackRelationOptions, error) {
	return parseAtlasRelationOptions("atlas-pack "+query, query, arguments)
}

func parseAtlasRelationOptions(command, query string, arguments []string) (atlasPackRelationOptions, error) {
	options := atlasPackRelationOptions{}
	required := []string{"--package-uid", "--symbol", "--direction", "--max-records", "--max-relations", "--max-selection-bytes", "--max-output-bytes"}
	if query == "around" {
		required = append(required, "--max-depth")
	}
	flags := []flagDefinition{
		{Name: "--package-uid", Kind: flagScalar},
		{Name: "--symbol", Kind: flagScalar},
		{Name: "--direction", Kind: flagScalar},
		{Name: "--max-records", Kind: flagScalar},
		{Name: "--max-relations", Kind: flagScalar},
		{Name: "--max-selection-bytes", Kind: flagScalar},
		{Name: "--max-output-bytes", Kind: flagScalar},
		{Name: "--relation", Kind: flagSlice, DuplicateValue: fmt.Sprintf("experimental command %q repeats relation %%q", command)},
		{Name: "--locked", Kind: flagSwitch},
	}
	if query == "around" {
		flags = append(flags, flagDefinition{Name: "--max-depth", Kind: flagScalar})
	}
	if command == "atlas-pack show" || command == "atlas-pack around" {
		flags = append(flags, flagDefinition{Name: "--pack-version", Kind: flagScalar})
	}
	_, err := parseCommandFlags(arguments, flagSchema{Command: command, Experimental: true, Flags: flags, Required: required}, func(flag, value string) error {
		switch flag {
		case "--locked":
			options.locked = true
		case "--pack-version":
			if value != "1" && value != "2" && value != "3" {
				return fmt.Errorf("experimental command %q requires --pack-version as 1, 2 or 3", command)
			}
			options.packVersion = value
		case "--package-uid":
			if err := validateAtlasPackageUID(command, value); err != nil {
				return err
			}
			options.packageUID = value
		case "--symbol":
			if err := validateAtlasSymbol(value); err != nil {
				return err
			}
			options.symbol = value
		case "--direction":
			options.direction = atlasmap.QueryDirection(value)
			if !slices.Contains([]atlasmap.QueryDirection{atlasmap.DirectionOutgoing, atlasmap.DirectionIncoming, atlasmap.DirectionBoth}, options.direction) {
				return fmt.Errorf("experimental command %q does not support direction %q", command, value)
			}
		case "--relation":
			relation := atlasmap.RelationKind(value)
			if !slices.Contains(atlasmap.AllRelationKinds(), relation) {
				return fmt.Errorf("experimental command %q does not support relation %q", command, value)
			}
			options.relations = append(options.relations, relation)
		case "--max-records", "--max-relations", "--max-selection-bytes", "--max-output-bytes":
			maximum := maxAtlasPackBytes
			if flag == "--max-records" || flag == "--max-relations" {
				maximum = maxAtlasPackRecords
			}
			parsed, err := parseAtlasPositiveLimit(command, flag, value, maximum)
			if err != nil {
				return err
			}
			switch flag {
			case "--max-records":
				options.maxRecords = int(parsed)
			case "--max-relations":
				options.maxRelations = int(parsed)
			case "--max-selection-bytes":
				options.maxSelectionBytes = parsed
			case "--max-output-bytes":
				options.maxOutputBytes = parsed
			}
		case "--max-depth":
			parsed, err := parseAtlasNonnegativeLimit(command, flag, value, maxAtlasPackDepth)
			if err != nil {
				return err
			}
			options.maxDepth = int(parsed)
		}
		return nil
	})
	if err != nil {
		return atlasPackRelationOptions{}, err
	}
	if err := validateFlagRules(flagRule{Invalid: len(options.relations) == 0, Message: fmt.Sprintf("experimental command %q requires at least one --relation", command)}); err != nil {
		return atlasPackRelationOptions{}, err
	}
	slices.Sort(options.relations)
	return options, nil
}

func validateAtlasSymbol(value string) error {
	if strings.HasPrefix(value, symbolidentity.VisiblePrefix) {
		if _, err := symbolidentity.ParseVisibleID(value); err == nil {
			return nil
		}
	}
	if strings.HasPrefix(value, symbolidentity.DigestPrefix) {
		if _, err := symbolidentity.ParseFullDigest(value); err == nil {
			return nil
		}
	}
	return fmt.Errorf("experimental Atlas relation query requires --symbol as a canonical visible or full SymbolId")
}

func atlasPackSchema(version string) string {
	switch version {
	case "2":
		return atlasmap.LogicalSchemaV2
	case "3":
		return atlasmap.LogicalSchemaV3
	default:
		return atlasmap.LogicalSchema
	}
}
