package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestAtlasEditBodyOutsideProductionRoots(t *testing.T) {
	root, args := editFixture(t)
	for _, path := range []string{"src/body.txt", "libs/answer/src/body.txt"} {
		writeWorkspaceFile(t, root, path, "{return 1;}")
		before := remoteTree(t, root)
		current := append([]string{}, args...)
		current[8] = path
		code, out, diagnostic := runForTest(current, root)
		if code != 7 || out != "" || !strings.Contains(diagnostic, "outside production source roots") || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("production body %s: %d %s %s", path, code, out, diagnostic)
		}
	}
	writeWorkspaceFile(t, root, "libs/answer/src2/body.txt", "{return 1;}")
	current := append([]string{}, args...)
	current[8] = "libs/answer/src2/body.txt"
	before := remoteTree(t, root)
	if code, out, diagnostic := runForTest(current, root); code != 0 || !strings.Contains(out, `"verified":true`) || diagnostic != "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("source root sibling: %d %s %s", code, out, diagnostic)
	}
}
