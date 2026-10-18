package main

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/cli"
)

func runStableNativeTestTimeoutProof(root string) (int, error) {
	testRoot := filepath.Join(root, "examples", "native_test_timeout_quickstart")
	cases := []struct {
		name      string
		arguments []string
	}{
		{name: "development", arguments: []string{"test", "--native", "--timeout-seconds", "1"}},
		{name: "release", arguments: []string{"test", "--native", "--release", "--timeout-seconds", "1", "--locked"}},
	}
	for _, testCase := range cases {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.Run(testCase.arguments, testRoot, &stdout, &stderr)
		wantStderr := "fairyc: build failed: execute native test tests::timeout::hangs: MSVC target executable exceeded 1s\n"
		if code != cli.ExitBuildFailure || stdout.Len() != 0 || stderr.String() != wantStderr {
			return 0, fmt.Errorf("stable native-test timeout %s returned code %d, stdout %q, stderr %q", testCase.name, code, stdout.String(), stderr.String())
		}
	}
	return len(cases), nil
}
