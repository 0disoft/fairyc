package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

type candidateScannerCorpusProfile struct {
	Profile         string `json:"profile"`
	Passed          bool   `json:"passed"`
	Batches         int    `json:"batches"`
	MalformedFrames int    `json:"malformedFrames"`
}

type candidateScannerCorpusBuild struct {
	Profile         string `json:"profile"`
	CompilerPath    string `json:"compilerPath"`
	CompilerVersion string `json:"compilerVersion"`
	CSourceSHA256   string `json:"cSourceSHA256"`
	RuntimeSHA256   string `json:"runtimeSHA256"`
	ImageSHA256     string `json:"imageSHA256"`
}

type candidateScannerCorpusReport struct {
	Schema         string                          `json:"schema"`
	Passed         bool                            `json:"passed"`
	FixtureFiles   int                             `json:"fixtureFiles"`
	ProjectFiles   int                             `json:"projectFiles"`
	FramedCases    int                             `json:"framedCases"`
	InputsSHA256   string                          `json:"inputsSHA256"`
	ExpectedSHA256 string                          `json:"expectedSHA256"`
	Builds         []candidateScannerCorpusBuild   `json:"builds"`
	Profiles       []candidateScannerCorpusProfile `json:"profiles"`
}

func candidateScannerDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Opt-in native harness: Go supplies inputs and expectations, while only the
// Fairy candidate's emitted C performs recognition in the tested process.
func TestWindowsCandidateEmittedLexerCorpus(t *testing.T) {
	unit := os.Getenv("FAIRYC_CANDIDATE_SCANNER_C")
	if unit == "" {
		t.Skip("requires a candidate-emitted scanner C unit")
	}
	receipt := os.Getenv("FAIRYC_CANDIDATE_SCANNER_RECEIPT")
	if !filepath.IsAbs(unit) || !filepath.IsAbs(receipt) {
		t.Fatal("candidate C and receipt paths must be absolute")
	}
	report := candidateScannerCorpusReport{Schema: "fairyc.candidate-scanner-corpus-native/v1"}
	defer func() {
		if t.Failed() {
			report.Passed = false
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.MkdirAll(filepath.Dir(receipt), 0o700)
		}
		if err == nil {
			err = os.WriteFile(receipt, append(data, '\n'), 0o600)
		}
		if err != nil {
			t.Errorf("write corpus receipt: %v", err)
		}
	}()
	runWindowsFairyLexerCorpus(t, unit, &report)
}

func buildCandidateScannerCorpus(t *testing.T, path string, profile cdriver.Profile, report *candidateScannerCorpusReport) cdriver.WindowsHostArtifact {
	t.Helper()
	unit, err := os.ReadFile(path)
	if err != nil || len(unit) == 0 || len(unit) > 2<<20 {
		t.Fatalf("candidate C unit: size=%d err=%v", len(unit), err)
	}
	runtimeSource, err := cruntime.WindowsRunSource(120)
	if err != nil {
		t.Fatal(err)
	}
	sdkSource, err := cstdlib.WindowsMSVCSourceForSymbols([]string{"fy_std_buffer_initialize_v0", "fy_std_stdout_write_usize_v0", "fy_std_stdin_read_v0"})
	if err != nil {
		t.Fatal(err)
	}
	vectorSource, err := os.ReadFile(filepath.Join("..", "cstdlib", "stage0_vector_storage.c"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource += "\n" + sdkSource + "\n" + string(vectorSource)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := cdriver.BuildWindowsHost(toolchain, profile, string(unit), runtimeSource)
	if err != nil {
		t.Fatal(err)
	}
	artifact.TrapControl = true
	report.Builds = append(report.Builds, candidateScannerCorpusBuild{Profile: profile.Name, CompilerPath: toolchain.Path,
		CompilerVersion: toolchain.Version, CSourceSHA256: candidateScannerDigest(unit), RuntimeSHA256: candidateScannerDigest([]byte(runtimeSource)), ImageSHA256: candidateScannerDigest(artifact.Bytes)})
	return artifact
}

func candidateScannerCorpusIdentity(report *candidateScannerCorpusReport, input []byte, expected string) {
	report.InputsSHA256 = candidateScannerDigest(input)
	report.ExpectedSHA256 = candidateScannerDigest([]byte(expected))
}
