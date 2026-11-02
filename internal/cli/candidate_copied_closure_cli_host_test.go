package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// The authored main and every command are retained. The Go companion consumes
// fresh Fairy C packets only; it cannot substitute a Go frontend or cached AST.
func copiedClosureCLIHost(t *testing.T, profile cdriver.Profile) string {
	t.Helper()
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	build, support := candidateParityBuildPartsWithSystemImports(t, root, map[string]bool{"CloseHandle": true, "CreateFileW": true, "GetFileInformationByHandleEx": true, "GetFinalPathNameByHandleW": true, "GetLastError": true})
	imageDir := t.TempDir()
	if parent := os.Getenv("FAIRYC_CLOSURE_CLI_ARTIFACTS"); parent != "" {
		imageDir = filepath.Join(parent, profile.Name)
		if err := os.MkdirAll(imageDir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	image := filepath.Join(imageDir, "fairyc.exe")
	digest := sha256.New()
	for _, part := range append([]string{profile.Name, build.Header, support}, build.Sources...) {
		digest.Write([]byte(part))
		digest.Write([]byte{0})
	}
	sourceID := hex.EncodeToString(digest.Sum(nil))
	type receipt struct {
		Source string
		Image  string
	}
	record := filepath.Join(imageDir, "host.json")
	var saved receipt
	data, _ := os.ReadFile(record)
	binary, _ := os.ReadFile(image)
	hash := sha256.Sum256(binary)
	reusable := json.Unmarshal(data, &saved) == nil && len(binary) > 0 && saved.Source == sourceID && saved.Image == hex.EncodeToString(hash[:])
	if !reusable {
		tc, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("full authored CLI %s: %d C parts", profile.Name, len(build.Sources))
		artifact, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		hash = sha256.Sum256(artifact.Bytes)
		data, err = json.Marshal(receipt{sourceID, hex.EncodeToString(hash[:])})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(record, data, 0600); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Logf("reuse exact C/image digest for %s CLI", profile.Name)
	}
	companion := filepath.Join(imageDir, "fairyc-candidate-build.exe")
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", companion, "./tools/candidate/fairyc-candidate-build")
	command.Dir = filepath.Join("..", "..")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build C-only companion: %v %s", err, output)
	}
	return image
}
