package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestVersion2EpilogRanges(t *testing.T) {
	for _, test := range []struct {
		name         string
		codes        []byte
		prolog, size uint32
		want         string
	}{
		{"end", []byte{3, 0x16, 0, 6, 2, 0x60, 1, 0x70}, 2, 16, ""},
		{"explicit", []byte{1, 6, 2, 6, 4, 0x42}, 4, 22, ""},
		{"twelve-bit-offset", []byte{2, 6, 1, 0x16}, 2, 512, ""},
		{"zero-size", []byte{0, 0x16}, 0, 16, "size/flags"},
		{"reserved-info", []byte{3, 0x26}, 0, 16, "size/flags"},
		{"missing-offset", []byte{3, 6}, 0, 16, "truncated"},
		{"offset-before-begin", []byte{3, 6, 17, 6}, 0, 16, "range"},
		{"range-past-end", []byte{3, 6, 2, 6}, 0, 16, "range"},
		{"prolog-overlap", []byte{3, 0x16}, 14, 16, "range"},
		{"duplicate", []byte{3, 6, 4, 6, 4, 6}, 0, 16, "overlapping"},
		{"overlap", []byte{3, 6, 4, 6, 5, 6}, 0, 16, "overlapping"},
	} {
		t.Run(test.name, func(t *testing.T) {
			image := minimalAMD64PEWithUnwind()
			binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset+4:], 0x1000+test.size)
			binary.LittleEndian.PutUint32(image[testUnwindSectionsOffset+8:], test.size)
			image[testUnwindXDataRawOffset] = 2
			image[testUnwindXDataRawOffset+1] = byte(test.prolog)
			image[testUnwindXDataRawOffset+2] = byte(len(test.codes) / 2)
			copy(image[testUnwindXDataRawOffset+4:], test.codes)
			_, err := InspectAMD64Unwind(image)
			if test.want == "" && err != nil || test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("error = %v; want %q", err, test.want)
			}
			_, err = plainUnwindSignatureSize(image[testUnwindXDataRawOffset:], test.size)
			if test.want == "" && err != nil || test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("plain signature error = %v; want %q", err, test.want)
			}
		})
	}
}
