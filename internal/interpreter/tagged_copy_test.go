package interpreter

import (
	"fmt"
	"testing"
)

func nestedCopyValue(depth int, leaf runtimeValue) runtimeValue {
	for index := 0; index < depth; index++ {
		child := leaf
		kind := valueOption
		variant := "some"
		if index%2 != 0 {
			kind = valueResult
			variant = "ok"
		}
		leaf = runtimeValue{kind: kind, payload: &taggedState{variant: variant, payload: &child}}
	}
	return leaf
}

func BenchmarkNestedTaggedValueCopy(b *testing.B) {
	for _, depth := range []int{16, 64, 128, 255} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			value := nestedCopyValue(depth, i32Value(42))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				benchmarkValue = value.cloneForValueCopy()
			}
		})
	}
}

func TestNestedTaggedValueCopy(t *testing.T) {
	for _, depth := range []int{1, 16, 255, 256} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			original := nestedCopyValue(depth, i32Value(42))
			copy := original.cloneForValueCopy()
			if depth >= maxValueNesting {
				if copy.tagged() != original.tagged() {
					t.Fatal("copy bypassed the nesting bound")
				}
				return
			}
			for original.tagged() != nil {
				if original.tagged() == copy.tagged() || original.tagged().payload == copy.tagged().payload {
					t.Fatal("inert tagged copy aliases original storage")
				}
				original, copy = *original.tagged().payload, *copy.tagged().payload
			}
			if original != copy {
				t.Fatal("copy changed scalar payload")
			}
		})
	}
	resource := runtimeValue{kind: valueResource, payload: &resourceState{}}
	owned := nestedCopyValue(16, resource)
	if owned.cloneForValueCopy().tagged() != owned.tagged() {
		t.Fatal("copy duplicated nested resource ownership")
	}
	cyclic := runtimeValue{kind: valueOption, payload: &taggedState{}}
	cyclic.tagged().payload = &cyclic
	if cyclic.cloneForValueCopy().tagged() != cyclic.tagged() {
		t.Fatal("copy detached cyclic state")
	}
}

func TestTaggedValueCopyProofIsLocalToTraversal(t *testing.T) {
	leaf := copyTestStruct(t)
	inner := nestedCopyValue(16, leaf)
	array := runtimeValue{kind: valueArray, payload: &arrayState{elements: []runtimeValue{inner}}}
	outer, err := optionValue("some", &array)
	if err != nil {
		t.Fatal(err)
	}
	copy := outer.cloneForValueCopy()
	originalChild := outer.tagged().payload.array().elements[0]
	copyChild := copy.tagged().payload.array().elements[0]
	for originalChild.tagged() != nil {
		if originalChild.tagged() == copyChild.tagged() {
			t.Fatal("copy through aggregate shares tagged state")
		}
		originalChild, copyChild = *originalChild.tagged().payload, *copyChild.tagged().payload
	}
	copyChild.structure().fields[0] = i32Value(99)
	if originalChild.structure().fields[0].i32() != 7 {
		t.Fatal("copy through aggregate changed original field")
	}
	// Changing the original after a successful copy must require a fresh proof.
	outer.tagged().payload.array().elements[0] = runtimeValue{kind: valueResource, payload: &resourceState{}}
	if outer.cloneForValueCopy().tagged() != outer.tagged() {
		t.Fatal("a stale inert proof duplicated resource ownership")
	}
}
