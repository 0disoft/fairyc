package packageartifact

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkValidateStableIndirection(b *testing.B) {
	for _, depth := range []int{16, 256, 1024} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			value := strings.Repeat("ptr ", depth) + "i32"
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if !validStableType(value) {
					b.Fatal("rejected stable pointer type")
				}
			}
		})
	}
}

func TestStableTypeValidationGrammar(t *testing.T) {
	nominal := "pkg[" + testCoordinateID(t, "validation").String() + "]::api::Item"
	for _, value := range []string{
		"i32", "[i32]", "ptr mut i32", "ref mut Option<ptr u8>",
		"Option< i32 >", "Result<i32, Option<u8>>", "u8[2][3]",
		nominal, nominal + "[]", nominal + "[32]", "Option<" + nominal + "[]>",
		"Result<ref " + nominal + ", ptr mut u8>",
	} {
		if !validStableType(value) {
			t.Errorf("rejected valid stable type %q", value)
		}
	}
	for _, value := range []string{
		"", "ref ", "ptr mut ", "$p3::api::Item", "pkg[broken]::api::Item",
		"u8[0]", "u8[18446744073709551616]", "Result<i32>", "Result<i32, bad>",
		"Result<bad, i32>", "Option<>", "Option<i32, u8>", "ptr bad", "ref i32[]",
		nominal + "[][]", nominal + "[2][]", nominal + "::", "ptr \xff",
	} {
		for _, wrap := range []func(string) string{
			func(s string) string { return s },
			func(s string) string { return "ptr mut " + s },
			func(s string) string { return "Result<i32, " + s + ">" },
		} {
			if wrapped := wrap(value); validStableType(wrapped) {
				t.Errorf("accepted invalid stable type %q", wrapped)
			}
		}
	}
}

func TestStableTypeValidationDeepIndirection(t *testing.T) {
	// Deep persisted spellings must not consume one call frame or intern one
	// opaque HIR node for every pointer/borrow layer.
	prefix := strings.Repeat("ptr mut ref ", 16384)
	for _, leaf := range []string{"i32", "Option<i32>", "Result<i32, u8>"} {
		if !validStableType(prefix + leaf) {
			t.Fatalf("rejected deep valid type ending in %s", leaf)
		}
	}
	for _, leaf := range []string{"", "bad", "Result<i32, bad>", "$p1::api::Item", "\xff"} {
		if validStableType(prefix + leaf) {
			t.Fatalf("accepted deep invalid type ending in %q", leaf)
		}
	}
}

func TestStableCompoundValidationAcrossParserDepth(t *testing.T) {
	for _, depth := range []int{16, 255, 256, 257, 512} {
		for _, shape := range []string{"option", "result-success", "result-failure"} {
			for _, leaf := range []struct {
				value string
				valid bool
			}{{"i32", true}, {"ptr ref Option<u8>", true}, {"bad", false}, {"Result<i32, bad>", false}} {
				t.Run(fmt.Sprintf("%s/%d/%s", shape, depth, leaf.value), func(t *testing.T) {
					var value string
					switch shape {
					case "option":
						value = strings.Repeat("Option<", depth) + leaf.value + strings.Repeat(">", depth)
					case "result-success":
						value = strings.Repeat("Result<", depth) + leaf.value + strings.Repeat(", i32>", depth)
					case "result-failure":
						value = strings.Repeat("Result<i32, ", depth) + leaf.value + strings.Repeat(">", depth)
					}
					if got := validStableType(value); got != leaf.valid {
						t.Fatalf("validation=%v, want %v", got, leaf.valid)
					}
				})
			}
		}
	}
}

func BenchmarkValidateStableCompound(b *testing.B) {
	for _, depth := range []int{16, 64, 256} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			value := strings.Repeat("Option<", depth) + "i32" + strings.Repeat(">", depth)
			b.ReportAllocs()
			for b.Loop() {
				if !validStableType(value) {
					b.Fatal("rejected nested stable Option type")
				}
			}
		})
	}
}
