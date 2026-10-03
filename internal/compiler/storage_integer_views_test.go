package compiler_test

import (
	"github.com/0disoft/fairyc/internal/hir"
	"strings"
	"testing"
)

func TestIntegerStorageViewsMIR(t *testing.T) {
	for _, element := range []string{"i32", "u32", "usize"} {
		for _, mutable := range []bool{false, true} {
			t.Run(element+map[bool]string{false: "/shared", true: "/exclusive"}[mutable], func(t *testing.T) {
				source := i64StorageViewSource
				if mutable {
					source = mutableI64StorageViewSource()
				}
				checkStorageViewMIR(t, strings.ReplaceAll(source, "i64", element), mutable, hir.Type("["+element+"]"))
			})
		}
	}
}
