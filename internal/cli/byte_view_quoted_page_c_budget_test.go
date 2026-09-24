package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

// This opt-in process-wall comparison reuses each built image. Its time includes
// process startup, input parsing, scanning, console output and collection.
func TestWindowsQuotedPageCBudget(t *testing.T) {
	if os.Getenv("FAIRYC_QUOTED_PAGE_BUDGET") != "1" {
		t.Skip("opt-in quoted-page C comparison")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	pages, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "pages.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/pages.fy", string(pages))
	writeTestSource(t, root, "src/main.fy", `module main;
import pages as pages;
@effects(console) fn i32 main() { return pages::run(); }`)
	reference, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "quoted_page.c"))
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics strings.Builder
	buildStarted := time.Now()
	fairy, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
	fairyBuildNS := time.Since(buildStarted).Nanoseconds()
	if code != ExitSuccess {
		t.Fatalf("Fairy compile: %d %s", code, diagnostics.String())
	}
	buildStarted = time.Now()
	c, err := cdriver.BuildWindowsHost(toolchain, cdriver.OptimizedProfile, string(reference), "int fairyc_quoted_page_support(void) { return 0; }\n")
	cBuildNS := time.Since(buildStarted).Nanoseconds()
	if err != nil {
		t.Fatal(err)
	}
	artifacts := []struct {
		name, path string
		bytes      []byte
	}{
		{"c", filepath.Join(t.TempDir(), "c.exe"), c.Bytes},
		{"fairy", filepath.Join(t.TempDir(), "fairy.exe"), fairy.artifact.Bytes},
	}
	for _, artifact := range artifacts {
		if err := os.WriteFile(artifact.path, artifact.bytes, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	input := []byte(`"` + strings.Repeat("a", 10000) + `"`)
	frame := streamPageFrame(input, false, 0)
	var expected strings.Builder
	for _, value := range streamObservation(t, input, false)[:4096] {
		fmt.Fprintf(&expected, "%d ", value)
	}
	type sample struct {
		Pair        int   `json:"pair"`
		CWallNS     int64 `json:"c_wall_ns"`
		FairyWallNS int64 `json:"fairy_wall_ns"`
		CCPUNS      int64 `json:"c_cpu_ns"`
		FairyCPUNS  int64 `json:"fairy_cpu_ns"`
	}
	samples := make([]sample, 0, executionPairs)
	cWall := make([]float64, 0, executionPairs)
	fairyWall := make([]float64, 0, executionPairs)
	const warmupPairs = 3
	for pair := -warmupPairs; pair < executionPairs; pair++ {
		order := [2]int{0, 1}
		if pair%2 == 0 {
			order = [2]int{1, 0}
		}
		row := sample{Pair: pair}
		for _, laneIndex := range order {
			lane := artifacts[laneIndex]
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			command := exec.CommandContext(ctx, lane.path)
			command.Stdin = bytes.NewReader(frame)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			started := time.Now()
			err := command.Run()
			wallNS := time.Since(started).Nanoseconds()
			cancel()
			if err != nil || stdout.String() != expected.String() || stderr.Len() != 0 {
				t.Fatalf("pair %d %s: err=%v output_bytes=%d expected_bytes=%d stderr=%q", pair, lane.name, err, stdout.Len(), expected.Len(), stderr.String())
			}
			cpuNS := (command.ProcessState.UserTime() + command.ProcessState.SystemTime()).Nanoseconds()
			if lane.name == "c" {
				row.CWallNS, row.CCPUNS = wallNS, cpuNS
			} else {
				row.FairyWallNS, row.FairyCPUNS = wallNS, cpuNS
			}
		}
		if pair >= 0 {
			samples = append(samples, row)
			cWall = append(cWall, float64(row.CWallNS))
			fairyWall = append(fairyWall, float64(row.FairyWallNS))
		}
	}
	report := struct {
		Schema         string                 `json:"schema"`
		Profile        string                 `json:"profile"`
		MSVC           string                 `json:"msvc"`
		InputBytes     int                    `json:"input_bytes"`
		OutputBytes    int                    `json:"output_bytes"`
		WarmupPairs    int                    `json:"warmup_pairs"`
		MeasuredPairs  int                    `json:"measured_pairs"`
		CSourceSHA     string                 `json:"c_source_sha256"`
		PagesSourceSHA string                 `json:"pages_source_sha256"`
		InputSHA       string                 `json:"input_sha256"`
		CEXESHA        string                 `json:"c_exe_sha256"`
		FairyEXESHA    string                 `json:"fairy_exe_sha256"`
		CEXEBytes      int                    `json:"c_exe_bytes"`
		FairyEXEBytes  int                    `json:"fairy_exe_bytes"`
		CBuildNS       int64                  `json:"c_single_build_ns"`
		FairyBuildNS   int64                  `json:"fairy_single_build_ns"`
		Samples        []sample               `json:"samples"`
		Summary        executionBudgetSummary `json:"summary"`
	}{
		Schema: "fairyc.quoted-page-c-budget/v1", Profile: cdriver.OptimizedProfile.Name,
		MSVC: toolchain.Version, InputBytes: len(frame), OutputBytes: expected.Len(),
		WarmupPairs: warmupPairs, MeasuredPairs: executionPairs,
		CSourceSHA: fmt.Sprintf("%x", sha256.Sum256(reference)), PagesSourceSHA: fmt.Sprintf("%x", sha256.Sum256(pages)),
		InputSHA: fmt.Sprintf("%x", sha256.Sum256(frame)), CEXESHA: fmt.Sprintf("%x", sha256.Sum256(c.Bytes)),
		FairyEXESHA: fmt.Sprintf("%x", sha256.Sum256(fairy.artifact.Bytes)),
		CEXEBytes:   len(c.Bytes), FairyEXEBytes: len(fairy.artifact.Bytes), CBuildNS: cBuildNS, FairyBuildNS: fairyBuildNS,
		Samples: samples, Summary: summarizeExecutionBudget(cWall, fairyWall),
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("QUOTED_PAGE_C_BUDGET %s", encoded)
}
