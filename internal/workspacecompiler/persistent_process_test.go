package workspacecompiler

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	persistentProcessModeEnv      = "FAIRYC_TEST_PERSISTENT_SESSION_MODE"
	persistentProcessWorkspaceEnv = "FAIRYC_TEST_PERSISTENT_SESSION_WORKSPACE"
	persistentProcessCacheEnv     = "FAIRYC_TEST_PERSISTENT_SESSION_CACHE"
	persistentProcessReportEnv    = "FAIRYC_TEST_PERSISTENT_SESSION_REPORT"
)

type persistentProcessReport struct {
	Storage           string `json:"storage"`
	Outcome           string `json:"outcome"`
	Hits              int    `json:"hits"`
	Misses            int    `json:"misses"`
	Stored            int    `json:"stored"`
	Lowered           bool   `json:"lowered"`
	InterpreterResult int32  `json:"interpreterResult"`
}

func TestPersistentSessionReusesStableMIRAcrossProcesses(t *testing.T) {
	workspaceRoot := writeCacheWorkspace(t, testAnswer42)
	cacheRoot := t.TempDir()
	writer := runPersistentProcessHelper(t, workspaceRoot, cacheRoot, filepath.Join(t.TempDir(), "writer.json"))
	if writer.Storage != StorageDisk || writer.Outcome != OutcomeFallback || !writer.Lowered || writer.Misses != 2 || writer.Stored != 2 || writer.InterpreterResult != 42 {
		t.Fatalf("writer process report = %+v", writer)
	}
	reader := runPersistentProcessHelper(t, workspaceRoot, cacheRoot, filepath.Join(t.TempDir(), "reader.json"))
	if reader.Storage != StorageDisk || reader.Outcome != OutcomeAllHit || reader.Lowered || reader.Hits != 2 || reader.Misses != 0 || reader.InterpreterResult != 42 {
		t.Fatalf("reader process report = %+v", reader)
	}
}

func TestPersistentSessionProcessHelper(t *testing.T) {
	if os.Getenv(persistentProcessModeEnv) == "" {
		t.Skip("subprocess helper")
	}
	workspace, err := workspaceloader.Load(os.Getenv(persistentProcessWorkspaceEnv))
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(persistentTestConfiguration(os.Getenv(persistentProcessCacheEnv), "test-cross-process-compiler-build"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Check(workspace)
	if err != nil {
		t.Fatal(err)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	outcome, err := interpreter.Execute(result.Compiler.MIR, entry)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != interpreter.OutcomeReturn {
		t.Fatalf("persistent Session subprocess outcome = %+v", outcome)
	}
	report := persistentProcessReport{
		Storage: result.Cache.Storage, Outcome: result.Cache.Outcome, Hits: result.Cache.Hits,
		Misses: result.Cache.Misses, Stored: result.Cache.Stored, Lowered: result.Cache.Lowered,
		InterpreterResult: outcome.Value,
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv(persistentProcessReportEnv), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}

func runPersistentProcessHelper(t *testing.T, workspaceRoot, cacheRoot, reportPath string) persistentProcessReport {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestPersistentSessionProcessHelper$", "-test.v=false")
	command.Env = append(os.Environ(),
		persistentProcessModeEnv+"=run",
		persistentProcessWorkspaceEnv+"="+workspaceRoot,
		persistentProcessCacheEnv+"="+cacheRoot,
		persistentProcessReportEnv+"="+reportPath,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("persistent Session subprocess failed: %v\n%s", err, output)
	}
	encoded, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report persistentProcessReport
	if err := json.Unmarshal(encoded, &report); err != nil {
		t.Fatal(err)
	}
	return report
}
