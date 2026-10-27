package moduleinterface

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
)

func TestComputeMatchesV5Encoding(t *testing.T) {
	t.Parallel()
	descriptor := Descriptor{
		StateLayouts: []StateLayout{{ContractUID: "example.stream", Hash: contractstate.LayoutHash{1}}},
		Modules: []Module{{Path: "a_empty"}, {Path: "api", Items: []Item{
			{Kind: "enum", Name: "CState", CBase: "c.uint", Variants: []Variant{{Name: "Ready"}, {Name: "Busy"}}},
			{Kind: "enum", Name: "Payload", Variants: []Variant{{Name: "Data", Fields: []Field{{Name: "code", Type: "i32"}}}}},
			{Kind: "foreign-function", Name: "raw", Unsafe: true, ABI: "C", LinkName: "raw_v1", Parameters: []string{"i32", "u32"}, ResultType: "i32"},
			{Kind: "function", Name: "read", MustUse: true, OwnerType: "Thing", Effects: []string{"console"}, Parameters: []string{"ref Thing"}, ResultType: "u32"},
			{Kind: "resource", Name: "Handle", Abilities: []string{"drop"}},
			{Kind: "struct", Name: "Thing", ReprC: true, Fields: []Field{{Name: "x", Type: "u32"}, {Name: "y", Type: "i32"}}},
		}}, {Path: "z_empty"}},
	}
	encoded, err := Encode(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	want := Hash(sha256.Sum256(encoded))
	// Captured from the v5 encoder before incremental hashing was introduced.
	const golden = "9d374d69f864549b83ba1d4f6450637bf1f8827b37b98af70493a7df93bd806d"
	if len(encoded) != 1024 || fmt.Sprintf("%x", [32]byte(want)) != golden {
		t.Fatalf("v5 encoding changed: bytes=%d hash=%x", len(encoded), [32]byte(want))
	}
	for range 2 {
		got, err := Compute(descriptor)
		if err != nil || got != want {
			t.Fatalf("Compute() = %x, %v; want %x", [32]byte(got), err, [32]byte(want))
		}
	}
}

func TestComputeAndEncodeRejectInvalidDescriptors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value Descriptor
	}{
		{"no modules", Descriptor{}},
		{"empty module path", Descriptor{Modules: []Module{{}}}},
		{"unordered modules", Descriptor{Modules: []Module{{Path: "z"}, {Path: "a"}}}},
		{"duplicate items", Descriptor{Modules: []Module{{Path: "api", Items: []Item{{Kind: "function", Name: "f"}, {Kind: "function", Name: "f"}}}}}},
		{"empty layout hash", Descriptor{StateLayouts: []StateLayout{{ContractUID: "example.stream"}}, Modules: []Module{{Path: "api"}}}},
		{"empty resource abilities", Descriptor{Modules: []Module{{Path: "api", Items: []Item{{Kind: "resource", Name: "Asset"}}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, encodeErr := Encode(tc.value)
			hash, computeErr := Compute(tc.value)
			if encodeErr == nil || computeErr == nil || encodeErr.Error() != computeErr.Error() || hash != (Hash{}) {
				t.Fatalf("Encode() error=%v; Compute() hash=%v error=%v", encodeErr, hash, computeErr)
			}
		})
	}
}

func BenchmarkComputeLargeInterface(b *testing.B) {
	for _, count := range []int{64, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			items := make([]Item, count)
			for i := range items {
				items[i] = Item{Kind: "function", Name: fmt.Sprintf("function_%05d", i),
					Parameters: []string{"i32", "i32", "i32", "i32"}, ResultType: "i32"}
			}
			descriptor := Descriptor{Modules: []Module{{Path: "main", Items: items}}}
			encoded, err := Encode(descriptor)
			if err != nil {
				b.Fatal(err)
			}
			want := Hash(sha256.Sum256(encoded))
			b.SetBytes(int64(len(encoded)))
			b.ReportAllocs()
			for b.Loop() {
				if got, err := Compute(descriptor); err != nil || got != want {
					b.Fatalf("interface hash = %x, %v", got, err)
				}
			}
		})
	}
}
