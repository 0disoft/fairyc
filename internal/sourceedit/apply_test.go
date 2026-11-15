package sourceedit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/durablefile"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func editApplyFixture(t *testing.T) Plan {
	t.Helper()
	root := t.TempDir()
	editFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.edit"
name = "edit"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	const original = "module main;fn i32 helper(){return 0;}fn i32 main(){return helper();}\n"
	editFile(t, root, "src/main.fy", original)
	editFile(t, root, "fairy.lock", "retain lock bytes\n")
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanBody(workspace, "main", "helper", Digest([]byte(original)), []byte("{return 1;}"))
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestApplyRejectsSameSizeStageMutation(t *testing.T) {
	plan := editApplyFixture(t)
	err := plan.ApplyWithOptions(ApplyOptions{Close: func(file *os.File) error {
		if err := file.Close(); err != nil {
			return err
		}
		tampered := strings.Replace(string(plan.after), "return 1", "return 9", 1)
		return os.WriteFile(file.Name(), []byte(tampered), 0600)
	}})
	if err == nil || plan.Report.Applied {
		t.Fatalf("unverified staged bytes published: applied=%v err=%v", plan.Report.Applied, err)
	}
	assertEditFile(t, plan.root, plan.path, plan.before)
}

func assertEditFile(t *testing.T, root, name string, want []byte) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil || !bytes.Equal(data, want) {
		t.Fatalf("file %s changed: bytes=%q err=%v", name, data, err)
	}
}

func assertEditCleanup(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(root, ".fairyc-edit.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("edit lock remained: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "src"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "main.fy" {
		t.Fatalf("edit staging remained: entries=%v err=%v", entries, err)
	}
	assertEditFile(t, root, "fairy.lock", []byte("retain lock bytes\n"))
}

func TestApplyRejectsStageReplacement(t *testing.T) {
	plan := editApplyFixture(t)
	var stage string
	err := plan.ApplyWithOptions(ApplyOptions{Close: func(file *os.File) error {
		if err := file.Close(); err != nil {
			return err
		}
		stage = file.Name()
		if err := os.Rename(stage, stage+".held"); err != nil {
			t.Fatal(err)
		}
		return os.WriteFile(stage, plan.after, 0600)
	}})
	if !errors.Is(err, durablefile.ErrIdentityChanged) || plan.Report.Applied {
		t.Fatalf("replacement stage published: applied=%v err=%v", plan.Report.Applied, err)
	}
	assertEditFile(t, plan.root, plan.path, plan.before)
	data, readErr := os.ReadFile(stage)
	if readErr != nil || !bytes.Equal(data, plan.after) {
		t.Fatalf("replacement removed: %v", readErr)
	}
}

func TestApplyCommitAndCancellationBoundaries(t *testing.T) {
	for _, phase := range []string{"success", "before rename", "after rename", "rename failed"} {
		t.Run(phase, func(t *testing.T) {
			plan := editApplyFixture(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("edit publication interrupted")
			policy := ApplyOptions{Context: ctx}
			var reused string
			switch phase {
			case "before rename":
				policy.Close = func(file *os.File) error { err := file.Close(); cancel(cause); return err }
			case "after rename":
				policy.Rename = func(root *os.Root, from, to string) error {
					if err := root.Rename(from, to); err != nil {
						return err
					}
					reused = from
					if err := root.WriteFile(from, []byte("later user file"), 0600); err != nil {
						t.Fatal(err)
					}
					cancel(cause)
					return nil
				}
			case "rename failed":
				policy.Rename = func(*os.Root, string, string) error { return cause }
			}
			var err error
			if phase == "success" {
				err = plan.Apply()
			} else {
				err = plan.ApplyWithOptions(policy)
			}
			committed := phase == "success" || phase == "after rename"
			if plan.Report.Applied != committed || phase == "success" && err != nil || phase != "success" && !errors.Is(err, cause) {
				t.Fatalf("commit boundary wrong: applied=%v err=%v", plan.Report.Applied, err)
			}
			if ctx.Err() != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
			want := plan.before
			if committed {
				want = plan.after
			}
			assertEditFile(t, plan.root, plan.path, want)
			if reused != "" {
				assertEditFile(t, plan.root, reused, []byte("later user file"))
				if err := os.Remove(filepath.Join(plan.root, reused)); err != nil {
					t.Fatal(err)
				}
			}
			assertEditCleanup(t, plan.root)
		})
	}
}

func TestEditStageWriteFailures(t *testing.T) {
	for _, phase := range []string{"write", "chmod", "sync", "close", "cancel write", "short write"} {
		t.Run(phase, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "stage-")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			failure := errors.New("staging failed")
			closes := 0
			policy, err := editApplyPolicy([]ApplyOptions{{Context: ctx, Close: func(file *os.File) error {
				closes++
				err := file.Close()
				if phase == "close" {
					return errors.Join(err, failure)
				}
				return err
			}}})
			if err != nil {
				t.Fatal(err)
			}
			switch phase {
			case "write":
				policy.Write = func(*os.File, []byte) (int, error) { return 0, failure }
			case "chmod":
				policy.Chmod = func(*os.File, os.FileMode) error { return failure }
			case "sync":
				policy.Sync = func(*os.File) error { return failure }
			case "cancel write":
				policy.Write = func(file *os.File, data []byte) (int, error) {
					n, err := file.Write(data)
					cancel(failure)
					return n, err
				}
			case "short write":
				policy.Write = func(_ *os.File, data []byte) (int, error) { return len(data) - 1, nil }
			}
			err = writeEditStage(file, []byte("verified"), 0600, policy)
			want := failure
			if phase == "short write" {
				want = io.ErrShortWrite
			}
			if !errors.Is(err, want) || closes != 1 {
				t.Fatalf("lost failure or close: closes=%d err=%v", closes, err)
			}
			if phase == "cancel write" && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
		})
	}
}

func TestEditStageVerificationByteBoundaries(t *testing.T) {
	for _, sizes := range [][2]int{{32767, 32767}, {32768, 32768}, {32769, 32769}, {8, 7}, {8, 8}, {8, 9}} {
		t.Run(fmt.Sprint(sizes), func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			file, name, err := durablefile.CreateTemporary(root, ".")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			identity, err := file.Stat()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write(bytes.Repeat([]byte("x"), sizes[1])); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			err = verifyEditStage(context.Background(), root, name, identity, bytes.Repeat([]byte("x"), sizes[0]))
			if sizes[0] == sizes[1] && err != nil || sizes[0] != sizes[1] && !errors.Is(err, durablefile.ErrIdentityChanged) {
				t.Fatalf("verification boundary: %v", err)
			}
		})
	}
}

func TestApplyPreconditions(t *testing.T) {
	var plan *Plan
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("pre-cancelled edit")
	cancel(cause)
	if err := plan.ApplyWithOptions(ApplyOptions{Context: ctx}); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatalf("lost pre-cancellation: %v", err)
	}
	if err := plan.Apply(); err == nil {
		t.Fatal("nil plan accepted")
	}
	if err := plan.ApplyWithOptions(ApplyOptions{}, ApplyOptions{}); err == nil {
		t.Fatal("extra options accepted")
	}
}
