package workspaceloader

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// ExactSource is a detached package-relative source view recomputed from the retained validated
// workspace bytes. It contains no absolute path and does not imply caller authority to reopen one.
type ExactSource struct {
	Package SemanticPackage
	File    SemanticFile
	Bytes   []byte
}

// ResolveExactSource selects one retained source by exact package instance and canonical path. It
// hashes the complete package source closure, cloning the selected file before hashing so a caller
// never receives bytes that disagree with the validated package identity.
func (workspace Workspace) ResolveExactSource(packageInstance, sourcePath string) (ExactSource, error) {
	instance, err := packagegraph.ParseInstanceID(packageInstance)
	if err != nil {
		return ExactSource{}, fmt.Errorf("resolve exact workspace source: %w", err)
	}
	packages, err := workspace.SemanticPackages()
	if err != nil {
		return ExactSource{}, fmt.Errorf("resolve exact workspace source packages: %w", err)
	}
	var selectedPackage SemanticPackage
	packageFound := false
	for _, packageValue := range packages {
		if packageValue.Instance != instance {
			continue
		}
		if packageFound {
			return ExactSource{}, fmt.Errorf("resolve exact workspace source: package instance is duplicated")
		}
		selectedPackage, packageFound = packageValue, true
	}
	if !packageFound {
		return ExactSource{}, fmt.Errorf("resolve exact workspace source: package instance is absent")
	}
	loaded, found := workspace.Package(selectedPackage.Path)
	if !found {
		return ExactSource{}, fmt.Errorf("resolve exact workspace source: package path is absent")
	}
	files := make([]packagecontent.File, 0, len(loaded.Sources))
	selectedIndex := -1
	for index, sourceFile := range loaded.Sources {
		contents := sourceFile.Bytes
		if sourceFile.Path == sourcePath {
			if selectedIndex >= 0 {
				return ExactSource{}, fmt.Errorf("resolve exact workspace source: source path is duplicated")
			}
			selectedIndex = index
			contents = slices.Clone(contents)
		}
		files = append(files, packagecontent.File{Path: sourceFile.Path, Bytes: contents})
	}
	if selectedIndex < 0 {
		return ExactSource{}, fmt.Errorf("resolve exact workspace source: source path is absent")
	}
	digest, err := packagecontent.Compute(files)
	if err != nil {
		return ExactSource{}, fmt.Errorf("resolve exact workspace source package digest: %w", err)
	}
	if digest != loaded.Digest || digest != selectedPackage.Source {
		return ExactSource{}, fmt.Errorf("resolve exact workspace source: retained package bytes are stale")
	}
	selected := files[selectedIndex]
	fileDigest, lineMap := packagecontent.ComputeFileIdentities(selected.Bytes)
	kind := SemanticFileSource
	if loaded.Sources[selectedIndex].Test {
		kind = SemanticFileTest
	}
	return ExactSource{
		Package: selectedPackage,
		File: SemanticFile{
			Package: selectedPackage.ID, Path: selected.Path, Digest: fileDigest, LineMap: lineMap, Kind: kind,
		},
		Bytes: selected.Bytes,
	}, nil
}
