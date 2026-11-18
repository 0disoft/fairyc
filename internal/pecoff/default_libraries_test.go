package pecoff

import (
	"reflect"
	"strings"
	"testing"
)

func TestMSVCLibraryTraceIgnoresUnrelatedDiagnosticBytes(t *testing.T) {
	const search = "Searching C:\\sdk\\libcmt.lib:\r\n"
	const load = "Loaded libcmt.lib(exe_main.obj)\r\n"
	want := []MSVCLibrary{{Name: "libcmt.lib", Path: `C:\sdk\libcmt.lib`, Members: []string{"exe_main.obj"}}}
	for _, diagnostic := range []string{"ICF note: \xff\r\n", "ICF note: \x00\r\n"} {
		got, err := inspectMSVCLibraryTrace([]byte(search + diagnostic + load))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("diagnostic %q changed library evidence: %+v, %v", diagnostic, got, err)
		}
	}
}

func TestMSVCLibraryTraceBoundaries(t *testing.T) {
	base := "Searching C:\\sdk\\libcmt.lib:\nLoaded libcmt.lib(exe_main.obj)\n"
	r, err := inspectMSVCLibraryTrace([]byte(base))
	if err != nil || len(r) != 1 || len(r[0].Members) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	for _, text := range []string{"", strings.ReplaceAll(base, `C:\sdk\`, ""), base + "Searching D:\\other\\libcmt.lib:\n", "Loaded libcmt.lib(exe_main.obj)\n", strings.ReplaceAll(base, "exe_main.obj", ""), strings.Repeat("x", 64*1024+1)} {
		if _, err := inspectMSVCLibraryTrace([]byte(text)); err == nil {
			t.Fatal("accepted incomplete or ambiguous library trace")
		}
	}
	for _, invalid := range []string{"\x00", "\xff"} {
		for _, field := range []string{"sdk", "exe_main.obj"} {
			if _, err := inspectMSVCLibraryTrace([]byte(strings.Replace(base, field, field+invalid, 1))); err == nil {
				t.Errorf("accepted invalid byte %q in library field %q", invalid, field)
			}
		}
	}
	for _, name := range []string{"", `C:\sdk\libcmt.lib`, "../libcmt", "two libs", `lib"cmt`} {
		if _, err := defaultLibraryName(name); err == nil {
			t.Fatalf("accepted unbounded default name %q", name)
		}
	}
}
