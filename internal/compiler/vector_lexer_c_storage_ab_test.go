package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// The same independent C scanner is compiled with two storage strategies.
// This isolates storage cost; it is not a Fairy/C budget verdict.
func TestPrivateVectorSDKCandidateIndependentCScannerStorageAB(t *testing.T) {
	mode := os.Getenv("FAIRYC_C_SCANNER_STORAGE_AB")
	if mode == "" {
		t.Skip("opt-in paired C scanner storage comparison")
	}
	if mode != "smoke" && mode != "measure" {
		t.Fatalf("unknown C scanner storage mode %q", mode)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" || runtime.GOMAXPROCS(0) != 1 {
		t.Fatal("requires Windows AMD64 with GOMAXPROCS=1")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const sample = "let n=12345; /*open"
	const seed = 12345
	data := []byte(sample)
	lexed := lexer.Lex(source.NewFile("input.fy", data), source.NewBudget(source.DefaultLimits(), len(data)))
	perRound := len(data) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	if len(lexed.Tokens) < 9 || len(lexed.Diagnostics) == 0 {
		t.Fatal("storage sample no longer crosses the first token capacity")
	}
	wantPrefix := scannerGoObservation(data)
	type image struct {
		name     string
		artifact cdriver.WindowsHostArtifact
		source   string
	}
	images := make([]image, 0, 2)
	for _, item := range []struct {
		name      string
		geometric bool
	}{{"preallocated", false}, {"geometric", true}} {
		code := scannerCBenchmarkSourceWithStorage(t, item.geometric)
		artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, cdriver.OptimizedProfile, code)
		if err != nil {
			t.Fatalf("build %s C: %v", item.name, err)
		}
		images = append(images, image{name: item.name, artifact: artifact, source: code})
	}
	if images[0].source == images[1].source {
		t.Fatal("C storage modes produced identical source")
	}
	rounds, warmup, measured := 1000, 0, 1
	if mode == "measure" {
		rounds, warmup, measured = 250000, 1, 12
	}
	samples := make([]map[string]any, 0, warmup+measured)
	for pair := 0; pair < warmup+measured; pair++ {
		order := make([]string, 0, len(images))
		lanes := make([]map[string]any, 0, len(images))
		for position := range images {
			item := images[(pair+position)%len(images)]
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(item.artifact,
				strings.NewReader(fmt.Sprintf("%d %d\n", seed, rounds)))
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("pair %d %s execution: %+v %v", pair, item.name, outcome, err)
			}
			if !strings.HasPrefix(outcome.Stdout, wantPrefix) {
				t.Fatalf("pair %d %s reporter differs", pair, item.name)
			}
			fields := strings.Fields(strings.TrimPrefix(outcome.Stdout, wantPrefix))
			if len(fields) != 4 {
				t.Fatalf("pair %d %s timing fields=%v", pair, item.name, fields)
			}
			completed, completedErr := strconv.ParseUint(fields[0], 10, 32)
			ticks, ticksErr := strconv.ParseUint(fields[1], 10, 64)
			frequency, frequencyErr := strconv.ParseUint(fields[2], 10, 64)
			total, totalErr := strconv.ParseUint(fields[3], 10, 64)
			if completedErr != nil || ticksErr != nil || frequencyErr != nil || totalErr != nil ||
				completed != uint64(rounds) || ticks == 0 || frequency == 0 ||
				total != uint64(perRound*rounds) {
				t.Fatalf("pair %d %s timing=%v errors=%v,%v,%v,%v", pair, item.name,
					fields, completedErr, ticksErr, frequencyErr, totalErr)
			}
			order = append(order, item.name)
			lanes = append(lanes, map[string]any{
				"lane": item.name, "qpc_ticks": ticks, "qpc_frequency": frequency,
				"completed": completed, "total": total,
			})
		}
		samples = append(samples, map[string]any{
			"pair": pair, "warmup": pair < warmup, "order": order, "lanes": lanes,
		})
	}
	imageRows := make([]map[string]any, 0, len(images))
	for _, item := range images {
		imageRows = append(imageRows, map[string]any{
			"lane": item.name, "source_bytes": len(item.source),
			"source_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(item.source))),
			"exe_bytes":     len(item.artifact.Bytes),
			"exe_sha256":    fmt.Sprintf("%x", sha256.Sum256(item.artifact.Bytes)),
		})
	}
	report, err := json.Marshal(map[string]any{
		"schema": "fairyc.c-scanner-storage-ab/v1", "mode": mode,
		"sample_sha256": fmt.Sprintf("%x", sha256.Sum256(data)),
		"go":            runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH,
		"msvc": toolchain.Version, "gomaxprocs": runtime.GOMAXPROCS(0),
		"profile": cdriver.OptimizedProfile.Name, "seed": seed,
		"rounds": rounds, "warmup_pairs": warmup, "measured_pairs": measured,
		"per_round_result": perRound, "images": imageRows, "samples": samples,
		"stability_rule": "(max-min)/median <= 0.10 for both lane times and paired geometric/preallocated ratios",
		"scope":          "Independent C scanner storage A/B only; QPC excludes process startup and final output. No retries or exclusions. No Fairy/C 8% verdict.",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("C_SCANNER_STORAGE_AB " + string(report))
}
