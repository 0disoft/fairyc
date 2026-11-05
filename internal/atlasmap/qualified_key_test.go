package atlasmap

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestCodecDeclarationPathsPreserveQualifiedNames(t *testing.T) {
	instance := "fypi1-sha256-" + strings.Repeat("a", 64)
	modules := map[string]struct{}{"main": {}}
	check := func(name string, valid bool) {
		t.Helper()
		key := semanticsnapshot.DeclarationKey{PackageInstance: instance, Module: "main", Kind: semanticsnapshot.DeclarationFunction, Name: name}
		for _, err := range []error{validateLocalKey(key, instance, modules), validateCallTarget(key)} {
			if (err == nil) != valid {
				t.Fatalf("path %q: valid=%v, error=%v", name, valid, err)
			}
		}
	}
	for _, name := range []string{"get", "Cell::get", "outer::Cell::get", "_Cell::_get2"} {
		check(name, true)
	}
	for _, name := range []string{"", "::get", "Cell::", "Cell:::get", "Cell:get", "Cell/get", "Cell.get", "Cell::9get", "Cell::<get>", "Cell::get\x00", "Cell::get\n"} {
		check(name, false)
	}
}
