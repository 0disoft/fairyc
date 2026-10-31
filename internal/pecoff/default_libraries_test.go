package pecoff

import (
	"strings"
	"testing"
)

func TestMSVCLibraryTraceBoundaries(t *testing.T) {
	base := "Searching C:\\sdk\\libcmt.lib:\nLoaded libcmt.lib(exe_main.obj)\n"
	r, err := inspectMSVCLibraryTrace([]byte(base))
	if err != nil || len(r) != 1 || len(r[0].Members) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	for _, text := range []string{"", strings.ReplaceAll(base, `C:\sdk\`, ""), base + "Searching D:\\other\\libcmt.lib:\n", "Loaded libcmt.lib(exe_main.obj)\n", strings.ReplaceAll(base, "exe_main.obj", ""), base + "\x00", strings.Repeat("x", 64*1024+1)} {
		if _, err := inspectMSVCLibraryTrace([]byte(text)); err == nil {
			t.Fatal("accepted incomplete or ambiguous library trace")
		}
	}
	for _, name := range []string{"", `C:\sdk\libcmt.lib`, "../libcmt", "two libs", `lib"cmt`} {
		if _, err := defaultLibraryName(name); err == nil {
			t.Fatalf("accepted unbounded default name %q", name)
		}
	}
}
