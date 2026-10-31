package interfacereview

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"

	"github.com/0disoft/fairyc/internal/moduleinterface"
)

func reviewWorkspace(t *testing.T, sources map[string]string) string {
	t.Helper()
	root := t.TempDir()
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"example.review-test\"\nname = \"review-test\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
	if err := os.WriteFile(filepath.Join(root, "fairy.toml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, text := range sources {
		if err := os.WriteFile(filepath.Join(root, "src", name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func captureReview(t *testing.T, source string) Snapshot {
	t.Helper()
	snapshot := Capture(context.Background(), reviewWorkspace(t, map[string]string{"main.fy": source}))
	if snapshot.State != Bounded {
		t.Fatalf("capture: %+v", snapshot)
	}
	return snapshot
}

func TestTypedInterfaceReviewFindsSignatureEffectsAndLayoutChanges(t *testing.T) {
	for _, test := range []struct{ name, before, after, field string }{
		{"effects", "pub fn i32 value(i32 x) {return x;}", "@effects(console) pub fn i32 value(i32 x) {return x;}", "effects"},
		{"parameters", "pub fn i32 value(i32 x) {return x;}", "pub fn i32 value(u32 x) {return 42;}", "parameters"},
		{"result", "pub fn i32 value() {return 42;}", "pub fn u32 value() {return 42;}", "result-type"},
		{"unsafe", "pub fn i32 value() {return 42;}", "pub unsafe fn i32 value() {return 42;}", "unsafe"},
		{"field-order", "pub struct Pair {i32 a; u32 b;}", "pub struct Pair {u32 b; i32 a;}", "fields"},
		{"repr", "pub struct Pair {i32 a;}", "@repr(C) pub struct Pair {i32 a;}", "repr-c"},
		{"variants", "pub enum State { ready; failed; }", "pub enum State { failed; ready; }", "variants"},
		{"link-name", "@link_name(\"old_v1\") pub unsafe extern C fn i32 raw(i32 x);", "@link_name(\"new_v1\") pub unsafe extern C fn i32 raw(i32 x);", "link-name"},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := captureReview(t, "module main; "+test.before+" fn i32 main(){return 0;}")
			after := captureReview(t, "module main; "+test.after+" fn i32 main(){return 0;}")
			report := Compare(before, after)
			if report.State != Bounded || report.Changed == nil || !*report.Changed || len(report.Changes) != 1 || report.Changes[0].State != Exact || !slices.Contains(report.Changes[0].Fields, test.field) {
				t.Fatalf("report: %+v", report)
			}
			if test.field == "effects" && !slices.Equal(report.Changes[0].EffectsAdded, []string{"console"}) {
				t.Fatal("effect delta lost")
			}
		})
	}
}

func TestInterfaceReviewIgnoresPrivateBodyAndDeclarationOrder(t *testing.T) {
	before := captureReview(t, "module main; pub fn i32 first(){return 1;} pub fn i32 second(){return 2;} fn i32 private_body(){return 3;}")
	after := captureReview(t, "module main; fn i32 private_body(){return 999;} pub fn i32 second(){return 22;} pub fn i32 first(){return 11;}")
	report := Compare(before, after)
	if report.Changed == nil || *report.Changed || report.BehaviorCompatibility != Unknown || len(report.Changes) != 0 || before.SourceDigest == after.SourceDigest {
		t.Fatalf("private/body delta: %+v", report)
	}
	a, _ := json.Marshal(report)
	b, _ := json.Marshal(Compare(before, after))
	if string(a) != string(b) {
		t.Fatal("nondeterministic comparison")
	}
}

func TestInterfaceReviewDoesNotInferRenamesOrMissingFacts(t *testing.T) {
	before := captureReview(t, "module main; pub fn i32 old_name(){return 42;}")
	after := captureReview(t, "module main; pub fn i32 new_name(){return 42;}")
	report := Compare(before, after)
	if len(report.Changes) != 2 || report.Changes[0].Action != "added" || report.Changes[1].Action != "removed" {
		t.Fatalf("rename guessed: %+v", report)
	}
	invalid := Capture(context.Background(), reviewWorkspace(t, map[string]string{"main.fy": "module main; pub fn i32 broken(){return false;}"}))
	report = Compare(before, invalid)
	if report.State != Unknown || report.Changed != nil || len(report.Changes) != 0 {
		t.Fatalf("invalid after became removal: %+v", report)
	}
	after.Package.UID = "example.other"
	if report := Compare(before, after); report.State != Unknown || report.Changed != nil || len(report.Changes) != 0 {
		t.Fatal("different package compared")
	}
	after = before
	after.ProjectionHash = strings.Repeat("0", 64)
	if report := Compare(before, after); report.State != Unknown || len(report.Changes) != 0 {
		t.Fatal("invalid projection accepted")
	}
}

func TestGenericModuleIsExplicitlyOmittedOnBothSides(t *testing.T) {
	ctx := context.Background()
	before := Capture(ctx, reviewWorkspace(t, map[string]string{"ordinary.fy": "module ordinary; pub fn i32 value(){return 1;}", "generic.fy": "module generic; pub fn T identity<T>(T x){return x;}"}))
	after := Capture(ctx, reviewWorkspace(t, map[string]string{"ordinary.fy": "module ordinary; @effects(console) pub fn i32 value(){return 2;}", "generic.fy": "module generic; pub fn i32 identity(i32 x){return x;}"}))
	if before.State != Partial || after.State != Bounded {
		t.Fatalf("snapshots: %+v %+v", before, after)
	}
	report := Compare(before, after)
	if report.State != Partial || report.Changed == nil || !*report.Changed || len(report.Changes) != 1 || report.Changes[0].Module != "ordinary" {
		t.Fatalf("unknown module became additions/removals: %+v", report)
	}
	after = Capture(ctx, reviewWorkspace(t, map[string]string{"ordinary.fy": "module ordinary; pub fn i32 value(){return 1;}", "generic.fy": "module generic; pub fn T identity<T>(T x){let T copy=x;return copy;}"}))
	report = Compare(before, after)
	if report.State != Partial || report.Changed != nil || len(report.Changes) != 0 {
		t.Fatal("partial equality presented as no change")
	}
}

func TestCompareResourceEffectsAndOpaqueStateLayout(t *testing.T) {
	makeSnapshot := func(effect string) Snapshot {
		descriptor := moduleinterface.Descriptor{Modules: []moduleinterface.Module{{Path: "api", Items: []moduleinterface.Item{{Kind: "resource", Name: "Handle", Abilities: []string{"drop"}, Effects: []string{effect}}}}}}
		hash, err := moduleinterface.Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		return Snapshot{State: Bounded, Package: Package{UID: "example.review-test"}, ProjectionHash: hash.String(), Descriptor: &descriptor}
	}
	report := Compare(makeSnapshot("filesystem"), makeSnapshot("console"))
	if len(report.Changes) != 1 || !slices.Equal(report.Changes[0].EffectsAdded, []string{"console"}) || !slices.Equal(report.Changes[0].EffectsRemoved, []string{"filesystem"}) {
		t.Fatalf("resource authority change: %+v", report)
	}
	a, b := makeSnapshot("console"), makeSnapshot("console")
	a.Descriptor.StateLayouts = []moduleinterface.StateLayout{{ContractUID: "example.review-state", Hash: contractstate.LayoutHash{1}}}
	b.Descriptor.StateLayouts = []moduleinterface.StateLayout{{ContractUID: "example.review-state", Hash: contractstate.LayoutHash{2}}}
	for _, snapshot := range []*Snapshot{&a, &b} {
		hash, err := moduleinterface.Compute(*snapshot.Descriptor)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.ProjectionHash = hash.String()
	}
	report = Compare(a, b)
	if len(report.Changes) != 1 || report.Changes[0].Kind != "state-layout" || report.Changes[0].BeforeLayout.Hash == report.Changes[0].AfterLayout.Hash {
		t.Fatal("opaque state identity change lost")
	}
}

func TestCaptureCanceledOrMissingInputHasNoConclusion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, snapshot := range []Snapshot{Capture(ctx, t.TempDir()), Capture(context.Background(), filepath.Join(t.TempDir(), "absent"))} {
		if snapshot.State != Unknown || snapshot.Descriptor != nil || len(snapshot.Issues) == 0 {
			t.Fatalf("unknown lost: %+v", snapshot)
		}
	}
}
