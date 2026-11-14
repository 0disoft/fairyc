package durablefile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"testing/iotest"
)

func TestPublicationPreservesDestinationOnUnverifiedComparison(t *testing.T) {
	for _, scenario := range []string{"read", "close", "permission", "pre-cancel", "cancel-compare", "compare-nonregular"} {
		t.Run(scenario, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			payload := []byte("previous valid generation")
			if err := root.WriteFile("final", payload, 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			failure := errors.New("injected " + scenario + " failure")
			if scenario == "permission" {
				failure = os.ErrPermission
			}
			if scenario == "compare-nonregular" {
				failure = ErrNotRegularFile
			}
			moves, compares, removes, flushes := 0, 0, 0, 0
			options := PublicationOptions{Context: ctx,
				Move: func(context.Context, *os.Root, string, string) error { moves++; return os.ErrPermission },
				Compare: func(context.Context, *os.Root, string, []byte) (bool, error) {
					compares++
					if scenario == "cancel-compare" {
						cancel()
						return false, nil
					}
					return false, failure
				},
				Remove: func(ctx context.Context, root *os.Root, name string) error { removes++; return root.Remove(name) },
				Flush:  func(context.Context, *os.Root, string, string) error { flushes++; return nil },
			}
			if scenario == "pre-cancel" {
				cancel()
			}
			err = PublishImmutableWithOptions(root, "temporary", "final", []byte("new generation"), options)
			got, readErr := root.ReadFile("final")
			if readErr != nil || !bytes.Equal(got, payload) {
				t.Fatalf("destination lost: %q %v", got, readErr)
			}
			want := failure
			if scenario == "pre-cancel" || scenario == "cancel-compare" {
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			if removes != 0 || flushes != 0 {
				t.Fatalf("unverified mutation/success: remove=%d flush=%d", removes, flushes)
			}
			if scenario == "pre-cancel" && (moves != 0 || compares != 0) {
				t.Fatal("canceled publication invoked I/O")
			}
			if scenario != "pre-cancel" && (moves != 1 || compares != 1) {
				t.Fatalf("unexpected publication attempts: move=%d compare=%d", moves, compares)
			}
		})
	}
}

func TestCompareReaderBoundariesAndFailures(t *testing.T) {
	failure := errors.New("read failure")
	for _, size := range []int{compareChunkBytes - 1, compareChunkBytes, compareChunkBytes + 1} {
		for _, scenario := range []string{"equal", "short", "long", "error", "error-with-last-bytes", "error-at-eof", "cancel"} {
			t.Run(fmt.Sprintf("%d/%s", size, scenario), func(t *testing.T) {
				expected := bytes.Repeat([]byte("x"), size)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				var reader io.Reader = bytes.NewReader(expected)
				var want error
				switch scenario {
				case "short":
					reader = bytes.NewReader(expected[:size-1])
				case "long":
					reader = bytes.NewReader(append(bytes.Clone(expected), 'x'))
				case "error":
					reader = iotest.ErrReader(failure)
					want = failure
				case "error-with-last-bytes":
					reader = &failingFinalRead{data: bytes.Clone(expected), failure: failure}
					want = failure
				case "error-at-eof":
					reader = io.MultiReader(reader, iotest.ErrReader(failure))
					want = failure
				case "cancel":
					reader = publicationReaderFunc(func(buffer []byte) (int, error) { cancel(); return copy(buffer, expected), nil })
					want = context.Canceled
				}
				counted := &countedReader{Reader: reader}
				same, err := compareReader(ctx, counted, expected)
				if same != (scenario == "equal") || !errors.Is(err, want) {
					t.Fatalf("same=%v err=%v want=%v", same, err, want)
				}
				if counted.largestRead > compareChunkBytes || counted.total > size+1 {
					t.Fatalf("read bounds exceeded: %+v", counted)
				}
			})
		}
	}
}

func TestCompareNoProgressBoundary(t *testing.T) {
	for _, empty := range []int{maxComparisonEmptyReads - 1, maxComparisonEmptyReads, maxComparisonEmptyReads + 1} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			calls := 0
			source := bytes.NewReader([]byte("x"))
			reader := publicationReaderFunc(func(buffer []byte) (int, error) {
				calls++
				if calls <= empty {
					return 0, nil
				}
				return source.Read(buffer)
			})
			same, err := compareReader(t.Context(), reader, []byte("x"))
			if empty < maxComparisonEmptyReads {
				if !same || err != nil {
					t.Fatalf("progress rejected: %v %v", same, err)
				}
			} else if same || !errors.Is(err, io.ErrNoProgress) || calls != maxComparisonEmptyReads {
				t.Fatalf("unbounded empty reads: same=%v calls=%d err=%v", same, calls, err)
			}
		})
	}
}

func TestCompareFileDoesNotAdmitDirectories(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if same, err := compareFile(t.Context(), root, ".", []byte("x")); same || !errors.Is(err, ErrNotRegularFile) {
		t.Fatalf("directory: %v %v", same, err)
	}
	if same, err := compareFile(t.Context(), root, "missing", []byte("x")); same || err != nil {
		t.Fatalf("absent destination: %v %v", same, err)
	}
	if err := PublishImmutableWithOptions(root, "temporary", "final", []byte("x"), PublicationOptions{}, PublicationOptions{}); err == nil {
		t.Fatal("ambiguous options admitted")
	}
}

func TestPublicationCancellationStopsBeforeFlush(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("temporary", []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	flushes := 0
	err = PublishImmutableWithOptions(root, "temporary", "final", []byte("x"), PublicationOptions{Context: ctx,
		Move: func(ctx context.Context, root *os.Root, from, to string) error {
			err := root.Rename(from, to)
			cancel()
			return err
		},
		Flush: func(context.Context, *os.Root, string, string) error { flushes++; return nil },
	})
	if !errors.Is(err, context.Canceled) || flushes != 0 {
		t.Fatalf("cancel ignored: error=%v flushes=%d", err, flushes)
	}
	if got, err := root.ReadFile("final"); err != nil || string(got) != "x" {
		t.Fatalf("completed rename was incorrectly rolled back: %q %v", got, err)
	}
}

type publicationReaderFunc func([]byte) (int, error)

func (read publicationReaderFunc) Read(buffer []byte) (int, error) { return read(buffer) }

type failingFinalRead struct {
	data    []byte
	failure error
}

func (reader *failingFinalRead) Read(buffer []byte) (int, error) {
	if len(reader.data) == 0 {
		return 0, io.EOF
	}
	count := copy(buffer, reader.data)
	reader.data = reader.data[count:]
	if len(reader.data) == 0 {
		return count, reader.failure
	}
	return count, nil
}
