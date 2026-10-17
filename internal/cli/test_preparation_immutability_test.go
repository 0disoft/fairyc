package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestWorkspaceTestPreparationDoesNotMutateLoadedAST(t *testing.T) {
	for _, valid := range []bool{true, false} {
		name := "accepted"
		declaration := `@test @effects(console) fn void works() { assert(true); }`
		if !valid {
			name = "rejected"
			declaration = `@test @effects(console) fn i32 invalid() { return 0; }`
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeStableTestWorkspace(t, root)
			writeWorkspaceFile(t, root, "tests/reuse.fy", "module tests::reuse; "+declaration)
			workspace, err := workspaceloader.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			workspace, err = workspaceloader.WithPackageTests(workspace, ".")
			if err != nil {
				t.Fatal(err)
			}
			input, err := workspace.SemanticInput()
			if err != nil {
				t.Fatal(err)
			}
			hasTest := false
			for _, file := range input.Files {
				hasTest = hasTest || file.Test
			}
			if !hasTest {
				t.Fatal("loader dropped test source admission")
			}
			before, _ := json.Marshal(input.Files)
			var previous []workspaceTest
			for attempt := 0; attempt < 2; attempt++ {
				var stderr bytes.Buffer
				tests, _, code := prepareWorkspaceTests(workspace, ".", "", true, &stderr)
				if valid {
					if code != ExitSuccess || stderr.Len() != 0 || len(tests) != 1 {
						t.Fatalf("prepare: %d %s %+v", code, &stderr, tests)
					}
					if attempt > 0 && !reflect.DeepEqual(previous, tests) {
						t.Fatal("repeated discovery changed test identities or locations")
					}
					previous = tests
					checked, err := workspacecompiler.Check(workspace)
					if err != nil || !checked.Verified {
						t.Fatalf("reanalysis: %v %+v", err, checked.Semantic)
					}
				} else if code != ExitSourceDiagnostics || stderr.Len() == 0 {
					t.Fatalf("rejection changed on attempt %d: %d %s", attempt, code, &stderr)
				}
			}
			if valid {
				session, err := workspacecompiler.NewSourceSession(workspacecompiler.Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20})
				if err != nil {
					t.Fatal(err)
				}
				defer session.Close()
				cold, err := session.Check(workspace)
				if err != nil || !cold.Compiler.Verified {
					t.Fatalf("cold: %v %+v", err, cold.Cache)
				}
				warm, err := session.Check(workspace)
				if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != workspacecompiler.OutcomeAllHit {
					t.Fatalf("warm: %v %+v", err, warm.Cache)
				}
			}
			afterInput, err := workspace.SemanticInput()
			if err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(afterInput.Files)
			if !bytes.Equal(before, after) {
				t.Fatal("test preparation or reanalysis mutated original AST")
			}
		})
	}
}
