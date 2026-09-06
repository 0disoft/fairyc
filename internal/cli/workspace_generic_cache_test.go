package cli

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

const cachedGenericMain = "module main; import answer as answer; fn i32 main() { return answer::value<i32>(42); }"
const cachedGenericAnswer = "module answer; pub fn T value<T>(T input) { return input; }"

func TestWorkspaceSessionSwitchesSourceFamiliesAndRecovers(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	reader := &workspaceSessionStagedReader{stages: []workspaceSessionReaderStage{
		{data: []byte(workspaceSessionRequestLine(1))},
		{data: []byte(workspaceSessionRequestLine(2))},
		{before: func() {
			writeWorkspaceFile(t, root, "src/main.fy", cachedGenericMain)
			writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", cachedGenericAnswer)
		}, data: []byte(workspaceSessionRequestLine(3))},
		{data: []byte(workspaceSessionRequestLine(4))},
		{before: func() {
			writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(cachedGenericMain, "<i32>", "<Unknown>", 1))
		}, data: []byte(workspaceSessionRequestLine(5))},
		{before: func() { writeWorkspaceFile(t, root, "src/main.fy", cachedGenericMain) }, data: []byte(workspaceSessionRequestLine(6))},
		{before: func() {
			writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootSource)
			writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", workspaceSessionMemberSource)
		}, data: []byte(workspaceSessionRequestLine(7))},
		{data: []byte(workspaceSessionRequestLine(8))},
	}}
	code, stdout, stderr := runWithInputForTest(workspaceSessionArguments(8), root, reader)
	if code != ExitSuccess || !strings.Contains(stderr, "E4502") {
		t.Fatalf("session: (%d, %q, %q)", code, stdout, stderr)
	}
	responses := decodeWorkspaceSessionResponses(t, stdout)
	if len(responses) != 8 {
		t.Fatalf("responses: %d", len(responses))
	}
	for _, i := range []int{0, 2, 6} {
		if r := responses[i]; r.Status != workspaceSessionStatusOK || r.Cache.Outcome != workspacecompiler.OutcomeFallback || r.Cache.Hits != 0 || !r.Cache.Lowered || r.Cache.RetainedEntries != 2 {
			t.Fatalf("cold %d: %+v", i, r)
		}
	}
	for _, i := range []int{1, 3, 5, 7} {
		if r := responses[i]; r.Status != workspaceSessionStatusOK || r.Cache.Outcome != workspacecompiler.OutcomeAllHit || r.Cache.Hits != 2 || r.Cache.Lowered {
			t.Fatalf("warm %d: %+v", i, r)
		}
	}
	if r := responses[4]; r.Status != workspaceSessionStatusDiagnostics || r.ExitCode != ExitSourceDiagnostics || r.Cache.Hits != 0 || r.Cache.Outcome != workspacecompiler.OutcomeNotEligible {
		t.Fatalf("invalid source: %+v", r)
	}
}

func TestWorkspaceGenericPersistentCacheCommand(t *testing.T) {
	root := writeGenericCLIWorkspace(t)
	args := append([]string{"experimental", "check-workspace-cache"}, workspaceCacheArguments()...)
	for i := 0; i < 2; i++ {
		code, stdout, stderr := runForTest(args, root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("invocation %d: (%d, %q, %q)", i, code, stdout, stderr)
		}
		report := decodeWorkspaceCacheReport(t, stdout)
		if i == 0 && (report.Cache.Stored != 2 || !report.Cache.Lowered) || i == 1 && (report.Cache.Outcome != workspacecompiler.OutcomeAllHit || report.Cache.Hits != 2 || report.Cache.Lowered) {
			t.Fatalf("invocation %d: %+v", i, report.Cache)
		}
	}
	writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(cachedGenericMain, "<i32>", "<Unknown>", 1))
	code, stdout, stderr := runForTest(args, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E4502") {
		t.Fatalf("invalid cached source: (%d, %q, %q)", code, stdout, stderr)
	}
}
