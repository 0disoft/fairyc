package cli

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestAtlasDeferredResultCalls(t *testing.T) {
	const prefix = "module main;fn Result<i32,i32> read(){return .ok(7);}fn void notice(){}"
	for _, f := range []struct {
		name, body              string
		owners, perOwner, total int
	}{
		{"cleanup", "resource Token{i32 value;cleanup{}}fn Result<i32,i32> exercise(){let token=Token{.value=1};defer notice();let n=read()?;return .ok(n);}fn i32 main(){discard exercise();return 0;}", 1, 2, 3},
		{"two-source-defers", "fn Result<i32,i32> exercise(){defer notice();{defer notice();discard read()?;}return .ok(0);}fn i32 main(){discard exercise();return 0;}", 1, 3, 4},
		{"generic", "fn Result<T,i32> exercise<T>(T value){defer notice();discard read()?;return .ok(value);}fn i32 main(){discard exercise<i32>(7);discard exercise<u32>(8);return 0;}", 2, 2, 6},
	} {
		t.Run(f.name, func(t *testing.T) {
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(writeWorkspaceSessionFixture(t, prefix+f.body), false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("deferred source: %d %s", code, &diagnostic)
			}
			owners, total := 0, 0
			for _, p := range prepared.partitions {
				encoded, err := atlasmap.EncodeV3(p, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				check := func(summary semanticsnapshot.FunctionSummary) {
					total += len(summary.Calls)
					if summary.Function.Module != "main" || summary.Function.Name != "exercise" {
						return
					}
					owners++
					if len(summary.Calls) != f.perOwner || len(summary.ReturnErrorSites) != 1 {
						t.Fatalf("deferred source-site inventory: %+v", summary)
					}
					seen := map[semanticsnapshot.Call]bool{}
					for _, call := range summary.Calls {
						if seen[call] {
							t.Fatalf("duplicate source call: %+v", call)
						}
						seen[call] = true
					}
				}
				for _, summary := range decoded.Functions() {
					check(summary)
				}
				for _, instance := range decoded.Specializations() {
					if instance.Aggregate == nil {
						check(instance.Function)
					}
					if instance.Cleanup != nil {
						check(*instance.Cleanup)
					}
				}
				for _, c := range decoded.Completeness() {
					if c.Relation == "calls" && (c.Status != semanticsnapshot.CompletenessExact || c.Resolved != f.total || c.External != 0 || c.Unknown != 0) {
						t.Fatalf("inflated source-call completeness: %+v, want %d", c, f.total)
					}
				}
			}
			if owners != f.owners || total != f.total {
				t.Fatalf("source-call inventory: owners=%d/%d calls=%d/%d", owners, f.owners, total, f.total)
			}
		})
	}
}
