package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// Compare only the prepare-push reserve path; both images use one verified
// scanner MIR program, SDK, workload, C toolchain and optimization profile.
func TestPrivateVectorPreparePushAB(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_PREPARE_PUSH_AB") != "1" {
		t.Skip("opt-in native prepare-push A/B measurement")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("requires Windows AMD64 MSVC")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[4].name != "all vector for" {
		t.Fatal("real scanner vector source changed")
	}
	program, entry, specializations := scannerBudgetFairyProgram(t, variants[4])
	storage, err := cstdlib.WindowsMSVCVectorStorageSource()
	if err != nil {
		t.Fatal(err)
	}
	storage = strings.ReplaceAll(storage, "\r\n", "\n")
	const current = `    /* A validated spare slot needs no second descriptor check or allocation. */
    if (value.length == value.block.size / value.element_size) {
        uint32_t status = fy_private_vector_reserve_v0(storage, value.length + 1);
        if (status != FY_STD_ALLOC_OK_V0) return status;
        memcpy(&value, storage, sizeof value);
    }
`
	const previous = `    uint32_t status = fy_private_vector_reserve_v0(storage, value.length + 1);
    if (status != FY_STD_ALLOC_OK_V0) return status;
    memcpy(&value, storage, sizeof value);
`
	if strings.Count(storage, current) != 1 {
		t.Fatal("prepare-push source changed; refresh the exact A/B baseline")
	}
	baseline := strings.Replace(storage, current, previous, 1)
	if strings.Count(baseline, previous) != 1 {
		t.Fatal("prepare-push baseline replacement was not unique")
	}
	support, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "support.c"))
	if err != nil {
		t.Fatal(err)
	}
	extra := "\n" + string(support) + "\nvoid fc_budget_flush(void) { fflush(stdout); }\n"
	type image struct {
		name     string
		storage  string
		artifact cdriver.WindowsHostArtifact
	}
	images := []image{{name: "prior reserve", storage: baseline}, {name: "spare-slot fast path", storage: storage}}
	var generatedC string
	for i := range images {
		artifact, unit := buildRealScannerVectorNativeWithStorage(t, program, entry,
			cdriver.OptimizedProfile, extra, images[i].storage)
		images[i].artifact = artifact
		if generatedC == "" {
			generatedC = unit.Source
		} else if unit.Source != generatedC {
			t.Fatal("A/B images generated different scanner C units")
		}
	}
	const sample = "let n=12345; /*open"
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(sample)),
		source.NewBudget(source.DefaultLimits(), len(sample)))
	perRound := len(sample) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	const seed = 12345
	const rounds = 250000
	const measuredPairs = 12
	var reporter string
	samples := make([]map[string]any, 0, measuredPairs+1)
	for pair := 0; pair <= measuredPairs; pair++ {
		order := [2]string{}
		lanes := make([]map[string]any, 0, len(images))
		for position := range images {
			image := images[(pair+position)%len(images)]
			order[position] = image.name
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image.artifact,
				strings.NewReader(fmt.Sprintf("%d %d\n", seed, rounds)))
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("pair %d %s execution: %+v %v", pair, image.name, outcome, err)
			}
			fields := strings.Fields(outcome.Stdout)
			if len(fields) < 5 {
				t.Fatalf("pair %d %s missing timing fields: %q", pair, image.name, outcome.Stdout)
			}
			prefix := strings.Join(fields[:len(fields)-4], " ")
			if reporter == "" {
				reporter = prefix
			} else if prefix != reporter {
				t.Fatalf("pair %d %s reporter differs", pair, image.name)
			}
			last := fields[len(fields)-4:]
			completed, completedErr := strconv.ParseUint(last[0], 10, 32)
			ticks, ticksErr := strconv.ParseUint(last[1], 10, 64)
			frequency, frequencyErr := strconv.ParseUint(last[2], 10, 64)
			total, totalErr := strconv.ParseUint(last[3], 10, 64)
			if completedErr != nil || ticksErr != nil || frequencyErr != nil || totalErr != nil ||
				completed != rounds || ticks == 0 || frequency == 0 || total != uint64(perRound*rounds) {
				t.Fatalf("pair %d %s timing fields=%v errors=%v,%v,%v,%v", pair, image.name,
					last, completedErr, ticksErr, frequencyErr, totalErr)
			}
			lanes = append(lanes, map[string]any{
				"lane": image.name, "qpc_ticks": ticks, "qpc_frequency": frequency,
				"completed": completed, "total": total,
			})
		}
		samples = append(samples, map[string]any{"pair": pair, "warmup": pair == 0,
			"order": order, "lanes": lanes})
	}
	imageRows := make([]map[string]any, 0, len(images))
	for _, image := range images {
		objects := map[string]int{}
		for _, object := range image.artifact.Objects {
			objects[object.Name] = len(object.Bytes)
		}
		imageRows = append(imageRows, map[string]any{
			"lane": image.name, "storage_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(image.storage))),
			"exe_sha256": fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
			"exe_bytes":  len(image.artifact.Bytes), "objects": objects,
		})
	}
	report, err := json.Marshal(map[string]any{
		"schema": "fairyc.vector-prepare-push-ab/v1", "go": runtime.Version(),
		"msvc": images[0].artifact.Toolchain.Version, "gomaxprocs": runtime.GOMAXPROCS(0),
		"profile": cdriver.OptimizedProfile.Name, "sample_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(sample))),
		"generated_c_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(generatedC))),
		"specializations":    specializations, "seed": seed, "rounds": rounds,
		"per_round_result": perRound, "warmup_pairs": 1, "measured_pairs": measuredPairs,
		"images": imageRows, "samples": samples,
		"scope": "Exact prior/current private prepare-push C source with shared verified scanner MIR; one warmup and twelve alternating measured pairs. QPC excludes startup and final output. Raw samples only; no automatic 8% verdict.",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_PREPARE_PUSH_AB " + string(report))
}
