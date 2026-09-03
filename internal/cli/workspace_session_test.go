package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

const (
	workspaceSessionRootManifest = `schema = "fairyc.manifest/v1"
[package]
uid = "example.session.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/13"
source = "src"
[workspace]
members = ["libs/answer"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`
	workspaceSessionMemberManifest = `schema = "fairyc.manifest/v1"
[package]
uid = "example.session.answer"
name = "answer"
version = "1.0.0"
language = "fairyc-language/13"
source = "src"
`
	workspaceSessionRootSource      = "module main;\nimport answer as answer;\nfn i32 main() { return answer::value(); }\n"
	workspaceSessionRootPrivateEdit = "module main;\nimport answer as answer;\nstruct Hidden { i32 value; }\nfn i32 main() { return answer::value(); }\n"
	workspaceSessionMemberSource    = "module answer;\npub fn i32 value() { return 42; }\n"
)

func TestExperimentalWorkspaceSessionReusesCompleteProcessCache(t *testing.T) {
	t.Parallel()
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	stdin := strings.NewReader(workspaceSessionRequestLine(1) + workspaceSessionRequestLine(2))
	code, stdout, stderr := runWithInputForTest(workspaceSessionArguments(2), root, stdin)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("session = code %d, stderr %q", code, stderr)
	}
	responses := decodeWorkspaceSessionResponses(t, stdout)
	if len(responses) != 2 {
		t.Fatalf("responses = %d, want 2", len(responses))
	}
	first, second := responses[0], responses[1]
	if first.Status != workspaceSessionStatusOK || first.LockState != "updated" || first.Cache.Outcome != workspacecompiler.OutcomeFallback ||
		first.Cache.Hits != 0 || first.Cache.Misses != 2 || !first.Cache.Lowered || first.Cache.RetainedEntries != 2 {
		t.Fatalf("cold response = %+v", first)
	}
	if second.Status != workspaceSessionStatusOK || second.LockState != "unchanged" || second.Cache.Outcome != workspacecompiler.OutcomeAllHit ||
		second.Cache.Hits != 2 || second.Cache.Misses != 0 || second.Cache.Lowered || second.Cache.VerifiedHits != 2 {
		t.Fatalf("warm response = %+v", second)
	}
}

func TestExperimentalWorkspaceSessionObservesEditBetweenRequestsAndRecovers(t *testing.T) {
	t.Parallel()
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	reader := &workspaceSessionStagedReader{stages: []workspaceSessionReaderStage{
		{data: []byte(workspaceSessionRequestLine(1))},
		{before: func() { writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootPrivateEdit) }, data: []byte(workspaceSessionRequestLine(2))},
		{data: []byte(workspaceSessionRequestLine(3))},
	}}
	code, stdout, stderr := runWithInputForTest(workspaceSessionArguments(3), root, reader)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("session = code %d, stderr %q", code, stderr)
	}
	responses := decodeWorkspaceSessionResponses(t, stdout)
	if len(responses) != 3 {
		t.Fatalf("responses = %d, want 3", len(responses))
	}
	if changed := responses[1]; changed.LockState != "updated" || changed.Cache.Outcome != workspacecompiler.OutcomeFallback ||
		changed.Cache.Hits != 1 || changed.Cache.Misses != 1 || !changed.Cache.Lowered || changed.Cache.RetainedEntries != 3 {
		t.Fatalf("changed response = %+v", changed)
	}
	if healed := responses[2]; healed.LockState != "unchanged" || healed.Cache.Outcome != workspacecompiler.OutcomeAllHit ||
		healed.Cache.Hits != 2 || healed.Cache.Misses != 0 || healed.Cache.Lowered {
		t.Fatalf("healed response = %+v", healed)
	}
}

func TestExperimentalWorkspaceSessionKeepsTransientDiagnosticsInsideSession(t *testing.T) {
	t.Parallel()
	root := writeWorkspaceSessionFixture(t, "module main;\nfn i32 main() { return missing; }\n")
	reader := &workspaceSessionStagedReader{stages: []workspaceSessionReaderStage{
		{data: []byte(workspaceSessionRequestLine(1))},
		{before: func() { writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootSource) }, data: []byte(workspaceSessionRequestLine(2))},
	}}
	code, stdout, stderr := runWithInputForTest(workspaceSessionArguments(2), root, reader)
	if code != ExitSuccess || !strings.Contains(stderr, "E0101: unresolved name or path") {
		t.Fatalf("session = code %d, stderr %q", code, stderr)
	}
	responses := decodeWorkspaceSessionResponses(t, stdout)
	if len(responses) != 2 || responses[0].Status != workspaceSessionStatusDiagnostics || responses[0].ExitCode != ExitSourceDiagnostics ||
		responses[0].Diagnostics == 0 || responses[0].Cache.Outcome != "not-eligible" {
		t.Fatalf("diagnostic response = %+v", responses)
	}
	if responses[1].Status != workspaceSessionStatusOK || responses[1].Cache.Outcome != workspacecompiler.OutcomeFallback {
		t.Fatalf("recovered response = %+v", responses[1])
	}

	code, _, _ = runWithInputForTest(workspaceSessionArguments(1), writeWorkspaceSessionFixture(t, "module main;\nfn i32 main() { return missing; }\n"), strings.NewReader(workspaceSessionRequestLine(1)))
	if code != ExitSourceDiagnostics {
		t.Fatalf("single diagnostic request exit = %d", code)
	}
}

func TestParseWorkspaceSessionOptionsRejectsAmbiguousOrUnboundedPolicies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing", arguments: nil, contains: "requires --target"},
		{name: "unknown target", arguments: replaceWorkspaceSessionArgument(workspaceSessionArguments(1), "--target", "unknown"), contains: "known --target"},
		{name: "zero requests", arguments: replaceWorkspaceSessionArgument(workspaceSessionArguments(1), "--max-requests", "0"), contains: "1 through 256"},
		{name: "noncanonical requests", arguments: replaceWorkspaceSessionArgument(workspaceSessionArguments(1), "--max-requests", "01"), contains: "canonical decimal"},
		{name: "too many requests", arguments: replaceWorkspaceSessionArgument(workspaceSessionArguments(1), "--max-requests", "257"), contains: "1 through 256"},
		{name: "too many entries", arguments: replaceWorkspaceSessionArgument(workspaceSessionArguments(1), "--cache-max-entries", "65537"), contains: "1 through 65536"},
		{name: "too many bytes", arguments: replaceWorkspaceSessionArgument(workspaceSessionArguments(1), "--cache-max-bytes", "1073741825"), contains: "1 through 1073741824"},
		{name: "artifact exceeds cache", arguments: replaceWorkspaceSessionArgument(workspaceSessionArguments(1), "--cache-max-bytes", "1024"), contains: "not to exceed"},
		{name: "duplicate", arguments: append(workspaceSessionArguments(1), "--locked", "--locked"), contains: "more than once"},
		{name: "unknown", arguments: append(workspaceSessionArguments(1), "--daemon"), contains: "does not accept"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseWorkspaceSessionOptions(test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("parse error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func TestExperimentalWorkspaceSessionRejectsMalformedFraming(t *testing.T) {
	t.Parallel()
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	tests := []struct {
		name     string
		input    []byte
		contains string
	}{
		{name: "no requests", input: nil, contains: "at least one NDJSON request"},
		{name: "empty line", input: []byte("\n"), contains: "line is empty"},
		{name: "not object", input: []byte("[]\n"), contains: "one JSON object"},
		{name: "missing fields", input: []byte("{}\n"), contains: "requires schemaVersion"},
		{name: "duplicate field", input: []byte(`{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":1,"sequence":1}` + "\n"), contains: "repeats field"},
		{name: "unknown field", input: []byte(`{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":1,"path":"src/main.fy"}` + "\n"), contains: "unknown field"},
		{name: "wrong type", input: []byte(`{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":"1"}` + "\n"), contains: "wrong type"},
		{name: "wrong sequence", input: []byte(strings.Replace(workspaceSessionRequestLine(1), `"sequence":1`, `"sequence":2`, 1)), contains: "want 1"},
		{name: "trailing JSON", input: []byte(strings.TrimSpace(workspaceSessionRequestLine(1)) + ` {}` + "\n"), contains: "trailing JSON"},
		{name: "invalid utf8", input: []byte{0xff, '\n'}, contains: "valid UTF-8"},
		{name: "oversize", input: []byte(strings.Repeat("x", maxWorkspaceSessionRequestBytes+1) + "\n"), contains: "exceeds 1024 bytes"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runWithInputForTest(workspaceSessionArguments(1), root, bytes.NewReader(test.input))
			if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, test.contains) {
				t.Fatalf("session = (%d, %q, %q), want error containing %q", code, stdout, stderr, test.contains)
			}
		})
	}
}

func TestExperimentalWorkspaceSessionStopsAtBoundAndClassifiesIOFailures(t *testing.T) {
	t.Parallel()
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	input := workspaceSessionRequestLine(1) + "not-json\n"
	code, stdout, stderr := runWithInputForTest(workspaceSessionArguments(1), root, strings.NewReader(input))
	if code != ExitSuccess || len(decodeWorkspaceSessionResponses(t, stdout)) != 1 || stderr != "" {
		t.Fatalf("bounded session = (%d, %q, %q)", code, stdout, stderr)
	}

	code, stdout, stderr = runWithInputForTest(workspaceSessionArguments(1), root, workspaceSessionErrorReader{})
	if code != ExitInternalFailure || stdout != "" || !strings.Contains(stderr, "input read failed") {
		t.Fatalf("input failure = (%d, %q, %q)", code, stdout, stderr)
	}

	var errorOutput bytes.Buffer
	code = RunWithInput([]string{"experimental", "check-workspace-session"}, root, strings.NewReader(""), &errorOutput, &errorOutput)
	if code != ExitInvalidInvocation || !strings.Contains(errorOutput.String(), "requires --target") {
		t.Fatalf("missing options = (%d, %q)", code, errorOutput.String())
	}

	var errorText bytes.Buffer
	code = RunWithInput(append([]string{"experimental", "check-workspace-session"}, workspaceSessionArguments(1)...), root, strings.NewReader(workspaceSessionRequestLine(1)), failingWriter{}, &errorText)
	if code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatalf("output failure = (%d, %q)", code, errorText.String())
	}
}

func writeWorkspaceSessionFixture(t *testing.T, rootSource string) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", workspaceSessionRootManifest)
	writeWorkspaceFile(t, root, "src/main.fy", rootSource)
	writeWorkspaceFile(t, root, "libs/answer/fairy.toml", workspaceSessionMemberManifest)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", workspaceSessionMemberSource)
	return root
}

func workspaceSessionArguments(maxRequests int) []string {
	return []string{
		"--target", "x86_64-windows-msvc",
		"--max-requests", strconv.Itoa(maxRequests),
		"--cache-max-entries", "8",
		"--cache-max-bytes", "33554432",
		"--cache-max-artifact-bytes", "8388608",
	}
}

func workspaceSessionRequestLine(sequence int) string {
	return fmt.Sprintf(`{"schemaVersion":%q,"kind":%q,"sequence":%d}`+"\n", workspaceSessionSchema, workspaceSessionRequestKind, sequence)
}

func replaceWorkspaceSessionArgument(arguments []string, flag, value string) []string {
	result := append([]string(nil), arguments...)
	for index := range result {
		if result[index] == flag && index+1 < len(result) {
			result[index+1] = value
			return result
		}
	}
	return result
}

func runWithInputForTest(arguments []string, root string, stdin io.Reader) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	args := append([]string{"experimental", "check-workspace-session"}, arguments...)
	code := RunWithInput(args, root, stdin, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func decodeWorkspaceSessionResponses(t *testing.T, output string) []workspaceSessionResponse {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	var result []workspaceSessionResponse
	for {
		var response workspaceSessionResponse
		if err := decoder.Decode(&response); errors.Is(err, io.EOF) {
			return result
		} else if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		result = append(result, response)
	}
}

type workspaceSessionReaderStage struct {
	before func()
	data   []byte
}

type workspaceSessionStagedReader struct {
	stages []workspaceSessionReaderStage
	index  int
	offset int
}

func (reader *workspaceSessionStagedReader) Read(destination []byte) (int, error) {
	if reader.index >= len(reader.stages) {
		return 0, io.EOF
	}
	stage := &reader.stages[reader.index]
	if reader.offset == 0 && stage.before != nil {
		stage.before()
	}
	written := copy(destination, stage.data[reader.offset:])
	reader.offset += written
	if reader.offset == len(stage.data) {
		reader.index++
		reader.offset = 0
	}
	return written, nil
}

type workspaceSessionErrorReader struct{}

func (workspaceSessionErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("synthetic input failure")
}
