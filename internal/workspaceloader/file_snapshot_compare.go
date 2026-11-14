package workspaceloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
)

const snapshotCompareBufferBytes = 32 << 10

func matchBoundedRootFile(ctx context.Context, root *os.Root, name string, maximum int, expected, buffer []byte) (bool, error) {
	file, err := openBoundedRootFile(root, name, maximum)
	if err != nil {
		return false, err
	}
	defer file.Close()
	return matchBoundedSourceReader(ctx, file, maximum, expected, buffer)
}

// Continue after a byte mismatch so later I/O/size errors retain precedence.
// The buffer belongs to one Matches call, never to the reusable snapshot.
func matchBoundedSourceReader(ctx context.Context, input io.Reader, maximum int, expected, buffer []byte) (bool, error) {
	if len(buffer) == 0 {
		return false, fmt.Errorf("source comparison buffer is empty")
	}
	reader := io.LimitReader(input, int64(maximum)+1)
	offset, matches := 0, true
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		n, err := reader.Read(buffer)
		if offset > len(expected) || n > len(expected)-offset || !bytes.Equal(buffer[:n], expected[offset:offset+n]) {
			matches = false
		}
		offset += n
		if err != nil {
			if err != io.EOF {
				return false, err
			}
			break
		}
	}
	if offset > maximum {
		return false, fmt.Errorf("file exceeds %d-byte implementation guard", maximum)
	}
	return matches && offset == len(expected), nil
}
