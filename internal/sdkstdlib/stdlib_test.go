package sdkstdlib

import (
	"slices"
	"strings"
	"testing"
)

func TestEmbeddedStandardLibraryIdentity(t *testing.T) {
	t.Parallel()
	definition, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Sources) != 2 || len(definition.Modules) != 2 {
		t.Fatalf("definition = %+v", definition)
	}
	paths := []string{definition.Sources[0].Path, definition.Sources[1].Path}
	modules := []string{definition.Modules[0].Path, definition.Modules[1].Path}
	if !slices.Equal(paths, []string{"std/alloc.fy", "std/io.fy"}) ||
		!slices.Equal(modules, []string{"std::alloc", "std::io"}) {
		t.Fatalf("source paths = %v, modules = %v", paths, modules)
	}
	if !strings.HasPrefix(definition.SourceIdentity, SDKIdentity+";"+RuntimeABI+";fysrc1-sha256-") {
		t.Fatalf("source identity = %q", definition.SourceIdentity)
	}
}
