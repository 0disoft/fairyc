package sdkstdlib

import (
	"slices"
	"strings"
	"testing"
)

func TestEmbeddedStandardLibraryIdentity(t *testing.T) {
	t.Parallel()
	if Version != "0.0.0-stage0.29" {
		t.Fatalf("Version = %q", Version)
	}
	if LanguageRevision != "fairyc-language/10" {
		t.Fatalf("LanguageRevision = %q", LanguageRevision)
	}
	definition, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Sources) != 6 || len(definition.Modules) != 6 {
		t.Fatalf("definition = %+v", definition)
	}
	paths := []string{
		definition.Sources[0].Path, definition.Sources[1].Path, definition.Sources[2].Path,
		definition.Sources[3].Path, definition.Sources[4].Path, definition.Sources[5].Path,
	}
	modules := []string{
		definition.Modules[0].Path, definition.Modules[1].Path, definition.Modules[2].Path,
		definition.Modules[3].Path, definition.Modules[4].Path, definition.Modules[5].Path,
	}
	if !slices.Equal(paths, []string{"std/alloc.fy", "std/bytes.fy", "std/fs.fy", "std/io.fy", "std/process.fy", "std/text.fy"}) ||
		!slices.Equal(modules, []string{"std::alloc", "std::bytes", "std::fs", "std::io", "std::process", "std::text"}) {
		t.Fatalf("source paths = %v, modules = %v", paths, modules)
	}
	if !strings.HasPrefix(definition.SourceIdentity, SDKIdentity+";"+RuntimeABI+";fysrc1-sha256-") {
		t.Fatalf("source identity = %q", definition.SourceIdentity)
	}
	var fileKindVariants []string
	for _, source := range definition.Sources {
		if source.Path != "std/fs.fy" {
			continue
		}
		for _, structure := range source.Module.Structs {
			if structure.Name == "FileKind" {
				t.Fatal("FileKind remains a struct")
			}
		}
		for _, enumeration := range source.Module.Enums {
			if enumeration.Name != "FileKind" {
				continue
			}
			if !enumeration.Public {
				t.Fatal("FileKind enum is not public")
			}
			for _, variant := range enumeration.Variants {
				if len(variant.Fields) != 0 {
					t.Fatalf("FileKind variant %q has payload fields", variant.Name)
				}
				fileKindVariants = append(fileKindVariants, variant.Name)
			}
		}
	}
	if !slices.Equal(fileKindVariants, []string{"Unknown", "Disk", "Character", "Pipe"}) {
		t.Fatalf("FileKind variants = %v", fileKindVariants)
	}
}
