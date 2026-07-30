// Package architecturecontract owns deterministic repository-local dependency checks that keep
// compiler correctness boundaries executable without adding those checks to the fairyc runtime.
package architecturecontract

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const modulePath = "github.com/0disoft/fairyc"

var atlasPackageRoots = []string{
	modulePath + "/internal/semanticsnapshot",
	modulePath + "/internal/symbolidentity",
	modulePath + "/internal/atlasmap",
	modulePath + "/internal/contextpack",
	modulePath + "/internal/symbolcatalog",
	modulePath + "/internal/workspacecallcatalog",
	modulePath + "/internal/attributedtext",
	modulePath + "/internal/evidencebundle",
}

var atlasConsumerRoots = append(slices.Clone(atlasPackageRoots),
	modulePath+"/internal/cli",
	modulePath+"/cmd/fairyc-windows-host-smoke",
)

// ImportViolation identifies one direct import that would let Atlas or its export bridge flow back
// into a compiler, backend, runtime, library, or unreviewed adapter package.
type ImportViolation struct {
	Importer string
	Imported string
	File     string
}

// CheckAtlasOneWayImports parses every Go file below cmd and internal, including tests and files for
// other build tags. It returns canonical sorted violations and never invokes the Go toolchain.
func CheckAtlasOneWayImports(root string) ([]ImportViolation, error) {
	if root == "" {
		return nil, fmt.Errorf("Atlas import boundary requires a repository root")
	}
	var violations []ImportViolation
	for _, subtree := range []string{"cmd", "internal"} {
		base := filepath.Join(root, subtree)
		info, err := os.Stat(base)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("Atlas import boundary requires repository subtree %q", subtree)
		}
		if err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return fmt.Errorf("derive repository-relative Go path %q", path)
			}
			relative = filepath.ToSlash(relative)
			importerDirectory := filepath.ToSlash(filepath.Dir(relative))
			importer := modulePath + "/" + importerDirectory
			source, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s", relative)
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), relative, source, parser.ImportsOnly)
			if err != nil {
				return fmt.Errorf("parse imports from %s: %w", relative, err)
			}
			for _, importedSpec := range parsed.Imports {
				imported, err := strconv.Unquote(importedSpec.Path.Value)
				if err != nil {
					return fmt.Errorf("decode import in %s: %w", relative, err)
				}
				if isWithinAny(imported, atlasPackageRoots) && !isWithinAny(importer, atlasConsumerRoots) {
					violations = append(violations, ImportViolation{
						Importer: importer, Imported: imported, File: relative,
					})
				}
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("check Atlas imports below %s: %w", subtree, err)
		}
	}
	slices.SortFunc(violations, func(left, right ImportViolation) int {
		if value := strings.Compare(left.Importer, right.Importer); value != 0 {
			return value
		}
		if value := strings.Compare(left.Imported, right.Imported); value != 0 {
			return value
		}
		return strings.Compare(left.File, right.File)
	})
	return violations, nil
}

func isWithinAny(packagePath string, roots []string) bool {
	for _, root := range roots {
		if packagePath == root || strings.HasPrefix(packagePath, root+"/") {
			return true
		}
	}
	return false
}
