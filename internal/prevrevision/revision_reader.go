package prevrevision

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	revisionReadTimeout    = 2 * time.Minute
	maxRevisionBlobBytes   = 16 << 20
	maxRevisionDiagnostics = 64 << 10
)

// A single lazily started process serves sequential, NUL-framed blob requests.
// Close must be called even after a malformed response or a cancelled read.
type revisionReader struct {
	ctx         context.Context
	cancel      context.CancelFunc
	root, ref   string
	command     *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	output      *bufio.Reader
	diagnostics revisionDiagnostics
	done        chan struct{}
	closed      bool
}

func newRevisionReader(ctx context.Context, root, ref string) *revisionReader {
	ctx, cancel := context.WithTimeout(ctx, revisionReadTimeout)
	return &revisionReader{ctx: ctx, cancel: cancel, root: root, ref: ref}
}

func (r *revisionReader) start() error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if r.closed {
		return fmt.Errorf("Git revision reader is closed")
	}
	if r.command != nil {
		return nil
	}
	command := exec.CommandContext(r.ctx, "git", "-C", r.root, "cat-file", "--batch", "-Z")
	command.WaitDelay = 2 * time.Second
	command.Stderr = &r.diagnostics
	stdin, err := command.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		stdin.Close()
		return err
	}
	if err := command.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return err
	}
	r.command, r.stdin, r.stdout = command, stdin, stdout
	r.output, r.done = bufio.NewReaderSize(stdout, 4096), make(chan struct{})
	go func() {
		select {
		case <-r.ctx.Done():
			// CommandContext kills Git; closing pipes also releases a blocked
			// caller if another process inherited Git's pipe handles.
			stdout.Close()
			stdin.Close()
		case <-r.done:
		}
	}()
	return nil
}

func (r *revisionReader) Read(path string) (data []byte, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("git show %s:%s: %w", r.ref, path, errors.Join(err, r.ctx.Err()))
		}
	}()
	if strings.ContainsRune(r.ref, 0) || strings.ContainsRune(path, 0) {
		return nil, fmt.Errorf("NUL in revision object name")
	}
	if err := r.start(); err != nil {
		return nil, err
	}
	if _, err := io.WriteString(r.stdin, r.ref+":"+path+"\x00"); err != nil {
		return nil, err
	}
	return readRevisionBlob(r.output, maxRevisionBlobBytes)
}

func readRevisionBlob(output *bufio.Reader, limit int64) ([]byte, error) {
	header, err := output.ReadSlice(0)
	if err != nil {
		return nil, fmt.Errorf("read bounded Git blob header: %w", err)
	}
	fields := strings.Fields(string(header[:len(header)-1]))
	if len(fields) != 3 || fields[1] != "blob" {
		return nil, fmt.Errorf("Git revision object is missing or is not a blob")
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || size < 0 || size > limit {
		return nil, fmt.Errorf("Git revision blob exceeds %d byte limit or has invalid size", limit)
	}
	data := make([]byte, int(size))
	if _, err := io.ReadFull(output, data); err != nil {
		return nil, fmt.Errorf("read Git revision blob: %w", err)
	}
	end, err := output.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("invalid Git revision blob terminator: %w", err)
	}
	if end != 0 {
		return nil, fmt.Errorf("invalid Git revision blob terminator")
	}
	return data, nil
}

func (r *revisionReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	defer r.cancel()
	if r.command == nil {
		return r.ctx.Err()
	}
	r.stdin.Close()
	// A rejected blob may still be filling stdout. Close that pipe before
	// waiting so Git cannot wait for us to drain a body we deliberately refused.
	r.stdout.Close()
	err := r.command.Wait()
	close(r.done)
	if err != nil {
		return fmt.Errorf("Git revision reader failed: %w: %s", errors.Join(err, r.ctx.Err()), string(r.diagnostics))
	}
	return r.ctx.Err()
}

type revisionDiagnostics []byte

func (d *revisionDiagnostics) Write(p []byte) (int, error) {
	*d = append(*d, p[:min(len(p), maxRevisionDiagnostics-len(*d))]...)
	return len(p), nil
}
