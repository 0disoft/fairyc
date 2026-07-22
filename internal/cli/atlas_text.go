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

type atlasTextSourceOptions struct {
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
		return invocationError(stderr, `experimental command "atlas-text" requires "source"`)
	}
	if arguments[0] != "source" {
		return invocationError(stderr, fmt.Sprintf("experimental command \"atlas-text\" does not support action %q", arguments[0]))
	}
	options, err := parseAtlasTextSourceOptions(arguments[1:])
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	workspace, index, code := prepareAtlasQuery(sourceRoot, options.packageUID, options.locked, stderr)
	if code != ExitSuccess {
		return code
	}
	encoded, err := attributedtext.EncodeWorkspaceSource(workspace, index, attributedtext.Request{
		SymbolID: options.symbol,
		Kinds:    []attributedtext.Kind{attributedtext.KindSourceExcerpt},
		Origins:  []attributedtext.Origin{options.origin},
		Budget: attributedtext.Budget{
			MaxFragments: options.maxFragments, MaxFragmentBytes: options.maxFragmentBytes,
			MaxTextBytes: options.maxTextBytes,
		},
	}, options.maxOutputBytes)
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

func parseAtlasTextSourceOptions(arguments []string) (atlasTextSourceOptions, error) {
	const command = "atlas-text source"
	options := atlasTextSourceOptions{}
	seen := make(map[string]bool, 8)
	for position := 0; position < len(arguments); {
		flag := arguments[position]
		if flag == "--locked" {
			if seen[flag] {
				return atlasTextSourceOptions{}, fmt.Errorf("experimental command %q received %s more than once", command, flag)
			}
			seen[flag], options.locked = true, true
			position++
			continue
		}
		switch flag {
		case "--package-uid", "--symbol", "--origin", "--max-fragments", "--max-fragment-bytes", "--max-text-bytes", "--max-output-bytes":
		default:
			return atlasTextSourceOptions{}, fmt.Errorf("experimental command %q does not accept flag %q", command, flag)
		}
		if seen[flag] {
			return atlasTextSourceOptions{}, fmt.Errorf("experimental command %q received %s more than once", command, flag)
		}
		if position+1 >= len(arguments) || strings.HasPrefix(arguments[position+1], "--") {
			return atlasTextSourceOptions{}, fmt.Errorf("experimental command %q requires a value for %s", command, flag)
		}
		seen[flag] = true
		value := arguments[position+1]
		switch flag {
		case "--package-uid":
			if err := validateAtlasPackageUID(command, value); err != nil {
				return atlasTextSourceOptions{}, err
			}
			options.packageUID = value
		case "--symbol":
			if err := validateAtlasTextSymbol(value); err != nil {
				return atlasTextSourceOptions{}, err
			}
			options.symbol = value
		case "--origin":
			options.origin = attributedtext.Origin(value)
			if options.origin != attributedtext.OriginWorkspaceAuthored {
				return atlasTextSourceOptions{}, fmt.Errorf("experimental command %q supports only --origin workspace-authored", command)
			}
		case "--max-fragments":
			parsed, err := parseAtlasPositiveLimit(command, flag, value, attributedtext.MaxFragmentsGuard)
			if err != nil {
				return atlasTextSourceOptions{}, err
			}
			options.maxFragments = int(parsed)
		case "--max-fragment-bytes":
			parsed, err := parseAtlasPositiveLimit(command, flag, value, attributedtext.MaxFragmentBytesGuard)
			if err != nil {
				return atlasTextSourceOptions{}, err
			}
			options.maxFragmentBytes = parsed
		case "--max-text-bytes":
			parsed, err := parseAtlasPositiveLimit(command, flag, value, attributedtext.MaxTextBytesGuard)
			if err != nil {
				return atlasTextSourceOptions{}, err
			}
			options.maxTextBytes = parsed
		case "--max-output-bytes":
			parsed, err := parseAtlasPositiveLimit(command, flag, value, maxAtlasPackBytes)
			if err != nil {
				return atlasTextSourceOptions{}, err
			}
			options.maxOutputBytes = parsed
		}
		position += 2
	}
	for _, flag := range []string{
		"--package-uid", "--symbol", "--origin", "--max-fragments",
		"--max-fragment-bytes", "--max-text-bytes", "--max-output-bytes",
	} {
		if !seen[flag] {
			return atlasTextSourceOptions{}, fmt.Errorf("experimental command %q requires %s", command, flag)
		}
	}
	if options.maxFragmentBytes > options.maxTextBytes {
		return atlasTextSourceOptions{}, fmt.Errorf("experimental command %q requires --max-fragment-bytes no greater than --max-text-bytes", command)
	}
	return options, nil
}

func validateAtlasTextSymbol(value string) error {
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
	return fmt.Errorf("experimental command \"atlas-text source\" requires --symbol as a canonical visible or full SymbolId")
}
