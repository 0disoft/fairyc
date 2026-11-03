package pecoff

import (
	"bytes"
	"fmt"
	"math"
	"slices"
	"testing"
)

func dependencyStringFixture(length, sections int) ([]byte, Report, uint32) {
	image := append(bytes.Repeat([]byte{'x'}, length), 0)
	report := Report{}
	for i := range sections {
		report.Sections = append(report.Sections, Section{VirtualAddress: uint32(0x1000 * (i + 1)), RawSize: uint32(len(image)), VirtualSize: uint32(len(image))})
	}
	return image, report, report.Sections[sections-1].VirtualAddress
}

func TestDependencyASCIIReadBoundariesAndOwnership(t *testing.T) {
	for _, length := range []int{0, 1, 255, 256, 1023, 1024} {
		for _, limit := range []uint32{0, 1, 256, 1024} {
			image, report, rva := dependencyStringFixture(length, 1)
			before := slices.Clone(image)
			got, err := dependencyASCII(image, report, rva, limit)
			wantErr := ""
			switch {
			case limit == 0 || uint32(length) >= limit:
				wantErr = "unterminated import string"
			case length == 0:
				wantErr = "empty import string"
			}
			if wantErr != "" {
				if err == nil || err.Error() != wantErr {
					t.Fatalf("length=%d limit=%d: error=%v, want %q", length, limit, err, wantErr)
				}
			} else {
				if err != nil || got != string(image[:length]) {
					t.Fatalf("length=%d limit=%d: string=%q error=%v", length, limit, got, err)
				}
				image[0] = 'y'
				if got != string(before[:length]) {
					t.Fatal("returned name aliases the caller's image")
				}
			}
		}
	}
}

func TestDependencyASCIICrossSectionAndErrorPrecedence(t *testing.T) {
	report := Report{Sections: []Section{
		{VirtualAddress: 0x1000, RawSize: 2, VirtualSize: 2},
		{VirtualAddress: 0x1002, RawOffset: 4, RawSize: 3, VirtualSize: 3},
	}}
	image := []byte{'a', 'b', '!', '!', 'c', 'd', 0}
	if got, err := dependencyASCII(image, report, 0x1000, 5); err != nil || got != "abcd" {
		t.Fatalf("adjacent virtual sections: %q/%v", got, err)
	}
	for _, invalid := range []byte{32, 127, 255} {
		bad := slices.Clone(image)
		bad[1] = invalid
		if _, err := dependencyASCII(bad, report, 0x1000, 5); err == nil || err.Error() != "non-ASCII import string" {
			t.Fatalf("non-ASCII before a section boundary: %v", err)
		}
	}
	// The next virtual byte is unmapped even though the raw bytes are present.
	report.Sections[1].VirtualAddress++
	if _, err := dependencyASCII(image, report, 0x1000, 5); err == nil || err.Error() != "generated COMDAT range 0x1002+1 is not file-backed" {
		t.Fatalf("unmapped boundary: %v", err)
	}
	if _, err := dependencyASCII(image, report, 0, 5); err == nil || err.Error() != "zero import string RVA" {
		t.Fatalf("zero RVA: %v", err)
	}
	last := Report{Sections: []Section{{VirtualAddress: math.MaxUint32, RawSize: 1, VirtualSize: 1}}}
	if _, err := dependencyASCII([]byte{'x'}, last, math.MaxUint32, 2); err == nil || err.Error() != "import string RVA overflow" {
		t.Fatalf("RVA overflow: %v", err)
	}
}

func TestDependencyASCIISectionValidationStopsAtTerminator(t *testing.T) {
	for _, test := range []struct {
		data []byte
		want string
	}{
		{[]byte{'!', '~', 0}, "!~"},
		{[]byte{'a', 0, 255}, "a"},
		{[]byte{'a', 32, 0}, ""},
		{[]byte{'a', 127, 0}, ""},
		{[]byte{'a', 255, 0}, ""},
	} {
		report := Report{Sections: []Section{{VirtualAddress: 0x1000, RawSize: 3, VirtualSize: 3}}}
		got, err := dependencyASCII(test.data, report, 0x1000, 3)
		if test.want != "" {
			if err != nil || got != test.want {
				t.Fatalf("data=%v: got %q/%v", test.data, got, err)
			}
		} else if err == nil || err.Error() != "non-ASCII import string" {
			t.Fatalf("data=%v: accepted invalid ASCII: %v", test.data, err)
		}
	}
	for _, virtualShorter := range []bool{false, true} {
		section := Section{VirtualAddress: 0x1000, RawSize: 3, VirtualSize: 2}
		if !virtualShorter {
			section.RawSize, section.VirtualSize = section.VirtualSize, section.RawSize
		}
		if _, err := dependencyASCII([]byte{'a', 'b', 0}, Report{Sections: []Section{section}}, 0x1000, 3); err == nil ||
			err.Error() != "generated COMDAT range 0x1002+1 is not file-backed" {
			t.Fatalf("read escaped raw/virtual section bounds: %v", err)
		}
	}
}

func BenchmarkDependencyASCII(b *testing.B) {
	for _, length := range []int{32, 1023} {
		b.Run(fmt.Sprint(length), func(b *testing.B) {
			image, report, rva := dependencyStringFixture(length, 64)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				name, err := dependencyASCII(image, report, rva, 1024)
				if err != nil || len(name) != length {
					b.Fatalf("import name: %q/%v", name, err)
				}
			}
		})
	}
}
