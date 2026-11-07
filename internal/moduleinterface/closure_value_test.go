package moduleinterface

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const closurePublicSource = `module api;
pub struct Holder{fn i32(i32) action;}
pub fn fn i32(i32) make(i32 seed){let fn i32(i32) action=fn [i32 seed] i32(i32 x){return seed+x;};return action;}
pub fn i32 apply(fn i32(i32) action,i32 x){return action(x);}`

// Fixed v5 wire vectors, assembled independently with big-endian uint64 length
// prefixes, three false item flags, and empty legacy ABI/effect/ability fields.
// Do not compute expected bytes through Encode or update them for a body edit.
const closurePublicBytes = "ZmFpcnljLnB1YmxpYy1pbnRlcmZhY2UABQAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA2FwaQAAAAAAAAADAAAAAAAAAAhmdW5jdGlvbgAAAAAAAAAFYXBwbHkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAC2ZuIGkzMihpMzIpAAAAAAAAAANpMzIAAAAAAAAAA2kzMgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACGZ1bmN0aW9uAAAAAAAAAARtYWtlAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAANpMzIAAAAAAAAAC2ZuIGkzMihpMzIpAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAGc3RydWN0AAAAAAAAAAZIb2xkZXIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAZhY3Rpb24AAAAAAAAAC2ZuIGkzMihpMzIpAAAAAAAAAAA="
const legacyClosureFreePublicBytes = "ZmFpcnljLnB1YmxpYy1pbnRlcmZhY2UABQAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA2FwaQAAAAAAAAABAAAAAAAAAAhmdW5jdGlvbgAAAAAAAAAFdmFsdWUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA2kzMgAAAAAAAAAAAAAAAAAAAAA="

func TestClosureValuePublicInterfaceExactBytesAndImmutability(t *testing.T) {
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: testCoordinateID(t, "closure-public")}
	for _, tc := range []struct{ name, source, bytes, hash string }{
		{"closure", closurePublicSource, closurePublicBytes, "fyif5-sha256-bde0c1544468f3fe6d0150dde8c0d14068d65225e1ab9b6d14c3524a6e5aded5"},
		{"closure-body-edit", strings.Replace(closurePublicSource, "return seed+x;", "return seed+x+1;", 1), closurePublicBytes, "fyif5-sha256-bde0c1544468f3fe6d0150dde8c0d14068d65225e1ab9b6d14c3524a6e5aded5"},
		{"legacy", `module api;pub fn i32 value(){return 42;}`, legacyClosureFreePublicBytes, "fyif5-sha256-2a5358d58cba908ccc28bb1758e764a2b13ad69ebd6d713ff4f03eccf738f41f"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := analyzeSource(t, tc.source)
			before, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			descriptor, err := Extract(value, hir.DefaultPackage, coordinates)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Encode(descriptor)
			if err != nil {
				t.Fatal(err)
			}
			want, err := base64.StdEncoding.DecodeString(tc.bytes)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("public interface bytes changed: error=%v got=%x want=%x", err, got, want)
			}
			hash, err := Compute(descriptor)
			if err != nil || hash.String() != tc.hash {
				t.Fatalf("fixed public interface hash: %s %v want %s", hash, err, tc.hash)
			}
			after, err := json.Marshal(value)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("public interface extraction/encoding/hash mutated source HIR", err)
			}
			t.Logf("%s bytes=%d hash=%s", tc.name, len(got), hash)
		})
	}
}

func TestClosureValuePublicInterfaceAdmitsOnlyExactFunctionLeaf(t *testing.T) {
	if !builtinType(hir.TypeClosureI32) {
		t.Fatal("exact canonical closure leaf missing")
	}
	for _, text := range []string{"fn i32()", "fn i32(i32,i32)", "fn bool(i32)", "fn i32(u32)", "fn  i32(i32)", "Fn i32(i32)", "fn i32(i32) trailing"} {
		value := hir.ParseType(text)
		if builtinType(value) {
			t.Fatalf("arbitrary function type admitted as builtin: %q", text)
		}
		if _, err := stableType(value, hir.DefaultPackage, nil, nil); err == nil {
			t.Fatalf("arbitrary function type admitted as canonical public type: %q", text)
		}
	}
}
