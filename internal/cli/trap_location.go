package cli

import (
	"fmt"
	"path"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type trapSource struct {
	display string
	bytes   []byte
}

// Retain the compilation's source snapshot rather than reading files after execution.
// Match the emitter's build-local package/file keys, including distinct feature instances.
func workspaceTrapSources(workspace workspaceloader.Workspace) map[string]trapSource {
	sources := make(map[string]trapSource)
	packages := make(map[string]workspaceloader.Package, len(workspace.Packages))
	for _, pkg := range workspace.Packages {
		packages[pkg.Path] = pkg
	}
	instances := make(map[string]string, len(workspace.Instances))
	for _, instance := range workspace.Instances {
		instances[instance.NodeID] = instance.PackagePath
	}
	for index, class := range workspace.Resolution.InstanceClasses {
		if len(class.Members) == 0 {
			continue
		}
		packagePath, found := instances[class.Members[0]]
		if !found {
			continue
		}
		pkg, found := packages[packagePath]
		if !found {
			continue
		}
		for _, file := range pkg.Sources {
			display := path.Join(pkg.Path, file.Path)
			if pkg.SourceKind == workspaceloader.SourceKindLocal {
				display = path.Join(pkg.Path, pkg.Manifest.Package.Source, file.Path)
			}
			key := cbackend.PackageTrapSource(hir.PackageID(index+1), file.Path)
			sources[key] = trapSource{display: display, bytes: file.Bytes}
		}
	}
	return sources
}

func formatNativeTrap(outcome cdriver.NativeOutcome, sources map[string]trapSource) string {
	if outcome.TrapCategory == "" {
		return ""
	}
	text := fmt.Sprintf("trap: %s\n", outcome.TrapCategory)
	location := outcome.TrapLocation
	if location == nil {
		return text
	}
	source, found := sources[location.Source]
	if !found || source.display == "" || location.StartByte > location.EndByte ||
		location.EndByte > uint64(len(source.bytes)) || !utf8.Valid(source.bytes) {
		return text
	}
	start, end := int(location.StartByte), int(location.EndByte)
	if (start < len(source.bytes) && !utf8.RuneStart(source.bytes[start])) ||
		(end < len(source.bytes) && !utf8.RuneStart(source.bytes[end])) {
		return text
	}
	line, column := 1, 1
	for _, value := range string(source.bytes[:start]) {
		if value == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return text + fmt.Sprintf("  at %s:%d:%d\n", safeDisplayPath(source.display), line, column)
}
