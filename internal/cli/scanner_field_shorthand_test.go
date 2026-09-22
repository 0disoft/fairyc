package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScannerFieldShorthandNativeSizes(t *testing.T) {
	root, _, _ := parserLabelWorkspace(t)
	data, err := os.ReadFile(filepath.Join(root, "src", "scanner.fy"))
	if err != nil {
		t.Fatal(err)
	}
	changed := string(data)
	plain := changed
	for _, pair := range [][2]string{
		{"Token { .kind, .trivia, .start, .end, .cooked_index = 0 }", "Token { .kind = kind, .trivia = trivia, .start = start, .end = end, .cooked_index = 0 }"},
		{"Token { .kind, .trivia, .start, .end, .cooked_index }", "Token { .kind = kind, .trivia = trivia, .start = start, .end = end, .cooked_index = cooked_index }"},
	} {
		if strings.Count(plain, pair[0]) != 1 {
			t.Fatalf("missing or repeated scanner initializer: %s", pair[0])
		}
		plain = strings.Replace(plain, pair[0], pair[1], 1)
	}
	compareParserNativeEdit(t, root, "src/scanner.fy", plain, changed, "SCANNER_SHORTHAND_NATIVE_SIZES")
}
