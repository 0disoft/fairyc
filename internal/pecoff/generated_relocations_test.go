package pecoff

import (
	"math"
	"testing"
)

func TestGeneratedRelocationFormulas(t *testing.T) {
	for _, test := range []struct {
		name         string
		kind         uint16
		base         uint64
		target, site uint32
		addend, want uint64
	}{
		{"rel32", 4, 0x140000000, 0x2000, 0x1000, 0, 0xffc},
		{"rel32-negative-addend", 4, 0x140000000, 0x2000, 0x1000, 0xfffffff8, 0xff4},
		{"rel32-backwards", 4, 0x140000000, 0x1000, 0x2000, 0, 0xffffeffc},
		{"rel32-1", 5, 0x140000000, 0x2000, 0x1000, 0, 0xffb},
		{"rel32-2", 6, 0x140000000, 0x2000, 0x1000, 0, 0xffa},
		{"rel32-3", 7, 0x140000000, 0x2000, 0x1000, 0, 0xff9},
		{"rel32-4", 8, 0x140000000, 0x2000, 0x1000, 0, 0xff8},
		{"rel32-5", 9, 0x140000000, 0x2000, 0x1000, 0, 0xff7},
		{"addr32nb", 3, 0x140000000, 0x2000, 0, 7, 0x2007},
		{"addr64", 1, 0x140000000, 0x2000, 0, 9, 0x140002009},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := generatedRelocationExpected(test.kind, test.base, test.target, test.site, test.addend)
			if err != nil || got != test.want {
				t.Fatalf("got %#x, %v; want %#x", got, err, test.want)
			}
		})
	}
	for _, test := range []struct {
		kind         uint16
		base         uint64
		target, site uint32
		addend       uint64
	}{
		{4, 0, math.MaxUint32, 0, 0}, {4, 0, 0, math.MaxUint32, 0},
		{3, 0, math.MaxUint32, 0, 1}, {1, math.MaxUint64, 1, 0, 0}, {1, math.MaxUint64 - 1, 1, 0, 1},
		{0, 0, 0, 0, 0}, {2, 0, 0, 0, 0}, {10, 0, 0, 0, 0}, {11, 0, 0, 0, 0}, {16, 0, 0, 0, 0},
	} {
		if _, err := generatedRelocationExpected(test.kind, test.base, test.target, test.site, test.addend); err == nil {
			t.Fatalf("accepted overflow/unmeasured kind: %+v", test)
		}
	}
}
