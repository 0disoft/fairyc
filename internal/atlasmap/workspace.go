package atlasmap

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// PrepareWorkspace creates the immutable compiler-fact snapshot and deterministic package plans
// from one already-loaded workspace and its exact compiler result. Compiler executable discovery is
// deliberately caller-owned so this function performs no filesystem or process access.
func PrepareWorkspace(workspace workspaceloader.Workspace, result compiler.Result, compilerBuild buildidentity.Identity) (semanticsnapshot.Snapshot, []Partition, error) {
	return prepareWorkspace(workspace, result, compilerBuild, LogicalSchema)
}

// PrepareWorkspaceV2 prepares generic-aware logical partitions without publishing them.
func PrepareWorkspaceV2(workspace workspaceloader.Workspace, result compiler.Result, compilerBuild buildidentity.Identity) (semanticsnapshot.Snapshot, []Partition, error) {
	return prepareWorkspace(workspace, result, compilerBuild, LogicalSchemaV2)
}

// PrepareWorkspaceV3 retains aggregate specializations and their cleanup bodies.
func PrepareWorkspaceV3(workspace workspaceloader.Workspace, result compiler.Result, compilerBuild buildidentity.Identity) (semanticsnapshot.Snapshot, []Partition, error) {
	return prepareWorkspace(workspace, result, compilerBuild, LogicalSchemaV3)
}

func prepareWorkspace(workspace workspaceloader.Workspace, result compiler.Result, compilerBuild buildidentity.Identity, schema string) (semanticsnapshot.Snapshot, []Partition, error) {
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		return semanticsnapshot.Snapshot{}, nil, err
	}
	fileMetadata, err := workspace.SemanticFiles()
	if err != nil {
		return semanticsnapshot.Snapshot{}, nil, err
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return semanticsnapshot.Snapshot{}, nil, err
	}
	packageIDs := make([]hir.PackageID, 0, len(metadata))
	for _, packageValue := range metadata {
		packageIDs = append(packageIDs, packageValue.ID)
	}
	interfacesByID, genericByID, err := workspaceInterfaceFacts(result, packageIDs, coordinates, schema)
	if err != nil {
		return semanticsnapshot.Snapshot{}, nil, err
	}
	filesByPackage := make(map[hir.PackageID][]semanticsnapshot.FileInput)
	for _, file := range fileMetadata {
		filesByPackage[file.Package] = append(filesByPackage[file.Package], semanticsnapshot.FileInput{
			Path: file.Path, Digest: file.Digest, LineMap: file.LineMap, Kind: semanticsnapshot.FileKind(file.Kind),
		})
	}
	packageInputs := make([]semanticsnapshot.PackageInput, 0, len(metadata))
	interfaceHashes := make(map[string]moduleinterface.Hash, len(metadata))
	genericHashes := make(map[string]moduleinterface.GenericHash, len(genericByID))
	for _, packageValue := range metadata {
		interfaceHash, found := interfacesByID[packageValue.ID]
		if !found {
			return semanticsnapshot.Snapshot{}, nil, fmt.Errorf("package %d has no public-interface hash", packageValue.ID)
		}
		packageInputs = append(packageInputs, semanticsnapshot.PackageInput{
			BuildID: packageValue.ID, UID: packageValue.PackageUID, Name: packageValue.Name,
			Version: packageValue.Version, Language: packageValue.Language, Instance: packageValue.Instance,
			Coordinate: packageValue.Coordinate, Source: packageValue.Source, SourceKind: packageValue.SourceKind,
			RuntimeABI: packageValue.RuntimeABI, Role: packageValue.Role,
			Features: slices.Clone(packageValue.Features), Dependencies: slices.Clone(packageValue.Dependencies),
			Files: slices.Clone(filesByPackage[packageValue.ID]),
		})
		interfaceHashes[packageValue.Instance.String()] = interfaceHash
		if hash, found := genericByID[packageValue.ID]; found {
			genericHashes[packageValue.Instance.String()] = hash
		}
		delete(filesByPackage, packageValue.ID)
	}
	if len(filesByPackage) != 0 {
		return semanticsnapshot.Snapshot{}, nil, fmt.Errorf("source file metadata references an unknown semantic package")
	}
	snapshot, err := semanticsnapshot.BuildCompiler(result, packageInputs)
	if err != nil {
		return semanticsnapshot.Snapshot{}, nil, err
	}
	partitions, err := buildPlan(snapshot, Input{CompilerBuild: compilerBuild, Interfaces: interfaceHashes, GenericInterfaces: genericHashes}, schema)
	if err != nil {
		return semanticsnapshot.Snapshot{}, nil, err
	}
	return snapshot, partitions, nil
}
