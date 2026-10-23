package foreignstructprobe

import (
	"fmt"
	"strings"
)

// Reuse the same independent value/copy oracle with nominal C enum elements.
// Values outside the declared variants must survive transport without narrowing.
func cEnumArraySources(posix bool) (string, string) {
	var fairy, c strings.Builder
	fairyTemplate := strings.NewReplacer("c_array_", "enum_array_", "c::%[1]s", "FixtureEnumArray_%[1]s").Replace(cScalarArrayFairyTemplate)
	cTemplate := strings.ReplaceAll(cScalarArrayCTemplate, "c_array_", "enum_array_")
	var calls []string
	for _, typ := range cArrayProbeTypes(posix) {
		if typ.name == "bool" || typ.name == "float" || typ.name == "double" {
			continue
		}
		fmt.Fprintf(&fairy, "\n@repr(c::%s) enum FixtureEnumArray_%s{Zero;One;Two;}\n", typ.name, typ.name)
		fmt.Fprintf(&fairy, fairyTemplate, typ.name)
		fmt.Fprintf(&c, cTemplate, typ.name, typ.c, typ.first, typ.middle, typ.last)
		calls = append(calls, "check_enum_array_"+typ.name+"(0)", "check_enum_array_"+typ.name+"(1)", "check_enum_array_"+typ.name+"(2)")
	}
	fmt.Fprintf(&fairy, "fn bool check_c_enum_arrays(){return %s;}\n", strings.Join(calls, "&&"))
	return fairy.String(), c.String()
}
