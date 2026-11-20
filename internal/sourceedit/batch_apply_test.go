package sourceedit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestApplyBatchRestoresEarlierFilesOnPublicationFailure(t *testing.T) {
	for _, failRollback := range []bool{false, true} {
		name := "restore"
		if failRollback {
			name = "preserve recovery"
		}
		t.Run(name, func(t *testing.T) {
			first := editApplyFixture(t)
			const second = "module other;fn i32 helper(){return 2;}\n"
			editFile(t, first.root, "src/other.fy", second)
			workspace, err := workspaceloader.Load(first.root)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := PlanBodies(workspace, []BodyEdit{
				{Module: "main", Function: "helper", Expected: Digest(first.before), Body: []byte("{return 1;}")},
				{Module: "other", Function: "helper", Expected: Digest([]byte(second)), Body: []byte("{return 3;}")},
			})
			if err != nil {
				t.Fatal(err)
			}
			renameFailure := errors.New("injected rename failure")
			calls := 0
			err = plan.ApplyWithOptions(ApplyOptions{Rename: func(root *os.Root, from, to string) error {
				calls++
				if calls == 2 || failRollback && calls == 3 {
					return renameFailure
				}
				return root.Rename(from, to)
			}})
			if !errors.Is(err, renameFailure) || calls != 3 || plan.Report.Edits[0].Applied {
				t.Fatalf("publication failure = %v, renames = %d", err, calls)
			}
			wantMain := first.before
			lock := filepath.Join(first.root, editLock)
			if failRollback {
				wantMain = first.after
				if !strings.Contains(err.Error(), "batch recovery required") {
					t.Fatal(err)
				}
				entries, err := os.ReadDir(lock)
				if err != nil || len(entries) != 1 {
					t.Fatalf("recovery journal missing: %v %v", entries, err)
				}
				journal, err := os.ReadFile(filepath.Join(lock, entries[0].Name()))
				if err != nil || !strings.Contains(string(journal), Digest(first.before)) {
					t.Fatalf("journal lost original identity: %s %v", journal, err)
				}
				stages, err := filepath.Glob(filepath.Join(first.root, "src", ".tmp-*"))
				if err != nil || len(stages) != 2 {
					t.Fatalf("original staging files missing: %v %v", stages, err)
				}
			} else {
				if _, err := os.Lstat(lock); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("restored batch left lock: %v", err)
				}
				stages, err := filepath.Glob(filepath.Join(first.root, "src", ".tmp-*"))
				if err != nil || len(stages) != 0 {
					t.Fatalf("restored batch left staging files: %v %v", stages, err)
				}
			}
			for path, want := range map[string]string{"src/main.fy": string(wantMain), "src/other.fy": second, "fairy.lock": "retain lock bytes\n"} {
				got, err := os.ReadFile(filepath.Join(first.root, filepath.FromSlash(path)))
				if err != nil || string(got) != want {
					t.Fatalf("%s = %q, expected %q (%v)", path, got, want, err)
				}
			}
		})
	}
}
