package cli

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCommandFlagsPreserveTokenAndDiagnosticOrder(t *testing.T) {
	for _, tt := range []struct {
		name       string
		args       []string
		valueFirst bool
		want       string
	}{
		{"unknown", []string{"--other"}, false, `command "test" does not accept flag "--other"`},
		{"no-equals", []string{"--value=1"}, false, `command "test" does not accept flag "--value=1"`},
		{"missing", []string{"--value", "--switch"}, false, `command "test" requires a value for --value`},
		{"duplicate-first", []string{"--value", "1", "--value"}, false, `command "test" received --value more than once`},
		{"value-first", []string{"--value", "1", "--value"}, true, `command "test" requires a value for --value`},
		{"switch-duplicate", []string{"--switch", "--switch"}, false, `command "test" received --switch more than once`},
		{"domain-before-later-token", []string{"--value", "bad", "--unknown"}, false, "domain rejection"},
		{"required-after-parse", nil, false, `command "test" requires --value`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCommandFlags(tt.args, flagSchema{Command: "test", Flags: []flagDefinition{{Name: "--switch", Kind: flagSwitch}, {Name: "--value", Kind: flagScalar, ValueFirst: tt.valueFirst}}, Required: []string{"--value"}}, func(_, v string) error {
				if v == "bad" {
					return errors.New("domain rejection")
				}
				return nil
			})
			if err == nil || err.Error() != tt.want {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCommandFlagsOrderedSlicesAndForwarding(t *testing.T) {
	var applied, forwarded []string
	schema := flagSchema{Command: "test", Flags: []flagDefinition{{Name: "--item", Kind: flagSlice, DuplicateValue: `repeated item %q`}}, Forward: &forwarded}
	seen, err := parseCommandFlags([]string{"--foreign", "--item", "", "--item", "-x", "tail"}, schema, func(_, v string) error { applied = append(applied, v); return nil })
	if err != nil || !seen["--item"] || !reflect.DeepEqual(applied, []string{"", "-x"}) || !reflect.DeepEqual(forwarded, []string{"--foreign", "tail"}) {
		t.Fatalf("tokens: %v %v %v %v", seen, applied, forwarded, err)
	}
	_, err = parseCommandFlags([]string{"--item", "x", "--item", "x"}, schema, func(_, v string) error { return nil })
	if err == nil || err.Error() != `repeated item "x"` {
		t.Fatalf("duplicates: %v", err)
	}
}

func TestStableTestFlagErrorPrecedence(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"--package", "pkg", "--package"}, `command "test" requires a package UID after --package`},
		{[]string{"--package", "pkg", "--package", "other"}, `command "test" received --package more than once`},
		{[]string{"--list", "--native", "--release"}, `command "test" does not combine --list with --native, --release, --stdin, --stdin-file, --capture-output, or --exact`},
		{[]string{"--workspace", "--package", ""}, ""},
		{[]string{"--capture-output"}, `command "test" requires --native when --capture-output is supplied`},
		{[]string{"--native", "--capture-output"}, `command "test" requires --json when --capture-output is supplied`},
		{[]string{"--file", "a.fy", "--file", "a.fy"}, `command "test" received test file "a.fy" more than once`},
	} {
		_, err := parseStableTestCompilerOptions(tt.args)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != tt.want {
			t.Fatalf("%q: got %q, want %q", tt.args, got, tt.want)
		}
	}
}

// Assert the public error envelope as well as parser errors: status, stdout,
// diagnostic prefix, blank line and help bytes remain compiler-owned.
func TestDeclarativeFlagsInvocationContract(t *testing.T) {
	for _, tt := range []struct {
		args    []string
		message string
	}{
		{[]string{"test", "--package", "pkg", "--package"}, `command "test" requires a package UID after --package`},
		{[]string{"test", "--list", "--native"}, `command "test" does not combine --list with --native, --release, --stdin, --stdin-file, --capture-output, or --exact`},
		{[]string{"test", "--native", "--capture-output"}, `command "test" requires --json when --capture-output is supplied`},
		{[]string{"test", "--file", "a.fy", "--file", "a.fy"}, `command "test" received test file "a.fy" more than once`},
		{[]string{"build", "--release", "--release"}, `command "build" received --release more than once`},
		{[]string{"run", "--unknown", "--", "--release"}, `command "run" accepts only optional --release and --locked`},
		{[]string{"experimental", "run", "main.fy", "--source"}, `experimental command "run" requires a value for --source`},
		{[]string{"experimental", "build-target", "main.fy", "--target", "x86_64-windows-msvc", "--profile", "low", "--output"}, `experimental command "build-target" requires a value for --output`},
		{[]string{"experimental", "build-workspace", "--target", "x86_64-linux-gnu", "--profile", "low", "--output", "build/a"}, `experimental build-workspace with Linux target requires --cc gcc or --cc clang`},
		{[]string{"experimental", "atlas-calls", "list"}, `experimental command "atlas-calls list" requires --package-uid`},
		{[]string{"experimental", "atlas-evidence", "show", "--unknown"}, `experimental command "atlas-evidence show" requires --max-fragment-bytes`},
	} {
		var stdout, stderr bytes.Buffer
		code := RunWithInput(tt.args, t.TempDir(), strings.NewReader(""), &stdout, &stderr)
		want := "fairyc: " + tt.message + "\n\n" + usage
		if code != 2 || stdout.Len() != 0 || stderr.String() != want {
			t.Fatalf("%q: code=%d stdout=%q stderr=%q", tt.args, code, stdout.String(), stderr.String())
		}
	}
}
