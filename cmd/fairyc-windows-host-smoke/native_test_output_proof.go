package main

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/cli"
)

func runStableNativeTestOutputProof(root string) (int, error) {
	testRoot := filepath.Join(root, "examples", "native_test_output_quickstart")
	cases := []struct {
		name      string
		arguments []string
		profile   string
		lockState string
	}{
		{
			name:      "development",
			arguments: []string{"test", "--native", "--json", "--capture-output"},
			profile:   "development",
			lockState: "updated",
		},
		{
			name:      "release",
			arguments: []string{"test", "--native", "--release", "--json", "--capture-output", "--locked"},
			profile:   "release",
			lockState: "verified",
		},
	}
	for _, testCase := range cases {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.Run(testCase.arguments, testRoot, &stdout, &stderr)
		wantStdout := `{"schemaVersion":"fairyc.test-report/v4","engine":"native","target":"x86_64-windows-msvc","profile":"` + testCase.profile + `","tests":[{"identity":"tests::output::alpha_passes","outcome":"passed","trapCategory":null,"stdoutBase64":"AP8=","stderrBase64":""},{"identity":"tests::output::zeta_fails","outcome":"failed","trapCategory":"assertion-failed","stdoutBase64":"","stderrBase64":"CoA="}],"count":2,"passed":1,"failed":1,"status":"failed","lock":"` + testCase.lockState + `"}` + "\n"
		if code != cli.ExitProgramTrap || stdout.String() != wantStdout || stderr.Len() != 0 {
			return 0, fmt.Errorf("stable native-test output %s returned code %d, stdout %q, stderr %q", testCase.name, code, stdout.String(), stderr.String())
		}
	}
	return len(cases) * 2, nil
}
