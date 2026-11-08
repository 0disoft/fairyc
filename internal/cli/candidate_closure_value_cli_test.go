package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Original sources are shared with the independent Go CLI controls. The authored
// Fairy CLI binary receives only normal command arguments and the real workspace.
func TestCandidateClosureValueCLI(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		if !t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			root := closureValueCLIWorkspace(t)
			if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
				t.Fatal(err)
			}
			command := copiedClosureCommand{t: t, image: image, root: root}
			if out := command.cache(0, "cache: cold stored", "check"); !strings.HasPrefix(out, "check ok:") {
				t.Fatal(out)
			}
			if out := command.cache(0, "cache: hit", "check", "--locked"); !strings.HasPrefix(out, "check ok:") {
				t.Fatal(out)
			}
			lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
			if err != nil {
				t.Fatal(err)
			}
			for _, flags := range [][]string{nil, {"--release"}} {
				if out := command.cache(0, "cache: hit", append([]string{"build", "--locked"}, flags...)...); !strings.Contains(out, "built:") {
					t.Fatal(out)
				}
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				out, err := exec.CommandContext(ctx, filepath.Join(root, "build", "app.exe")).CombinedOutput()
				cancel()
				if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 42 || len(out) != 0 {
					t.Fatalf("built closure-value program: %v %s", err, out)
				}
				if out := command.cache(42, "cache: hit", append([]string{"run", "--locked"}, flags...)...); out != "" {
					t.Fatal(out)
				}
			}
			out, diag := command.run(0, "test", "--list", "--json", "--locked")
			if diag != "" || !strings.Contains(out, `"count":4`) || !strings.Contains(out, "z_values") {
				t.Fatalf("closure-value list: %s %s", out, diag)
			}
			for _, flags := range [][]string{nil, {"--native", "--capture-output"}, {"--native", "--capture-output", "--release"}} {
				args := append([]string{"test", "--json", "--locked"}, flags...)
				out, diag := command.run(ExitProgramTrap, args...)
				if diag != "" {
					t.Fatal(diag)
				}
				var report stableTestExecutionReport
				if err := json.Unmarshal([]byte(out), &report); err != nil {
					t.Fatal(err)
				}
				if report.Count != 4 || report.Passed != 1 || report.Failed != 3 || report.Status != "failed" || len(report.Tests) != 4 || report.Lock != "verified" {
					t.Fatalf("closure-value report: %+v", report)
				}
				native := len(flags) != 0
				wantEngine, wantSchema := "interpreter", stableTestReportSchema
				if native {
					wantEngine, wantSchema = "native", stableCapturedTestReportSchema
				}
				if report.Engine != wantEngine || report.SchemaVersion != wantSchema {
					t.Fatalf("closure-value engine/schema: %+v", report)
				}
				categories := []string{"division-by-zero", "division-by-zero", "integer-overflow", ""}
				identities := []string{"a_body_trap", "b_argument_first", "c_overflow", "z_values"}
				for i, result := range report.Tests {
					want := "failed"
					if categories[i] == "" {
						want = "passed"
					}
					if result.Identity != "tests::closure::"+identities[i] || result.Outcome != want || categories[i] == "" && result.TrapCategory != nil || categories[i] != "" && (result.TrapCategory == nil || *result.TrapCategory != categories[i]) {
						t.Fatalf("closure-value outcome %d: %+v", i, result)
					}
					if native && (result.StdoutBase64 == nil || *result.StdoutBase64 != "" || result.StderrBase64 == nil || *result.StderrBase64 != "") {
						t.Fatalf("closure-value trap streams: %+v", result)
					}
				}
				out, diag = command.run(0, append(args, "--exact", "tests::closure::z_values")...)
				if err := json.Unmarshal([]byte(out), &report); err != nil || diag != "" || report.Count != 1 || report.Passed != 1 || report.Failed != 0 || report.Status != "ok" || len(report.Tests) != 1 || report.Tests[0].Identity != "tests::closure::z_values" || report.Tests[0].Outcome != "passed" {
					t.Fatalf("successful exact test: %v %s %s", err, out, diag)
				}
			}
			current, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
			if err != nil || string(current) != string(lock) {
				t.Fatal("locked commands changed the source lock", err)
			}
			t.Logf("%d authored Fairy CLI processes and two built images: returned factories, different helpers, struct copies, snapshots and ordered traps", command.calls)
		}) {
			return
		}
	}
}
