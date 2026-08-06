package sdkstdlib

import (
	"slices"
	"strings"
	"testing"
)

func TestEmbeddedStandardLibraryIdentity(t *testing.T) {
	t.Parallel()
	if Version != "0.0.0-stage0.43" {
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
	var pathKindVariants []string
	createDirAllPublic := false
	copyFileBytesPublic := false
	pathKindPublic := false
	seekFunctions := make(map[string]bool)
	readOnlyPublic := false
	hardLinkCountPublic := false
	allocatedSizePublic := false
	hardLinkPublic := false
	appendBytePublic := false
	asBytesPublic := false
	stderrScalarFunctions := make(map[string]bool)
	for _, source := range definition.Sources {
		if source.Path == "std/bytes.fy" {
			for _, function := range source.Module.Functions {
				if function.Name == "append_byte" {
					appendBytePublic = function.Public
				}
			}
		}
		if source.Path == "std/text.fy" {
			for _, function := range source.Module.Functions {
				if function.Name == "as_bytes" {
					asBytesPublic = function.Public
				}
			}
		}
		if source.Path == "std/io.fy" {
			for _, function := range source.Module.Functions {
				switch function.Name {
				case "write_stderr_i32", "write_stderr_u32", "write_stderr_usize", "write_stderr_bool":
					stderrScalarFunctions[function.Name] = function.Public
				}
			}
		}
		if source.Path != "std/fs.fy" {
			continue
		}
		for _, structure := range source.Module.Structs {
			if structure.Name == "FileKind" {
				t.Fatal("FileKind remains a struct")
			}
		}
		for _, enumeration := range source.Module.Enums {
			switch enumeration.Name {
			case "FileKind":
				if !enumeration.Public {
					t.Fatal("FileKind enum is not public")
				}
				for _, variant := range enumeration.Variants {
					if len(variant.Fields) != 0 {
						t.Fatalf("FileKind variant %q has payload fields", variant.Name)
					}
					fileKindVariants = append(fileKindVariants, variant.Name)
				}
			case "PathKind":
				if !enumeration.Public {
					t.Fatal("PathKind enum is not public")
				}
				for _, variant := range enumeration.Variants {
					if len(variant.Fields) != 0 {
						t.Fatalf("PathKind variant %q has payload fields", variant.Name)
					}
					pathKindVariants = append(pathKindVariants, variant.Name)
				}
			}
		}
		for _, function := range source.Module.Functions {
			switch function.Name {
			case "seek_current", "seek_end":
				seekFunctions[function.Name] = function.Public
			}
			if function.Name == "create_dir_all" {
				createDirAllPublic = function.Public
			}
			if function.Name == "copy_file_bytes" {
				copyFileBytesPublic = function.Public
			}
			if function.Name == "path_kind" {
				pathKindPublic = function.Public
			}
			if function.Name == "is_read_only" {
				readOnlyPublic = function.Public
			}
			if function.Name == "hard_link_count" {
				hardLinkCountPublic = function.Public
			}
			if function.Name == "allocated_size" {
				allocatedSizePublic = function.Public
			}
			if function.Name == "hard_link" {
				hardLinkPublic = function.Public
			}
		}
	}
	if !slices.Equal(fileKindVariants, []string{"Unknown", "Disk", "Character", "Pipe"}) {
		t.Fatalf("FileKind variants = %v", fileKindVariants)
	}
	if !slices.Equal(pathKindVariants, []string{"File", "Directory", "Other"}) {
		t.Fatalf("PathKind variants = %v", pathKindVariants)
	}
	if !createDirAllPublic {
		t.Fatal("std::fs::create_dir_all is not public")
	}
	if !copyFileBytesPublic {
		t.Fatal("std::fs::copy_file_bytes is not public")
	}
	if !pathKindPublic {
		t.Fatal("std::fs::path_kind is not public")
	}
	if !readOnlyPublic {
		t.Fatal("std::fs::is_read_only is not public")
	}
	if !hardLinkCountPublic {
		t.Fatal("std::fs::hard_link_count is not public")
	}
	if !allocatedSizePublic {
		t.Fatal("std::fs::allocated_size is not public")
	}
	if !hardLinkPublic {
		t.Fatal("std::fs::hard_link is not public")
	}
	for _, name := range []string{"seek_current", "seek_end"} {
		if !seekFunctions[name] {
			t.Errorf("std::fs::%s is not public", name)
		}
	}
	if !appendBytePublic {
		t.Fatal("std::bytes::append_byte is not public")
	}
	if !asBytesPublic {
		t.Fatal("std::text::as_bytes is not public")
	}
	for _, name := range []string{"write_stderr_i32", "write_stderr_u32", "write_stderr_usize", "write_stderr_bool"} {
		if !stderrScalarFunctions[name] {
			t.Errorf("std::io::%s is not public", name)
		}
	}
}
