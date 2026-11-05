package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func candidateAtlasFamilyWorkspace(t *testing.T) string {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, atlasTypeSource+atlasSchemaExtra+`
unsafe fn i32 ffi_root(i32 value){return renamed_external(value);}
fn i32 bridge(){return unsafe{ffi_root(1)};}
fn i32 resource_use(){let token=Token{.value=Pair{.value=4}};return token.value.value;}
const i32 ignored_constant=1;static_assert(ignored_constant==1);
`)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", `module answer;
pub struct Pair{i32 value;}
pub unsafe extern C fn i32 native_value();
pub fn i32 value(){return unsafe{native_value()};}
`)
	// File byte order differs from module order here: a/b.fy < a0.fy,
	// while module a0 < a::b. Both tables must retain their own canonical order.
	writeWorkspaceFile(t, root, "src/a0.fy", "module a0;fn i32 zero(){return 0;}")
	writeWorkspaceFile(t, root, "src/a/b.fy", "module a::b;fn i32 nested(){return 0;}")
	return root
}

func verifyCandidateAtlasFamilies(t *testing.T, image string) {
	t.Helper()
	root := candidateAtlasFamilyWorkspace(t)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("Go family oracle: %d %s", code, &diagnostic)
	}
	index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	selected := map[string]bool{
		"main": true, "scalars": true, "Pair": true, "Container": true,
		"Token": true, "Loud": true, "Empty": true, "Payload": true,
		"unused_external": true, "renamed_external": true, "native_value": true,
		"ffi_root": true, "value": true, "zero": true, "nested": true, "resource_use": true,
	}
	before := remoteTree(t, root)
	selections := 0
	for _, p := range prepared.partitions {
		for _, symbol := range p.Symbols() {
			if !selected[symbol.Declaration.Name] {
				continue
			}
			selections++
			for _, selector := range []string{symbol.Visible, symbol.Full} {
				args := append([]string{"experimental", "atlas-impact"}, impactArgs(p.Header().PackageUID, selector)...)
				// Two resolved and two external calls: only the resolved ones
				// consume the documented workspace call-join allowance.
				args = replaceWorkspaceSessionArgument(args, "--max-calls", "2")
				code, out, stderr := fetchCLIProcess(t, image, root, args, "")
				if code != 0 || stderr != "" {
					t.Fatalf("family %s: %d %s %s", symbol.Declaration.Name, code, out, stderr)
				}
				var got atlasmap.ImpactResult
				if err := json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatalf("family output: %v %s", err, out)
				}
				want, err := index.Impact(p.Header().PackageInstance, selector, 8, 64)
				if err != nil || got.State != "partial" || !reflect.DeepEqual(got.Subject, want.Subject) || !reflect.DeepEqual(got.Nodes, want.Nodes) || got.TraversalComplete != want.TraversalComplete || !reflect.DeepEqual(got.Omissions, want.Omissions) || len(got.Contexts) != len(want.Contexts) {
					t.Fatalf("family %s differs: err=%v\ngot %s\nwant %+v", symbol.Declaration.Name, err, out, want)
				}
				for i, c := range got.Contexts {
					w := want.Contexts[i]
					verifyCandidateAtlasFamilyContext(t, c, w, compiler)
				}
			}
		}
	}
	if selections != 17 {
		t.Fatalf("family selections=%d", selections)
	}
	for _, p := range prepared.partitions {
		for _, symbol := range p.Symbols() {
			if symbol.Declaration.Name != "main" {
				continue
			}
			args := append([]string{"experimental", "atlas-impact"}, impactArgs(p.Header().PackageUID, symbol.Visible)...)
			args = replaceWorkspaceSessionArgument(args, "--max-calls", "1")
			code, out, stderr := fetchCLIProcess(t, image, root, args, "")
			if code != 2 || out != "" || stderr == "" {
				t.Fatalf("resolved-call cutoff: %d %q %q", code, out, stderr)
			}
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("family queries changed workspace bytes")
	}
}

func verifyCandidateAtlasFamilyContext(t *testing.T, got, want atlasmap.QueryContext, compiler string) {
	t.Helper()
	// Identity and observed calls come from the independent Go source oracle.
	// The candidate explicitly reports its narrower non-call collection scope.
	if len(want.Completeness) != 1 && len(want.Completeness) != 3 || want.Completeness[0].Relation != "calls" {
		t.Fatalf("unexpected Go call oracle: %+v", want.Completeness)
	}
	want.CompilerBuild = compiler
	want.Generation = "partial"
	if len(want.Completeness) == 1 {
		want.Completeness = append(want.Completeness[:1:1],
			semanticsnapshot.Completeness{Relation: "generic-specializations", Status: "exact"},
			semanticsnapshot.Completeness{Relation: "generic-template-bodies", Status: "unavailable", Reasons: []string{"bodies-checked-only-after-specialization"}},
		)
	} else if want.Completeness[1].Relation != "generic-specializations" || want.Completeness[2].Relation != "generic-template-bodies" {
		t.Fatalf("unexpected Go generic oracle: %+v", want.Completeness)
	}
	want.MapOmissions = []string{"generic-template-bodies:unavailable:bodies-checked-only-after-specialization"}
	for _, relation := range []string{"allocates", "borrows", "effects", "moves", "reads", "returns-error", "test-calls", "uses-unsafe", "writes"} {
		want.MapOmissions = append(want.MapOmissions, relation+":unavailable:candidate-atlas-not-collected")
	}
	if want.Completeness[0].External != 0 {
		want.MapOmissions = append(want.MapOmissions, "calls:partial:external-ffi")
	}
	sort.Strings(want.MapOmissions)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("family context %s differs\ngot %#v\nwant %#v", got.PackageUID, got, want)
	}
}
