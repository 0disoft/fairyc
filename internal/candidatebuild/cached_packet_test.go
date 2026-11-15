package candidatebuild

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func TestCachedPacketRejectsDeviceFile(t *testing.T) {
	t.Parallel()
	sum := sha256.Sum256(nil)
	if data, err := cachedPacket(os.DevNull, hex.EncodeToString(sum[:])); err == nil || data != nil {
		t.Fatalf("device accepted as a cached packet: bytes=%d err=%v", len(data), err)
	}
}

type cachedPacketTestInfo struct {
	size int64
	mode fs.FileMode
}

func (info cachedPacketTestInfo) Name() string       { return "packet" }
func (info cachedPacketTestInfo) Size() int64        { return info.size }
func (info cachedPacketTestInfo) Mode() fs.FileMode  { return info.mode }
func (info cachedPacketTestInfo) ModTime() time.Time { return time.Time{} }
func (info cachedPacketTestInfo) IsDir() bool        { return info.mode.IsDir() }
func (info cachedPacketTestInfo) Sys() any           { return nil }

type cachedPacketTestFile struct {
	reader            io.Reader
	info              fs.FileInfo
	statErr, closeErr error
	reads, bytesRead  int
	closes            int
	afterRead         func(int, error)
	afterClose        func()
}

func (file *cachedPacketTestFile) Stat() (fs.FileInfo, error) { return file.info, file.statErr }
func (file *cachedPacketTestFile) Close() error {
	file.closes++
	if file.afterClose != nil {
		file.afterClose()
	}
	return file.closeErr
}
func (file *cachedPacketTestFile) Read(buffer []byte) (int, error) {
	file.reads++
	n, err := file.reader.Read(buffer)
	file.bytesRead += n
	if file.afterRead != nil {
		file.afterRead(n, err)
	}
	return n, err
}
func cachedPacketTestDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func cachedPacketTestPolicy(file *cachedPacketTestFile) cachedPacketReadOptions {
	return cachedPacketReadOptions{Open: func(string) (fs.File, error) { return file, nil }}
}

func TestCachedPacketByteBoundaries(t *testing.T) {
	t.Parallel()
	for _, size := range []int{31, 32, 33} {
		for _, staleMetadata := range []bool{false, true} {
			t.Run(fmt.Sprintf("bytes=%d/stale-stat=%t", size, staleMetadata), func(t *testing.T) {
				contents := bytes.Repeat([]byte("a"), size)
				metadataSize := int64(size)
				if staleMetadata {
					metadataSize = 0 // File grew after Stat; the streaming bound must still apply.
				}
				file := &cachedPacketTestFile{reader: bytes.NewReader(contents), info: cachedPacketTestInfo{size: metadataSize}}
				policy := cachedPacketTestPolicy(file)
				policy.MaxBytes = 32
				data, err := cachedPacketWithOptions("packet", cachedPacketTestDigest(contents), policy)
				if size > 32 {
					if err == nil || data != nil || !staleMetadata && file.reads != 0 || file.bytesRead > 33 {
						t.Fatalf("oversized packet: bytes=%d reads=%d err=%v", file.bytesRead, file.reads, err)
					}
				} else if err != nil || !bytes.Equal(data, contents) {
					t.Fatalf("bounded packet: data=%q err=%v", data, err)
				}
				if file.closes != 1 {
					t.Fatalf("file closes=%d", file.closes)
				}
			})
		}
	}
	// Exercise the real 96 MiB default without allocating a 96 MiB fixture.
	// Metadata is a preflight observation, not a substitute for the pinned digest.
	for _, size := range []int64{maxPacket - 1, maxPacket, maxPacket + 1} {
		t.Run(fmt.Sprintf("default-stat=%d", size), func(t *testing.T) {
			contents := []byte("packet")
			file := &cachedPacketTestFile{reader: bytes.NewReader(contents), info: cachedPacketTestInfo{size: size}}
			data, err := cachedPacketWithOptions("packet", cachedPacketTestDigest(contents), cachedPacketTestPolicy(file))
			if size > maxPacket {
				if err == nil || data != nil || file.reads != 0 {
					t.Fatalf("oversized metadata: reads=%d err=%v", file.reads, err)
				}
			} else if err != nil || !bytes.Equal(data, contents) {
				t.Fatalf("default boundary: data=%q err=%v", data, err)
			}
			if file.closes != 1 {
				t.Fatalf("file closes=%d", file.closes)
			}
		})
	}
}

func TestCachedPacketReadFailureOwnership(t *testing.T) {
	t.Parallel()
	failure := errors.New("injected I/O failure")
	for _, phase := range []string{"open", "stat", "read", "close", "nil-info", "negative-size", "directory", "pipe", "digest"} {
		t.Run(phase, func(t *testing.T) {
			contents := []byte("packet")
			file := &cachedPacketTestFile{reader: bytes.NewReader(contents), info: cachedPacketTestInfo{size: int64(len(contents))}}
			policy := cachedPacketTestPolicy(file)
			switch phase {
			case "open":
				policy.Open = func(string) (fs.File, error) { return file, failure }
			case "stat":
				file.statErr = failure
			case "read":
				file.reader = iotest.ErrReader(failure)
			case "close":
				file.closeErr = failure
			case "nil-info":
				file.info = nil
			case "negative-size":
				file.info = cachedPacketTestInfo{size: -1}
			case "directory":
				file.info = cachedPacketTestInfo{mode: fs.ModeDir}
			case "pipe":
				file.info = cachedPacketTestInfo{mode: fs.ModeNamedPipe}
			case "digest":
				file.reader = strings.NewReader("changed packet")
			}
			data, err := cachedPacketWithOptions("packet", cachedPacketTestDigest(contents), policy)
			if err == nil || data != nil || file.closes != 1 {
				t.Fatalf("failure ownership: data=%q closes=%d err=%v", data, file.closes, err)
			}
			if (phase == "open" || phase == "stat" || phase == "read" || phase == "close") && !errors.Is(err, failure) {
				t.Fatalf("lost I/O cause: %v", err)
			}
			if phase != "read" && phase != "close" && phase != "digest" && file.reads != 0 {
				t.Fatalf("read rejected file %d times", file.reads)
			}
		})
	}
}

func TestCachedPacketCancellation(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"before-open", "after-open", "after-read", "at-EOF", "at-close"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("cached read stopped")
			contents := []byte("packet")
			file := &cachedPacketTestFile{reader: bytes.NewReader(contents), info: cachedPacketTestInfo{size: int64(len(contents))}}
			opens := 0
			policy := cachedPacketReadOptions{Context: ctx, Open: func(string) (fs.File, error) {
				opens++
				if phase == "after-open" {
					cancel(cause)
				}
				return file, nil
			}}
			file.afterRead = func(n int, err error) {
				if phase == "after-read" || phase == "at-EOF" && err == io.EOF {
					cancel(cause)
				}
			}
			file.afterClose = func() {
				if phase == "at-close" {
					cancel(cause)
				}
			}
			if phase == "before-open" {
				cancel(cause)
			}
			data, err := cachedPacketWithOptions("packet", cachedPacketTestDigest(contents), policy)
			if data != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || file.closes != opens {
				t.Fatalf("cancellation: data=%q opens=%d closes=%d err=%v", data, opens, file.closes, err)
			}
			if phase == "before-open" && opens != 0 || phase == "after-open" && file.reads != 0 || phase == "after-read" && file.reads != 1 {
				t.Fatalf("I/O continued: opens=%d reads=%d", opens, file.reads)
			}
		})
	}
}

type cachedPacketEmptyReader struct{}

func (cachedPacketEmptyReader) Read([]byte) (int, error) { return 0, nil }

func TestCachedPacketNoProgressIsBounded(t *testing.T) {
	t.Parallel()
	file := &cachedPacketTestFile{reader: cachedPacketEmptyReader{}, info: cachedPacketTestInfo{}}
	data, err := cachedPacketWithOptions("packet", cachedPacketTestDigest(nil), cachedPacketTestPolicy(file))
	if data != nil || !errors.Is(err, io.ErrNoProgress) || file.reads != 100 || file.closes != 1 {
		t.Fatalf("no progress: reads=%d closes=%d err=%v", file.reads, file.closes, err)
	}
}

func TestCachedPacketReadOptionsAdmission(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"missing-path", "invalid-digest", "negative-limit", "raised-limit", "multiple-options"} {
		t.Run(name, func(t *testing.T) {
			path, digest := "packet", cachedPacketTestDigest(nil)
			opens := 0
			options := []cachedPacketReadOptions{{Open: func(string) (fs.File, error) {
				opens++
				return nil, nil
			}}}
			switch name {
			case "missing-path":
				path = ""
			case "invalid-digest":
				digest = strings.Repeat("z", 64)
			case "negative-limit":
				options[0].MaxBytes = -1
			case "raised-limit":
				options[0].MaxBytes = maxPacket + 1
			case "multiple-options":
				options = append(options, cachedPacketReadOptions{})
			}
			if data, err := cachedPacketWithOptions(path, digest, options...); err == nil || data != nil || opens != 0 {
				t.Fatalf("invalid policy performed I/O: opens=%d data=%q err=%v", opens, data, err)
			}
		})
	}
	policy := cachedPacketReadOptions{Open: func(string) (fs.File, error) { return nil, nil }}
	if data, err := cachedPacketWithOptions("packet", cachedPacketTestDigest(nil), policy); err == nil || data != nil {
		t.Fatalf("nil opened file: data=%q err=%v", data, err)
	}
	file := &cachedPacketTestFile{reader: strings.NewReader("x"), info: cachedPacketTestInfo{size: 1}}
	policy = cachedPacketTestPolicy(file)
	policy.MaxBytes = 1
	if data, err := cachedPacketWithOptions("packet", cachedPacketTestDigest([]byte("x")), policy); err != nil || string(data) != "x" || file.closes != 1 {
		t.Fatalf("minimum limit: data=%q closes=%d err=%v", data, file.closes, err)
	}
}
