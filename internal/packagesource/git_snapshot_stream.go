package packagesource

import (
	"archive/zip"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

func streamGitObjectCommand(ctx context.Context, repository string, input []byte, consume func(io.Reader) error, args ...string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return consumeGitProcess(configuredGitObjectCommand(ctx, repository, input, args...), cancel, consume)
}

// Stop a rejected producer before Wait: draining arbitrary blob output would
// defeat early admission. Every successfully started child is reaped once.
func consumeGitProcess(command *exec.Cmd, cancel context.CancelFunc, consume func(io.Reader) error) error {
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
	command.Stderr = &boundedGitBuffer{limit: 4096}
	if err = command.Start(); err != nil {
		return fmt.Errorf("local Git object command failed: %w", err)
	}
	readErr := consume(stdout)
	if readErr != nil {
		cancel()
	}
	closeErr := stdout.Close()
	waitErr := command.Wait()
	if readErr != nil {
		return readErr
	}
	if err = errors.Join(closeErr, waitErr); err != nil {
		return fmt.Errorf("local Git object command failed: %w", err)
	}
	return nil
}

func writeGitSnapshotBlobs(ctx context.Context, input io.Reader, entries []gitTreeBlob, writer *zip.Writer) ([]byte, error) {
	reader := bufio.NewReaderSize(input, 128)
	var scratch, manifest []byte
	expanded := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, err := reader.ReadSlice('\n')
		if err != nil {
			return nil, fmt.Errorf("invalid Git blob framing: %w", err)
		}
		header := strings.Split(string(line[:len(line)-1]), " ")
		if len(header) != 3 || header[0] != entry.object || header[1] != "blob" {
			return nil, fmt.Errorf("missing or mismatched Git blob")
		}
		size, err := strconv.ParseUint(header[2], 10, 32)
		if err != nil || strconv.FormatUint(size, 10) != header[2] || size > MaxFileBytes || int(size) > MaxTotalBytes-expanded {
			return nil, fmt.Errorf("Git blob byte bound or framing")
		}
		if entry.name == projectmanifest.FileName && size > projectmanifest.MaxBytes {
			return nil, fmt.Errorf("Git package manifest byte bound")
		}
		if cap(scratch) < int(size) {
			scratch = make([]byte, max(int(size), min(MaxFileBytes, 2*cap(scratch))))
		}
		data := scratch[:int(size)]
		if _, err = io.ReadFull(reader, data); err != nil {
			return nil, fmt.Errorf("truncated Git blob: %w", err)
		}
		if terminator, err := reader.ReadByte(); err != nil || terminator != '\n' {
			return nil, fmt.Errorf("invalid Git blob terminator")
		}
		expanded += int(size)
		if entry.name == projectmanifest.FileName {
			manifest = append([]byte(nil), data...)
		}
		fileHeader := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		fileHeader.SetMode(0644)
		fileHeader.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		file, err := writer.CreateHeader(fileHeader)
		if err != nil {
			return nil, err
		}
		if _, err = file.Write(data); err != nil {
			return nil, err
		}
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		return nil, fmt.Errorf("trailing Git blob data or read failure")
	}
	return manifest, nil
}
