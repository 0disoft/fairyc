package packagecache

import (
	"bytes"
	"io"
	"os"
)

// Cap speculative allocation from metadata for sparse files, and retain the
// byte-limit sentinel if a file changes after its metadata was read.
const maxDiskReadHint int64 = 8 << 20

func readDiskRecordBytes(file *os.File, maxBytes uint64) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return readDiskBytes(file, info.Size(), maxBytes)
}

func readDiskBytes(reader io.Reader, sizeHint int64, maxBytes uint64) ([]byte, error) {
	if sizeHint > 0 && uint64(sizeHint) > maxBytes {
		return nil, ErrRecordTooLarge
	}
	var buffer bytes.Buffer
	// ReadFrom needs room for its EOF probe to avoid growing a full buffer.
	buffer.Grow(int(max(0, min(sizeHint, maxDiskReadHint))) + bytes.MinRead)
	if _, err := buffer.ReadFrom(io.LimitReader(reader, diskReadLimit(maxBytes))); err != nil {
		return nil, err
	}
	if uint64(buffer.Len()) > maxBytes {
		return nil, ErrRecordTooLarge
	}
	return buffer.Bytes(), nil
}
