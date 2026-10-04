package cli

import (
	"os"
	"strings"
	"testing"
)

func TestSharedIntegerForNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/shared_integer_for.fy")
	if err != nil {
		t.Fatal(err)
	}
	runI64ViewNativeCases(t, "shared-integer-for", []struct{ name, source, trap string }{
		{"stored-matrix", string(source), ""},
		{"body-trap", `module main;fn i32 main(){let i64[1] data=[7];for(let ref i64 item in data){assert(*item==8);}return 0;}`, "assertion-failed"},
	})
}

func TestSharedIntegerForCheckRejections(t *testing.T) {
	for name, sample := range map[string]struct{ source, diagnostic string }{
		"mutable-owner": {`fn void bad(ref mut i64[] values){for(let ref i64 item in values){assert(*item>0);values[0]=9;break;}}`, "E2101"},
		"array-owner":   {`fn void bad(){var i64[1] data=[1];for(let ref i64 item in data){assert(*item>0);data[0]=9;return;}}`, "E2101"},
		"local-escape":  {`fn ref i64 bad(){var i64[1] data=[1];for(let ref i64 item in data){return item;}return ref data[0];}`, "E2102"},
		"shared-store":  {`fn void bad(ref i64[] values){for(let ref i64 item in values){*item=9;}}`, "E1008"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, "module main;"+sample.source+"fn i32 main(){return 0;}")
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, sample.diagnostic) {
				t.Fatalf("expected %s: %d %s", sample.diagnostic, code, diag)
			}
		})
	}
}
