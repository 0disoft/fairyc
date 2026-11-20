package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func runAtlasDiff(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	const command = "atlas-diff"
	var baseline, uid string
	var declarations, output uint64
	required := []string{"--baseline", "--package-uid", "--max-declarations", "--max-output-bytes"}
	flags := make([]flagDefinition, 0, len(required))
	for _, flag := range required {
		flags = append(flags, flagDefinition{Name: flag, Kind: flagScalar})
	}
	_, err := parseCommandFlags(arguments, flagSchema{Command: command, Experimental: true, Flags: flags, Required: required}, func(flag, value string) error {
		switch flag {
		case "--baseline":
			if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) {
				return fmt.Errorf("atlas-diff requires a baseline workspace directory")
			}
			baseline = value
		case "--package-uid":
			uid = value
			return validateAtlasPackageUID(command, value)
		case "--max-declarations":
			var err error
			declarations, err = parseAtlasPositiveLimit(command, flag, value, maxAtlasPackRecords)
			return err
		case "--max-output-bytes":
			var err error
			output, err = parseAtlasPositiveLimit(command, flag, value, maxAtlasPackBytes)
			return err
		}
		return nil
	})
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	if !filepath.IsAbs(baseline) {
		baseline = filepath.Join(sourceRoot, baseline)
	}
	before, code := prepareAtlasWorkspaceSchema(baseline, true, stderr, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		return code
	}
	after, code := prepareAtlasWorkspaceSchema(sourceRoot, true, stderr, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		return code
	}
	old, ok := uniqueDiffPartition(before.partitions, uid)
	new, found := uniqueDiffPartition(after.partitions, uid)
	if !ok || !found {
		return invocationError(stderr, "atlas-diff requires exactly one selected package in each workspace")
	}
	report, err := atlasmap.CompareV3(old, new, before.expectations[old.Header().PackageInstance],
		after.expectations[new.Header().PackageInstance], int(declarations))
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	data, err := json.Marshal(report)
	if err != nil {
		return internalError(stderr)
	}
	if uint64(len(data)) > output {
		return invocationError(stderr, "atlas-diff output exceeds byte limit")
	}
	if _, err := stdout.Write(data); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func uniqueDiffPartition(partitions []atlasmap.Partition, uid string) (atlasmap.Partition, bool) {
	var selected atlasmap.Partition
	found := false
	for _, partition := range partitions {
		if partition.Header().PackageUID != uid {
			continue
		}
		if found {
			return atlasmap.Partition{}, false
		}
		selected, found = partition, true
	}
	return selected, found
}
