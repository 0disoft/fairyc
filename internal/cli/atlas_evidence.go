package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/attributedtext"
	"github.com/0disoft/fairyc/internal/contextpack"
	"github.com/0disoft/fairyc/internal/evidencebundle"
)

type atlasEvidenceOptions struct {
	bundleVersion        string
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
	schema := atlasmap.LogicalSchema
	if options.bundleVersion == "1" {
		schema = atlasmap.LogicalSchemaV2
	}
	if options.bundleVersion == "2" {
		schema = atlasmap.LogicalSchemaV3
	}
	prepared, code := prepareAtlasQuerySchema(
		sourceRoot, options.relation.packageUID, options.relation.locked, stderr, schema,
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
	if options.bundleVersion == "1" {
		encode = evidencebundle.EncodeShowV1
	} else if options.bundleVersion == "2" {
		encode = evidencebundle.EncodeShowV2
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
	relationArguments := make([]string, 0, len(arguments))
	_, err := parseCommandFlags(arguments, flagSchema{Command: command, Experimental: true,
		Flags: []flagDefinition{
			{Name: "--bundle-version", Kind: flagScalar},
			{Name: "--max-fragment-bytes", Kind: flagScalar},
			{Name: "--max-text-bytes", Kind: flagScalar},
			{Name: "--max-facts-bytes", Kind: flagScalar},
			{Name: "--max-text-document-bytes", Kind: flagScalar},
		},
		Required: []string{
			"--max-fragment-bytes", "--max-text-bytes", "--max-facts-bytes", "--max-text-document-bytes",
		}, Forward: &relationArguments}, func(flag, valueText string) error {
		if flag == "--bundle-version" {
			if valueText != "0" && valueText != "1" && valueText != "2" {
				return fmt.Errorf("experimental command %q requires --bundle-version as 0, 1 or 2", command)
			}
			result.bundleVersion = valueText
			return nil
		}
		value, err := parseAtlasPositiveLimit(command, flag, valueText, maxAtlasPackBytes)
		if err != nil {
			return err
		}
		switch flag {
		case "--max-fragment-bytes":
			if value > attributedtext.MaxFragmentBytesGuard {
				return fmt.Errorf("experimental command %q requires %s as a canonical decimal from 1 through %d", command, flag, attributedtext.MaxFragmentBytesGuard)
			}
			result.maxFragmentBytes = value
		case "--max-text-bytes":
			if value > attributedtext.MaxTextBytesGuard {
				return fmt.Errorf("experimental command %q requires %s as a canonical decimal from 1 through %d", command, flag, attributedtext.MaxTextBytesGuard)
			}
			result.maxTextBytes = value
		case "--max-facts-bytes":
			result.maxFactsBytes = value
		case "--max-text-document-bytes":
			result.maxTextDocumentBytes = value
		}
		return nil
	})
	if err != nil {
		return atlasEvidenceOptions{}, err
	}
	if err := validateFlagRules(flagRule{Invalid: result.maxFragmentBytes > result.maxTextBytes, Message: fmt.Sprintf("experimental command %q requires --max-fragment-bytes no greater than --max-text-bytes", command)}); err != nil {
		return atlasEvidenceOptions{}, err
	}
	relation, err := parseAtlasRelationOptions(command, "show", relationArguments)
	if err != nil {
		return atlasEvidenceOptions{}, err
	}
	result.relation = relation
	return result, nil
}
