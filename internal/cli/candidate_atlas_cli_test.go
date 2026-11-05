package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Keep the authored main and dispatcher in the real CLI image. No companion is
// built or installed: Atlas has to resolve, check and traverse entirely in Fairy.
func TestCandidateAtlasCLI(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows full CLI; Linux remains batched")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for full CLI replay")
	}
	unsupportedRoots := candidateAtlasUnsupportedFixtures(t)
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	build, support := candidateParityBuildPartsWithLimit(t, root, map[string]bool{"CloseHandle": true, "CreateFileW": true, "GetFileInformationByHandleEx": true, "GetFinalPathNameByHandleW": true, "GetLastError": true}, 2<<20)
	largest := 0
	for _, source := range build.Sources {
		if len(source) > largest {
			largest = len(source)
		}
	}
	t.Logf("C input bytes: header=%d runtime=%d units=%d largest=%d", len(build.Header), len(support), len(build.Sources), largest)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			imageDir := root
			if retained := os.Getenv("FAIRYC_ATLAS_CLI_ARTIFACTS"); retained != "" {
				imageDir = filepath.Join(retained, profile.Name)
				if err := os.MkdirAll(imageDir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			image := filepath.Join(imageDir, "atlas-cli-"+profile.Name+".exe")
			digest := sha256.New()
			policy, _ := json.Marshal(profile)
			for _, part := range append([]string{string(policy), tc.Name, tc.Path, tc.Version, "stack-reserve=16777216", build.Header, support}, build.Sources...) {
				digest.Write([]byte(part))
				digest.Write([]byte{0})
			}
			sourceID := hex.EncodeToString(digest.Sum(nil))
			type receipt struct{ Source, Image string }
			var saved receipt
			record := filepath.Join(imageDir, "atlas-image.json")
			data, _ := os.ReadFile(record)
			binary, _ := os.ReadFile(image)
			hash := sha256.Sum256(binary)
			if json.Unmarshal(data, &saved) != nil || len(binary) == 0 || saved.Source != sourceID || saved.Image != hex.EncodeToString(hash[:]) {
				artifact, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(image, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				hash = sha256.Sum256(artifact.Bytes)
				data, err = json.Marshal(receipt{sourceID, hex.EncodeToString(hash[:])})
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(record, data, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				t.Log("reuse verified generated-C/profile/toolchain/image digest")
			}

			t.Setenv("PATH", "")
			verifyCandidateAtlasCommand(t, image)
			verifyCandidateAtlasFamilies(t, image)
			verifyCandidateMCPCommand(t, image)
			verifyCandidateMCPAggregateCommand(t, image)
			verifyCandidateEditCommand(t, image)
			verifyCandidateEditCLIApply(t, image)
			verifyCandidateEditAggregateCommand(t, image)
			verifyCandidateAtlasUnsupported(t, image, unsupportedRoots)
			verifyCandidateAtlasGenericCommand(t, image)
			verifyCandidateAtlasAggregateCommand(t, image)
			verifyCandidateAtlasSDKCommand(t, image)
			verifyCandidateAtlasSDKVectorCommand(t, image)
			verifyCandidateMCPSDKCommand(t, image)
			verifyCandidateMCPSDKVectorCommand(t, image)
			verifyCandidateEditSDKCommand(t, image)
			for _, args := range [][]string{{"--help"}, {"experimental", "--help"}, {"experimental", "fetch-packages", "--help"}, {"experimental", "atlas-impact", "--help"}, {"experimental", "atlas-mcp", "--help"}, {"experimental", "atlas-edit", "--help"}} {
				code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), args, "")
				if code != 0 || out != usage || diagnostic != "" {
					t.Fatalf("help %v: %d %q %q", args, code, out, diagnostic)
				}
			}
		})
	}
}

func candidateAtlasUnsupportedFixtures(t *testing.T) []string {
	t.Helper()
	var roots []string
	for _, source := range []string{
		`module main;fn i32 main(){let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(2);}`,
	} {
		root := writeWorkspaceSessionFixture(t, source)
		if code, _, diagnostic := runForTest([]string{"check"}, root); code != 0 {
			t.Fatalf("unsupported source must be otherwise valid: %d %s", code, diagnostic)
		}
		roots = append(roots, root)
	}
	return roots
}
func verifyCandidateAtlasUnsupported(t *testing.T, image string, roots []string) {
	t.Helper()
	for _, root := range roots {
		before := remoteTree(t, root)
		args := append([]string{"experimental", "atlas-impact"}, impactArgs("example.session.app", "fysym1-sha256-128-"+strings.Repeat("a", 32))...)
		code, out, diagnostic := fetchCLIProcess(t, image, root, args, "")
		if code != 7 || out != "" || diagnostic == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("unsupported source: %d %q %q", code, out, diagnostic)
		}
	}
}
