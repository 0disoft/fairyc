package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/attributedtext"
	"github.com/0disoft/fairyc/internal/contextpack"
	"github.com/0disoft/fairyc/internal/evidencebundle"
)

type atlasEvidenceOptions struct {
	generic              bool
	relation             atlasPackRelationOptions
	maxFragmentBytes     uint64
	maxTextBytes         uint64
	maxFactsBytes        uint64
	maxTextDocumentBytes uint64
}

func runAtlasEvidence(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		return invocationError(stderr, `experimental command "atlas-evidence" requires "show"`)
	}
	if arguments[0] != "show" {
		return invocationError(stderr, fmt.Sprintf("experimental command \"atlas-evidence\" does not support action %q", arguments[0]))
	}
	options, err := parseAtlasEvidenceOptions(arguments[1:])
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	prepared, code := prepareAtlasQueryVersion(
		sourceRoot, options.relation.packageUID, options.relation.locked, stderr, options.generic,
	)
	if code != ExitSuccess {
		return code
	}
	request := atlasmap.RelationRequest{
		Direction: options.relation.direction, Relations: options.relation.relations,
		Budget: atlasmap.QueryBudget{
			MaxRecords: options.relation.maxRecords, MaxRelations: options.relation.maxRelations,
			MaxDepth: 0, MaxSelectionBytes: options.relation.maxSelectionBytes,
		},
	}
	encode := evidencebundle.EncodeShow
	if options.generic {
		encode = evidencebundle.EncodeShowV1
	}
	encoded, err := encode(
		prepared.workspace, prepared.index, prepared.snapshot, options.relation.symbol,
		request,
		attributedtext.Budget{
			MaxFragments: 1, MaxFragmentBytes: options.maxFragmentBytes,
			MaxTextBytes: options.maxTextBytes,
		},
		evidencebundle.Limits{
			FactsBytes: options.maxFactsBytes, TextDocumentBytes: options.maxTextDocumentBytes,
			BundleBytes: options.relation.maxOutputBytes,
		},
	)
	if err != nil {
		if errors.Is(err, atlasmap.ErrSymbolNotFound) || errors.Is(err, atlasmap.ErrQueryBudgetTooSmall) ||
			errors.Is(err, contextpack.ErrPackTooLarge) || errors.Is(err, attributedtext.ErrDocumentTooLarge) ||
			errors.Is(err, evidencebundle.ErrBundleTooLarge) {
			return invocationError(stderr, err.Error())
		}
		return internalError(stderr)
	}
	if _, err := stdout.Write(encoded); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseAtlasEvidenceOptions(arguments []string) (atlasEvidenceOptions, error) {
	const command = "atlas-evidence show"
	result := atlasEvidenceOptions{}
	seen := make(map[string]bool, 4)
	relationArguments := make([]string, 0, len(arguments))
	for position := 0; position < len(arguments); {
		flag := arguments[position]
		if !isAtlasEvidenceLimit(flag) && flag != "--bundle-version" {
			relationArguments = append(relationArguments, flag)
			position++
			continue
		}
		if seen[flag] {
			return atlasEvidenceOptions{}, fmt.Errorf("experimental command %q received %s more than once", command, flag)
		}
		if position+1 >= len(arguments) || strings.HasPrefix(arguments[position+1], "--") {
			return atlasEvidenceOptions{}, fmt.Errorf("experimental command %q requires a value for %s", command, flag)
		}
		seen[flag] = true
		if flag == "--bundle-version" {
			version := arguments[position+1]
			if version != "0" && version != "1" {
				return atlasEvidenceOptions{}, fmt.Errorf("experimental command %q requires --bundle-version as 0 or 1", command)
			}
			result.generic = version == "1"
			position += 2
			continue
		}
		value, err := parseAtlasPositiveLimit(command, flag, arguments[position+1], maxAtlasPackBytes)
		if err != nil {
			return atlasEvidenceOptions{}, err
		}
		switch flag {
		case "--max-fragment-bytes":
			if value > attributedtext.MaxFragmentBytesGuard {
				return atlasEvidenceOptions{}, fmt.Errorf("experimental command %q requires %s as a canonical decimal from 1 through %d", command, flag, attributedtext.MaxFragmentBytesGuard)
			}
			result.maxFragmentBytes = value
		case "--max-text-bytes":
			if value > attributedtext.MaxTextBytesGuard {
				return atlasEvidenceOptions{}, fmt.Errorf("experimental command %q requires %s as a canonical decimal from 1 through %d", command, flag, attributedtext.MaxTextBytesGuard)
			}
			result.maxTextBytes = value
		case "--max-facts-bytes":
			result.maxFactsBytes = value
		case "--max-text-document-bytes":
			result.maxTextDocumentBytes = value
		}
		position += 2
	}
	for _, flag := range []string{
		"--max-fragment-bytes", "--max-text-bytes", "--max-facts-bytes", "--max-text-document-bytes",
	} {
		if !seen[flag] {
			return atlasEvidenceOptions{}, fmt.Errorf("experimental command %q requires %s", command, flag)
		}
	}
	if result.maxFragmentBytes > result.maxTextBytes {
		return atlasEvidenceOptions{}, fmt.Errorf("experimental command %q requires --max-fragment-bytes no greater than --max-text-bytes", command)
	}
	relation, err := parseAtlasRelationOptions(command, "show", relationArguments)
	if err != nil {
		return atlasEvidenceOptions{}, err
	}
	result.relation = relation
	return result, nil
}

func isAtlasEvidenceLimit(flag string) bool {
	return flag == "--max-fragment-bytes" || flag == "--max-text-bytes" ||
		flag == "--max-facts-bytes" || flag == "--max-text-document-bytes"
}
