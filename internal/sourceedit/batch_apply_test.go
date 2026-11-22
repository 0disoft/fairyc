package sourceedit

import (
	"encoding/json"
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
			publishFailure := errors.New("injected publish failure")
			rollbackFailure := errors.New("injected rollback failure")
			calls := 0
			err = plan.ApplyWithOptions(ApplyOptions{Rename: func(root *os.Root, from, to string) error {
				calls++
				if calls == 2 {
					return publishFailure
				}
				if failRollback && calls == 3 {
					return rollbackFailure
				}
				return root.Rename(from, to)
			}})
			if !errors.Is(err, publishFailure) || errors.Is(err, rollbackFailure) != failRollback || calls != 3 ||
				plan.Report.Edits[0].Applied || plan.Report.Edits[1].Applied {
				t.Fatalf("publication failure = %v, renames = %d", err, calls)
			}
			wantMain := first.before
			lock := filepath.Join(first.root, editLock)
			if failRollback {
				wantMain = first.after
				checkBatchRecoveryJournal(t, first.root, err, map[string]batchRecoveryContents{
					"src/main.fy":  {string(first.before), string(first.after)},
					"src/other.fy": {second, strings.Replace(second, "return 2", "return 3", 1)},
				})
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

type batchRecoveryContents struct {
	before, after string
}

func checkBatchRecoveryJournal(t *testing.T, root string, applyErr error, want map[string]batchRecoveryContents) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, editLock))
	if err != nil || len(entries) != 1 {
		t.Fatalf("recovery journal missing: %v %v", entries, err)
	}
	journalPath := filepath.Join(editLock, entries[0].Name())
	if !strings.Contains(applyErr.Error(), "batch recovery required; inspect "+filepath.ToSlash(journalPath)+":") {
		t.Fatalf("error does not identify recovery journal %q: %v", journalPath, applyErr)
	}
	data, err := os.ReadFile(filepath.Join(root, journalPath))
	if err != nil {
		t.Fatal(err)
	}
	var journal struct {
		Schema string `json:"schema"`
		Files  []struct {
			Path   string `json:"path"`
			Backup string `json:"backup"`
			Before string `json:"beforeSha256"`
			After  string `json:"afterSha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &journal); err != nil || journal.Schema != "fairyc.edit-batch-recovery/v1" || len(journal.Files) != len(want) {
		t.Fatalf("invalid recovery journal: %s (%v)", data, err)
	}
	for _, file := range journal.Files {
		expected, ok := want[file.Path]
		if !ok || file.Before != Digest([]byte(expected.before)) || file.After != Digest([]byte(expected.after)) {
			t.Fatalf("invalid recovery record: %+v", file)
		}
		delete(want, file.Path)
		backup := filepath.FromSlash(file.Backup)
		if filepath.Dir(backup) != filepath.Dir(filepath.FromSlash(file.Path)) || !strings.HasPrefix(filepath.Base(backup), ".tmp-") {
			t.Fatalf("invalid backup path for %s: %q", file.Path, file.Backup)
		}
		got, err := os.ReadFile(filepath.Join(root, backup))
		if err != nil || string(got) != expected.before {
			t.Fatalf("unusable backup for %s: %q (%v)", file.Path, got, err)
		}
	}
}
