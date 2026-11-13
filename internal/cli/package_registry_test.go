package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/packagesource"
)

func preparationJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func packageRegistryFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "패키지")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	trust := packagesource.RegistryTrust{Schema: packagesource.RegistryTrustSchema, MinimumSequence: 7, Keys: []packagesource.RegistryKey{{ID: "fixture", PublicKey: hex.EncodeToString(public)}}}
	catalog := packagesource.RegistryCatalog{Schema: packagesource.RegistryCatalogSchema, Sequence: 7, Expires: time.Now().UTC().Add(time.Hour).Truncate(time.Second).Format(time.RFC3339), Packages: []packagesource.RegistryRelease{
		{PackageUID: "example.math", Version: "1.0.0", URL: "https://packages.example/math-1.0.0.zip", SHA256: strings.Repeat("a", 64)},
		{PackageUID: "example.math", Version: "1.2.0", URL: "https://packages.example/math-1.2.0.zip", SHA256: strings.Repeat("b", 64)},
	}}
	data := preparationJSON(t, catalog)
	sig := ed25519.Sign(private, append([]byte(packagesource.RegistrySignatureDomain), data...))
	requests := packagesource.RegistryRequests{Schema: packagesource.RegistryRequestsSchema, Packages: []packagesource.RegistryRequest{{Path: "vendor/math", PackageUID: "example.math", Version: "^1.0.0"}}}
	current := packagesource.Plan{Schema: packagesource.Schema, Packages: []packagesource.Pin{{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: catalog.Packages[0].URL, SHA256: catalog.Packages[0].SHA256}}}
	for name, data := range map[string][]byte{"trust.json": preparationJSON(t, trust), "catalog.json": data, "signature.json": preparationJSON(t, packagesource.RegistrySignature{Schema: packagesource.RegistrySignatureSchema, KeyID: "fixture", Signature: hex.EncodeToString(sig)}), "requests.json": preparationJSON(t, requests), "fairy.sources.json": preparationJSON(t, current), "fairy.toml": []byte("unchanged manifest"), "fairy.lock": []byte("unchanged lock")} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

type packagePreparationInvoker func(args []string, root string) (int, string, string)

func checkPackageRegistryCLI(t *testing.T, invoke packagePreparationInvoker) {
	t.Helper()
	root := packageRegistryFixture(t)
	before := map[string][]byte{}
	for _, name := range []string{"trust.json", "fairy.sources.json", "fairy.toml", "fairy.lock"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		before[name] = data
	}
	args := []string{"experimental", "registry-resolve", "trust.json", "catalog.json", "signature.json", "requests.json"}
	code, out, stderr := invoke(args, root)
	if code != 0 || stderr != "" {
		t.Fatalf("resolve: %d %s %s", code, out, stderr)
	}
	var resolution packagesource.RegistryResolution
	if err := json.Unmarshal([]byte(out), &resolution); err != nil || resolution.Sequence != 7 || resolution.SourcePlan.Packages[0].Version != "1.2.0" {
		t.Fatalf("resolution: %s %v", out, err)
	}
	update := append([]string{}, args...)
	update[1] = "registry-update"
	update = append(update, "fairy.sources.json")
	code, out, stderr = invoke(update, root)
	var proposal packagesource.UpdateProposal
	if code != 0 || stderr != "" {
		t.Fatalf("update: %d %s %s", code, out, stderr)
	}
	if err := json.Unmarshal([]byte(out), &proposal); err != nil || len(proposal.Changes) != 1 || proposal.Changes[0].Before.Version != "1.0.0" || proposal.Changes[0].After.Version != "1.2.0" {
		t.Fatalf("proposal: %s %v", out, err)
	}
	for name, old := range before {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(old, data) {
			t.Fatalf("preparation changed %s", name)
		}
	}
	catalog, err := os.ReadFile(filepath.Join(root, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalog = append(catalog, ' ')
	if err := os.WriteFile(filepath.Join(root, "catalog.json"), catalog, 0600); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := invoke(args, root); code != ExitPackageFailure || out != "" {
		t.Fatalf("tampered catalog: %d %s", code, out)
	}
}

func TestPackageRegistryCLI(t *testing.T) {
	checkPackageRegistryCLI(t, runForTest)
	for _, command := range []string{"registry-resolve", "registry-update", "git-snapshot"} {
		if code, out, _ := runForTest([]string{"experimental", command}, t.TempDir()); code != ExitInvalidInvocation || out != "" {
			t.Fatal("invalid invocation admitted")
		}
	}
	root := packageRegistryFixture(t)
	var stderr bytes.Buffer
	if code := Run([]string{"experimental", "registry-resolve", "trust.json", "catalog.json", "signature.json", "requests.json"}, root, failingWriter{}, &stderr); code != ExitInternalFailure {
		t.Fatalf("writer failure: %d", code)
	}
}

func TestGitSnapshotExclusiveOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "snapshot.zip"), []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := writeGitSnapshot(root, "snapshot.zip", []byte("new")); err == nil {
		t.Fatal("output overwrite accepted")
	}
	data, err := os.ReadFile(filepath.Join(dir, "snapshot.zip"))
	if err != nil || string(data) != "existing" {
		t.Fatal("existing output changed")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatal("snapshot temporary leaked")
	}
}

func TestPackagePreparationCLIProcess(t *testing.T) {
	executable := os.Getenv("FAIRYC_PACKAGE_PREPARATION_CLI")
	if executable == "" {
		t.Skip("set explicit current Go CLI executable for process replay")
	}
	invoke := func(args []string, root string) (int, string, string) {
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, args...)
		cmd.Dir = root
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err == nil {
			return 0, stdout.String(), stderr.String()
		}
		if failure, ok := err.(*exec.ExitError); ok {
			return failure.ExitCode(), stdout.String(), stderr.String()
		}
		t.Fatal(err)
		return -1, "", ""
	}
	checkPackageRegistryCLI(t, invoke)
	root := t.TempDir()
	repository := filepath.Join(root, "repository")
	if err := os.Mkdir(repository, 0700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", repository}, args...)...)
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if !strings.HasPrefix(strings.ToUpper(key), "GIT_") {
				cmd.Env = append(cmd.Env, entry)
			}
		}
		cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture git: %v %s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	if err := os.Mkdir(filepath.Join(repository, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, repository, "fairy.toml", "schema=\"fairyc.manifest/v1\"\n[package]\nuid=\"example.math\"\nname=\"math\"\nversion=\"1.0.0\"\nlanguage=\"fairyc-language/20\"\nsource=\"src\"\n")
	writeWorkspaceFile(t, repository, "src/main.fy", "module main;pub fn i32 main(){return 42;}\n")
	git("add", ".")
	git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath="+filepath.Join(repository, ".git", "empty-hooks"), "commit", "-qm", "fixture")
	commit := git("rev-parse", "HEAD")
	args := []string{"experimental", "git-snapshot", "repository", commit, "snapshot.zip"}
	code, out, stderr := invoke(args, root)
	if code != 0 || stderr != "" {
		t.Fatalf("snapshot: %d %s %s", code, out, stderr)
	}
	var receipt packagesource.GitSnapshot
	if err := json.Unmarshal([]byte(out), &receipt); err != nil || receipt.Commit != commit || receipt.PackageUID != "example.math" {
		t.Fatalf("snapshot receipt: %s %v", out, err)
	}
	archive, err := os.ReadFile(filepath.Join(root, "snapshot.zip"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(archive)
	if receipt.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatal("snapshot digest mismatch")
	}
	// Replay the generated archive through the existing exact-pin offline fetcher.
	writeWorkspaceFile(t, root, "fairy.toml", "schema=\"fairyc.manifest/v1\"\n[package]\nuid=\"example.app\"\nname=\"app\"\nversion=\"1.0.0\"\nlanguage=\"fairyc-language/20\"\nsource=\"src\"\n[workspace]\nmembers=[\"vendor/math\"]\n[dependencies.math]\npath=\"vendor/math\"\nversion=\"1.0.0\"\nfeatures=[]\n")
	plan := packagesource.Plan{Schema: packagesource.Schema, Packages: []packagesource.Pin{{Path: "vendor/math", PackageUID: receipt.PackageUID, Version: receipt.Version, URL: "https://packages.example/math.zip", SHA256: receipt.SHA256}}}
	writeWorkspaceFile(t, root, "fairy.sources.json", string(preparationJSON(t, plan)))
	if err := os.MkdirAll(filepath.Join(root, "build", "package-sources"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build", "package-sources", receipt.SHA256+".zip"), archive, 0600); err != nil {
		t.Fatal(err)
	}
	if code, out, stderr := invoke([]string{"experimental", "fetch-packages", "--offline"}, root); code != 0 || stderr != "" || !strings.Contains(out, "1 installed") {
		t.Fatalf("snapshot offline consumption: %d %s %s", code, out, stderr)
	}
	if code, out, _ := invoke(args, root); code != ExitPackageFailure || out != "" {
		t.Fatal("existing output accepted")
	}
	args[3] = "HEAD"
	args[4] = "failed.zip"
	if code, out, _ := invoke(args, root); code != ExitPackageFailure || out != "" {
		t.Fatal("mutable Git source admitted")
	}
	if _, err := os.Lstat(filepath.Join(root, "failed.zip")); !os.IsNotExist(err) {
		t.Fatal("failed snapshot left output")
	}
}
