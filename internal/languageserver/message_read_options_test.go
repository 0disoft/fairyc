package languageserver

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

const framingTestBody = `{"jsonrpc":"2.0","method":"shutdown","id":1}`

func TestMessageEveryTruncationBoundary(t *testing.T) {
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(framingTestBody), framingTestBody)
	for length := 0; length < len(frame); length++ {
		_, err := readMessage(bufio.NewReaderSize(strings.NewReader(frame[:length]), 8192))
		if length == 0 {
			if !errors.Is(err, io.EOF) {
				t.Fatalf("empty stream: %v", err)
			}
		} else if !errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			t.Fatalf("truncated prefix %d accepted as clean EOF: %v", length, err)
		}
	}
	reader := bufio.NewReader(strings.NewReader(frame))
	if m, err := readMessage(reader); err != nil || m.Method != "shutdown" {
		t.Fatalf("complete frame rejected: %+v %v", m, err)
	}
	if _, err := readMessage(reader); !errors.Is(err, io.EOF) {
		t.Fatalf("EOF after complete frame: %v", err)
	}
}

func TestServeTruncatedFrameFails(t *testing.T) {
	for _, input := range []string{"Content-Len", "Content-Length: 12\r\n", "Content-Length: 12\r\n\r\n", "Content-Length: 12\r\n\r\n{"} {
		t.Run(fmt.Sprintf("%d", len(input)), func(t *testing.T) {
			var output, diagnostic bytes.Buffer
			if code := Serve(strings.NewReader(input), &output, &diagnostic); code != 1 || !strings.Contains(diagnostic.String(), "unexpected EOF") {
				t.Fatalf("truncated request silently accepted: code=%d diagnostic=%q", code, diagnostic.String())
			}
			if output.Len() != 0 {
				t.Fatal("invalid frame reached handler")
			}
		})
	}
	if code := Serve(strings.NewReader(""), io.Discard, io.Discard); code != 0 {
		t.Fatalf("empty input exit=%d", code)
	}
}

func TestMessageHeaderAndBodyLimits(t *testing.T) {
	for _, size := range []int{8191, 8192, 8193} {
		t.Run(fmt.Sprintf("header/%d", size), func(t *testing.T) {
			prefix := fmt.Sprintf("Content-Length: %d\r\nX-Pad: ", len(framingTestBody))
			header := prefix + strings.Repeat("a", size-len(prefix)-4) + "\r\n\r\n"
			_, err := readMessage(bufio.NewReaderSize(strings.NewReader(header+framingTestBody), 8192))
			if (err == nil) != (size <= 8192) {
				t.Fatalf("header limit=%d err=%v", size, err)
			}
		})
	}
	body := framingTestBody + strings.Repeat(" ", maxMessage+1-len(framingTestBody))
	for _, size := range []int{maxMessage - 1, maxMessage, maxMessage + 1} {
		t.Run(fmt.Sprintf("body/%d", size), func(t *testing.T) {
			header := fmt.Sprintf("Content-Length: %d\r\n\r\n", size)
			calls := 0
			input := bufio.NewReaderSize(io.MultiReader(strings.NewReader(header), strings.NewReader(body[:size])), 8192)
			_, err := readMessageWithOptions(input, messageReadOptions{ReadFull: func(ctx context.Context, in io.Reader, out []byte) (int, error) { calls++; return io.ReadFull(in, out) }})
			if (err == nil) != (size <= maxMessage) {
				t.Fatalf("body limit=%d err=%v", size, err)
			}
			if size > maxMessage && calls != 0 {
				t.Fatal("oversize body admitted to I/O")
			}
		})
	}
	for _, lengths := range []string{"-1\r\nContent-Length: " + fmt.Sprint(len(framingTestBody)), fmt.Sprint(len(framingTestBody)) + "\r\nContent-Length: -1"} {
		_, err := readMessage(bufio.NewReader(strings.NewReader("Content-Length: " + lengths + "\r\n\r\n" + framingTestBody)))
		if err == nil {
			t.Fatal("negative length sentinel hid a duplicate")
		}
	}
}

func TestMessageReadCancellationAndFailures(t *testing.T) {
	failure := errors.New("injected stream failure")
	for _, scenario := range []string{"pre-cancel", "cancel-header", "cancel-body", "read-error", "short-success", "body-eof"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(framingTestBody))
			inputReads, bodyReads := 0, 0
			source := strings.NewReader(header + framingTestBody)
			input := bufio.NewReader(messageTestReader(func(buffer []byte) (int, error) {
				inputReads++
				count, err := source.Read(buffer)
				if scenario == "cancel-header" {
					cancel()
				}
				return count, err
			}))
			if scenario == "pre-cancel" {
				cancel()
			}
			m, err := readMessageWithOptions(input, messageReadOptions{Context: ctx, ReadFull: func(ctx context.Context, reader io.Reader, buffer []byte) (int, error) {
				bodyReads++
				switch scenario {
				case "cancel-body":
					count, _ := io.ReadFull(reader, buffer)
					cancel()
					return count, nil
				case "read-error":
					return 0, failure
				case "short-success":
					return 0, nil
				case "body-eof":
					return 0, io.EOF
				}
				return io.ReadFull(reader, buffer)
			}})
			want := context.Canceled
			switch scenario {
			case "read-error":
				want = failure
			case "short-success", "body-eof":
				want = io.ErrUnexpectedEOF
			}
			if !errors.Is(err, want) || m.Method != "" {
				t.Fatalf("partial/canceled frame accepted: %+v err=%v want=%v", m, err, want)
			}
			if scenario == "pre-cancel" && (inputReads != 0 || bodyReads != 0) {
				t.Fatal("pre-canceled reader touched I/O")
			}
			if scenario == "cancel-header" && bodyReads != 0 {
				t.Fatal("canceled header admitted body")
			}
		})
	}
	input := bufio.NewReader(strings.NewReader(""))
	if _, err := readMessageWithOptions(input, messageReadOptions{}, messageReadOptions{}); err == nil {
		t.Fatal("ambiguous options admitted")
	}
}

func TestMessageBodyReaderProgressAndCancellation(t *testing.T) {
	for _, empty := range []int{messageMaxEmptyReads - 1, messageMaxEmptyReads, messageMaxEmptyReads + 1} {
		calls := 0
		reader := &messageBodyReader{ctx: t.Context(), reader: messageTestReader(func(buffer []byte) (int, error) {
			calls++
			if calls <= empty {
				return 0, nil
			}
			return copy(buffer, "x"), nil
		})}
		_, err := io.ReadFull(reader, make([]byte, 1))
		if empty < messageMaxEmptyReads {
			if err != nil {
				t.Fatal(err)
			}
		} else if !errors.Is(err, io.ErrNoProgress) || calls != messageMaxEmptyReads {
			t.Fatalf("empty reads=%d calls=%d err=%v", empty, calls, err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	largest := 0
	reader := &messageBodyReader{ctx: ctx, reader: messageTestReader(func(buffer []byte) (int, error) { largest = len(buffer); cancel(); return len(buffer), nil })}
	_, err := reader.Read(make([]byte, messageReadChunkBytes+1))
	if !errors.Is(err, context.Canceled) || largest > messageReadChunkBytes {
		t.Fatalf("cancellation/read bound: %d %v", largest, err)
	}
}

type messageTestReader func([]byte) (int, error)

func (read messageTestReader) Read(buffer []byte) (int, error) { return read(buffer) }
