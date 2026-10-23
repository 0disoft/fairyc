package workspacecompiler

import (
	"path/filepath"
	"testing"
)

func TestRegionCapabilityStorageIsSourceDiagnostic(t *testing.T) {
	for _, item := range []struct{ name, declaration, primary string }{
		{"return", "fn Region bad(Region input) { return input; }", "Region"},
		{"let", "fn i32 bad(Region input) { let Region copy = input; return 0; }", "copy"},
		{"var", "fn i32 bad(Region input) { var Region copy = input; return 0; }", "copy"},
		{"inferred", "fn i32 bad(Region input) { let copy = input; return 0; }", "copy"},
	} {
		t.Run(item.name, func(t *testing.T) {
			input := "module main;\n" + item.declaration + "\nfn i32 main() { return 0; }\n"
			root := writeCacheWorkspace(t, testAnswer42)
			writeFile(t, filepath.Join(root, "src", "main.fy"), input)
			checked, err := Check(loadCacheWorkspace(t, root))
			if err != nil || checked.Verified || len(checked.Semantic.Diagnostics) != 1 {
				t.Fatalf("Region storage: verified=%v diagnostics=%+v error=%v", checked.Verified, checked.Semantic.Diagnostics, err)
			}
			diagnostic := checked.Semantic.Diagnostics[0]
			if diagnostic.Code != "E3103" || input[diagnostic.Primary.StartByte:diagnostic.Primary.EndByte] != item.primary {
				t.Fatalf("Region storage diagnostic: %+v", diagnostic)
			}
		})
	}
}
