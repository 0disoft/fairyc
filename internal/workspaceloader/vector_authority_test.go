package workspaceloader

import (
	"testing"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

func TestVectorCandidateAuthorityMatchesBytesAndAST(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), map[string]string{"src/main.fy": "module main; import std::vec as vec;"})
	load := func() Workspace {
		t.Helper()
		workspace, err := LoadVectorCandidate(root)
		if err != nil {
			t.Fatal(err)
		}
		return workspace
	}
	workspace := load()
	owners, err := workspace.VectorCandidateStorageOwners()
	if err != nil || len(owners) != 1 {
		t.Fatalf("exact candidate authority: %v %v", owners, err)
	}
	for name, mutate := range map[string]func(*Package){
		"source bytes":   func(p *Package) { p.Sources[0].Bytes[0] ^= 1 },
		"AST only":       func(p *Package) { p.Sources[0].Module.Path = "forged" },
		"file identity":  func(p *Package) { p.Sources[0].Path = "forged.fy" },
		"test source":    func(p *Package) { p.Sources[0].Test = true },
		"missing file":   func(p *Package) { p.Sources = p.Sources[1:] },
		"source order":   func(p *Package) { p.Sources[0], p.Sources[1] = p.Sources[1], p.Sources[0] },
		"digest":         func(p *Package) { p.Digest[0] ^= 1 },
		"SDK label":      func(p *Package) { p.SDKIdentity = sdkstdlib.SDKIdentity },
		"runtime ABI":    func(p *Package) { p.RuntimeABI = "forged" },
		"module catalog": func(p *Package) { p.Modules[0].Path = "forged" },
	} {
		t.Run(name, func(t *testing.T) {
			workspace := load()
			changed := false
			for i := range workspace.Packages {
				if workspace.Packages[i].Path == sdkstdlib.PackagePath {
					mutate(&workspace.Packages[i])
					changed = true
				}
			}
			if !changed {
				t.Fatal("SDK missing from mutation fixture")
			}
			if _, err := workspace.VectorCandidateStorageOwners(); err == nil {
				t.Fatal("altered SDK gained storage authority")
			}
		})
	}
	workspace = load()
	workspace.vectorCandidate = false
	if _, err := workspace.VectorCandidateStorageOwners(); err == nil {
		t.Fatal("ordinary selection gained candidate authority")
	}
	workspace = load()
	workspace.Packages[0].SDKIdentity = sdkstdlib.VectorCandidateIdentity
	if _, err := workspace.VectorCandidateStorageOwners(); err == nil {
		t.Fatal("client label gained SDK authority")
	}
}

func TestStandardVectorAuthorityUsesExactBundle(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), map[string]string{"src/main.fy": "module main; import std::vec as vec;"})
	for _, mutation := range []string{"none", "identity", "AST"} {
		workspace, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		for i := range workspace.Packages {
			pkg := &workspace.Packages[i]
			if pkg.Path != sdkstdlib.PackagePath {
				continue
			}
			if mutation == "identity" {
				pkg.SDKIdentity = sdkstdlib.VectorCandidateIdentity
			}
			if mutation == "AST" {
				pkg.Sources[0].Module.Path = "forged"
			}
		}
		owners, err := workspace.SDKStorageOwners()
		if mutation == "none" {
			if err != nil || len(owners) != 1 {
				t.Fatalf("standard vector authority: %v %v", owners, err)
			}
		} else if err == nil {
			t.Fatalf("forged %s accepted", mutation)
		}
	}
	writeWorkspaceLoaderFile(t, root, "src/main.fy", "module main; import std::io as io;")
	core, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := core.SDKStorageOwners()
	if err != nil || len(owners) != 0 {
		t.Fatalf("core SDK selected vector authority: %v %v", owners, err)
	}
}
