package windowsbuild

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsBuildConfig(t *testing.T) {
	valid := `{"schema":"fairyc.windows-build/v1","subsystem":"windows","libraries":["user32.lib"],"sources":["native/ui.c"],"resources":["app.rc"]}`
	if _, err := Parse([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		strings.Replace(valid, `"windows"`, `"driver"`, 1),
		strings.Replace(valid, `"user32.lib"`, `"/ENTRY:bad"`, 1),
		strings.Replace(valid, `"native/ui.c"`, `"../ui.c"`, 1),
		strings.Replace(valid, `"native/ui.c"`, `"C:/ui.c"`, 1),
		strings.Replace(valid, `"native/ui.c"`, `"native/ui.cpp"`, 1),
		strings.Replace(valid, `"user32.lib"`, `"user32.lib","user32.lib"`, 1),
		strings.Replace(valid, `"sources":`, `"sources":null,"sources":`, 1),
		strings.Replace(valid, `"resources":`, `"cflags":`, 1),
		valid + `{}`, `null`,
	} {
		if _, err := Parse([]byte(source)); err == nil {
			t.Fatalf("accepted %s", source)
		}
	}
	root := t.TempDir()
	if p, err := Load(root); p != nil || err != nil {
		t.Fatalf("absent config: %v %v", p, err)
	}
	if err := os.Mkdir(filepath.Join(root, "native"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{FileName: valid, "native/ui.c": "int ui(void){return 42;}\n", "app.rc": "1 RCDATA {42}\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Load(root)
	if err != nil || len(p.CInputs) != 1 || len(p.RCInputs) != 1 {
		t.Fatalf("load: %v %v", p, err)
	}
	if err := os.Remove(filepath.Join(root, "app.rc")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("missing declared input was ignored")
	}
}
