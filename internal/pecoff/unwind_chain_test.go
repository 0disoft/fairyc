package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestInspectAMD64UnwindChains(t *testing.T) {
	for _, links := range []int{1, 2, maximumUnwindChainDepth} {
		image := chainTestImage(links)
		report, err := InspectAMD64Unwind(image)
		if err != nil || report.MaximumChainDepth != links || report.ChainedInfos != 1 {
			t.Fatalf("links=%d report=%+v error=%v", links, report, err)
		}
	}
}

func TestInspectAMD64UnwindRejectsInvalidChains(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		edit       func([]byte)
	}{
		{"self cycle", "cycle", func(b []byte) { binary.LittleEndian.PutUint32(b[testUnwindXDataRawOffset+12:], 0x3000) }},
		{"multi cycle", "cycle", func(b []byte) { binary.LittleEndian.PutUint32(b[testUnwindXDataRawOffset+28:], 0x3000) }},
		{"missing target", "map UNWIND_INFO", func(b []byte) { binary.LittleEndian.PutUint32(b[testUnwindXDataRawOffset+12:], 0x7000) }},
		{"target flags", "invalid version/flags", func(b []byte) { b[testUnwindXDataRawOffset+16] = 1 | 5<<3 }},
		{"target version", "invalid version/flags", func(b []byte) { b[testUnwindXDataRawOffset+32] = 0 }},
		{"frame mismatch", "frame register/offset mismatch", func(b []byte) { b[testUnwindXDataRawOffset+32+3] = 5 }},
		{"frame offset mismatch", "frame register/offset mismatch", func(b []byte) {
			for i := 0; i < 3; i++ {
				b[testUnwindXDataRawOffset+i*16+3] = 5
			}
			b[testUnwindXDataRawOffset+32+3] = 0x15
		}},
		{"invalid tail range", "invalid chained runtime function", func(b []byte) { binary.LittleEndian.PutUint32(b[testUnwindXDataRawOffset+8:], 0x1000) }},
		{"unaligned target", "invalid chained runtime function", func(b []byte) { binary.LittleEndian.PutUint32(b[testUnwindXDataRawOffset+12:], 0x3011) }},
		{"primary prolog outside copied interval", "invalid prolog/frame", func(b []byte) { b[testUnwindXDataRawOffset+32+1] = 2 }},
		{"secondary allocation", "changes fixed stack/frame", func(b []byte) {
			o := testUnwindXDataRawOffset
			copy(b[o+8:o+20], b[o+4:o+16])
			b[o+2], b[o+4], b[o+5] = 1, 0, 2
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			image := chainTestImage(2)
			tc.edit(image)
			if _, err := InspectAMD64Unwind(image); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
	if _, err := InspectAMD64Unwind(chainTestImage(maximumUnwindChainDepth + 1)); err == nil || !strings.Contains(err.Error(), "depth bound") {
		t.Fatalf("depth overflow: %v", err)
	}
}

// Only the secondary entry has a .pdata row. All other links must still be
// inspected, and a primary handler remains legal.
func chainTestImage(links int) []byte {
	image := minimalAMD64PEWithUnwind()
	image = append(image, make([]byte, 0x600)...)
	binary.LittleEndian.PutUint32(image[testUnwindSectionsOffset+80+16:], 0x800)
	for i := 0; i <= links; i++ {
		o := testUnwindXDataRawOffset + i*16
		image[o] = 1 | 4<<3
		binary.LittleEndian.PutUint32(image[o+4:], 0x1000)
		binary.LittleEndian.PutUint32(image[o+8:], 0x1001)
		binary.LittleEndian.PutUint32(image[o+12:], 0x3000+uint32(i+1)*16)
	}
	o := testUnwindXDataRawOffset + links*16
	image[o] = 1 | 1<<3
	binary.LittleEndian.PutUint32(image[o+4:], 0x1000)
	return image
}
