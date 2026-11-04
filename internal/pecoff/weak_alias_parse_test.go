package pecoff

import (
	"encoding/binary"
	"fmt"
	"slices"
	"testing"
)

func weakPlanLargeObject(count int) []byte {
	object := ordinaryObjectWithSharedNames(count, "local_symbol")
	for index := range count {
		object[testCOFFSymbolOffset+index*18+16] = coffStorageClassStatic
	}
	return object
}

func TestWeakPlanLargeObjectPreservesSelectionAndOwnership(t *testing.T) {
	chain, err := BuildAMD64WeakChainProbeObject(false)
	if err != nil {
		t.Fatal(err)
	}
	large := weakPlanLargeObject(16384)
	beforeChain, beforeLarge := slices.Clone(chain), slices.Clone(large)
	plan, err := PlanDirectWeakAlias([]WeakAliasObject{
		{Name: "large.obj", Bytes: large}, {Name: "weak.obj", Bytes: chain},
	}, "weak.obj", "fy_wa")
	if err != nil || plan.Selected != "fy_fa" || plan.Owner != "weak.obj" || !slices.Equal(plan.Path, []string{"fy_wa", "fy_wb", "fy_fa"}) {
		t.Fatalf("plan=%+v error=%v", plan, err)
	}
	if !slices.Equal(chain, beforeChain) || !slices.Equal(large, beforeLarge) {
		t.Fatal("weak inventory or planning modified caller-owned objects")
	}
}

func BenchmarkWeakPlanObjectParsing(b *testing.B) {
	chain, err := BuildAMD64WeakChainProbeObject(false)
	if err != nil {
		b.Fatal(err)
	}
	for _, count := range []int{16384, 65536} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			inputs := []WeakAliasObject{{Name: "large.obj", Bytes: weakPlanLargeObject(count)}, {Name: "weak.obj", Bytes: chain}}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				plan, err := PlanDirectWeakAlias(inputs, "weak.obj", "fy_wa")
				if err != nil || plan.Selected != "fy_fa" {
					b.Fatalf("plan=%+v error=%v", plan, err)
				}
			}
		})
	}
}

func weakPlanDirectiveObject(text string) []byte {
	object := testPickAnyObject("entry", "fallback", 7)
	start := len(object)
	object = append(object, text...)
	header := object[coffFileHeaderBytes+coffSectionHeaderBytes:]
	copy(header, ".drectve")
	binary.LittleEndian.PutUint32(header[16:], uint32(len(text)))
	binary.LittleEndian.PutUint32(header[20:], uint32(start))
	symbolStart := binary.LittleEndian.Uint32(object[8:])
	definition := object[symbolStart+18:]
	copy(definition, ".drectve")
	binary.LittleEndian.PutUint32(definition[18:], uint32(len(text)))
	return object
}

func TestWeakPlanReusesValidatedDirectiveInventory(t *testing.T) {
	object := weakPlanDirectiveObject("/ALTERNATENAME:root=fallback")
	before := slices.Clone(object)
	inventory, err := InspectWeakAliases(object)
	if err != nil || !slices.Equal(inventory.Directives, []WeakAliasDirective{{Name: "root", Fallback: "fallback"}}) {
		t.Fatalf("inventory=%+v error=%v", inventory, err)
	}
	plan, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "directive.obj", Bytes: object}}, "directive.obj", "root")
	if err != nil || plan.Owner != "directive.obj" || plan.Selected != "fallback" || !slices.Equal(plan.Path, []string{"root", "fallback"}) {
		t.Fatalf("plan=%+v error=%v", plan, err)
	}
	if !slices.Equal(object, before) {
		t.Fatal("directive inspection or planning modified caller-owned bytes")
	}
	// Returned directive strings and plan paths remain detached from the input.
	clear(object)
	if inventory.Directives[0].Name != "root" || inventory.Directives[0].Fallback != "fallback" || plan.Path[0] != "root" {
		t.Fatal("returned inventory or plan aliases caller-owned bytes")
	}
}

func TestWeakPlanPreservesAdmissionAndDirectiveErrorOrder(t *testing.T) {
	malformed := weakPlanDirectiveObject("/alternatename:root=")
	for _, inspect := range []func([]byte) error{
		func(raw []byte) error { _, err := InspectWeakAliases(raw); return err },
		func(raw []byte) error {
			_, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "directive.obj", Bytes: raw}}, "directive.obj", "root")
			return err
		},
	} {
		if err := inspect(malformed); err == nil || err.Error() != "invalid alternate-name directive" {
			t.Fatalf("malformed directive: %v", err)
		}
		badObject := slices.Clone(malformed)
		binary.LittleEndian.PutUint16(badObject, 0x14c)
		_, want := InspectObject(badObject)
		if err := inspect(badObject); want == nil || err == nil || err.Error() != want.Error() {
			t.Fatalf("object admission must precede directive parsing: got %v, want %v", err, want)
		}
	}
	// The second object's malformed directive must still precede its conflicting
	// strong definitions; collecting definitions first would change the error.
	_, err := PlanDirectWeakAlias([]WeakAliasObject{
		{Name: "first.obj", Bytes: weakPlanDirectiveObject("/alternatename:other=fallback")},
		{Name: "second.obj", Bytes: malformed},
	}, "first.obj", "other")
	if err == nil || err.Error() != "invalid alternate-name directive" {
		t.Fatalf("directive validation must precede strong-definition ambiguity: %v", err)
	}
}
