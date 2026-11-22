package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPublicationCleanupPreservesPromotedOutput(t *testing.T) {
	for _, concurrent := range []bool{false, true} {
		name := "own output"
		if concurrent {
			name = "concurrent replacement"
		}
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			local := filepath.FromSlash("build/app.exe")
			contents := []byte("complete image")
			want := bytes.Clone(contents)
			cleanupErr := errors.New("staging deletion denied")
			failed, finalRemovals := false, 0
			err := publishBuildArtifactWithOptions(directory, "build/app.exe", contents, false, buildPublicationOptions{Remove: func(root *os.Root, name string) error {
				if name == local {
					finalRemovals++
				}
				if !failed && name != local {
					failed = true
					if concurrent {
						// A second publisher wins after our successful hard link.
						if err := root.Remove(local); err != nil {
							t.Fatal(err)
						}
						want = []byte("other publisher's complete image")
						if err := root.WriteFile(local, want, 0700); err != nil {
							t.Fatal(err)
						}
					}
					return cleanupErr
				}
				return root.Remove(name)
			}})
			if !errors.Is(err, cleanupErr) {
				t.Fatalf("lost cleanup failure: %v", err)
			}
			got, readErr := os.ReadFile(filepath.Join(directory, local))
			if readErr != nil || !bytes.Equal(got, want) || finalRemovals != 0 {
				t.Fatalf("published output deleted: got=%q want=%q removals=%d error=%v", got, want, finalRemovals, readErr)
			}
		})
	}
}

func fixedBuildStaging(root *os.Root, parent string) (string, *os.File, error) {
	name := filepath.Join(parent, ".fairyc-fixture.tmp")
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	return name, file, err
}

func TestBuildPublicationPersistentCleanupFailureRetainsImage(t *testing.T) {
	directory := t.TempDir()
	contents := []byte("complete image")
	want := errors.New("cleanup remains denied")
	removals := 0
	err := publishBuildArtifactWithOptions(directory, "build/app.exe", contents, false, buildPublicationOptions{
		CreateStaging: fixedBuildStaging,
		Remove: func(_ *os.Root, name string) error {
			if name != filepath.Join("build", ".fairyc-fixture.tmp") {
				t.Fatalf("cleanup targeted final output: %q", name)
			}
			removals++
			return want
		},
	})
	if !errors.Is(err, want) || removals != 2 {
		t.Fatalf("cleanup failure/retry lost: removals=%d error=%v", removals, err)
	}
	for _, name := range []string{"app.exe", ".fairyc-fixture.tmp"} {
		got, err := os.ReadFile(filepath.Join(directory, "build", name))
		if err != nil || !bytes.Equal(got, contents) {
			t.Fatalf("retained %s: %q %v", name, got, err)
		}
	}
}

func TestBuildPublicationReportsCleanupFailureBeforePromotion(t *testing.T) {
	directory := t.TempDir()
	promotionErr := errors.New("promotion denied")
	staging := filepath.Join("build", ".fairyc-fixture.tmp")
	err := publishBuildArtifactWithOptions(directory, "build/app.exe", []byte("image"), false, buildPublicationOptions{
		CreateStaging: fixedBuildStaging,
		Promote:       func(*os.Root, string, string, bool) error { return promotionErr },
		Remove: func(_ *os.Root, name string) error {
			if name != staging {
				t.Fatalf("cleanup targeted %q, want %q", name, staging)
			}
			return os.ErrPermission
		},
	})
	if !errors.Is(err, promotionErr) || !errors.Is(err, os.ErrPermission) || !strings.Contains(err.Error(), filepath.Base(staging)) {
		t.Fatalf("missing promotion or staging cleanup failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, staging)); err != nil {
		t.Fatalf("retained staging file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "build", "app.exe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected final output: %v", err)
	}
}

func TestBuildPublicationCancellationBeforeAndAfterCommit(t *testing.T) {
	for _, replace := range []bool{false, true} {
		for _, afterCommit := range []bool{false, true} {
			t.Run(fmt.Sprintf("replace=%v/afterCommit=%v", replace, afterCommit), func(t *testing.T) {
				directory := t.TempDir()
				output := filepath.Join(directory, "build", "app.exe")
				old, image := []byte("prior image"), []byte("new image")
				if err := os.Mkdir(filepath.Join(directory, "build"), 0700); err != nil {
					t.Fatal(err)
				}
				if replace {
					if err := os.WriteFile(output, old, 0700); err != nil {
						t.Fatal(err)
					}
				}
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("caller cancelled publication")
				promoted := 0
				var staged *os.File
				err := publishBuildArtifactWithOptions(directory, "build/app.exe", image, replace, buildPublicationOptions{
					Context: ctx,
					CreateStaging: func(root *os.Root, parent string) (string, *os.File, error) {
						name, file, err := fixedBuildStaging(root, parent)
						staged = file
						if !afterCommit {
							cancel(cause)
						}
						return name, file, err
					},
					Promote: func(root *os.Root, temporary, final string, replace bool) error {
						promoted++
						var err error
						if replace {
							err = root.Rename(temporary, final)
						} else {
							err = root.Link(temporary, final)
						}
						cancel(cause)
						return err
					},
				})
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatalf("cancellation cause lost: %v", err)
				}
				if staged == nil {
					t.Fatal("staging file was never created")
				}
				if _, err := staged.Write([]byte{'!'}); !errors.Is(err, os.ErrClosed) {
					t.Fatalf("staging handle leaked: %v", err)
				}
				got, readErr := os.ReadFile(output)
				switch {
				case afterCommit:
					if promoted != 1 || readErr != nil || !bytes.Equal(got, image) {
						t.Fatalf("committed image rolled back: promotions=%d image=%q error=%v", promoted, got, readErr)
					}
				case replace:
					if promoted != 0 || readErr != nil || !bytes.Equal(got, old) {
						t.Fatalf("pre-commit cancellation changed old image: %q %v", got, readErr)
					}
				default:
					if promoted != 0 || !errors.Is(readErr, os.ErrNotExist) {
						t.Fatalf("pre-commit cancellation created an output: %q %v", got, readErr)
					}
				}
				leftovers, err := filepath.Glob(filepath.Join(directory, "build", ".fairyc-*.tmp"))
				if err != nil || len(leftovers) != 0 {
					t.Fatalf("cancellation skipped staging cleanup: %v %v", leftovers, err)
				}
			})
		}
	}
}

func TestBuildPublicationSizeAndPathBoundaries(t *testing.T) {
	for _, size := range []int{0, 1, 2} {
		t.Run(fmt.Sprintf("bytes=%d", size), func(t *testing.T) {
			directory := t.TempDir()
			image := bytes.Repeat([]byte{'x'}, size)
			err := publishBuildArtifactWithOptions(directory, "build/app.exe", image, false, buildPublicationOptions{CreateStaging: fixedBuildStaging})
			if (err != nil) != (size == 0) {
				t.Fatalf("size=%d error=%v", size, err)
			}
			got, readErr := os.ReadFile(filepath.Join(directory, "build", "app.exe"))
			if size == 0 {
				if !errors.Is(readErr, os.ErrNotExist) {
					t.Fatalf("empty image created output: %q %v", got, readErr)
				}
			} else if readErr != nil || !bytes.Equal(got, image) {
				t.Fatalf("image changed: %q %v", got, readErr)
			}
		})
	}
	reachedStaging := errors.New("valid path reached staging")
	for _, size := range []int{maxBuildComponentSize - 1, maxBuildComponentSize, maxBuildComponentSize + 1} {
		t.Run(fmt.Sprintf("componentBytes=%d", size), func(t *testing.T) {
			err := publishBuildArtifactWithOptions(t.TempDir(), "build/"+strings.Repeat("a", size), []byte("x"), false, buildPublicationOptions{
				CreateStaging: func(*os.Root, string) (string, *os.File, error) { return "", nil, reachedStaging },
			})
			if err == nil || (size <= maxBuildComponentSize) != errors.Is(err, reachedStaging) {
				t.Fatalf("path bytes=%d error=%v", size, err)
			}
		})
	}
}

func TestBuildPublicationOptionsAndPromotionFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := publishBuildArtifactWithOptions(filepath.Join(t.TempDir(), "missing"), "build/app.exe", []byte("x"), false, buildPublicationOptions{Context: ctx}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancelled publication performed I/O: %v", err)
	}
	if err := publishBuildArtifactWithOptions("", "", nil, false, buildPublicationOptions{}, buildPublicationOptions{}); err == nil {
		t.Fatal("multiple options values accepted")
	}
	if err := publishBuildArtifactWithOptions(t.TempDir(), "build/app.exe", []byte("x"), false, buildPublicationOptions{
		CreateStaging: func(*os.Root, string) (string, *os.File, error) { return "", nil, nil },
	}); err == nil {
		t.Fatal("missing staging handle accepted")
	}
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprintf("replace=%v", replace), func(t *testing.T) {
			directory := t.TempDir()
			if err := os.Mkdir(filepath.Join(directory, "build"), 0700); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(directory, "build", "app.exe")
			old := []byte("prior image")
			if err := os.WriteFile(output, old, 0700); err != nil {
				t.Fatal(err)
			}
			want := errors.New("promotion failed")
			err := publishBuildArtifactWithOptions(directory, "build/app.exe", []byte("new image"), replace, buildPublicationOptions{
				CreateStaging: fixedBuildStaging,
				Promote:       func(*os.Root, string, string, bool) error { return want },
			})
			got, readErr := os.ReadFile(output)
			if !errors.Is(err, want) || readErr != nil || !bytes.Equal(got, old) {
				t.Fatalf("failed publication changed prior image: %q %v %v", got, err, readErr)
			}
			entries, err := os.ReadDir(filepath.Dir(output))
			if err != nil || len(entries) != 1 {
				t.Fatalf("failed publication leaked staging state: %v %v", entries, err)
			}
		})
	}
}
