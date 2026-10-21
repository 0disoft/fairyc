package cli

import (
	"fmt"
	"path"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type trapSource struct {
	display string
	bytes   []byte
}

// Retain the compilation's source snapshot rather than reading files after execution.
// Receipts do not carry a package ID: duplicate source names must remain unknown,
// even when their contents match, to avoid attributing a trap to the wrong package.
func workspaceTrapSources(workspace workspaceloader.Workspace) map[string]trapSource {
	sources := make(map[string]trapSource)
	for _, pkg := range workspace.Packages {
		for _, file := range pkg.Sources {
			if _, duplicate := sources[file.Path]; duplicate {
				sources[file.Path] = trapSource{}
				continue
			}
			display := path.Join(pkg.Path, file.Path)
			if pkg.SourceKind == workspaceloader.SourceKindLocal {
				display = path.Join(pkg.Path, pkg.Manifest.Package.Source, file.Path)
			}
			sources[file.Path] = trapSource{display: display, bytes: file.Bytes}
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
