package cli

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestCandidateCleanupRecipeIndex(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows cleanup recipe index native probe; Linux execution is batched")
	}
	read := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_unit_cleanup_mir_lower.fy")
	fixturePath := filepath.Join("testdata", "cleanup_recipe_index", "main.fy")
	production, fixture := read(path), read(fixturePath)
	productionHash, fixtureHash := sha256.Sum256(production), sha256.Sum256(fixture)
	t.Logf("lower sha256=%x fixture sha256=%x", productionHash, fixtureHash)
	unchanged := func() {
		t.Helper()
		if sha256.Sum256(read(path)) != productionHash || sha256.Sum256(read(fixturePath)) != fixtureHash {
			t.Fatal("cleanup index inputs changed during replay")
		}
	}
	replace := func(s, from, to string) string {
		t.Helper()
		if strings.Count(s, from) != 1 {
			t.Fatalf("expected one private cleanup test splice: %q", from)
		}
		return strings.Replace(s, from, to, 1)
	}
	source := strings.ReplaceAll(string(production), "\r\n", "\n")
	start := strings.Index(source, "fn Result<ir::Program,ir::Problem> lower_policy(")
	end := strings.Index(source, "\npub fn Result<ir::Program,ir::Problem> lower_checked_catalog(")
	if start < 0 || end <= start {
		t.Fatal("missing private lowering boundary")
	}
	// Retain the production policy and field/operand/error ordering; the baseline
	// substitutes only the original ordered shape and frame scans. Both versions
	// execute in one image over the very same borrowed, real checked inputs.
	baseline := replace(source[start:end], " lower_policy(", " lower_policy_scan(")
	baseline = replace(baseline, "    var Lookup index=Lookup{.rows=v::empty<Selection>()};\n", "")
	baseline = replace(baseline,
		`if(v::len<Selection>(ref index.rows)==0){index=switch(index_recipes(values::type_count(typed),interfaces::shape_records(imported),ref frames,t.module_index,ref mut work)){case .ok(p):{p}case .err(p):{return .err(p);}};}if(!m::charge(ref mut work,1)){return .err(fail(1002,t.module_index,0,0));}let found=v::at<Selection>(ref index.rows,i).shape;var usize j=0;`,
		`var usize found=0;var usize j=0;while(j<interfaces::shape_count(imported)){if(!m::charge(ref mut work,1)){return .err(fail(1002,t.module_index,0,0));}let s=interfaces::shape_at(imported,j);if(s.type==i){found=j+1;}j+=1;}`)
	baseline = replace(baseline,
		`if(g.nonempty){g.frame=switch(recipe_frame(ref index,i,v::len<ir::Frame>(ref frames),t.module_index,s.start,s.end,ref mut work)){case .ok(frame):{frame}case .err(p):{return .err(p);}};}`,
		`if(g.nonempty){j=0;while(j<v::len<ir::Frame>(ref frames)){if(!m::charge(ref mut work,1)){return .err(fail(1002,t.module_index,s.start,s.end));}let f=*v::at<ir::Frame>(ref frames,j);if(f.kind==2 && f.receiver_type==i){if(g.frame!=0){return .err(fail(1001,t.module_index,s.start,s.end));}g.frame=j+1;}j+=1;}if(g.frame==0){return .err(fail(1001,t.module_index,s.start,s.end));}}`)
	// Keep private helpers private: merge the fixture with the unchanged production
	// functions in a temporary main module instead of expanding the product API.
	fixtureText := strings.ReplaceAll(string(fixture), "\r\n", "\n")
	header, body, ok := strings.Cut(fixtureText, "\n\n")
	if !ok {
		t.Fatal("missing private fixture header")
	}
	source = replace(source, "module module_unit_cleanup_mir_lower;", header)
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_unit_cleanup_mir_lower", "module_unit_bind", "module_unit_call_closure", "sha256")
	writeTestSource(t, root, "src/main.fy", source+"\n"+baseline+"\n"+body)
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
			imagePath := filepath.Join(root, "cleanup-index-"+profile.Name+".exe")
			if err = os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			t.Logf("native PE sha256=%x", sha256.Sum256(image.Bytes))
			code, out := candidateAtlasProcess(t, imagePath, nil)
			if code != 0 || !strings.HasSuffix(out, "selection=last frame=unique errors=ordered inputs=unchanged\n") {
				t.Fatalf("cleanup index: exit=%d output=%q", code, out)
			}
			unchanged()
			t.Log(out)
		})
	}
	unchanged()
}
