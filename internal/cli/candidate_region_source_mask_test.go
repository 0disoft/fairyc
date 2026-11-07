package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateRegionSourceConsumersCheck(t *testing.T) {
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_unit_region_demands", "module_unit_checked_bodies")
	writeTestSource(t, root, "src/main.fy", `module main;
import module_unit_region_demands as regions;
import module_unit_checked_bodies as checked;
fn i32 main(){return 0;}
`)
	if code, out, err := runForTest([]string{"check"}, root); code != 0 || err != "" || !strings.HasPrefix(out, "check ok:") {
		t.Fatalf("actual source-count consumers: exit=%d stdout=%q stderr=%q", code, out, err)
	}
}

func TestCandidateRegionSourceMask(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_unit_region_demands.fy"))
	if err != nil {
		t.Fatal(err)
	}
	// Compile the actual private helper, without a second implementation or the
	// unrelated complete body-checker dependency closure.
	_, helper, ok := strings.Cut(string(data), "fn Result<bytes::Buffer,u32> selection(")
	if !ok {
		t.Fatal("missing production selection helper")
	}
	helper, _, ok = strings.Cut(helper, "fn Result<bool,")
	if !ok {
		t.Fatal("missing helper boundary")
	}
	main := "module main;\nimport candidate_limits as limits;\nimport std::bytes as bytes;\nfn Result<bytes::Buffer,u32> selection(" + helper + regionSourceMaskCases
	candidateAtlasProbeSize(t, "candidate_limits", main, 1<<10, func(t *testing.T, image string) {
		if code, out := candidateAtlasProcess(t, image, nil); code != 0 || out != "" {
			t.Fatalf("actual source-selection mask: exit=%d output=%q", code, out)
		}
	})
}

const regionSourceMaskCases = `
fn void check(usize count,usize first,usize stop){
 let mask=switch(selection(count,first,stop)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let view=bytes::view(ref mask);assert(view.len==count);var usize i=0;
 while(i<count){if(i>=first && i<stop){assert(view[i]==1);}else{assert(view[i]==0);}i+=1;}
}
fn void invalid(usize count,usize first,usize stop){switch(selection(count,first,stop)){case .ok(_):{assert(false);}case .err(code):{assert(code==9);}};}
fn i32 main(){
 check(1,0,1);check(512,511,512);check(513,512,513);check(646,512,646);
 check(1024,0,1024);check(1024,1023,1024);check(1024,0,1);
 invalid(0,0,0);invalid(1025,0,1);invalid(1024,1024,1024);invalid(1024,0,1025);invalid(1024,2,1);
 return 0;
}
`
