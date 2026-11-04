package packagesource

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func directoryOnlyZIP(count int) []byte {
	data := make([]byte, count*(zipDirectoryHeaderBytes+1)+zipEndBytes)
	for index := range count {
		header := data[index*(zipDirectoryHeaderBytes+1):]
		binary.LittleEndian.PutUint32(header, zipDirectorySignature)
		binary.LittleEndian.PutUint16(header[28:], 1)
		header[zipDirectoryHeaderBytes] = 'a'
	}
	end := data[len(data)-zipEndBytes:]
	binary.LittleEndian.PutUint32(end, zipEndSignature)
	binary.LittleEndian.PutUint16(end[8:], uint16(count))
	binary.LittleEndian.PutUint16(end[10:], uint16(count))
	binary.LittleEndian.PutUint32(end[12:], uint32(len(data)-zipEndBytes))
	return data
}

func TestArchiveDirectoryAdmissionBoundaries(t *testing.T) {
	for _, count := range []int{0, 8191, 8192, 8193, 65536 + 8192} {
		data := directoryOnlyZIP(count)
		before := bytes.Clone(data)
		err := admitArchiveDirectory(data)
		wantLimit := count == 0 || count > maxArchivePathEntries
		if errors.Is(err, errArchiveEntryLimit) != wantLimit || (!wantLimit && err != nil) {
			t.Fatalf("count=%d error=%v", count, err)
		}
		if !bytes.Equal(before, data) {
			t.Fatal("admission mutated archive bytes")
		}
		if err64 := admitArchiveDirectory(zip64Archive(data)); errors.Is(err64, errArchiveEntryLimit) != wantLimit || (!wantLimit && err64 != nil) {
			t.Fatalf("ZIP64 count=%d error=%v", count, err64)
		}
	}
	// Go accepts the low 16 bits of the declared count. Admission must count
	// actual headers even when that hint and the declared byte size are false.
	data := directoryOnlyZIP(65536 + 8192)
	end := data[len(data)-zipEndBytes:]
	binary.LittleEndian.PutUint32(end[12:], 0)
	if reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil || len(reader.File) != 65536+8192 {
		t.Fatalf("underreported baseline: %v", err)
	}
	if allocs := testing.AllocsPerRun(2, func() {
		if admitArchiveDirectory(data) != errArchiveEntryLimit {
			panic("entry bomb admitted")
		}
	}); allocs != 0 {
		t.Fatalf("preflight made %g allocations", allocs)
	}
	if _, err := unpack(context.Background(), data, testPin(), "fairyc-language/20"); err != errArchiveEntryLimit {
		t.Fatalf("unpack did not use admission: %v", err)
	}
}

func BenchmarkArchiveDirectoryAdmission(b *testing.B) {
	data := directoryOnlyZIP(65536 + 8192)
	for _, preflight := range []bool{false, true} {
		name := "metadata-first"
		if preflight {
			name = "preflight"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if preflight {
					if admitArchiveDirectory(data) != errArchiveEntryLimit {
						b.Fatal("entry bomb admitted")
					}
				} else if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func zip64Archive(data []byte) []byte {
	endAt := len(data) - zipEndBytes
	oldEnd := data[endAt:]
	result := append(bytes.Clone(data[:endAt]), make([]byte, zip64EndBytes+zip64LocatorBytes)...)
	end64 := result[endAt:]
	binary.LittleEndian.PutUint32(end64, zip64EndSignature)
	binary.LittleEndian.PutUint64(end64[4:], 44)
	binary.LittleEndian.PutUint64(end64[24:], uint64(binary.LittleEndian.Uint16(oldEnd[8:])))
	binary.LittleEndian.PutUint64(end64[32:], uint64(binary.LittleEndian.Uint16(oldEnd[10:])))
	binary.LittleEndian.PutUint64(end64[40:], uint64(binary.LittleEndian.Uint32(oldEnd[12:])))
	binary.LittleEndian.PutUint64(end64[48:], uint64(binary.LittleEndian.Uint32(oldEnd[16:])))
	locator := end64[zip64EndBytes:]
	binary.LittleEndian.PutUint32(locator, zip64LocatorSignature)
	binary.LittleEndian.PutUint64(locator[8:], uint64(endAt))
	binary.LittleEndian.PutUint32(locator[16:], 1)
	result = append(result, oldEnd...)
	end := result[len(result)-zipEndBytes:]
	binary.LittleEndian.PutUint16(end[8:], 0xffff)
	binary.LittleEndian.PutUint16(end[10:], 0xffff)
	binary.LittleEndian.PutUint32(end[12:], 0xffffffff)
	binary.LittleEndian.PutUint32(end[16:], 0xffffffff)
	return result
}

func TestArchiveDirectoryPreservesZIPLayouts(t *testing.T) {
	original := archiveBytes(t)
	baseline, err := unpack(context.Background(), original, testPin(), "fairyc-language/20")
	if err != nil {
		t.Fatal(err)
	}
	commented := append(bytes.Clone(original), strings.Repeat("c", 65535)...)
	binary.LittleEndian.PutUint16(commented[len(original)-2:], 65535)
	misleadingSize := bytes.Clone(original)
	binary.LittleEndian.PutUint32(misleadingSize[len(misleadingSize)-zipEndBytes+12:], 1)
	for name, data := range map[string][]byte{
		"ZIP32":                original,
		"ZIP64":                zip64Archive(original),
		"prefix":               append([]byte("self-extracting prefix\n"), original...),
		"maximum comment":      commented,
		"unadjusted directory": misleadingSize,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := unpack(context.Background(), data, testPin(), "fairyc-language/20")
			if err != nil || !reflect.DeepEqual(baseline, got) {
				t.Fatalf("layout changed bundle: %v", err)
			}
		})
	}
}

func TestArchiveDirectoryRejectsMalformedZIP64(t *testing.T) {
	for _, mutate := range []func([]byte){
		func(data []byte) {
			binary.LittleEndian.PutUint64(data[len(data)-zipEndBytes-zip64LocatorBytes+8:], ^uint64(0))
		},
		func(data []byte) { data[len(data)-zipEndBytes-zip64LocatorBytes-zip64EndBytes] = 0 },
		func(data []byte) {
			binary.LittleEndian.PutUint64(data[len(data)-zipEndBytes-zip64LocatorBytes-zip64EndBytes+40:], ^uint64(0))
		},
		func(data []byte) {
			binary.LittleEndian.PutUint64(data[len(data)-zipEndBytes-zip64LocatorBytes-zip64EndBytes+32:], ^uint64(0))
		},
	} {
		data := zip64Archive(archiveBytes(t))
		mutate(data)
		if err := admitArchiveDirectory(data); err == nil {
			t.Fatal("invalid ZIP64 metadata admitted")
		}
	}
	for size := range zipEndBytes {
		if err := admitArchiveDirectory(make([]byte, size)); !errors.Is(err, zip.ErrFormat) {
			t.Fatalf("truncated ZIP size %d: %v", size, err)
		}
	}
}
