package main

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/cli"
)

func runStableNativeTestArgsProof(root string) (int, error) {
	const identity = "tests::process_args::program_arguments_are_forwarded"
	testRoot := filepath.Join(root, "examples", "native_test_args_quickstart")
	programArguments := []string{"--locked", "", "한글λ"}
	cases := []struct {
		name      string
		arguments []string
		lockState string
	}{
		{
			name: "development",
			arguments: []string{
				"test", "--native", "--exact", identity,
				"--", programArguments[0], programArguments[1], programArguments[2],
			},
			lockState: "updated",
		},
		{
			name: "release",
			arguments: []string{
				"test", "--native", "--release", "--exact", identity, "--locked",
				"--", programArguments[0], programArguments[1], programArguments[2],
			},
			lockState: "verified",
		},
	}
	for _, testCase := range cases {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.Run(testCase.arguments, testRoot, &stdout, &stderr)
		wantStdout := "test " + identity + " ... ok\n" +
			"test result: ok. 1 passed; 0 failed; lock " + testCase.lockState + "\n"
		if code != cli.ExitSuccess || stdout.String() != wantStdout || stderr.Len() != 0 {
			return 0, fmt.Errorf("stable native-test args %s returned code %d, stdout %q, stderr %q", testCase.name, code, stdout.String(), stderr.String())
		}
	}
	return len(cases), nil
}
