package cli

import (
	"reflect"
	"testing"
)

func TestWorkspaceTestDirectoriesStopsAtExactPackageTestRoot(t *testing.T) {
	t.Parallel()
	directories := workspaceTestDirectories(
		[]string{"libs/tests/member/tests/unit/example.fy"},
		"libs/tests/member/tests",
	)
	want := []string{"libs/tests/member/tests", "libs/tests/member/tests/unit"}
	if !reflect.DeepEqual(directories, want) {
		t.Fatalf("workspace test directories = %v, want %v", directories, want)
	}
}
