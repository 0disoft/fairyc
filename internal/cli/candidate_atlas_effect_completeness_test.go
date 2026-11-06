package cli

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestCandidateAtlasEffectCompleteness(t *testing.T) {
	const main = `module main;import project_atlas_header as header;import std::bytes as bytes;import std::io as io;
@effects(console) fn i32 main(){var usize work=0;discard io::write("[");var u32 version=1;
 while(version<=3){var usize external=0;while(external<=1){var usize borrowed=0;while(borrowed<=1){
  let usize templates=if(version==1){0}else{2};let usize implementations=if(version==1){0}else{3};
  let legacy=switch(header::completeness_observed_version(5,external,templates,implementations,version,borrowed==1,ref mut work)){case .ok(p):{p}case .err(_):{return 7;}};
  let collected=switch(header::completeness_collected_version(5,external,templates,implementations,version,borrowed==1,true,ref mut work)){case .ok(p):{p}case .err(_):{return 7;}};
  if(version!=1 || external!=0 || borrowed!=0){discard io::write(",");}
  discard io::write_bytes(bytes::view(ref legacy));discard io::write(",");discard io::write_bytes(bytes::view(ref collected));borrowed+=1;
 }external+=1;}version+=1;}
 discard io::write("]");return 0;}`
	candidateAtlasProbeSize(t, "project_atlas_header", main, 32<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		code, out := candidateAtlasProcess(t, image, nil)
		if code != 0 {
			t.Fatalf("effect completeness: %d %s", code, out)
		}
		var views [][]semanticsnapshot.Completeness
		if err := json.Unmarshal([]byte(out), &views); err != nil || len(views) != 24 {
			t.Fatalf("effect completeness views: %v %s", err, out)
		}
		for i := 0; i < len(views); i += 2 {
			before, after := views[i], views[i+1]
			count := 14
			if i < 8 {
				count = 12
			}
			if len(before) != count || len(after) != count {
				t.Fatalf("schema completeness count: %d/%d want %d", len(before), len(after), count)
			}
			found := false
			for j, old := range before {
				next := after[j]
				if old.Relation != "effects" {
					if !reflect.DeepEqual(old, next) {
						t.Fatalf("effect collection changed unrelated completeness: %+v %+v", old, next)
					}
					continue
				}
				found = true
				if !reflect.DeepEqual(old, semanticsnapshot.Completeness{Relation: "effects", Status: "unavailable", Reasons: []string{"candidate-atlas-not-collected"}}) || next.Relation != "effects" || next.Status != "exact" || next.Resolved != 0 || next.External != 0 || next.Unknown != 0 || len(next.Reasons) != 0 {
					t.Fatalf("uncollected/collected effects: %+v %+v", old, next)
				}
			}
			if !found {
				t.Fatal("missing effect completeness")
			}
		}
	})
}

func verifyCandidateAtlasEffectCompleteness(t *testing.T, p atlasmap.Partition) {
	t.Helper()
	for _, row := range p.Completeness() {
		if row.Relation != "effects" {
			continue
		}
		if row.Status != semanticsnapshot.CompletenessExact || row.Resolved != 0 || row.External != 0 || row.Unknown != 0 || len(row.Reasons) != 0 || slices.Contains(p.Header().Omissions, "effects:unavailable:candidate-atlas-not-collected") {
			t.Fatalf("source declaration effects must be exact: %+v", row)
		}
		return
	}
	t.Fatal("missing source declaration effect completeness")
}

// Scrambled spelling checks the entire sorted authority set. Function effects
// remain separate from cleanup effects and do not add executable relation rows.
const atlasAllDeclaredEffects = "storage_write,random,process,network,filesystem,external_call,environment,emit,console,clock,block_context,storage_read"

func atlasEffectProductionSources() []struct {
	name, source string
	minimum      int
} {
	return []struct {
		name, source string
		minimum      int
	}{
		{"effects-declarations", `module main;import answer as a;
@effects(` + atlasAllDeclaredEffects + `) fn void all(){}
@effects(console) resource Loud{i32 value;cleanup{discard 1;}}
struct Cell{i32 value;}
@effects(clock,filesystem) fn i32 Cell::get(ref Cell this){return this.value;}
@effects(filesystem,console,clock) fn i32 main(){let t=Loud{.value=1};let cell=Cell{.value=2};let view=ref cell;return view.get()+a::value();}`, 1},
		{"effects-foreign", `module main;
@effects(` + atlasAllDeclaredEffects + `) unsafe extern C fn i32 native_value(i32 n);
@effects(clock) unsafe extern C fn i32 unused_external();
@effects(` + atlasAllDeclaredEffects + `) fn i32 main(){return unsafe{native_value(1)};}`, 1},
		{"effects-generics", `module main;
@effects(clock) pub fn T id<T>(T value){return value;}
@effects(` + atlasAllDeclaredEffects + `) pub fn T unused<T>(T value){return value;}
@effects(console) pub resource Loud<T>{T value;cleanup{discard 1;}}
@effects(` + atlasAllDeclaredEffects + `) pub resource Unused<T>{T value;cleanup{discard 1;}}
@effects(clock,console) fn i32 main(){let t=Loud<i32>{.value=id<i32>(1)};discard id<u8>(2);return t.value;}`, 3},
	}
}
