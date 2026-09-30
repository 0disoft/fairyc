package sdkstdlib

import (
	"slices"
	"strings"
	"testing"
)

func TestBufferScalarFormattingSurfaceIsSDKOwned(t *testing.T) {
	t.Parallel()

	definition, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	var bytesSource *Source
	for index := range definition.Sources {
		if definition.Sources[index].Path == "std/bytes.fy" {
			bytesSource = &definition.Sources[index]
			break
		}
	}
	if bytesSource == nil {
		t.Fatal("embedded std::bytes source is missing")
	}

	public := make([]string, 0, 4)
	for _, function := range bytesSource.Module.Functions {
		switch function.Name {
		case "append_i32", "append_i64", "append_u32", "append_usize", "append_bool":
			if !function.Public {
				t.Errorf("std::bytes::%s is not public", function.Name)
			}
			public = append(public, function.Name)
		}
	}
	slices.Sort(public)
	if want := []string{"append_bool", "append_i32", "append_i64", "append_u32", "append_usize"}; !slices.Equal(public, want) {
		t.Fatalf("public formatting functions = %v, want %v", public, want)
	}

	sourceText := string(bytesSource.Bytes)
	for _, forbidden := range []string{
		"fmt\"",
		"format_plan",
		"va_list",
		"fy_std_buffer_append_i32",
		"fy_std_buffer_append_u32",
		"fy_std_buffer_append_usize",
		"fy_std_buffer_append_bool",
	} {
		if strings.Contains(sourceText, forbidden) {
			t.Errorf("std::bytes scalar formatting introduced forbidden runtime or syntax surface %q", forbidden)
		}
	}
	if strings.Count(sourceText, "return append(buffer, encoded[position..]);") != 4 {
		t.Fatal("integer formatting must commit through one existing Buffer append per value")
	}
}
