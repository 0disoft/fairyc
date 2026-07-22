package atlasmap

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

var (
	ErrStaleMap         = errors.New(".fymap does not match current workspace source")
	ErrIncompatibleMap  = errors.New(".fymap compatibility does not match the reader")
	ErrMapConflict      = errors.New("multiple compatible .fymap generations disagree")
	ErrStoredMapCorrupt = errors.New("stored .fymap is corrupt")
)

// CurrentFile is one caller-authorized current source identity, never source text.
type CurrentFile struct {
	Path    string
	Digest  packagecontent.FileDigest
	LineMap packagecontent.LineMapID
	Kind    semanticsnapshot.FileKind
}

// CurrentWorkspaceExpectations derives the complete safe-reader expectation set from one verified
// compiler result and the exact workspace that produced it.
func CurrentWorkspaceExpectations(
	workspace workspaceloader.Workspace,
	result compiler.Result,
	compilerBuild buildidentity.Identity,
) (map[string]LoadExpectation, error) {
	if !result.Verified || compilerBuild == (buildidentity.Identity{}) {
		return nil, fmt.Errorf("map load expectations require verified MIR and exact compiler identity")
	}
	current, err := CurrentWorkspacePackages(workspace)
	if err != nil {
		return nil, err
	}
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		return nil, err
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return nil, err
	}
	packageIDs := make([]hir.PackageID, 0, len(metadata))
	for _, packageValue := range metadata {
		packageIDs = append(packageIDs, packageValue.ID)
	}
	interfaces, err := moduleinterface.ComputePackages(result.Semantic.HIR, packageIDs, coordinates)
	if err != nil {
		return nil, err
	}
	expectations := make(map[string]LoadExpectation, len(metadata))
	for _, packageValue := range metadata {
		instance := packageValue.Instance.String()
		currentPackage, found := current[instance]
		if !found {
			return nil, fmt.Errorf("current package %q has no source expectation", instance)
		}
		interfaceHash, found := interfaces[packageValue.ID]
		if !found || interfaceHash == (moduleinterface.Hash{}) {
			return nil, fmt.Errorf("current package %q has no public-interface expectation", instance)
		}
		expectations[instance] = LoadExpectation{
			CompilerBuild: compilerBuild, PublicInterface: interfaceHash, Current: currentPackage,
		}
	}
	if len(expectations) != len(current) {
		return nil, fmt.Errorf("current source and interface expectation sets differ")
	}
	return expectations, nil
}

// CurrentPackage is the exact workspace state a stored map must match before spans are trusted.
type CurrentPackage struct {
	UID               string
	Language          string
	Instance          packagegraph.InstanceID
	Coordinate        packagegraph.CoordinateID
	Source            packagecontent.Digest
	SourceKind        string
	RuntimeABI        string
	Features          []string
	Dependencies      []string
	PackageReferences []PackageReference
	Files             []CurrentFile
}

// LoadExpectation combines current source with non-source compatibility selected by the caller.
type LoadExpectation struct {
	CompilerBuild   buildidentity.Identity
	PublicInterface moduleinterface.Hash
	Current         CurrentPackage
}

// CurrentWorkspacePackages recomputes exact file identities from already validated retained bytes
// and returns one canonical current-source expectation per package instance.
func CurrentWorkspacePackages(workspace workspaceloader.Workspace) (map[string]CurrentPackage, error) {
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		return nil, err
	}
	files, err := workspace.SemanticFiles()
	if err != nil {
		return nil, err
	}
	byID := make(map[hir.PackageID]workspaceloader.SemanticPackage, len(metadata))
	for _, packageValue := range metadata {
		if _, duplicate := byID[packageValue.ID]; duplicate {
			return nil, fmt.Errorf("current workspace repeats semantic package %d", packageValue.ID)
		}
		byID[packageValue.ID] = packageValue
	}
	filesByPackage := make(map[hir.PackageID][]CurrentFile, len(metadata))
	for _, file := range files {
		if _, found := byID[file.Package]; !found {
			return nil, fmt.Errorf("current source file %q references unknown package %d", file.Path, file.Package)
		}
		filesByPackage[file.Package] = append(filesByPackage[file.Package], CurrentFile{
			Path: file.Path, Digest: file.Digest, LineMap: file.LineMap, Kind: semanticsnapshot.FileKind(file.Kind),
		})
	}
	result := make(map[string]CurrentPackage, len(metadata))
	for _, packageValue := range metadata {
		dependencies := make([]string, 0, len(packageValue.Dependencies))
		references := make([]PackageReference, 0, len(packageValue.Dependencies)+1)
		references = append(references, PackageReference{UID: packageValue.PackageUID, Instance: packageValue.Instance.String(), Coordinate: packageValue.Coordinate.String()})
		for _, dependencyID := range packageValue.Dependencies {
			dependency, found := byID[dependencyID]
			if !found {
				return nil, fmt.Errorf("current package %d references unknown dependency %d", packageValue.ID, dependencyID)
			}
			dependencies = append(dependencies, dependency.Instance.String())
			references = append(references, PackageReference{UID: dependency.PackageUID, Instance: dependency.Instance.String(), Coordinate: dependency.Coordinate.String()})
		}
		slices.Sort(dependencies)
		slices.SortFunc(references, func(left, right PackageReference) int { return cmp.Compare(left.Instance, right.Instance) })
		currentFiles := filesByPackage[packageValue.ID]
		slices.SortFunc(currentFiles, func(left, right CurrentFile) int { return cmp.Compare(left.Path, right.Path) })
		if len(currentFiles) == 0 {
			return nil, fmt.Errorf("current package %d has no source files", packageValue.ID)
		}
		instance := packageValue.Instance.String()
		if _, duplicate := result[instance]; duplicate {
			return nil, fmt.Errorf("current workspace repeats package instance %q", instance)
		}
		result[instance] = CurrentPackage{
			UID: packageValue.PackageUID, Language: packageValue.Language, Instance: packageValue.Instance,
			Coordinate: packageValue.Coordinate, Source: packageValue.Source, SourceKind: packageValue.SourceKind,
			RuntimeABI: packageValue.RuntimeABI, Features: slices.Clone(packageValue.Features),
			Dependencies: dependencies, PackageReferences: references, Files: slices.Clone(currentFiles),
		}
		delete(filesByPackage, packageValue.ID)
	}
	if len(filesByPackage) != 0 {
		return nil, fmt.Errorf("current workspace has unassigned source files")
	}
	return result, nil
}

// ValidateCurrent rejects stale aggregate or per-file identities before a reader trusts map spans.
func ValidateCurrent(partition Partition, current CurrentPackage) error {
	if err := validateCurrentPackage(current); err != nil {
		return fmt.Errorf("validate current package: %w", err)
	}
	header := partition.Header()
	if header.PackageUID != current.UID || header.Language != current.Language ||
		header.PackageInstance != current.Instance.String() || header.PackageCoordinate != current.Coordinate.String() ||
		header.PackageSource != current.Source.String() || header.PackageSourceKind != current.SourceKind ||
		header.RuntimeABI != current.RuntimeABI || !slices.Equal(header.Features, current.Features) ||
		!slices.Equal(header.Dependencies, current.Dependencies) ||
		!slices.Equal(header.PackageReferences, current.PackageReferences) {
		return ErrStaleMap
	}
	mapFiles := partition.Files()
	if len(mapFiles) != len(current.Files) {
		return ErrStaleMap
	}
	for index, file := range mapFiles {
		currentFile := current.Files[index]
		if file.Path != currentFile.Path || file.Digest != currentFile.Digest.String() ||
			file.LineMap != currentFile.LineMap.String() || file.Kind != currentFile.Kind {
			return ErrStaleMap
		}
	}
	return nil
}

func validateLoadExpectation(value LoadExpectation) error {
	if value.CompilerBuild == (buildidentity.Identity{}) || value.PublicInterface == (moduleinterface.Hash{}) {
		return fmt.Errorf("map load expectation requires compiler and interface identities")
	}
	if err := validateCurrentPackage(value.Current); err != nil {
		return fmt.Errorf("validate map current-source expectation: %w", err)
	}
	return nil
}

func validateCurrentPackage(value CurrentPackage) error {
	if value.Instance == (packagegraph.InstanceID{}) || value.Coordinate == (packagegraph.CoordinateID{}) ||
		value.Source == (packagecontent.Digest{}) || !validTransportText(value.UID) ||
		!validTransportText(value.Language) || !validTransportText(value.SourceKind) ||
		value.RuntimeABI != "" && !validTransportText(value.RuntimeABI) {
		return fmt.Errorf("current package identity is incomplete or invalid")
	}
	if err := validateSortedText("current features", value.Features, false); err != nil {
		return err
	}
	if err := validateSortedText("current dependencies", value.Dependencies, false); err != nil {
		return err
	}
	for _, dependency := range value.Dependencies {
		parsed, err := packagegraph.ParseInstanceID(dependency)
		if err != nil || parsed == (packagegraph.InstanceID{}) {
			return fmt.Errorf("current dependency %q is invalid", dependency)
		}
	}
	if len(value.PackageReferences) != len(value.Dependencies)+1 {
		return fmt.Errorf("current package references do not cover self and direct dependencies")
	}
	selfFound := false
	for index, reference := range value.PackageReferences {
		if index > 0 && value.PackageReferences[index-1].Instance >= reference.Instance {
			return fmt.Errorf("current package references are not canonical")
		}
		instance, err := packagegraph.ParseInstanceID(reference.Instance)
		if err != nil || instance == (packagegraph.InstanceID{}) || !validTransportText(reference.UID) {
			return fmt.Errorf("current package reference is invalid")
		}
		coordinate, err := packagegraph.ParseCoordinateID(reference.Coordinate)
		if err != nil || coordinate == (packagegraph.CoordinateID{}) {
			return fmt.Errorf("current package reference coordinate is invalid")
		}
		if reference.Instance == value.Instance.String() {
			selfFound = reference.UID == value.UID && reference.Coordinate == value.Coordinate.String()
		} else if !slices.Contains(value.Dependencies, reference.Instance) {
			return fmt.Errorf("current package reference is not a direct dependency")
		}
	}
	if !selfFound {
		return fmt.Errorf("current package self reference is missing")
	}
	if len(value.Files) == 0 {
		return fmt.Errorf("current package has no files")
	}
	for index, file := range value.Files {
		if index > 0 && value.Files[index-1].Path >= file.Path {
			return fmt.Errorf("current files are not canonical")
		}
		if _, err := source.NewFile(file.Path, nil).ExpectedModule(); err != nil {
			return fmt.Errorf("current file %q is invalid", file.Path)
		}
		if file.Digest == (packagecontent.FileDigest{}) || file.LineMap == (packagecontent.LineMapID{}) ||
			file.Kind != semanticsnapshot.FileKindSource && file.Kind != semanticsnapshot.FileKindTest {
			return fmt.Errorf("current file %q has invalid identities or kind", file.Path)
		}
	}
	return nil
}

func compatibilityError(partition Partition, expectation LoadExpectation) error {
	if err := ValidateCurrent(partition, expectation.Current); err != nil {
		return err
	}
	header := partition.Header()
	if header.CompilerBuild != expectation.CompilerBuild.String() || header.InterfaceHash != expectation.PublicInterface.String() {
		return ErrIncompatibleMap
	}
	return nil
}
