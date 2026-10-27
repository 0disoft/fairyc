package packageartifact

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableTypeRebindMixedDeepIndirection(t *testing.T) {
	coordinate := testCoordinateID(t, "indirection-rebind")
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{7: coordinate}
	nominal := hir.ParseType("$p7::model::Packet")
	stable := "pkg[" + coordinate.String() + "]::model::Packet"
	want := nominal
	for range 512 {
		want = hir.SharedBorrow(hir.RawPointer(hir.MutableBorrow(hir.RawPointer(want, false)), true))
	}
	deep := strings.Repeat("ref ptr mut ref mut ptr ", 512) + stable
	for _, test := range []struct {
		name, value string
		want        hir.Type
	}{
		{"mixed", deep, want},
		{"option", "Option<" + deep + ">", hir.Option(want)},
		{"result", "Result<" + deep + ", i32>", hir.Result(want, hir.TypeI32)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got, err := rebindStableType(test.value, coordinates); err != nil || got != test.want {
				t.Fatalf("indirection order or nominal binding changed: %v", err)
			}
		})
	}
	for _, bindings := range []map[hir.PackageID]packagegraph.CoordinateID{
		nil, {7: coordinate, 9: coordinate},
	} {
		if got, err := rebindStableType(deep, bindings); err == nil || got != hir.TypeInvalid {
			t.Fatal("unresolved or ambiguous nominal binding accepted")
		}
	}
	for _, invalid := range []string{deep + "[0]", strings.Repeat("ptr ", 2048), "ref mut " + deep + "\xff"} {
		if got, err := rebindStableType(invalid, coordinates); err == nil || got != hir.TypeInvalid {
			t.Fatal("malformed deep indirection accepted")
		}
	}
}

func TestStableTypeRebindLongIndirection(t *testing.T) {
	const depth = 16384
	want := hir.TypeI32
	for range depth {
		want = hir.RawPointer(want, false)
	}
	if got, err := rebindStableType(strings.Repeat("ptr ", depth)+"i32", nil); err != nil || got != want {
		t.Fatalf("long indirection did not round trip: %v", err)
	}
}

func BenchmarkRebindStableIndirection(b *testing.B) {
	for _, depth := range []int{0, 64, 512, 2048} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			value := strings.Repeat("ptr ", depth) + "i32"
			want := hir.TypeI32
			for range depth {
				want = hir.RawPointer(want, false)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if got, err := rebindStableType(value, nil); err != nil || got != want {
					b.Fatalf("rebind failed: %v", err)
				}
			}
		})
	}
}
