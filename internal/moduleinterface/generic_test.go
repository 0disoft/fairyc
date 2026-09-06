package moduleinterface_test

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/moduleinterface"
)

func TestGenericInterfaceHashCanonicalAndDistinct(t *testing.T) {
	descriptor := moduleinterface.Descriptor{Modules: []moduleinterface.Module{{Path: "util"}}}
	a := moduleinterface.GenericTemplate{Module: "util", Name: "a", Closure: genericplan.DigestText("test", "a")}
	b := moduleinterface.GenericTemplate{Module: "util", Name: "b", Closure: genericplan.DigestText("test", "b")}
	input := []moduleinterface.GenericTemplate{b, a}
	first, err := moduleinterface.ComputeGeneric(descriptor, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := moduleinterface.ComputeGeneric(descriptor, []moduleinterface.GenericTemplate{a, b})
	if err != nil || first != second || !reflect.DeepEqual(input, []moduleinterface.GenericTemplate{b, a}) {
		t.Fatal("noncanonical or mutating generic interface hash")
	}
	if _, err := moduleinterface.ParseHash(first.String()); err == nil {
		t.Fatal("generic identity accepted as a stable v5 interface")
	}
	for _, invalid := range [][]moduleinterface.GenericTemplate{{a, a}, {{Module: "absent", Name: "a", Closure: a.Closure}}, {{Module: "util", Name: "a"}}} {
		if _, err := moduleinterface.ComputeGeneric(descriptor, invalid); err == nil {
			t.Fatal("invalid template interface accepted")
		}
	}
	descriptor.Modules[0].Items = []moduleinterface.Item{{Kind: "function", Name: "a", ResultType: "i32"}}
	if _, err := moduleinterface.ComputeGeneric(descriptor, []moduleinterface.GenericTemplate{a}); err == nil {
		t.Fatal("ordinary/generic public collision accepted")
	}
}
