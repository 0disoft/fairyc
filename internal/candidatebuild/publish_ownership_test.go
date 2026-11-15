package candidatebuild

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func TestPublishRejectsReplacedStage(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "app.exe")
	if err := os.WriteFile(output, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	var staged string
	err := publish(output, []byte("verified"), publishHooks{close: func(file *os.File) error {
		staged = file.Name()
		if err := file.Close(); err != nil {
			return err
		}
		if err := os.Rename(staged, staged+".held"); err != nil {
			return err
		}
		return os.WriteFile(staged, []byte("injected"), 0600)
	}})
	if !errors.Is(err, durablefile.ErrIdentityChanged) {
		t.Fatalf("published an image substituted after close: %v", err)
	}
	data, readErr := os.ReadFile(output)
	if readErr != nil || string(data) != "previous" {
		t.Fatalf("previous output changed: %q %v", data, readErr)
	}
	data, readErr = os.ReadFile(staged)
	if readErr != nil || string(data) != "injected" {
		t.Fatalf("replacement stage was removed: %q %v", data, readErr)
	}
}

func TestPublicationPreservesReusedStageNameAfterCommit(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			directory := t.TempDir()
			output := filepath.Join(directory, "app.exe")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("cancel after rename committed")
			var stage string
			err := PublishWithOptions(output, []byte("verified"), PublicationOptions{Context: ctx, Rename: func(root *os.Root, from, to string) error {
				if err := root.Rename(from, to); err != nil {
					return err
				}
				stage = filepath.Join(root.Name(), from)
				if err := root.WriteFile(from, []byte("later-file"), 0600); err != nil {
					return err
				}
				if cancelled {
					cancel(cause)
				}
				return nil
			}})
			if cancelled {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatalf("lost cancellation: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]string{output: "verified", stage: "later-file"} {
				data, err := os.ReadFile(name)
				if err != nil || string(data) != want {
					t.Fatalf("committed output/reused stage changed: %q %v", data, err)
				}
			}
		})
	}
}

func TestPublicationOptionsCancellationAndFailures(t *testing.T) {
	for _, mode := range []string{"failure", "cancel", "both"} {
		for _, phase := range []string{"random", "write", "sync", "close", "rename"} {
			t.Run(mode+"/"+phase, func(t *testing.T) {
				directory := t.TempDir()
				output := filepath.Join(directory, "app.exe")
				if err := os.WriteFile(output, []byte("previous"), 0600); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				failure, cause := errors.New("publication I/O failed"), errors.New("caller stopped publication")
				interrupt := func() error {
					if mode != "failure" {
						cancel(cause)
					}
					if mode != "cancel" {
						return failure
					}
					return nil
				}
				policy := PublicationOptions{Context: ctx}
				switch phase {
				case "random":
					policy.Random = func(buffer []byte) (int, error) { return len(buffer), interrupt() }
				case "write":
					policy.Write = func(file *os.File, data []byte) error {
						_, err := file.Write(data)
						return errors.Join(err, interrupt())
					}
				case "sync":
					policy.Sync = func(file *os.File) error { return errors.Join(file.Sync(), interrupt()) }
				case "close":
					policy.Close = func(file *os.File) error { return errors.Join(file.Close(), interrupt()) }
				case "rename":
					policy.Rename = func(*os.Root, string, string) error { return errors.Join(interrupt(), ctx.Err()) }
				}
				err := PublishWithOptions(output, []byte("verified"), policy)
				if mode != "cancel" && !errors.Is(err, failure) {
					t.Fatalf("lost I/O failure: %v", err)
				}
				if mode != "failure" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatalf("lost cancellation: %v", err)
				}
				data, readErr := os.ReadFile(output)
				if readErr != nil || string(data) != "previous" {
					t.Fatalf("prior output changed: %q %v", data, readErr)
				}
				files, readErr := os.ReadDir(directory)
				if readErr != nil || len(files) != 1 {
					t.Fatalf("staging residue: %v %v", files, readErr)
				}
			})
		}
	}
}

func TestPublicationNonceLengthBoundaries(t *testing.T) {
	for _, count := range []int{0, 15, 16, 17} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			directory := t.TempDir()
			output := filepath.Join(directory, "app.exe")
			calls := 0
			err := PublishWithOptions(output, []byte("verified"), PublicationOptions{Random: func(buffer []byte) (int, error) {
				calls++
				if len(buffer) != 16 {
					t.Fatalf("nonce request length=%d", len(buffer))
				}
				return count, nil
			}})
			if calls != 1 {
				t.Fatalf("random calls=%d", calls)
			}
			if count == 16 {
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(output)
				if err != nil || string(data) != "verified" {
					t.Fatalf("published data: %q %v", data, err)
				}
			} else {
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Fatalf("nonce count accepted: %v", err)
				}
				files, err := os.ReadDir(directory)
				if err != nil || len(files) != 0 {
					t.Fatalf("nonce failure performed staging I/O: %v %v", files, err)
				}
			}
		})
	}
}

func TestPublicationImageSizeBoundaries(t *testing.T) {
	image := make([]byte, maxPublishedImageBytes+1)
	for _, size := range []int{0, 1, maxPublishedImageBytes - 1, maxPublishedImageBytes, maxPublishedImageBytes + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			directory := t.TempDir()
			output := filepath.Join(directory, "app.exe")
			calls := 0
			stop := errors.New("image admitted; skip large disk write")
			err := PublishWithOptions(output, image[:size], PublicationOptions{Write: func(_ *os.File, data []byte) error {
				calls++
				if len(data) != size {
					t.Fatalf("write size=%d", len(data))
				}
				return stop
			}})
			if size > 0 && size <= maxPublishedImageBytes {
				if calls != 1 || !errors.Is(err, stop) {
					t.Fatalf("valid image rejected: calls=%d err=%v", calls, err)
				}
			} else if calls != 0 || err == nil {
				t.Fatalf("invalid image admitted: calls=%d err=%v", calls, err)
			}
			files, err := os.ReadDir(directory)
			if err != nil || len(files) != 0 {
				t.Fatalf("boundary check left output: %v %v", files, err)
			}
		})
	}
}

func TestPublicationRejectsShortStage(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "app.exe")
	if err := os.WriteFile(output, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	err := PublishWithOptions(output, []byte("verified"), PublicationOptions{Write: func(file *os.File, data []byte) error {
		_, err := file.Write(data[:len(data)-1])
		return err
	}})
	if !errors.Is(err, durablefile.ErrIdentityChanged) {
		t.Fatalf("short stage accepted: %v", err)
	}
	data, readErr := os.ReadFile(output)
	if readErr != nil || string(data) != "previous" {
		t.Fatalf("previous output changed: %q %v", data, readErr)
	}
}

func TestPublicationNonceCollisionPreservesExistingFiles(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "app.exe")
	stage := filepath.Join(directory, ".fairyc-candidate-"+strings.Repeat("0", 32)+".tmp")
	for name, data := range map[string]string{output: "previous", stage: "other operation"} {
		if err := os.WriteFile(name, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	err := PublishWithOptions(output, []byte("verified"), PublicationOptions{Random: func(buffer []byte) (int, error) { return len(buffer), nil }})
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("existing stage reused: %v", err)
	}
	for name, want := range map[string]string{output: "previous", stage: "other operation"} {
		data, err := os.ReadFile(name)
		if err != nil || string(data) != want {
			t.Fatalf("existing file changed: %q %v", data, err)
		}
	}
}

func TestPublicationRejectsCancellationAndOptionsBeforeIO(t *testing.T) {
	for _, rooted := range []bool{false, true} {
		for _, multiple := range []bool{false, true} {
			t.Run(fmt.Sprintf("rooted_%t/multiple_%t", rooted, multiple), func(t *testing.T) {
				directory := t.TempDir()
				ctx, cancel := context.WithCancelCause(context.Background())
				cause := errors.New("cancel before staging")
				cancel(cause)
				options := []PublicationOptions{{Context: ctx, Random: func([]byte) (int, error) { t.Fatal("unexpected randomness I/O"); return 0, nil }}}
				if multiple {
					options = append(options, PublicationOptions{})
				}
				var err error
				if rooted {
					err = PublishWithinWithOptions(directory, filepath.Join("new", "app.exe"), []byte("verified"), options...)
				} else {
					err = PublishWithOptions(filepath.Join(directory, "new", "app.exe"), []byte("verified"), options...)
				}
				if err == nil || !multiple && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatalf("invalid options/cancellation: %v", err)
				}
				files, readErr := os.ReadDir(directory)
				if readErr != nil || len(files) != 0 {
					t.Fatalf("performed filesystem I/O before validation: %v %v", files, readErr)
				}
			})
		}
	}
}
