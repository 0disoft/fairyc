package cli

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/projectscaffold"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateStarterTemplates(t *testing.T) {
	root := t.TempDir()
	if err := projectscaffold.Create(root, projectscaffold.Spec{Name: "hello-fairy", UID: "example.hello-fairy"}); err != nil {
		t.Fatal(err)
	}
	paths := []string{".gitignore", "README.md", "fairy.toml", "src/main.fy", "tests/main.fy"}
	var support strings.Builder
	support.WriteString("#include <assert.h>\n#include <string.h>\n#include <stddef.h>\n#include <stdint.h>\n")
	for i, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, "hello-fairy", filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&support, "static const uint8_t body_%d[]={", i)
		for _, b := range data {
			fmt.Fprintf(&support, "%d,", b)
		}
		support.WriteString("0};\n")
	}
	support.WriteString("void starter_template_check(size_t index,const uint8_t *data,size_t size){switch(index){\n")
	for i := range paths {
		fmt.Fprintf(&support, "case %d:assert(size==sizeof body_%d-1 && memcmp(data,body_%d,size)==0);break;\n", i, i, i)
	}
	support.WriteString("default:assert(0);}}\n")
	runLinuxCandidateFixture(t, []string{"project_starter"}, "project_starter", `fn i32 main(){matches_all_reference_starter_bytes();return 0;}`, support.String())
}
