package mir

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestMergeVerificationStateBoundaries(t *testing.T) {
	for _, test := range []struct {
		name         string
		left, right  map[hir.SymbolID]bool
		resources    map[hir.SymbolID]ResourceLocal
		want         map[hir.SymbolID]bool
		changed, bad bool
	}{
		{"same", map[hir.SymbolID]bool{1: true}, map[hir.SymbolID]bool{1: true}, nil, map[hir.SymbolID]bool{1: true}, false, false},
		{"scalar-loses-initialization", map[hir.SymbolID]bool{1: true, 2: true}, map[hir.SymbolID]bool{2: true}, nil, map[hir.SymbolID]bool{1: false, 2: true}, true, false},
		{"incoming-scalar-is-not-definite", map[hir.SymbolID]bool{1: true}, map[hir.SymbolID]bool{1: true, 2: true}, nil, map[hir.SymbolID]bool{1: true}, false, false},
		{"false-and-absent-equivalent", map[hir.SymbolID]bool{1: false}, nil, nil, map[hir.SymbolID]bool{1: false}, false, false},
		{"resource-missing-on-right", map[hir.SymbolID]bool{1: true}, nil, map[hir.SymbolID]ResourceLocal{1: {}}, nil, false, true},
		{"resource-missing-on-left", nil, map[hir.SymbolID]bool{1: true}, map[hir.SymbolID]ResourceLocal{1: {}}, nil, false, true},
		{"resource-inactive-on-left", map[hir.SymbolID]bool{1: false}, map[hir.SymbolID]bool{1: true}, map[hir.SymbolID]ResourceLocal{1: {}}, nil, false, true},
		{"inactive-resource-absent", map[hir.SymbolID]bool{}, map[hir.SymbolID]bool{1: false}, map[hir.SymbolID]ResourceLocal{1: {}}, map[hir.SymbolID]bool{}, false, false},
		{"matching-resource-and-scalar-change", map[hir.SymbolID]bool{1: true, 2: true}, map[hir.SymbolID]bool{1: true}, map[hir.SymbolID]ResourceLocal{1: {}}, map[hir.SymbolID]bool{1: true, 2: false}, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			left := verificationState{live: test.left, flags: map[string]flagState{"drop": {initialized: true}}, regions: map[hir.SymbolID]string{9: "region"}}
			right := verificationState{live: test.right, flags: map[string]flagState{"drop": {initialized: true}}, regions: map[hir.SymbolID]string{9: "region"}}
			originalLeft, originalRight := cloneVerificationState(left), cloneVerificationState(right)
			got, changed, bad := mergeVerificationState(left, right, test.resources)
			if changed != test.changed || bad != test.bad || !bad && !reflect.DeepEqual(got.live, test.want) {
				t.Fatalf("got live=%v changed=%t bad=%t; want live=%v changed=%t bad=%t", got.live, changed, bad, test.want, test.changed, test.bad)
			}
			if changed {
				got.live[777] = true
				got.flags["drop"] = flagState{}
				got.regions[9] = "changed"
				if !reflect.DeepEqual(left, originalLeft) || !reflect.DeepEqual(right, originalRight) {
					t.Fatal("merged mutation changed an input snapshot")
				}
			}
		})
	}
}

func TestMergeVerificationStateConvergesOnBackedge(t *testing.T) {
	entry := verificationState{live: map[hir.SymbolID]bool{1: true, 2: true}}
	backedge := verificationState{live: map[hir.SymbolID]bool{2: true}}
	merged, changed, bad := mergeVerificationState(entry, backedge, nil)
	if bad || !changed || merged.live[1] || !merged.live[2] {
		t.Fatal("backedge did not remove conditional initialization")
	}
	again, changed, bad := mergeVerificationState(merged, entry, nil)
	if bad || changed || !sameLiveState(merged.live, again.live) {
		t.Fatal("revisiting entry did not preserve the fixed point")
	}
}

func TestMergeVerificationStateRejectsCapabilityMismatch(t *testing.T) {
	for _, region := range []bool{false, true} {
		left := verificationState{live: map[hir.SymbolID]bool{1: true}, flags: map[string]flagState{"drop": {initialized: true}}, regions: map[hir.SymbolID]string{2: "left"}}
		right := cloneVerificationState(left)
		if region {
			right.regions[2] = "right"
		} else {
			right.flags["drop"] = flagState{}
		}
		if _, changed, bad := mergeVerificationState(left, right, nil); changed || !bad {
			t.Fatalf("region=%t: incompatible capability accepted", region)
		}
	}
}

func BenchmarkMergeVerificationState(b *testing.B) {
	for _, size := range []int{1024, 4096} {
		for _, changed := range []bool{false, true} {
			b.Run(fmt.Sprintf("locals=%d/changed=%t", size, changed), func(b *testing.B) {
				left := verificationState{live: make(map[hir.SymbolID]bool, size), flags: map[string]flagState{"drop": {initialized: true}}, regions: map[hir.SymbolID]string{0: "region"}}
				for i := range size {
					left.live[hir.SymbolID(i+1)] = true
				}
				right := cloneVerificationState(left)
				if changed {
					delete(right.live, 1)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					merged, gotChanged, bad := mergeVerificationState(left, right, nil)
					if bad || gotChanged != changed || merged.live[1] == changed {
						b.Fatal("incorrect merged initialization")
					}
				}
			})
		}
	}
}
