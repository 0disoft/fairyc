package contractstatemir

import (
	"strings"
	"testing"
)

func TestStableStateTypeDepthBudget(t *testing.T) {
	for _, prefix := range []string{"ref ", "ref mut ", "ptr ", "ptr mut "} {
		for _, depth := range []int{maxStableStateTypeDepth - 1, maxStableStateTypeDepth, maxStableStateTypeDepth + 1} {
			value := strings.Repeat(prefix, depth) + "i32"
			got, ok := canonicalStableStateBodyType(value)
			if want := depth <= maxStableStateTypeDepth; ok != want || ok && got != value {
				t.Fatalf("prefix %q depth %d: valid=%t", prefix, depth, ok)
			}
		}
	}
}

func TestStableStateTypeRejectsPrefixBombBeforeInterning(t *testing.T) {
	value := strings.Repeat("ref ", 16384) + "i32"
	allocations := testing.AllocsPerRun(3, func() {
		if _, ok := canonicalStableStateBodyType(value); ok {
			t.Fatal("deep type accepted")
		}
	})
	if allocations > 2 {
		t.Fatalf("deep rejection allocated %g times", allocations)
	}
	descriptor := stableStateFunctionDescriptor{name: "module::work", resultType: value}
	if _, err := decodeStableStateFunctionDescriptor(encodeStableStateFunctionDescriptor(descriptor)); err == nil {
		t.Fatal("descriptor accepted an over-depth type")
	}
}
