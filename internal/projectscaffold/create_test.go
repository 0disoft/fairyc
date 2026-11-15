package projectscaffold

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateRollbackPreservesReplacement(t *testing.T) {
	for _, regular := range []bool{false, true} {
		t.Run(fmt.Sprintf("regular=%v", regular), func(t *testing.T) {
			parent := t.TempDir()
			spec := Spec{Name: "new-project", UID: "example.new-project"}
			target := filepath.Join(parent, spec.Name)
			marker := target
			if !regular {
				marker = filepath.Join(target, "user-data.txt")
			}
			want := []byte("another owner's data")
			writeErr := errors.New("injected write failure")
			closes := 0
			err := CreateWithOptions(parent, spec, CreateOptions{
				WriteFile: func(*os.Root, string, []byte) error { return writeErr },
				CloseProject: func(root *os.Root) error {
					closes++
					if err := root.Close(); err != nil {
						return err
					}
					// Reproduce a path replacement after close, before rollback.
					if err := os.Rename(target, target+".partial"); err != nil {
						t.Fatal(err)
					}
					if !regular {
						if err := os.Mkdir(target, 0755); err != nil {
							t.Fatal(err)
						}
					}
					if err := os.WriteFile(marker, want, 0600); err != nil {
						t.Fatal(err)
					}
					return nil
				},
			})
			got, readErr := os.ReadFile(marker)
			if readErr != nil || string(got) != string(want) {
				t.Fatalf("rollback deleted replacement: data=%q error=%v", got, readErr)
			}
			var failure *Error
			if !errors.Is(err, writeErr) || !errors.As(err, &failure) || failure.Reason != ReasonRollback || closes != 1 {
				t.Fatalf("ownership failure lost: closes=%d error=%v", closes, err)
			}
		})
	}
}

func TestCreateCancellationBoundaries(t *testing.T) {
	for _, after := range []int{0, 1, len(scaffoldFiles) - 1, len(scaffoldFiles), len(scaffoldFiles) + 1} {
		t.Run(fmt.Sprintf("afterWrite=%d", after), func(t *testing.T) {
			parent := t.TempDir()
			spec := Spec{Name: "new-project", UID: "example.new-project"}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller cancelled")
			if after == 0 {
				cancel(cause)
			}
			writes, closes := 0, 0
			err := CreateWithOptions(parent, spec, CreateOptions{
				Context: ctx,
				WriteFile: func(root *os.Root, name string, data []byte) error {
					writes++
					err := writeExclusive(root, name, data)
					if writes == after {
						cancel(cause)
					}
					return err
				},
				CloseProject: func(root *os.Root) error { closes++; return root.Close() },
			})
			if after > len(scaffoldFiles) {
				if err != nil || writes != len(scaffoldFiles) || closes != 1 {
					t.Fatalf("uncancelled creation failed: writes=%d closes=%d error=%v", writes, closes, err)
				}
				return
			}
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || writes != after {
				t.Fatalf("cancel boundary ignored: writes=%d closes=%d error=%v", writes, closes, err)
			}
			wantCloses := 1
			if after == 0 {
				wantCloses = 0
			}
			entries, readErr := os.ReadDir(parent)
			if readErr != nil || len(entries) != 0 || closes != wantCloses {
				t.Fatalf("cancelled creation leaked state: entries=%v closes=%d error=%v", entries, closes, readErr)
			}
		})
	}
}

func TestCreateCloseFailureAndCancellation(t *testing.T) {
	for _, cancelAtClose := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancelAtClose=%v", cancelAtClose), func(t *testing.T) {
			parent := t.TempDir()
			spec := Spec{Name: "new-project", UID: "example.new-project"}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			closeErr, cause := errors.New("close failed"), errors.New("cancel during close")
			err := CreateWithOptions(parent, spec, CreateOptions{Context: ctx, CloseProject: func(root *os.Root) error {
				if err := root.Close(); err != nil {
					t.Fatal(err)
				}
				if cancelAtClose {
					cancel(cause)
					return nil
				}
				return closeErr
			}})
			if cancelAtClose {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatalf("close cancellation lost: %v", err)
				}
			} else if !errors.Is(err, closeErr) {
				t.Fatalf("close error lost: %v", err)
			}
			entries, readErr := os.ReadDir(parent)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("close failure leaked project: %v %v", entries, readErr)
			}
		})
	}
}

func TestCreateWriteFailureKeepsAllCauses(t *testing.T) {
	parent := t.TempDir()
	writeErr, closeErr := errors.New("write failed"), errors.New("close failed")
	err := CreateWithOptions(parent, Spec{Name: "new-project", UID: "example.new-project"}, CreateOptions{
		WriteFile: func(root *os.Root, name string, data []byte) error {
			if err := writeExclusive(root, name, data[:1]); err != nil {
				t.Fatal(err)
			}
			return writeErr
		},
		CloseProject: func(root *os.Root) error { return errors.Join(root.Close(), closeErr) },
	})
	if !errors.Is(err, writeErr) || !errors.Is(err, closeErr) {
		t.Fatalf("failure cause lost: %v", err)
	}
	entries, readErr := os.ReadDir(parent)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("partial write leaked project: %v %v", entries, readErr)
	}
}

func TestCreateOptionsAndIdentityBounds(t *testing.T) {
	for _, size := range []int{126, 127, 128} {
		err := Validate(Spec{Name: strings.Repeat("a", size), UID: "example.app", Template: "win32"})
		if (err != nil) != (size > 127) {
			t.Fatalf("name bytes=%d error=%v", size, err)
		}
	}
	for _, size := range []int{254, 255, 256} {
		err := Validate(Spec{Name: "app", UID: "a." + strings.Repeat("a", size-2), Template: "win32"})
		if (err != nil) != (size > 255) {
			t.Fatalf("UID bytes=%d error=%v", size, err)
		}
	}
	if err := CreateWithOptions(t.TempDir(), Spec{}, CreateOptions{}, CreateOptions{}); err == nil {
		t.Fatal("multiple options values accepted")
	}
}

func TestRollbackMissingAndUnprovenIdentity(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing=%v", missing), func(t *testing.T) {
			parent, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer parent.Close()
			if err := parent.Mkdir("target", 0755); err != nil {
				t.Fatal(err)
			}
			owned, err := parent.Lstat("target")
			if err != nil {
				t.Fatal(err)
			}
			if missing {
				if err := parent.Remove("target"); err != nil {
					t.Fatal(err)
				}
			} else {
				owned = nil
			}
			cause := errors.New("original failure")
			err = rollback(parent, "target", owned, cause)
			if !errors.Is(err, cause) {
				t.Fatalf("original failure lost: %v", err)
			}
			if missing {
				if err != cause {
					t.Fatalf("missing target introduced cleanup failure: %v", err)
				}
			} else {
				var failure *Error
				_, statErr := parent.Stat("target")
				if !errors.As(err, &failure) || failure.Reason != ReasonRollback || statErr != nil {
					t.Fatalf("unproven target was removed: %v %v", err, statErr)
				}
			}
		})
	}
}
