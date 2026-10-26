package cbackend_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestStorageLayoutSharedNestedStructs(t *testing.T) {
	var source strings.Builder
	source.WriteString("module main; struct Layer0 { i32 value; }\n")
	for depth := 1; depth <= 18; depth++ {
		fmt.Fprintf(&source, "struct Layer%d { Layer%d left; Layer%d right; }\n", depth, depth-1, depth-1)
	}
	source.WriteString("fn i32 main() { return 0; }\n")
	program := lowerSource(t, source.String())
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		result, err := cbackend.StorageLayout(program, hir.ParseType("main::Layer18"), profile)
		if err != nil || result.SizeBytes != 4<<18 || result.AlignmentBytes != 4 {
			t.Fatalf("shared nested layout for %s = %+v, %v", profile.Target, result, err)
		}
	}
}
