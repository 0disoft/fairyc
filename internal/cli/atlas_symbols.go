package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolcatalog"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

type atlasSymbolsOptions struct {
	packageUID        string
	module            string
	kinds             []semanticsnapshot.DeclarationKind
	after             string
	maxSymbols        int
	maxSelectionBytes uint64
	maxOutputBytes    uint64
	locked            bool
}

func runAtlasSymbols(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		return invocationError(stderr, `experimental command "atlas-symbols" requires "list"`)
	}
	if arguments[0] != "list" {
		return invocationError(stderr, fmt.Sprintf("experimental command \"atlas-symbols\" does not support action %q", arguments[0]))
	}
	options, err := parseAtlasSymbolsOptions(arguments[1:])
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	index, code := prepareAtlasQueryIndex(sourceRoot, options.packageUID, options.locked, stderr)
	if code != ExitSuccess {
		return code
	}
	encoded, err := symbolcatalog.Encode(index, atlasmap.SymbolCatalogRequest{
		Module: options.module, Kinds: options.kinds, After: options.after,
		Budget: atlasmap.SymbolCatalogBudget{
			MaxSymbols: options.maxSymbols, MaxSelectionBytes: options.maxSelectionBytes,
		},
	}, options.maxOutputBytes)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	if _, err := stdout.Write(encoded); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseAtlasSymbolsOptions(arguments []string) (atlasSymbolsOptions, error) {
	options := atlasSymbolsOptions{}
	seen := make(map[string]bool, 8)
	for position := 0; position < len(arguments); {
		flag := arguments[position]
		if flag == "--locked" {
			if seen[flag] {
				return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" received %s more than once", flag)
			}
			seen[flag], options.locked = true, true
			position++
			continue
		}
		switch flag {
		case "--package-uid", "--module", "--kind", "--after", "--max-symbols", "--max-selection-bytes", "--max-output-bytes":
		default:
			return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" does not accept flag %q", flag)
		}
		if flag != "--kind" && seen[flag] {
			return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" received %s more than once", flag)
		}
		if position+1 >= len(arguments) || strings.HasPrefix(arguments[position+1], "--") {
			return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" requires a value for %s", flag)
		}
		seen[flag] = true
		value := arguments[position+1]
		switch flag {
		case "--package-uid":
			if err := validateAtlasPackageUID(value); err != nil {
				return atlasSymbolsOptions{}, err
			}
			options.packageUID = value
		case "--module":
			if value == "" || len(value) > 512 {
				return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" requires --module as 1 through 512 bytes")
			}
			options.module = value
		case "--kind":
			kind := semanticsnapshot.DeclarationKind(value)
			if !slices.Contains(atlasmap.AllDeclarationKinds(), kind) {
				return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" does not support declaration kind %q", value)
			}
			if slices.Contains(options.kinds, kind) {
				return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" repeats declaration kind %q", value)
			}
			options.kinds = append(options.kinds, kind)
		case "--after":
			if _, err := symbolidentity.ParseFullDigest(value); err != nil {
				return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" requires --after as a canonical full SymbolId")
			}
			options.after = value
		case "--max-symbols":
			parsed, err := parseAtlasPositiveLimit("atlas-symbols list", flag, value, maxAtlasPackRecords)
			if err != nil {
				return atlasSymbolsOptions{}, err
			}
			options.maxSymbols = int(parsed)
		case "--max-selection-bytes":
			parsed, err := parseAtlasPositiveLimit("atlas-symbols list", flag, value, maxAtlasPackBytes)
			if err != nil {
				return atlasSymbolsOptions{}, err
			}
			options.maxSelectionBytes = parsed
		case "--max-output-bytes":
			parsed, err := parseAtlasPositiveLimit("atlas-symbols list", flag, value, maxAtlasPackBytes)
			if err != nil {
				return atlasSymbolsOptions{}, err
			}
			options.maxOutputBytes = parsed
		}
		position += 2
	}
	for _, flag := range []string{"--package-uid", "--max-symbols", "--max-selection-bytes", "--max-output-bytes"} {
		if !seen[flag] {
			return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" requires %s", flag)
		}
	}
	if len(options.kinds) == 0 {
		return atlasSymbolsOptions{}, fmt.Errorf("experimental command \"atlas-symbols list\" requires at least one --kind")
	}
	slices.Sort(options.kinds)
	return options, nil
}
