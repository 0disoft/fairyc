package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/symbolcatalog"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

type atlasSymbolsOptions struct {
	packageUID        string
	module            string
	name              string
	kinds             []semanticsnapshot.DeclarationKind
	after             string
	maxSymbols        int
	maxSelectionBytes uint64
	maxOutputBytes    uint64
	locked            bool
	catalogVersion    string
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
	schema := atlasmap.LogicalSchema
	if options.catalogVersion == "2" {
		schema = atlasmap.LogicalSchemaV2
	}
	if options.catalogVersion == "3" {
		schema = atlasmap.LogicalSchemaV3
	}
	prepared, code := prepareAtlasQuerySchema(sourceRoot, options.packageUID, options.locked, stderr, schema)
	if code != ExitSuccess {
		return code
	}
	encode := symbolcatalog.Encode
	if options.catalogVersion == "2" {
		encode = symbolcatalog.EncodeV2
	} else if options.catalogVersion == "3" {
		encode = symbolcatalog.EncodeV3
	}
	encoded, err := encode(prepared.index, atlasmap.SymbolCatalogRequest{
		Module: options.module, Name: options.name, Kinds: options.kinds, After: options.after,
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
	_, err := parseCommandFlags(arguments, flagSchema{Command: "atlas-symbols list", Experimental: true, Flags: []flagDefinition{
		{Name: "--catalog-version", Kind: flagScalar},
		{Name: "--package-uid", Kind: flagScalar},
		{Name: "--module", Kind: flagScalar},
		{Name: "--name", Kind: flagScalar},
		{Name: "--after", Kind: flagScalar},
		{Name: "--max-symbols", Kind: flagScalar},
		{Name: "--max-selection-bytes", Kind: flagScalar},
		{Name: "--max-output-bytes", Kind: flagScalar},
		{Name: "--kind", Kind: flagSlice, DuplicateValue: `experimental command "atlas-symbols list" repeats declaration kind %q`},
		{Name: "--locked", Kind: flagSwitch},
	}, Required: []string{"--package-uid", "--max-symbols", "--max-selection-bytes", "--max-output-bytes"}}, func(flag, value string) error {
		switch flag {
		case "--locked":
			options.locked = true
		case "--catalog-version":
			if value != "1" && value != "2" && value != "3" {
				return fmt.Errorf("experimental command \"atlas-symbols list\" requires --catalog-version as 1, 2 or 3")
			}
			options.catalogVersion = value
		case "--package-uid":
			if err := validateAtlasPackageUID("atlas-symbols list", value); err != nil {
				return err
			}
			options.packageUID = value
		case "--module":
			if value == "" || len(value) > 512 {
				return fmt.Errorf("experimental command \"atlas-symbols list\" requires --module as 1 through 512 bytes")
			}
			options.module = value
		case "--name":
			if len(value) > 512 || !source.ValidIdentifier(value) {
				return fmt.Errorf("experimental command \"atlas-symbols list\" requires --name as an exact Stage 0 identifier of at most 512 bytes")
			}
			options.name = value
		case "--kind":
			kind := semanticsnapshot.DeclarationKind(value)
			if !slices.Contains(atlasmap.AllDeclarationKinds(), kind) {
				return fmt.Errorf("experimental command \"atlas-symbols list\" does not support declaration kind %q", value)
			}
			options.kinds = append(options.kinds, kind)
		case "--after":
			if _, err := symbolidentity.ParseFullDigest(value); err != nil {
				return fmt.Errorf("experimental command \"atlas-symbols list\" requires --after as a canonical full SymbolId")
			}
			options.after = value
		case "--max-symbols":
			parsed, err := parseAtlasPositiveLimit("atlas-symbols list", flag, value, maxAtlasPackRecords)
			if err != nil {
				return err
			}
			options.maxSymbols = int(parsed)
		case "--max-selection-bytes":
			parsed, err := parseAtlasPositiveLimit("atlas-symbols list", flag, value, maxAtlasPackBytes)
			if err != nil {
				return err
			}
			options.maxSelectionBytes = parsed
		case "--max-output-bytes":
			parsed, err := parseAtlasPositiveLimit("atlas-symbols list", flag, value, maxAtlasPackBytes)
			if err != nil {
				return err
			}
			options.maxOutputBytes = parsed
		}
		return nil
	})
	if err != nil {
		return atlasSymbolsOptions{}, err
	}

	if err := validateFlagRules(flagRule{Invalid: len(options.kinds) == 0, Message: fmt.Sprintf("experimental command \"atlas-symbols list\" requires at least one --kind")}); err != nil {
		return atlasSymbolsOptions{}, err
	}
	slices.Sort(options.kinds)
	return options, nil
}
