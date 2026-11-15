package windowsbuild

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func TestReadRejectsReplacementBeforeOpen(t *testing.T) {
	for _, linked := range []bool{false, true} {
		t.Run(fmt.Sprintf("linked_%t", linked), func(t *testing.T) {
			directory := t.TempDir()
			for name, data := range map[string][]byte{
				"ui.c":    []byte("int ui(void){return 1;}\n"),
				"other.c": []byte("int ui(void){return 2;}\n"),
			} {
				if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			root, err := os.OpenRoot(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			var opened *observedInput
			policy, err := loadPolicy([]LoadOptions{{Open: func(root *os.Root, name string) (fs.File, error) {
				if err := root.Rename(name, "inspected.c"); err != nil {
					return nil, err
				}
				var err error
				if linked {
					err = root.Symlink("other.c", name)
				} else {
					err = root.Rename("other.c", name)
				}
				if err != nil {
					return nil, err
				}
				file, err := root.Open(name)
				if err != nil {
					return nil, err
				}
				opened = &observedInput{File: file}
				return opened, nil
			}}})
			if err != nil {
				t.Fatal(err)
			}
			data, err := read(root, "ui.c", MaxInputBytes, policy)
			if !errors.Is(err, durablefile.ErrIdentityChanged) || data != nil {
				t.Fatalf("replacement accepted: data=%q err=%v replaced=%v", data, err, bytes.Contains(data, []byte("return 2")))
			}
			if opened == nil || opened.reads != 0 || opened.closes != 1 {
				t.Fatalf("replacement read/closure: %+v", opened)
			}
		})
	}
}

type observedInput struct {
	fs.File
	readHook                              func([]byte) (int, error)
	statHook                              func() (fs.FileInfo, error)
	closeHook                             func() error
	reads, bytesRead, largestRead, closes int
}

func (file *observedInput) Read(buffer []byte) (int, error) {
	file.reads++
	file.largestRead = max(file.largestRead, len(buffer))
	read := file.File.Read
	if file.readHook != nil {
		read = file.readHook
	}
	n, err := read(buffer)
	file.bytesRead += n
	return n, err
}

func (file *observedInput) Stat() (fs.FileInfo, error) {
	if file.statHook != nil {
		return file.statHook()
	}
	return file.File.Stat()
}

func (file *observedInput) Close() error {
	file.closes++
	err := file.File.Close()
	if file.closeHook != nil {
		err = errors.Join(err, file.closeHook())
	}
	return err
}

func inputFixture(t *testing.T, data []byte) (*os.Root, LoadOptions, **observedInput) {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "ui.c"), data, 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	var observed *observedInput
	policy, err := loadPolicy([]LoadOptions{{Open: func(root *os.Root, name string) (fs.File, error) {
		file, err := durablefile.OpenRegular(root, name)
		if err != nil {
			return nil, err
		}
		observed = &observedInput{File: file}
		return observed, nil
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return root, policy, &observed
}

func TestReadInputSizeBoundaries(t *testing.T) {
	for _, limit := range []int{MaxConfigBytes, MaxInputBytes} {
		for _, size := range []int{0, 1, limit - 1, limit, limit + 1} {
			t.Run(fmt.Sprintf("limit_%d/size_%d", limit, size), func(t *testing.T) {
				want := bytes.Repeat([]byte("x"), size)
				root, policy, observed := inputFixture(t, want)
				data, err := read(root, "ui.c", limit, policy)
				if size == 0 || size > limit {
					if err == nil || data != nil || *observed != nil {
						t.Fatalf("invalid input opened: data=%d err=%v file=%v", len(data), err, *observed)
					}
					return
				}
				if err != nil || !bytes.Equal(data, want) || (*observed).closes != 1 || (*observed).largestRead > inputReadChunkBytes {
					t.Fatalf("read: bytes=%d err=%v observations=%+v", len(data), err, *observed)
				}
			})
		}
	}
}

func TestReadRejectsLinkedInputAndComponents(t *testing.T) {
	for _, directoryLink := range []bool{false, true} {
		t.Run(fmt.Sprint(directoryLink), func(t *testing.T) {
			root, policy, _ := inputFixture(t, []byte("int ui;"))
			name, target := "linked.c", "ui.c"
			if directoryLink {
				if err := root.Mkdir("native", 0700); err != nil {
					t.Fatal(err)
				}
				name, target = "linked", "native"
			}
			if err := root.Symlink(target, name); err != nil {
				t.Fatal(err)
			}
			if directoryLink {
				name += "/ui.c"
			}
			if data, err := read(root, name, MaxInputBytes, policy); err == nil || data != nil {
				t.Fatalf("accepted linked input: bytes=%d err=%v", len(data), err)
			}
		})
	}
}

func TestReadRejectsChangedOrInvalidInput(t *testing.T) {
	for _, scenario := range []string{"grow_before_open", "grow_during_read", "shrink_during_read", "same_size_edit", "nul", "utf8"} {
		t.Run(scenario, func(t *testing.T) {
			want := bytes.Repeat([]byte("x"), MaxConfigBytes)
			if scenario == "nul" {
				want[0] = 0
			}
			if scenario == "utf8" {
				want[0] = 0xff
			}
			root, policy, observed := inputFixture(t, want)
			open := policy.Open
			policy.Open = func(root *os.Root, name string) (fs.File, error) {
				if scenario == "grow_before_open" {
					if err := root.WriteFile(name, append(want, 'y'), 0600); err != nil {
						return nil, err
					}
				}
				file, err := open(root, name)
				if err != nil {
					return nil, err
				}
				changed := false
				(*observed).readHook = func(buffer []byte) (int, error) {
					if !changed {
						changed = true
						var err error
						if scenario == "grow_during_read" {
							err = root.WriteFile(name, append(want, 'y'), 0600)
						}
						if scenario == "shrink_during_read" {
							err = root.WriteFile(name, want[:len(want)-1], 0600)
						}
						if scenario == "same_size_edit" {
							want[0] = 'y'
							err = root.WriteFile(name, want, 0600)
							if err == nil {
								err = root.Chtimes(name, time.Unix(1, 0), time.Unix(1, 0))
							}
						}
						if err != nil {
							return 0, err
						}
					}
					return (*observed).File.Read(buffer)
				}
				return file, nil
			}
			data, err := read(root, "ui.c", MaxConfigBytes, policy)
			if err == nil || data != nil || (*observed).closes != 1 || (*observed).bytesRead > MaxConfigBytes+1 {
				t.Fatalf("invalid snapshot accepted: bytes=%d err=%v file=%+v", len(data), err, *observed)
			}
			if scenario == "grow_before_open" && (*observed).reads != 0 {
				t.Fatal("read a file that grew after inspection")
			}
		})
	}
}

func TestReadCancellationAndIOFailures(t *testing.T) {
	for _, cancelOperation := range []bool{false, true} {
		for _, stage := range []string{"inspect", "open", "stat", "read", "eof", "restat", "close"} {
			t.Run(fmt.Sprintf("cancel_%t/%s", cancelOperation, stage), func(t *testing.T) {
				root, policy, observed := inputFixture(t, []byte("int ui;"))
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				policy.Context = ctx
				failure, cause := errors.New("input I/O failure"), errors.New("caller stopped")
				interrupt := func() error {
					if cancelOperation {
						cancel(cause)
						return nil
					}
					return failure
				}
				inspect := policy.Lstat
				policy.Lstat = func(root *os.Root, name string) (fs.FileInfo, error) {
					info, err := inspect(root, name)
					if stage == "inspect" {
						err = errors.Join(err, interrupt())
					}
					return info, err
				}
				open := policy.Open
				policy.Open = func(root *os.Root, name string) (fs.File, error) {
					file, err := open(root, name)
					if err != nil {
						return nil, err
					}
					if stage == "open" {
						return file, interrupt()
					}
					stats := 0
					(*observed).statHook = func() (fs.FileInfo, error) {
						stats++
						info, err := (*observed).File.Stat()
						if stage == "stat" || stage == "restat" && stats == 2 {
							err = errors.Join(err, interrupt())
						}
						return info, err
					}
					(*observed).readHook = func(buffer []byte) (int, error) {
						n, err := (*observed).File.Read(buffer)
						if stage == "read" || stage == "eof" && errors.Is(err, io.EOF) {
							err = errors.Join(err, interrupt())
						}
						return n, err
					}
					(*observed).closeHook = func() error {
						if stage == "close" {
							return interrupt()
						}
						return nil
					}
					return file, nil
				}
				data, err := read(root, "ui.c", MaxInputBytes, policy)
				want := failure
				if cancelOperation {
					want = cause
				}
				if data != nil || !errors.Is(err, want) || cancelOperation && !errors.Is(err, context.Canceled) {
					t.Fatalf("failed operation accepted: bytes=%d err=%v want=%v", len(data), err, want)
				}
				if *observed != nil && (*observed).closes != 1 {
					t.Fatalf("close count=%d", (*observed).closes)
				}
				if cancelOperation && *observed != nil && (*observed).bytesRead > inputReadChunkBytes {
					t.Fatal("read after cancellation")
				}
			})
		}
	}
}

func TestReadEmptyProgressBoundaries(t *testing.T) {
	for _, empty := range []int{maxInputEmptyReads - 1, maxInputEmptyReads, maxInputEmptyReads + 1} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			root, policy, observed := inputFixture(t, []byte("x"))
			open := policy.Open
			policy.Open = func(root *os.Root, name string) (fs.File, error) {
				file, err := open(root, name)
				if err != nil {
					return nil, err
				}
				left := empty
				(*observed).readHook = func(buffer []byte) (int, error) {
					if left > 0 {
						left--
						return 0, nil
					}
					return (*observed).File.Read(buffer)
				}
				return file, nil
			}
			data, err := read(root, "ui.c", MaxInputBytes, policy)
			if empty < maxInputEmptyReads {
				if string(data) != "x" || err != nil {
					t.Fatalf("progress rejected: %q %v", data, err)
				}
			} else if data != nil || !errors.Is(err, io.ErrNoProgress) || (*observed).reads != maxInputEmptyReads {
				t.Fatalf("empty read bound: bytes=%d err=%v reads=%d", len(data), err, (*observed).reads)
			}
		})
	}
}

func TestLoadOptionsPreserveAbsenceAndCancellation(t *testing.T) {
	const config = `{"schema":"fairyc.windows-build/v1","subsystem":"windows"}`
	for _, stage := range []string{"default", "absent", "precanceled", "root", "root_error", "nil_root", "absent_canceled", "disappeared", "file_close", "multiple_options"} {
		t.Run(stage, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, FileName), []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("stop loading Windows inputs")
			failure := errors.New("injected I/O error")
			var opened *os.Root
			rootCalls := 0
			policy := LoadOptions{Context: ctx, OpenRoot: func(path string) (*os.Root, error) {
				rootCalls++
				if stage == "nil_root" {
					return nil, nil
				}
				root, err := os.OpenRoot(path)
				if err != nil {
					return nil, err
				}
				opened = root
				if stage == "root" {
					cancel(cause)
				}
				if stage == "root_error" {
					return root, failure
				}
				return root, nil
			}}
			if stage == "precanceled" {
				cancel(cause)
			}
			if stage == "absent" || stage == "absent_canceled" {
				policy.Lstat = func(root *os.Root, name string) (fs.FileInfo, error) {
					if stage == "absent_canceled" {
						cancel(cause)
					}
					return nil, &os.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
				}
			}
			if stage == "disappeared" || stage == "file_close" {
				policy.Open = func(root *os.Root, name string) (fs.File, error) {
					if stage == "disappeared" {
						if err := root.Remove(name); err != nil {
							return nil, err
						}
						return root.Open(name)
					}
					file, err := root.Open(name)
					if err != nil {
						return nil, err
					}
					return &observedInput{File: file, closeHook: func() error { return failure }}, nil
				}
			}
			options := []LoadOptions{policy}
			if stage == "multiple_options" {
				options = append(options, LoadOptions{})
			}
			project, err := LoadWithOptions(directory, options...)
			switch stage {
			case "default":
				if err != nil || project == nil || project.Subsystem != "windows" {
					t.Fatalf("load: %v %v", project, err)
				}
			case "absent":
				if project != nil || err != nil {
					t.Fatalf("absent config: %v %v", project, err)
				}
			default:
				if project != nil || err == nil {
					t.Fatalf("failure accepted: %v %v", project, err)
				}
			}
			if ctx.Err() != nil && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatalf("lost cancellation: %v", err)
			}
			if (stage == "precanceled" || stage == "multiple_options") && rootCalls != 0 {
				t.Fatal("performed unexpected root I/O")
			}
			if (stage == "root_error" || stage == "file_close") && !errors.Is(err, failure) {
				t.Fatalf("lost I/O error: %v", err)
			}
			if stage == "disappeared" && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("lost missing config error: %v", err)
			}
			if opened != nil {
				if _, err := opened.Stat("."); err == nil {
					t.Fatal("root was not closed")
				}
			}
		})
	}
}
