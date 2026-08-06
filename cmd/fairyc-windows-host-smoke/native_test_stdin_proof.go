package main

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/cli"
)

func runStableNativeTestStdinProof(root string) (int, error) {
	const identity = "tests::stdin::caller_input_is_forwarded"
	testRoot := filepath.Join(root, "examples", "native_test_stdin_quickstart")
	input := []byte{'F', 'a', 'i', 'r', 'y', 0, '\n'}
	cases := []struct {
		name      string
		arguments []string
		lockState string
	}{
		{
			name:      "development",
			arguments: []string{"test", "--native", "--stdin", "--exact", identity},
			lockState: "updated",
		},
		{
			name:      "release",
			arguments: []string{"test", "--native", "--release", "--stdin", "--exact", identity, "--locked"},
			lockState: "verified",
		},
	}
	for _, testCase := range cases {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.RunWithInput(testCase.arguments, testRoot, bytes.NewReader(input), &stdout, &stderr)
		wantStdout := "test " + identity + " ... ok\n" +
			"test result: ok. 1 passed; 0 failed; lock " + testCase.lockState + "\n"
		if code != cli.ExitSuccess || stdout.String() != wantStdout || stderr.Len() != 0 {
			return 0, fmt.Errorf("stable native-test stdin %s returned code %d, stdout %q, stderr %q", testCase.name, code, stdout.String(), stderr.String())
		}
	}
	return len(cases), nil
}
