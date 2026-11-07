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

func TestCandidateProjectMetadataMap(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows metadata map native replay; Linux execution is batched")
	}
	read := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	productionPath := filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_project_metadata.fy")
	fixturePath := filepath.Join("testdata", "project_metadata_map", "main.fy")
	production, fixture := read(productionPath), read(fixturePath)
	productionHash, fixtureHash := sha256.Sum256(production), sha256.Sum256(fixture)
	t.Logf("production sha256=%x fixture sha256=%x", productionHash, fixtureHash)
	unchanged := func() {
		t.Helper()
		if sha256.Sum256(read(productionPath)) != productionHash || sha256.Sum256(read(fixturePath)) != fixtureHash {
			t.Fatal("metadata map inputs changed during replay")
		}
	}
	header, body, ok := strings.Cut(strings.ReplaceAll(string(fixture), "\r\n", "\n"), "\n\n")
	if !ok || strings.Count(string(production), "module module_project_metadata;") != 1 {
		t.Fatal("missing private fixture boundary")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_project_metadata", "module_project_compile", "scanner", "sha256")
	irPath := filepath.Join(root, "src", "module_unit_body_ir_model.fy")
	// Test-only transformations operate on fresh owned real-source IR before it
	// becomes a frozen input to both builders. The product modules stay unchanged.
	extra := `
fn v::Vec<Layout> metadata_map_probe_order(v::Vec<Layout> input,usize fields){var v::Vec<Layout> layouts=input;if(fields==128){var usize i=0;let count=v::len<Layout>(ref layouts);while(i<count/2){let a=*v::at<Layout>(ref layouts,i);let b=*v::at<Layout>(ref layouts,count-i-1);{let ref mut Layout slot=v::at_mut<Layout>(ref mut layouts,i);*slot=b;};{let ref mut Layout slot=v::at_mut<Layout>(ref mut layouts,count-i-1);*slot=a;};i+=1;}}return layouts;}
`
	irSource := string(read(irPath))
	if !strings.Contains(irSource, ".bodies,.layouts,.fields") {
		t.Fatal("missing test-only IR constructor splice")
	}
	irSource = strings.ReplaceAll(irSource, ".bodies,.layouts,.fields", ".bodies,.layouts=metadata_map_probe_order(layouts,v::len<interfaces::Field>(ref fields)),.fields")
	writeTestSource(t, root, "src/module_unit_body_ir_model.fy", irSource+extra)
	writeTestSource(t, root, "src/main.fy", strings.Replace(string(production), "module module_project_metadata;", header, 1)+"\n"+body)
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
			imagePath := filepath.Join(root, "metadata-map-"+profile.Name+".exe")
			if err = os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			t.Logf("native PE sha256=%x", sha256.Sum256(image.Bytes))
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, imagePath).CombinedOutput()
			if err != nil || !strings.HasSuffix(string(out), "metadata=equal maps=equal conflict=refused inputs=unchanged\n") {
				t.Fatalf("metadata map: %v output=%s", err, out)
			}
			unchanged()
			t.Log(string(out))
		})
	}
	unchanged()
}
