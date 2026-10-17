package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/attributedtext"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

type atlasTextOptions struct {
	packageUID       string
	symbol           string
	origin           attributedtext.Origin
	maxFragments     int
	maxFragmentBytes uint64
	maxTextBytes     uint64
	maxOutputBytes   uint64
	locked           bool
}

func runAtlasText(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		return invocationError(stderr, `experimental command "atlas-text" requires "source" or "documentation"`)
	}
	action := arguments[0]
	if action != "source" && action != "documentation" {
		return invocationError(stderr, fmt.Sprintf("experimental command \"atlas-text\" does not support action %q", action))
	}
	options, err := parseAtlasTextOptions(action, arguments[1:])
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	prepared, code := prepareAtlasQuerySchema(sourceRoot, options.packageUID, options.locked, stderr, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		return code
	}
	kind := attributedtext.KindSourceExcerpt
	if action == "documentation" {
		kind = attributedtext.KindDeclarationDocumentation
	}
	request := attributedtext.Request{
		SymbolID: options.symbol,
		Kinds:    []attributedtext.Kind{kind},
		Origins:  []attributedtext.Origin{options.origin},
		Budget: attributedtext.Budget{
			MaxFragments: options.maxFragments, MaxFragmentBytes: options.maxFragmentBytes,
			MaxTextBytes: options.maxTextBytes,
		},
	}
	var encoded []byte
	if action == "source" {
		encoded, err = attributedtext.EncodeWorkspaceSource(
			prepared.workspace, prepared.index, request, options.maxOutputBytes,
		)
	} else {
		encoded, err = attributedtext.EncodeWorkspaceDocumentation(
			prepared.workspace, prepared.index, prepared.snapshot, request, options.maxOutputBytes,
		)
	}
	if err != nil {
		if errors.Is(err, atlasmap.ErrSymbolNotFound) || errors.Is(err, attributedtext.ErrDocumentTooLarge) {
			return invocationError(stderr, err.Error())
		}
		return internalError(stderr)
	}
	if _, err := stdout.Write(encoded); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseAtlasTextOptions(action string, arguments []string) (atlasTextOptions, error) {
	command := "atlas-text " + action
	options := atlasTextOptions{}
	_, err := parseCommandFlags(arguments, flagSchema{Command: command, Experimental: true, Flags: []flagDefinition{
		{Name: "--package-uid", Kind: flagScalar},
		{Name: "--symbol", Kind: flagScalar},
		{Name: "--origin", Kind: flagScalar},
		{Name: "--max-fragments", Kind: flagScalar},
		{Name: "--max-fragment-bytes", Kind: flagScalar},
		{Name: "--max-text-bytes", Kind: flagScalar},
		{Name: "--max-output-bytes", Kind: flagScalar},
		{Name: "--locked", Kind: flagSwitch},
	}, Required: []string{
		"--package-uid", "--symbol", "--origin", "--max-fragments",
		"--max-fragment-bytes", "--max-text-bytes", "--max-output-bytes",
	}}, func(flag, value string) error {
		switch flag {
		case "--locked":
			options.locked = true
		case "--package-uid":
			if err := validateAtlasPackageUID(command, value); err != nil {
				return err
			}
			options.packageUID = value
		case "--symbol":
			if err := validateAtlasTextSymbol(command, value); err != nil {
				return err
			}
			options.symbol = value
		case "--origin":
			options.origin = attributedtext.Origin(value)
			if options.origin != attributedtext.OriginWorkspaceAuthored {
				return fmt.Errorf("experimental command %q supports only --origin workspace-authored", command)
			}
		case "--max-fragments":
			parsed, err := parseAtlasPositiveLimit(command, flag, value, attributedtext.MaxFragmentsGuard)
			if err != nil {
				return err
			}
			options.maxFragments = int(parsed)
		case "--max-fragment-bytes":
			parsed, err := parseAtlasPositiveLimit(command, flag, value, attributedtext.MaxFragmentBytesGuard)
			if err != nil {
				return err
			}
			options.maxFragmentBytes = parsed
		case "--max-text-bytes":
			parsed, err := parseAtlasPositiveLimit(command, flag, value, attributedtext.MaxTextBytesGuard)
			if err != nil {
				return err
			}
			options.maxTextBytes = parsed
		case "--max-output-bytes":
			parsed, err := parseAtlasPositiveLimit(command, flag, value, maxAtlasPackBytes)
			if err != nil {
				return err
			}
			options.maxOutputBytes = parsed
		}
		return nil
	})
	if err != nil {
		return atlasTextOptions{}, err
	}

	if err := validateFlagRules(flagRule{Invalid: options.maxFragmentBytes > options.maxTextBytes, Message: fmt.Sprintf("experimental command %q requires --max-fragment-bytes no greater than --max-text-bytes", command)}); err != nil {
		return atlasTextOptions{}, err
	}
	return options, nil
}

func validateAtlasTextSymbol(command, value string) error {
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
	return fmt.Errorf("experimental command %q requires --symbol as a canonical visible or full SymbolId", command)
}
