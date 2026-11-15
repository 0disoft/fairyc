package atlasmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestMCPInputPayloadBoundary(t *testing.T) {
	t.Parallel()
	const ping = `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	for _, ending := range []struct{ name, text string }{
		{"LF", "\n"}, {"CRLF", "\r\n"}, {"EOF", ""},
	} {
		for _, size := range []int{MaxInput - 1, MaxInput, MaxInput + 1} {
			t.Run(fmt.Sprintf("%s/%d", ending.name, size), func(t *testing.T) {
				for _, fragmented := range []bool{false, true} {
					var input io.Reader = strings.NewReader(ping + strings.Repeat(" ", size-len(ping)) + ending.text)
					if fragmented {
						input = iotest.OneByteReader(input)
					}
					var output bytes.Buffer
					err := Serve(input, &output, 1, 2048, func(string, string) (json.RawMessage, error) {
						t.Fatal("ping called impact")
						return nil, nil
					})
					if size > MaxInput {
						if err == nil || output.Len() != 0 {
							t.Fatalf("fragmented=%t: oversized input: err=%v output=%q", fragmented, err, output.String())
						}
						continue
					}
					if err != nil || !strings.Contains(output.String(), `"result":{}`) {
						t.Fatalf("fragmented=%t: bounded ping: err=%v output=%q", fragmented, err, output.String())
					}
				}
			})
		}
	}
}

type mcpInputFunc func([]byte) (int, error)

func (read mcpInputFunc) Read(buffer []byte) (int, error) { return read(buffer) }

func TestMCPInputCancellationAndFailures(t *testing.T) {
	t.Parallel()
	const ping = `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
	for _, phase := range []string{"before-read", "during-complete-read", "during-partial-read", "during-final-EOF"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("session stopped")
			reads := 0
			input := mcpInputFunc(func(buffer []byte) (int, error) {
				reads++
				cancel(cause)
				switch phase {
				case "during-partial-read":
					return copy(buffer, ping[:8]), nil
				case "during-final-EOF":
					return 0, io.EOF
				default:
					return copy(buffer, ping), nil
				}
			})
			if phase == "before-read" {
				cancel(cause)
			}
			var output bytes.Buffer
			err := ServeWithOptions(input, &output, 1, 2048, func(string, string) (json.RawMessage, error) {
				t.Fatal("canceled input called impact")
				return nil, nil
			}, ServeOptions{Context: ctx})
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || output.Len() != 0 {
				t.Fatalf("cancellation: err=%v output=%q", err, output.String())
			}
			wantReads := 1
			if phase == "before-read" {
				wantReads = 0
			}
			if reads != wantReads {
				t.Fatalf("reads=%d want=%d", reads, wantReads)
			}
		})
	}
	t.Run("read-error", func(t *testing.T) {
		want := errors.New("read failed")
		var output bytes.Buffer
		err := ServeWithOptions(iotest.ErrReader(want), &output, 1, 2048, func(string, string) (json.RawMessage, error) {
			t.Fatal("failed input called impact")
			return nil, nil
		}, ServeOptions{})
		if !errors.Is(err, want) || output.Len() != 0 {
			t.Fatalf("read failure: err=%v output=%q", err, output.String())
		}
	})
}

func TestMCPCancellationStopsBufferedDispatch(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("query canceled")
	call := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"atlas_impact","arguments":{"packageUid":"example.app","symbol":"selected"}}}`
	var output bytes.Buffer
	calls := 0
	err := ServeWithOptions(strings.NewReader(strings.Join([]string{initialize, ready, call, call}, "\n")), &output, 4, 2048, func(string, string) (json.RawMessage, error) {
		calls++
		cancel(cause)
		return nil, nil // Cancellation takes precedence over an unusable host result.
	}, ServeOptions{Context: ctx})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 1 || bytes.Count(output.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("buffered cancellation: err=%v calls=%d output=%q", err, calls, output.String())
	}
}

func TestMCPDefaultSessionOptions(t *testing.T) {
	t.Parallel()
	input := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	handler := func(string, string) (json.RawMessage, error) { return nil, nil }
	var output bytes.Buffer
	if err := ServeWithOptions(strings.NewReader(input), &output, 1, 2048, handler, ServeOptions{}); err != nil || !json.Valid(bytes.TrimSpace(output.Bytes())) {
		t.Fatalf("default options: err=%v output=%q", err, output.String())
	}
	output.Reset()
	err := ServeWithOptions(strings.NewReader(input), &output, 1, 2048, handler, ServeOptions{}, ServeOptions{})
	if err == nil || output.Len() != 0 {
		t.Fatalf("multiple options: err=%v output=%q", err, output.String())
	}
}
