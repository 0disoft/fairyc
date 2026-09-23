package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type resultIsOkCheckTiming struct {
	GoVersion    string   `json:"go_version"`
	GOOS         string   `json:"goos"`
	GOARCH       string   `json:"goarch"`
	GOMAXPROCS   int      `json:"gomaxprocs"`
	Warmups      int      `json:"warmups_per_lane"`
	Pairs        int      `json:"pairs"`
	SwitchSHA256 string   `json:"switch_source_sha256"`
	IsOkSHA256   string   `json:"is_ok_source_sha256"`
	Functions    int      `json:"mir_functions"`
	SwitchNS     []int64  `json:"switch_ns"`
	IsOkNS       []int64  `json:"is_ok_ns"`
	FirstLane    []string `json:"first_lane"`
	SwitchMedian float64  `json:"switch_median_ns"`
	IsOkMedian   float64  `json:"is_ok_median_ns"`
	MedianRatio  float64  `json:"ratio_of_medians"`
	PairedMedian float64  `json:"median_paired_ratio"`
}

var measureResultIsOkCheckTiming = flag.Bool("fairyc-measure-result-is-ok-check", false, "run the opt-in Result.is_ok check-timing diagnostic")

func TestResultBranchReviewIsOkCheckTiming(t *testing.T) {
	if !*measureResultIsOkCheckTiming {
		t.Skip("run the check-timing diagnostic with fairyc_measure_result_is_ok_check_timing")
	}
	const pairs = 8
	const switchTail = ").is_ok"
	const switchArms = ")) { case .ok(_): { true } case .err(_): { false } }"
	current, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "body_state.fy"))
	if err != nil {
		t.Fatal(err)
	}
	isOK := string(current)
	switched := isOK
	for _, name := range []string{"Node", "Statement"} {
		old := "!v::push<" + name + ">(" + map[string]string{"Node": "nodes, node", "Statement": "statements, value"}[name] + switchTail
		newText := "!switch (v::push<" + name + ">(" + map[string]string{"Node": "nodes, node", "Statement": "statements, value"}[name] + switchArms
		if strings.Count(switched, old) != 1 {
			t.Fatalf("expected one %s is_ok site", name)
		}
		switched = strings.Replace(switched, old, newText, 1)
	}
	if strings.Count(isOK, ".is_ok") != 2 || strings.Contains(switched, ".is_ok") {
		t.Fatal("unexpected body_state comparison scope")
	}
	variants := []struct {
		name, source string
	}{
		{"switch", switched}, {"is_ok", isOK},
	}
	roots := make([]string, len(variants))
	for i, variant := range variants {
		root := t.TempDir()
		copyLexerStreamExample(t, root)
		for _, name := range []string{"syntax.fy", "integrity.fy"} {
			data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
			if err != nil {
				t.Fatal(err)
			}
			writeTestSource(t, root, "src/"+name, string(data))
		}
		writeTestSource(t, root, "src/main.fy", parserCorpusProgram)
		writeTestSource(t, root, "src/body_state.fy", variant.source)
		roots[i] = root
	}
	result := resultIsOkCheckTiming{
		GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		GOMAXPROCS: runtime.GOMAXPROCS(0), Warmups: 1, Pairs: pairs,
		SwitchSHA256: sourceHash(switched), IsOkSHA256: sourceHash(isOK),
		SwitchNS: make([]int64, pairs), IsOkNS: make([]int64, pairs),
		FirstLane: make([]string, pairs),
	}
	check := func(lane int) int64 {
		t.Helper()
		// Load outside the timer so every check sees fresh source facts without
		// folding filesystem work into the no-cache compiler measurement.
		workspace, err := workspaceloader.Load(roots[lane])
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		checked, err := workspacecompiler.Check(workspace)
		elapsed := time.Since(start).Nanoseconds()
		if err != nil || !checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
			t.Fatalf("%s check failed: err=%v verified=%t diagnostics=%+v unsupported=%+v", variants[lane].name, err, checked.Verified, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
		functions := len(checked.MIR.Functions)
		if result.Functions == 0 {
			result.Functions = functions
		} else if result.Functions != functions {
			t.Fatalf("MIR functions changed: %d vs %d", result.Functions, functions)
		}
		return elapsed
	}
	check(0)
	check(1)
	for pair := 0; pair < pairs; pair++ {
		first := pair % 2
		result.FirstLane[pair] = variants[first].name
		for _, lane := range []int{first, 1 - first} {
			elapsed := check(lane)
			if lane == 0 {
				result.SwitchNS[pair] = elapsed
			} else {
				result.IsOkNS[pair] = elapsed
			}
		}
	}
	result.SwitchMedian = medianInt64(result.SwitchNS)
	result.IsOkMedian = medianInt64(result.IsOkNS)
	result.MedianRatio = result.IsOkMedian / result.SwitchMedian
	paired := make([]float64, pairs)
	for i := range paired {
		paired[i] = float64(result.IsOkNS[i]) / float64(result.SwitchNS[i])
	}
	sort.Float64s(paired)
	result.PairedMedian = (paired[pairs/2-1] + paired[pairs/2]) / 2
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("RESULT_IS_OK_CHECK_TIMING %s", encoded)
}

func sourceHash(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

func medianInt64(values []int64) float64 {
	copyValues := append([]int64(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	return float64(copyValues[len(copyValues)/2-1]+copyValues[len(copyValues)/2]) / 2
}
