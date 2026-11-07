package cli

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestCandidateProjectTypePrefix(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows project prefix native probe; Linux execution is batched")
	}
	read := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	productionPath := filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_project_types.fy")
	fixturePath := filepath.Join("testdata", "project_type_prefix", "main.fy")
	production, fixture := read(productionPath), read(fixturePath)
	productionHash, fixtureHash := sha256.Sum256(production), sha256.Sum256(fixture)
	t.Logf("production sha256=%x fixture sha256=%x", productionHash, fixtureHash)
	unchanged := func() {
		t.Helper()
		if sha256.Sum256(read(productionPath)) != productionHash || sha256.Sum256(read(fixturePath)) != fixtureHash {
			t.Fatal("project prefix inputs changed during replay")
		}
	}
	header, body, ok := strings.Cut(strings.ReplaceAll(string(fixture), "\r\n", "\n"), "\n\n")
	if !ok || strings.Count(string(production), "module module_project_types;") != 1 {
		t.Fatal("missing private fixture boundary")
	}
	// Same-module temporary replay inspects retained map entries without exposing
	// private Plan fields or changing any production function signature.
	source := strings.Replace(string(production), "module module_project_types;", header, 1) + "\n" + body
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_project_types", "syntax", "scanner")
	// Only a temporary test module gets an unchecked constructor, so malformed
	// canonicality/coordinates reach build's real validation instead of being
	// refused by the fixture factory. Borrowed input objects are never mutated.
	typesPath := filepath.Join(root, "src", "module_types.fy")
	writeTestSource(t, root, "src/module_types.fy", string(read(typesPath))+"\npub fn Graph prefix_probe_raw(v::Vec<Type> types){return Graph{.types,.aggregates=false};}\n")
	writeTestSource(t, root, "src/main.fy", source)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 256<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			unchanged()
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			imagePath := filepath.Join(root, "project-type-prefix-"+profile.Name+".exe")
			if err = os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			t.Logf("native PE sha256=%x", sha256.Sum256(image.Bytes))
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, imagePath).CombinedOutput()
			if err != nil || !strings.HasSuffix(string(out), "mapping=equal edges=equal malformed=refused inputs=unchanged\n") {
				t.Fatalf("project prefix: %v output=%s", err, out)
			}
			unchanged()
			t.Log(string(out))
		})
	}
	unchanged()
}
