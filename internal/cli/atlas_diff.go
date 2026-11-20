package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packageartifact"
)

func runAtlasDiff(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	const command = "atlas-diff"
	var baseline, uid string
	var packages, declarations, output uint64
	seen, err := parseCommandFlags(arguments, flagSchema{Command: command, Experimental: true, Flags: []flagDefinition{
		{Name: "--baseline", Kind: flagScalar},
		{Name: "--package-uid", Kind: flagScalar},
		{Name: "--workspace", Kind: flagSwitch},
		{Name: "--max-packages", Kind: flagScalar},
		{Name: "--max-declarations", Kind: flagScalar},
		{Name: "--max-output-bytes", Kind: flagScalar},
	}, Required: []string{"--baseline", "--max-declarations", "--max-output-bytes"}}, func(flag, value string) error {
		switch flag {
		case "--baseline":
			if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) {
				return fmt.Errorf("atlas-diff requires a baseline workspace directory")
			}
			baseline = value
		case "--package-uid":
			uid = value
			return validateAtlasPackageUID(command, value)
		case "--max-packages":
			var err error
			packages, err = parseAtlasPositiveLimit(command, flag, value, maxAtlasPackRecords)
			return err
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
	if err := validateFlagRules(
		flagRule{Invalid: seen["--workspace"] == seen["--package-uid"], Message: "atlas-diff requires exactly one of --package-uid and --workspace"},
		flagRule{Invalid: seen["--workspace"] && packages == 0, Message: "atlas-diff --workspace requires --max-packages"},
		flagRule{Invalid: !seen["--workspace"] && packages != 0, Message: "atlas-diff --max-packages requires --workspace"},
	); err != nil {
		return invocationError(stderr, err.Error())
	}
	if !filepath.IsAbs(baseline) {
		baseline = filepath.Join(sourceRoot, baseline)
	}
	var oldCheck, newCheck compiler.Result
	before, code := prepareAtlasWorkspaceWithResult(baseline, true, stderr, atlasmap.LogicalSchemaV3, &oldCheck)
	if code != ExitSuccess {
		return code
	}
	after, code := prepareAtlasWorkspaceWithResult(sourceRoot, true, stderr, atlasmap.LogicalSchemaV3, &newCheck)
	if code != ExitSuccess {
		return code
	}
	oldAliases := atlasmap.DiffSpecializationAliases(before.partitions)
	newAliases := atlasmap.DiffSpecializationAliases(after.partitions)
	fingerprints := func(old, new atlasmap.Partition) (map[string][sha256.Size]byte, map[string][sha256.Size]byte) {
		return atlasDiffFingerprints(before, old, oldCheck.MIR, oldAliases), atlasDiffFingerprints(after, new, newCheck.MIR, newAliases)
	}
	var report any
	if seen["--workspace"] {
		report, err = atlasmap.CompareWorkspaceV3(before.partitions, after.partitions, before.expectations, after.expectations,
			int(packages), int(declarations), oldAliases, newAliases, fingerprints)
	} else {
		old, ok := uniqueDiffPartition(before.partitions, uid)
		new, found := uniqueDiffPartition(after.partitions, uid)
		if !ok || !found {
			return invocationError(stderr, "atlas-diff requires exactly one selected package in each workspace")
		}
		report, err = atlasmap.CompareV3(old, new, before.expectations[old.Header().PackageInstance],
			after.expectations[new.Header().PackageInstance], int(declarations), oldAliases, newAliases,
			func() (map[string][sha256.Size]byte, map[string][sha256.Size]byte) { return fingerprints(old, new) })
	}
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

func atlasDiffFingerprints(prepared preparedAtlasWorkspace, partition atlasmap.Partition, program mir.Program, aliases map[string]string) map[string][sha256.Size]byte {
	packages, err := prepared.workspace.SemanticPackages()
	if err != nil {
		return nil
	}
	instance := partition.Header().PackageInstance
	identities, err := prepared.workspace.PackageIdentities()
	if err != nil {
		return nil
	}
	coordinates, err := prepared.workspace.PackageCoordinates()
	if err != nil {
		return nil
	}
	for _, pkg := range packages {
		if pkg.Instance.String() != instance {
			continue
		}
		const maxBodyBytes = 32 << 20
		artifact, err := packageartifact.Extract(program, pkg.ID, identities, coordinates)
		if err != nil {
			return nil
		}
		fingerprints, err := packageartifact.ExecutableBodyFingerprints(artifact, maxBodyBytes, aliases)
		if err != nil {
			return nil
		}
		return fingerprints
	}
	return nil
}
