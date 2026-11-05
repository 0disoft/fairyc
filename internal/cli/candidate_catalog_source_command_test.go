package cli

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func verifyCandidateCatalogSourceCommand(t *testing.T, image string) {
	t.Helper()
	for _, fixture := range []struct {
		name, source, filter string
		version              int
	}{
		{"families-v1", "", "", 1},
		{"families-v2", "", "", 2},
		{"families-v3", "", "", 3},
		{"owner-namespaces", atlasOwnerQualifiedNamespaceSource(true), "Cell::get", 3},
		{"used-functions-v2", atlasGenericProductionUsedSource, "", 2},
		{"used-functions-v3", atlasGenericProductionUsedSource, "", 3},
		{"unused-functions", `module main;pub fn T unused<T>(T value){return value;}fn i32 main(){return 0;}`, "unused", 2},
		{"aggregates", `module main;import answer as a;pub struct Cell<T>{T value;}pub resource Token<T>{Cell<T> payload;cleanup{}}pub struct Unused<T>{T value;}fn Token<T> make<T>(T value){return Token<T>{.payload=Cell<T>{.value=value}};}fn i32 main(){let t=make<i32>(a::value());discard make<u32>(7);return t.payload.value;}`, "", 3},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			var root string
			if fixture.source == "" {
				root = candidateAtlasFamilyWorkspace(t)
			} else if fixture.source == atlasGenericProductionUsedSource {
				root = candidateAtlasGenericProductionWorkspace(t, fixture.source)
			} else {
				root = writeWorkspaceSessionFixture(t, fixture.source)
			}
			var diagnostic bytes.Buffer
			if _, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3); code != 0 {
				t.Fatalf("source setup: %d %s", code, &diagnostic)
			}
			f := candidateCatalogCommandFixture(t, image, root, fixture.version)
			r := atlasmap.SymbolCatalogRequest{Name: fixture.filter, Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 64, MaxSelectionBytes: 1 << 20}}
			golden, err := candidateCatalogOracle(t, f, r, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			before := remoteTree(t, root)
			code, out, stderr := fetchCLIProcess(t, image, root, append(candidateCatalogArgs(f, r, 1<<20), "--locked"), "")
			if code != 0 || stderr != "" || !bytes.Equal([]byte(out), golden) {
				t.Fatalf("source command: %d %.3500s %.500s; want %.3500s", code, out, stderr, golden)
			}
			minimum := 2
			if fixture.name == "owner-namespaces" || fixture.source == "" {
				minimum = 1
			} else if fixture.name == "aggregates" {
				minimum = 3
			}
			for version := 1; version < minimum; version++ {
				f.version = version
				code, out, stderr := fetchCLIProcess(t, image, root, append(candidateCatalogArgs(f, r, 1<<20), "--locked"), "")
				if code != 3 || out != "" || stderr == "" {
					t.Fatalf("unsupported family v%d: %d %q %q", version, code, out, stderr)
				}
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("source command changed workspace")
			}
		})
	}
}

func verifyCandidateCatalogSDKCommand(t *testing.T, image string) {
	t.Helper()
	for _, fixture := range []struct {
		name, source string
		version      int
	}{
		{"scalars", atlasSDKCommandSource, 1},
		{"vector", candidateAtlasSDKVectorUsedSource, 3},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, fixture.source)
			var diagnostic bytes.Buffer
			if _, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3); code != 0 {
				t.Fatalf("SDK source setup: %d %s", code, &diagnostic)
			}
			f := candidateCatalogCommandFixture(t, image, root, fixture.version)
			r := atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 64, MaxSelectionBytes: 1 << 20}}
			golden, err := candidateCatalogOracle(t, f, r, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			before := remoteTree(t, root)
			code, out, stderr := candidateAtlasSDKCLIProcess(t, image, root, append(candidateCatalogArgs(f, r, 1<<20), "--locked"))
			if code != 0 || stderr != "" || !bytes.Equal([]byte(out), golden) {
				t.Fatalf("SDK source command: %d %.3500s %.500s; want %.3500s", code, out, stderr, golden)
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("SDK source command changed workspace")
			}
		})
	}
}
