package packagesource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func TestPublishPreservesReplacedStageOnCancellation(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stage string
	during := &comparisonCheckContext{Context: ctx, onCheck: func(int) {
		if stage != "" {
			return
		}
		dir, err := root.Open("vendor")
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		names, readErr := dir.Readdirnames(-1)
		closeErr := dir.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			if !strings.HasPrefix(name, ".fairyc-source-") {
				continue
			}
			stage = path.Join("vendor", name)
			if err := root.Rename(stage, stage+".held"); err != nil {
				t.Fatal(err)
			}
			if err := root.Mkdir(stage, 0700); err != nil {
				t.Fatal(err)
			}
			if err := root.WriteFile(path.Join(stage, "keep.txt"), []byte("user data"), 0600); err != nil {
				t.Fatal(err)
			}
			cancel()
		}
	}}
	installed, err := publish(during, root, testPin(), bundle{files: map[string][]byte{"data": []byte("verified")}})
	if installed || !errors.Is(err, context.Canceled) || stage == "" {
		t.Fatalf("cancellation not exercised: installed=%v stage=%q err=%v", installed, stage, err)
	}
	if !errors.Is(err, durablefile.ErrIdentityChanged) {
		t.Fatalf("replacement not reported: %v", err)
	}
	data, err := root.ReadFile(path.Join(stage, "keep.txt"))
	if err != nil || string(data) != "user data" {
		t.Fatalf("replacement directory was deleted: data=%q err=%v", data, err)
	}
}

func sourceStageTestRoot(t *testing.T) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	return root
}

func assertNoSourceStage(t *testing.T, root *os.Root) {
	t.Helper()
	file, err := root.Open("vendor")
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	names, err := file.Readdirnames(-1)
	if err != nil || len(names) != 0 {
		t.Fatalf("unexpected vendor entries: %v err=%v", names, err)
	}
}

func TestPublishStageFailureBoundaries(t *testing.T) {
	for _, phase := range []string{"random", "write", "sync", "close", "rename"} {
		for _, mode := range []string{"failure", "cancel", "both"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				root := sourceStageTestRoot(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				failure, cause := errors.New("stage I/O failed"), errors.New("stop installing")
				interrupt := func() error {
					if mode != "failure" {
						cancel(cause)
					}
					if mode != "cancel" {
						return failure
					}
					return ctx.Err()
				}
				policy := sourcePublishOptions{}
				switch phase {
				case "random":
					policy.Random = func(buffer []byte) (int, error) { return len(buffer), interrupt() }
				case "write":
					policy.Write = func(file *os.File, data []byte) (int, error) {
						n, err := file.Write(data)
						return n, errors.Join(err, interrupt())
					}
				case "sync":
					policy.Sync = func(*os.File) error { return interrupt() }
				case "close":
					policy.Close = func(file *os.File) error { return errors.Join(file.Close(), interrupt()) }
				case "rename":
					policy.Rename = func(*os.Root, string, string) error { return interrupt() }
				}
				installed, err := publish(ctx, root, testPin(), bundle{files: map[string][]byte{"data": []byte("verified")}}, policy)
				if installed || err == nil {
					t.Fatalf("failed operation installed: %v %v", installed, err)
				}
				if mode != "cancel" && !errors.Is(err, failure) {
					t.Fatalf("lost I/O failure: %v", err)
				}
				if mode != "failure" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatalf("lost cancellation cause: %v", err)
				}
				assertNoSourceStage(t, root)
			})
		}
	}
}

func TestPublishStageNonceBoundaries(t *testing.T) {
	for _, count := range []int{15, 16, 17} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			root := sourceStageTestRoot(t)
			installed, err := publish(nil, root, testPin(), bundle{files: map[string][]byte{"data": []byte("verified")}}, sourcePublishOptions{Random: func(buffer []byte) (int, error) {
				if len(buffer) != 16 {
					t.Fatalf("nonce buffer=%d", len(buffer))
				}
				return count, nil
			}})
			if count == 16 {
				if !installed || err != nil {
					t.Fatalf("valid nonce rejected: %v %v", installed, err)
				}
			} else {
				if installed || !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Fatalf("invalid nonce admitted: %v %v", installed, err)
				}
				assertNoSourceStage(t, root)
			}
		})
	}
}

func TestPublishStageWriteCountBoundaries(t *testing.T) {
	for _, delta := range []int{-1, 0, 1} {
		t.Run(fmt.Sprint(delta), func(t *testing.T) {
			root := sourceStageTestRoot(t)
			installed, err := publish(nil, root, testPin(), bundle{files: map[string][]byte{"data": []byte("verified")}}, sourcePublishOptions{Write: func(file *os.File, data []byte) (int, error) {
				n, err := file.Write(data)
				return n + delta, err
			}})
			if delta == 0 {
				if !installed || err != nil {
					t.Fatalf("full write rejected: %v %v", installed, err)
				}
			} else {
				if installed || !errors.Is(err, io.ErrShortWrite) {
					t.Fatalf("invalid write count admitted: %v %v", installed, err)
				}
				assertNoSourceStage(t, root)
			}
		})
	}
}

func TestPublishStageCommitBoundary(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			root := sourceStageTestRoot(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("cancel after commit")
			var stage string
			removals := 0
			installed, err := publish(ctx, root, testPin(), bundle{files: map[string][]byte{"data": []byte("verified")}}, sourcePublishOptions{
				Rename: func(root *os.Root, from, to string) error {
					if err := root.Rename(from, to); err != nil {
						return err
					}
					stage = from
					if err := root.Mkdir(from, 0700); err != nil {
						t.Fatal(err)
					}
					if err := root.WriteFile(path.Join(from, "keep.txt"), []byte("later user data"), 0600); err != nil {
						t.Fatal(err)
					}
					if cancelled {
						cancel(cause)
					}
					return nil
				},
				RemoveAll: func(root *os.Root, name string) error { removals++; return root.RemoveAll(name) },
			})
			if !installed || removals != 0 || (!cancelled && err != nil) {
				t.Fatalf("commit lost ownership: installed=%v removals=%d err=%v", installed, removals, err)
			}
			if cancelled && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatalf("lost post-commit cancellation: %v", err)
			}
			for name, want := range map[string]string{"vendor/math/data": "verified", path.Join(stage, "keep.txt"): "later user data"} {
				data, err := root.ReadFile(name)
				if err != nil || string(data) != want {
					t.Fatalf("committed/replacement data lost: %s %q %v", name, data, err)
				}
			}
		})
	}
}

func TestPublishRejectsStageReplacementBeforeRename(t *testing.T) {
	root := sourceStageTestRoot(t)
	var stage string
	renames := 0
	installed, err := publish(nil, root, testPin(), bundle{files: map[string][]byte{"data": []byte("verified")}}, sourcePublishOptions{
		Close: func(file *os.File) error {
			if err := file.Close(); err != nil {
				return err
			}
			relative, err := filepath.Rel(root.Name(), filepath.Dir(file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			stage = filepath.ToSlash(relative)
			if err := root.Rename(stage, stage+".held"); err != nil {
				t.Fatal(err)
			}
			if err := root.Mkdir(stage, 0700); err != nil {
				t.Fatal(err)
			}
			return root.WriteFile(path.Join(stage, "keep.txt"), []byte("replacement"), 0600)
		},
		Rename: func(root *os.Root, from, to string) error { renames++; return root.Rename(from, to) },
	})
	if installed || renames != 0 || !errors.Is(err, durablefile.ErrIdentityChanged) {
		t.Fatalf("replacement published: installed=%v renames=%d err=%v", installed, renames, err)
	}
	data, err := root.ReadFile(path.Join(stage, "keep.txt"))
	if err != nil || string(data) != "replacement" {
		t.Fatalf("replacement not preserved: %q %v", data, err)
	}
}

func TestPublishStagePreconditionsAndCleanupFailure(t *testing.T) {
	t.Run("cancel before I/O", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if installed, err := publish(ctx, nil, testPin(), bundle{}); installed || !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-cancellation lost: %v %v", installed, err)
		}
	})
	t.Run("invalid options", func(t *testing.T) {
		if installed, err := publish(nil, nil, testPin(), bundle{}, sourcePublishOptions{}, sourcePublishOptions{}); installed || err == nil {
			t.Fatalf("extra options admitted: %v %v", installed, err)
		}
	})
	t.Run("nil root", func(t *testing.T) {
		if installed, err := publish(nil, nil, testPin(), bundle{}); installed || err == nil {
			t.Fatalf("nil root admitted: %v %v", installed, err)
		}
	})
	t.Run("existing stage", func(t *testing.T) {
		root := sourceStageTestRoot(t)
		stage := "vendor/.fairyc-source-" + strings.Repeat("0", 32)
		if err := root.MkdirAll(stage, 0700); err != nil {
			t.Fatal(err)
		}
		if err := root.WriteFile(path.Join(stage, "keep.txt"), []byte("old stage"), 0600); err != nil {
			t.Fatal(err)
		}
		installed, err := publish(nil, root, testPin(), bundle{}, sourcePublishOptions{Random: func(buffer []byte) (int, error) { return len(buffer), nil }})
		if installed || !errors.Is(err, os.ErrExist) {
			t.Fatalf("existing stage admitted: %v %v", installed, err)
		}
		data, err := root.ReadFile(path.Join(stage, "keep.txt"))
		if err != nil || string(data) != "old stage" {
			t.Fatalf("existing stage changed: %q %v", data, err)
		}
	})
	t.Run("cleanup failure", func(t *testing.T) {
		root := sourceStageTestRoot(t)
		writeErr, cleanupErr := errors.New("write failed"), errors.New("cleanup failed")
		installed, err := publish(nil, root, testPin(), bundle{files: map[string][]byte{"data": []byte("verified")}}, sourcePublishOptions{
			Write:     func(*os.File, []byte) (int, error) { return 0, writeErr },
			RemoveAll: func(*os.Root, string) error { return cleanupErr },
		})
		if installed || !errors.Is(err, writeErr) || !errors.Is(err, cleanupErr) {
			t.Fatalf("lost failure causes: %v %v", installed, err)
		}
	})
}
