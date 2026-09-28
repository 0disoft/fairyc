package compiler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

type linuxScannerNullTriad struct {
	Mode    string    `json:"mode"`
	PID     int       `json:"pid"`
	CPU     int       `json:"cpu"`
	Rounds  uint32    `json:"rounds"`
	Warmup  [3]uint64 `json:"warmup"`
	Work    [3]uint64 `json:"work"`
	Nanos   [3]uint64 `json:"nanos"`
	Session int       `json:"session,omitempty"`
	Index   int       `json:"index,omitempty"`
	Ratio   float64   `json:"ratio,omitempty"`
}

type linuxScannerNullSession struct {
	Triads  []linuxScannerNullTriad `json:"triads"`
	Median  float64                 `json:"median,omitempty"`
	Lower   float64                 `json:"lower,omitempty"`
	Upper   float64                 `json:"upper,omitempty"`
	Minimum float64                 `json:"minimum,omitempty"`
	Maximum float64                 `json:"maximum,omitempty"`
	Pass    bool                    `json:"pass"`
}

type linuxScannerNullReceipt struct {
	Schema       string                    `json:"schema"`
	Mode         string                    `json:"mode"`
	SourceSHA256 string                    `json:"source_sha256"`
	ELFSHA256    string                    `json:"elf_sha256,omitempty"`
	GCC          string                    `json:"gcc"`
	GCCArgv      []string                  `json:"gcc_argv"`
	Glibc        string                    `json:"glibc"`
	Kernel       string                    `json:"kernel"`
	Go           string                    `json:"go"`
	CPU          int                       `json:"cpu"`
	Smoke        *linuxScannerNullTriad    `json:"smoke,omitempty"`
	Sessions     []linuxScannerNullSession `json:"sessions,omitempty"`
	Verdict      string                    `json:"verdict"`
	Error        string                    `json:"error,omitempty"`
}

// Opt in explicitly. A Go PASS means the harness completed, not that the
// timing control passed its separate, predeclared equivalence interval.
func TestPrivateVectorSDKCandidateLinuxScannerNull(t *testing.T) {
	mode := os.Getenv("FAIRYC_SCANNER_LINUX_NULL_MODE")
	if mode == "" {
		t.Skip("opt-in Linux scanner timing null control")
	}
	if mode != "smoke" && mode != "full" {
		t.Fatalf("unknown Linux scanner null mode %q", mode)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux x86_64/glibc and GCC")
	}
	toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	core, err := os.ReadFile("../../tools/benchmarks/c-budget/scanner_reference.c")
	if err != nil {
		t.Fatal(err)
	}
	workload, err := os.ReadFile("../../tools/benchmarks/c-budget/scanner_workload_reference.c")
	if err != nil {
		t.Fatal(err)
	}
	source := "#define _GNU_SOURCE\n" + scannerCBudgetDefines(false) +
		"#define FC_SCANNER_LIBRARY\n#define FC_SCANNER_WORKLOAD_LIBRARY\n" +
		string(core) + "\n" + string(workload) + "\n" + linuxScannerNullMain
	root := filepath.Join("..", "..", ".cache")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	evidence, err := os.MkdirTemp(root, "linux-scanner-null-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Linux scanner null evidence: %s", evidence)
	sourcePath := filepath.Join(evidence, "source.c")
	program := filepath.Join(evidence, "program")
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-std=c11", "-pedantic-errors", "-Wall", "-Wextra", "-Werror", "-fno-common", "-O2", sourcePath, "-o", program}
	kernelOutput, err := exec.Command("uname", "-r").Output()
	if err != nil {
		t.Fatal(err)
	}
	receipt := linuxScannerNullReceipt{
		Schema: "fairyc.scanner-linux-null/v1", Mode: mode,
		SourceSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(source))),
		GCC:          toolchain.Version, GCCArgv: append([]string{toolchain.Path}, args...),
		Glibc: toolchain.Libc, Kernel: strings.TrimSpace(string(kernelOutput)),
		Go: runtime.Version(), CPU: -1, Verdict: "INCOMPLETE",
	}
	persist := func() error {
		data, marshalErr := json.MarshalIndent(receipt, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("encode Linux null receipt: %w", marshalErr)
		}
		temporary := filepath.Join(evidence, "receipt.json.tmp")
		if writeErr := os.WriteFile(temporary, append(data, '\n'), 0o600); writeErr != nil {
			return fmt.Errorf("write Linux null receipt: %w", writeErr)
		}
		if renameErr := os.Rename(temporary, filepath.Join(evidence, "receipt.json")); renameErr != nil {
			return fmt.Errorf("publish Linux null receipt: %w", renameErr)
		}
		return nil
	}
	defer func() {
		if err := persist(); err != nil {
			t.Error(err)
		}
	}()
	fail := func(format string, values ...any) {
		receipt.Error = fmt.Sprintf(format, values...)
		receipt.Verdict = "INVALID"
		t.Fatal(receipt.Error)
	}
	build := exec.Command(toolchain.Path, args...)
	build.Dir = "."
	var gccOut, gccErr bytes.Buffer
	build.Stdout, build.Stderr = &gccOut, &gccErr
	buildErr := build.Run()
	if err := os.WriteFile(filepath.Join(evidence, "gcc.stdout"), gccOut.Bytes(), 0o600); err != nil {
		fail("save GCC stdout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(evidence, "gcc.stderr"), gccErr.Bytes(), 0o600); err != nil {
		fail("save GCC stderr: %v", err)
	}
	if buildErr != nil || gccOut.Len() != 0 || gccErr.Len() != 0 {
		fail("GCC build failed or emitted diagnostics: %v stdout=%q stderr=%q", buildErr, gccOut.String(), gccErr.String())
	}
	elf, err := os.ReadFile(program)
	if err != nil || len(elf) < 4 || string(elf[:4]) != "\x7fELF" {
		fail("read linked ELF: %v", err)
	}
	receipt.ELFSHA256 = fmt.Sprintf("%x", sha256.Sum256(elf))
	probeBytes, err := linuxScannerNullRun(program, "probe")
	if err != nil {
		fail("CPU probe: %v", err)
	}
	var probe struct {
		CPU int `json:"cpu"`
	}
	if err := json.Unmarshal(probeBytes, &probe); err != nil || probe.CPU < 0 {
		fail("decode CPU probe %q: %v", probeBytes, err)
	}
	receipt.CPU = probe.CPU
	if mode == "smoke" {
		triad, err := linuxScannerNullMeasure(program, mode, probe.CPU)
		if err != nil {
			fail("smoke triad: %v", err)
		}
		receipt.Smoke = &triad
		receipt.Verdict = "SMOKE_ONLY"
		return
	}
	for session := 0; session < 2; session++ {
		receipt.Sessions = append(receipt.Sessions, linuxScannerNullSession{Triads: make([]linuxScannerNullTriad, 0, 24)})
		result := &receipt.Sessions[session]
		for index := 0; index < 24; index++ {
			triad, err := linuxScannerNullMeasure(program, mode, probe.CPU)
			if err != nil {
				fail("session %d triad %d: %v", session+1, index+1, err)
			}
			triad.Session, triad.Index = session+1, index+1
			result.Triads = append(result.Triads, triad)
			if err := persist(); err != nil {
				fail("preserve session %d triad %d: %v", session+1, index+1, err)
			}
		}
		ratios := make([]float64, 0, len(result.Triads))
		for _, triad := range result.Triads {
			ratios = append(ratios, triad.Ratio)
		}
		sort.Float64s(ratios)
		result.Minimum, result.Maximum = ratios[0], ratios[23]
		result.Median = (ratios[11] + ratios[12]) / 2
		result.Lower, result.Upper = ratios[6], ratios[17]
		result.Pass = result.Lower >= 0.98 && result.Upper <= 1.02
	}
	receipt.Verdict = "FAIL"
	if receipt.Sessions[0].Pass && receipt.Sessions[1].Pass {
		receipt.Verdict = "PASS"
	}
	t.Logf("Linux scanner null control: %s (session bounds %.5f..%.5f and %.5f..%.5f)", receipt.Verdict,
		receipt.Sessions[0].Lower, receipt.Sessions[0].Upper,
		receipt.Sessions[1].Lower, receipt.Sessions[1].Upper)
}

func linuxScannerNullRun(program string, arguments ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, program, arguments...)
	command.Stdin = nil
	var output, stderr bytes.Buffer
	command.Stdout, command.Stderr = &output, &stderr
	err := command.Run()
	if err != nil {
		return nil, fmt.Errorf("run %v: %w, stderr=%q", arguments, err, stderr.String())
	}
	if stderr.Len() != 0 {
		return nil, fmt.Errorf("null probe emitted stderr %q", stderr.String())
	}
	if output.Len() > 4096 {
		return nil, fmt.Errorf("null probe output exceeded 4096 bytes")
	}
	return output.Bytes(), nil
}

func linuxScannerNullMeasure(program, mode string, cpu int) (linuxScannerNullTriad, error) {
	output, err := linuxScannerNullRun(program, mode, fmt.Sprint(cpu))
	if err != nil {
		return linuxScannerNullTriad{}, err
	}
	var result linuxScannerNullTriad
	if err := json.Unmarshal(output, &result); err != nil {
		return result, fmt.Errorf("decode null triad %q: %w", output, err)
	}
	expectedRounds := uint32(250000)
	if mode == "smoke" {
		expectedRounds = 1000
	}
	expectedWork := uint64(expectedRounds) * 36
	if result.Mode != mode || result.PID <= 0 || result.CPU != cpu || result.Rounds != expectedRounds {
		return result, fmt.Errorf("triad identity drift: %+v", result)
	}
	for index := range result.Nanos {
		if result.Warmup[index] != expectedWork || result.Work[index] != expectedWork || result.Nanos[index] == 0 {
			return result, fmt.Errorf("invalid null triad call %d: %+v", index, result)
		}
	}
	result.Ratio = float64(result.Nanos[1]) / math.Sqrt(float64(result.Nanos[0])*float64(result.Nanos[2]))
	if math.IsNaN(result.Ratio) || math.IsInf(result.Ratio, 0) {
		return result, fmt.Errorf("non-finite null ratio")
	}
	return result, nil
}

const linuxScannerNullMain = `
#include <errno.h>
#include <inttypes.h>
#include <sched.h>
#include <time.h>
#include <unistd.h>

static uint64_t fc_null_work(uint32_t rounds) {
    static const unsigned char sample[] = "let n=12345; /*open";
    uint64_t total = 0;
    for (uint32_t index = 0; index < rounds; ++index) {
        uint64_t work = 0;
        if (fc_workload_round(sample, sizeof(sample) - 1u, 0, &work) != 0) return 0;
        total += work;
    }
    return total;
}

static int fc_null_cpu(void) {
    cpu_set_t allowed;
    if (sched_getaffinity(0, sizeof(allowed), &allowed) != 0) return -1;
    for (int cpu = 0; cpu < CPU_SETSIZE; ++cpu)
        if (CPU_ISSET(cpu, &allowed)) return cpu;
    return -1;
}

int main(int argc, char **argv) {
    int allowed = fc_null_cpu();
    if (allowed < 0) return 2;
    if (argc == 2 && strcmp(argv[1], "probe") == 0) {
        printf("{\"cpu\":%d}\n", allowed);
        return 0;
    }
    if (argc != 3) return 2;
    int smoke = strcmp(argv[1], "smoke") == 0;
    if (!smoke && strcmp(argv[1], "full") != 0) return 2;
    char *end = NULL;
    errno = 0;
    long requested = strtol(argv[2], &end, 10);
    if (errno != 0 || end == argv[2] || *end != '\0' || requested != allowed) return 2;
    cpu_set_t only;
    CPU_ZERO(&only);
    CPU_SET(allowed, &only);
    if (sched_setaffinity(0, sizeof(only), &only) != 0 || sched_getcpu() != allowed) return 2;
    uint32_t rounds = smoke ? 1000u : 250000u;
    uint64_t expected = (uint64_t)rounds * 36u;
    uint64_t warmup[3], work[3], nanos[3];
    for (int index = 0; index < 3; ++index) {
        warmup[index] = fc_null_work(rounds);
        if (warmup[index] != expected || sched_getcpu() != allowed) return 3;
    }
    for (int index = 0; index < 3; ++index) {
        struct timespec before, after;
        if (sched_getcpu() != allowed || clock_gettime(CLOCK_MONOTONIC_RAW, &before) != 0) return 4;
        work[index] = fc_null_work(rounds);
        if (clock_gettime(CLOCK_MONOTONIC_RAW, &after) != 0 || sched_getcpu() != allowed) return 4;
        int64_t elapsed = (int64_t)(after.tv_sec - before.tv_sec) * 1000000000LL +
                          (int64_t)(after.tv_nsec - before.tv_nsec);
        if (elapsed <= 0 || work[index] != expected) return 5;
        nanos[index] = (uint64_t)elapsed;
    }
    if (printf("{\"mode\":\"%s\",\"pid\":%ld,\"cpu\":%d,\"rounds\":%" PRIu32
           ",\"warmup\":[%" PRIu64 ",%" PRIu64 ",%" PRIu64
           "],\"work\":[%" PRIu64 ",%" PRIu64 ",%" PRIu64
           "],\"nanos\":[%" PRIu64 ",%" PRIu64 ",%" PRIu64 "]}\n",
           argv[1], (long)getpid(), allowed, rounds,
           warmup[0], warmup[1], warmup[2],
           work[0], work[1], work[2],
           nanos[0], nanos[1], nanos[2]) < 0 || fflush(stdout) != 0) return 6;
    return 0;
}
`
