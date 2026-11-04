package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

type impactOptions struct {
	uid, symbol                             string
	packages, calls, depth, symbols, output int
}

func parseImpact(arguments []string) (impactOptions, error) {
	o := impactOptions{}
	defs := []flagDefinition{{Name: "--package-uid", Kind: flagScalar}, {Name: "--symbol", Kind: flagScalar}, {Name: "--locked", Kind: flagSwitch}}
	numbers := map[string]*int{"--max-packages": &o.packages, "--max-calls": &o.calls, "--max-depth": &o.depth, "--max-symbols": &o.symbols, "--max-output-bytes": &o.output}
	required := []string{"--package-uid", "--symbol", "--max-packages", "--max-calls", "--max-depth", "--max-symbols", "--max-output-bytes"}
	for _, flag := range required[2:] {
		defs = append(defs, flagDefinition{Name: flag, Kind: flagScalar})
	}
	_, err := parseCommandFlags(arguments, flagSchema{Command: "atlas-impact", Experimental: true, Flags: defs, Required: required}, func(flag, value string) error {
		switch flag {
		case "--locked":
			return nil
		case "--package-uid":
			if err := validateAtlasPackageUID("atlas-impact", value); err != nil {
				return err
			}
			o.uid = value
		case "--symbol":
			if _, err := symbolidentity.ParseVisibleID(value); err != nil {
				if _, err := symbolidentity.ParseFullDigest(value); err != nil {
					return fmt.Errorf("canonical SymbolId required")
				}
			}
			o.symbol = value
		default:
			n, err := strconv.Atoi(value)
			maximum := 65536
			if flag == "--max-depth" {
				maximum = 64
			} else if flag == "--max-symbols" {
				maximum = 4096
			} else if flag == "--max-output-bytes" {
				maximum = 4 << 20
			}
			if err != nil || strconv.Itoa(n) != value || n < 0 || (n == 0 && flag != "--max-depth") || n > maximum {
				return fmt.Errorf("invalid %s limit", flag)
			}
			*numbers[flag] = n
		}
		return nil
	})
	return o, err
}

func impactDocument(sourceRoot string, o impactOptions, stderr io.Writer) ([]byte, int) {
	prepared, code := prepareAtlasWorkspaceSchema(sourceRoot, true, stderr, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		return nil, code
	}
	return (&preparedAtlasImpact{workspace: prepared}).document(o, stderr)
}

// The session owns this immutable preparation and at most one bounded call index.
type preparedAtlasImpact struct {
	workspace preparedAtlasWorkspace
	index     *atlasmap.WorkspaceCallIndex
}

func (impact *preparedAtlasImpact) document(o impactOptions, stderr io.Writer) ([]byte, int) {
	prepared := impact.workspace
	partition, found := selectAtlasPartition(prepared.partitions, o.uid)
	matches := 0
	for _, p := range prepared.partitions {
		if p.Header().PackageUID == o.uid {
			matches++
		}
	}
	if matches > 1 {
		return nil, invocationError(stderr, "impact package UID has multiple feature instances")
	}
	if !found {
		return nil, invocationError(stderr, "impact package UID was not found")
	}
	budget := atlasmap.WorkspaceCallBudget{MaxPackages: o.packages, MaxCalls: o.calls}
	if impact.index == nil || impact.index.Budget() != budget {
		index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, budget)
		if err != nil {
			return nil, invocationError(stderr, err.Error())
		}
		impact.index = index
	}
	r, err := impact.index.Impact(partition.Header().PackageInstance, o.symbol, o.depth, o.symbols)
	if err != nil {
		return nil, invocationError(stderr, err.Error())
	}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, internalError(stderr)
	}
	if len(data) > o.output {
		return nil, invocationError(stderr, "impact output exceeds byte limit")
	}
	return data, ExitSuccess
}

func runAtlasImpact(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	o, err := parseImpact(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	data, code := impactDocument(sourceRoot, o, stderr)
	if code != ExitSuccess {
		return code
	}
	if _, err := stdout.Write(data); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
