package cli

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"
)

type bytesCompareCase struct {
	Name     string `json:"name"`
	LeftHex  string `json:"leftHex"`
	RightHex string `json:"rightHex"`
	Order    int    `json:"order"`
}

func byteOrder(left, right []byte) int {
	result := bytes.Compare(left, right)
	if result < 0 {
		return -1
	}
	if result > 0 {
		return 1
	}
	return 0
}

func bytesCompareCases() []bytesCompareCase {
	var cases []bytesCompareCase
	add := func(name string, left, right []byte) {
		cases = append(cases, bytesCompareCase{name, hex.EncodeToString(left), hex.EncodeToString(right), byteOrder(left, right)})
	}
	for _, tc := range bytesEqualCases() {
		left, _ := hex.DecodeString(tc.LeftHex)
		right, _ := hex.DecodeString(tc.RightHex)
		add(tc.Name, left, right)
	}
	add("unsigned-high-bit", []byte{128}, []byte{127})
	add("first-difference-before-length", []byte{255}, []byte{0, 255})
	add("first-difference-before-later-byte", []byte{97, 255}, []byte{98, 0})
	add("not-numeric", []byte("10"), []byte("2"))
	prefix := bytes.Repeat([]byte{9}, 384)
	add("long-prefix-mismatch", append(bytes.Clone(prefix), 128), append(bytes.Clone(prefix), 127))
	add("long-proper-prefix", prefix, append(bytes.Clone(prefix), 0))
	return cases
}

func bytesCompareSource(cases []bytesCompareCase) (string, map[string]any) {
	var source strings.Builder
	source.WriteString(`module main;
import std::bytes as bytes;
fn i32 forward(ref u8[] left, ref u8[] right) { return bytes::compare(left, right); }
fn ref u8[] retain(ref u8[] view) { return view; }
fn void check(ref u8[] left, ref u8[] right, i32 want) {
    assert(bytes::compare(left, right) == want);
    assert(bytes::compare(right, left) == -want);
    assert(forward(left, right) == want && forward(right, left) == -want);
    assert(bytes::equal(left, right) == (want == 0));
    if (want < 0) { assert(bytes::compare(left, right) < 0); }
    else if (want > 0) { assert(bytes::compare(left, right) > 0); }
}
fn void ordered(ref u8[] a, ref u8[] b, ref u8[] c) {
    assert(bytes::compare(a, b) < 0 && bytes::compare(b, c) < 0);
    assert(bytes::compare(a, c) < 0 && bytes::compare(c, a) > 0);
}
fn void preserved(ref u8[] actual, ref u8[] saved) {
    assert(actual.len == saved.len);
    var usize at = 0;
    while (at < actual.len) { assert(actual[at] == saved[at]); at += 1; }
}
fn i32 main() {
`)
	literal := func(raw []byte) string {
		parts := make([]string, len(raw))
		for i, v := range raw {
			parts[i] = fmt.Sprint(v)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	for _, tc := range cases {
		left, _ := hex.DecodeString(tc.LeftHex)
		right, _ := hex.DecodeString(tc.RightHex)
		a := append(append([]byte{85}, left...), 170)
		b := append(append([]byte{17}, right...), 238)
		fmt.Fprintf(&source, "    { var u8[%d] a = %s; var u8[%d] b = %s; let u8[%d] savedA = %s; let u8[%d] savedB = %s; let ref u8[] av = ref a; let ref u8[] bv = ref b; check(av[1..%d], bv[1..%d], %d); preserved(av, ref savedA); preserved(bv, ref savedB); }\n",
			len(a), literal(a), len(b), literal(b), len(a), literal(a), len(b), literal(b), len(left)+1, len(right)+1, tc.Order)
	}

	all := make([]byte, 256)
	wants := make([]string, 256)
	for i := range all {
		all[i] = byte(i)
		wants[i] = fmt.Sprint(byteOrder([]byte{byte(i)}, []byte{128}))
	}
	fmt.Fprintf(&source, `    {
        var u8[256] all = %s;
        let u8[256] saved = %s;
        let i32[256] wants = [%s];
        let u8[1] pivot = [128];
        let ref u8[] view = ref all;
        var usize i = 0;
        while (i < view.len) { check(view[i..i+1], ref pivot, wants[i]); i += 1; }
        preserved(view, ref saved);
    }
`, literal(all), literal(all), strings.Join(wants, ", "))

	// Independent oracle sorts an intentionally unordered bounded domain. Each
	// strictly increasing triple must retain its order through the Fairy helper.
	domain := [][]byte{{255, 0}, {128}, {}, {1}, {0, 0}, {255}, {127}, {0}}
	sort.Slice(domain, func(i, j int) bool { return bytes.Compare(domain[i], domain[j]) < 0 })
	var triples [][3]string
	for i := 0; i < len(domain); i++ {
		for j := i + 1; j < len(domain); j++ {
			for k := j + 1; k < len(domain); k++ {
				// Numeric arrays preserve raw bytes through the current MIR codec;
				// its separate invalid-UTF-8 byte-literal limitation is documented.
				a := append([]byte{85}, domain[i]...)
				b := append([]byte{17}, domain[j]...)
				c := append([]byte{238}, domain[k]...)
				fmt.Fprintf(&source, "    { let u8[%d] a = %s; let u8[%d] b = %s; let u8[%d] c = %s; let ref u8[] av = ref a; let ref u8[] bv = ref b; let ref u8[] cv = ref c; ordered(av[1..], bv[1..], cv[1..]); }\n", len(a), literal(a), len(b), literal(b), len(c), literal(c))
				triples = append(triples, [3]string{hex.EncodeToString(domain[i]), hex.EncodeToString(domain[j]), hex.EncodeToString(domain[k])})
			}
		}
	}
	source.WriteString(`
    assert(bytes::compare(b"", b"") == 0);
    {
        var u8[5] owner = [77, 9, 9, 9, 88];
        let u8[5] saved = [77, 9, 9, 9, 88];
        let ref u8[] view = ref owner;
        check(view, view, 0);
        check(view[1..3], view[2..4], 0);
        check(view[0..0], view[5..5], 0);
        check(view[0..2], view[1..3], 1);
        preserved(view, ref saved);
        owner[0] = 66;
        assert(owner[0] == 66);
    }
    {
        var u8[2] data = [3, 4];
        let ref mut u8[] view = ref mut data;
        assert(bytes::compare(view, view) == 0);
        assert(forward(view, b"\x03\x04") == 0);
        let i32 result = bytes::compare(view, b"\x03\x05");
        view[0] = 9;
        assert(result == -1 && bytes::compare(view, b"\x09\x04") == 0);
        let ref u8[] child = retain(view);
        assert(bytes::compare(child, b"\x09\x04") == 0);
        view[1] = 7;
        assert(bytes::compare(view, b"\x09\x07") == 0);
    }
    return 0;
}
`)
	return source.String(), map[string]any{"pairs": cases, "singleBytePivot": 128, "singleByteOrders": wants, "orderedTriplesHex": triples}
}

func TestBytesCompareExecution(t *testing.T) {
	text, cases := bytesCompareSource(bytesCompareCases())
	runByteViewExecution(t, text, "compare", cases, "normalized Go bytes.Compare")
}

func TestBytesCompareBorrowConflicts(t *testing.T) {
	testByteViewBorrowConflicts(t, "compare")
}

func TestBytesCompareSDKCache(t *testing.T) {
	testByteViewSDKCache(t, "module main; import std::bytes as bytes; fn i32 main() { assert(bytes::compare(b\"abc\", b\"abd\") == -1); return 0; }")
}

func TestBytesCompareSDKExportIdentity(t *testing.T) {
	testByteViewSDKExportIdentity(t, "compare", "i32", "/// Orders unsigned logical bytes", "/// Borrows the initialized prefix for read-only access.", "0.0.0-stage0.77", "assert(bytes::compare(b\"a\", b\"a\") == 0);")
}
