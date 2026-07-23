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
}

func runAtlasPackRelations(query string, arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseAtlasPackRelationOptions(query, arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	index, code := prepareAtlasQueryIndex(sourceRoot, options.packageUID, options.locked, stderr)
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
	if query == "show" {
		encoded, err = contextpack.EncodeShow(index, options.symbol, request, options.maxOutputBytes)
	} else {
		encoded, err = contextpack.EncodeAround(index, options.symbol, request, options.maxOutputBytes)
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
	seen := make(map[string]bool, 10)
	for position := 0; position < len(arguments); {
		flag := arguments[position]
		if flag == "--locked" {
			if seen[flag] {
				return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q received %s more than once", command, flag)
			}
			seen[flag], options.locked = true, true
			position++
			continue
		}
		allowed := flag == "--package-uid" || flag == "--symbol" || flag == "--direction" ||
			flag == "--relation" || flag == "--max-records" || flag == "--max-relations" ||
			flag == "--max-selection-bytes" || flag == "--max-output-bytes" || query == "around" && flag == "--max-depth"
		if !allowed {
			return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q does not accept flag %q", command, flag)
		}
		if flag != "--relation" && seen[flag] {
			return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q received %s more than once", command, flag)
		}
		if position+1 >= len(arguments) || strings.HasPrefix(arguments[position+1], "--") {
			return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q requires a value for %s", command, flag)
		}
		seen[flag] = true
		value := arguments[position+1]
		switch flag {
		case "--package-uid":
			if err := validateAtlasPackageUID(command, value); err != nil {
				return atlasPackRelationOptions{}, err
			}
			options.packageUID = value
		case "--symbol":
			if err := validateAtlasSymbol(value); err != nil {
				return atlasPackRelationOptions{}, err
			}
			options.symbol = value
		case "--direction":
			options.direction = atlasmap.QueryDirection(value)
			if !slices.Contains([]atlasmap.QueryDirection{atlasmap.DirectionOutgoing, atlasmap.DirectionIncoming, atlasmap.DirectionBoth}, options.direction) {
				return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q does not support direction %q", command, value)
			}
		case "--relation":
			relation := atlasmap.RelationKind(value)
			if !slices.Contains(atlasmap.AllRelationKinds(), relation) {
				return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q does not support relation %q", command, value)
			}
			if slices.Contains(options.relations, relation) {
				return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q repeats relation %q", command, value)
			}
			options.relations = append(options.relations, relation)
		case "--max-records", "--max-relations", "--max-selection-bytes", "--max-output-bytes":
			maximum := maxAtlasPackBytes
			if flag == "--max-records" || flag == "--max-relations" {
				maximum = maxAtlasPackRecords
			}
			parsed, err := parseAtlasPositiveLimit(command, flag, value, maximum)
			if err != nil {
				return atlasPackRelationOptions{}, err
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
				return atlasPackRelationOptions{}, err
			}
			options.maxDepth = int(parsed)
		}
		position += 2
	}
	required := []string{"--package-uid", "--symbol", "--direction", "--max-records", "--max-relations", "--max-selection-bytes", "--max-output-bytes"}
	if query == "around" {
		required = append(required, "--max-depth")
	}
	for _, flag := range required {
		if !seen[flag] {
			return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q requires %s", command, flag)
		}
	}
	if len(options.relations) == 0 {
		return atlasPackRelationOptions{}, fmt.Errorf("experimental command %q requires at least one --relation", command)
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
