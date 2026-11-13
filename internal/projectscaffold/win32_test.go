package projectscaffold

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

func TestWin32BundleTracksCanonicalSources(t *testing.T) {
	var receipt struct {
		Version, Language string
		BundleSha256      string
		Inputs            map[string]string
	}
	data, err := os.ReadFile("assets/win32.inputs.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Version != DefaultVersion || receipt.Language != DefaultLanguage || len(receipt.Inputs) < 20 {
		t.Fatal("embedded starter defaults are stale; run go generate ./internal/projectscaffold")
	}
	for source, want := range receipt.Inputs {
		// Line ending normalization matches the bundle builder.
		data, err := os.ReadFile(filepath.Join("../..", filepath.FromSlash(source)))
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(source) != ".ico" {
			data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != want {
			t.Errorf("embedded starter input %s changed; run go generate ./internal/projectscaffold", source)
		}
	}
	digest := sha256.Sum256(win32Bundle)
	if hex.EncodeToString(digest[:]) != receipt.BundleSha256 {
		t.Fatal("embedded starter archive differs from its source receipt")
	}
}

func TestWin32CreateUsesEmbeddedStandaloneFiles(t *testing.T) {
	parent := t.TempDir()
	spec := Spec{Name: "my-tool", UID: "example.my-tool", Template: "win32"}
	if err := Create(parent, spec); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, spec.Name)
	for _, name := range []string{"src/domain.fy", "src/ui.fy", "native/window.c", "native/ime.c", "native/app.ico", "native/app.rc", "win32_app/portable.py", "package_release.py", "verify_cli.py", "LICENSE"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil || len(data) == 0 {
			t.Fatalf("missing standalone input %s: %v", name, err)
		}
	}
	manifest, err := projectmanifest.Parse([]byte(readScaffoldFile(t, root, "fairy.toml")))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Package.Name != spec.Name || manifest.Package.UID != spec.UID || manifest.Package.Version != DefaultVersion {
		t.Fatal("incorrect generated manifest")
	}
	for _, name := range []string{"app.toml", "native/app.rc", "README.md"} {
		text := readScaffoldFile(t, root, name)
		if strings.Contains(text, "@@") || !strings.Contains(text, spec.Name) {
			t.Fatalf("unrendered identity in %s", name)
		}
	}
	if !strings.Contains(readScaffoldFile(t, root, "app.toml"), `company_name = ""`) {
		t.Fatal("publisher must remain empty by default")
	}
	if _, err := os.Stat(filepath.Join(root, "fairy.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("generation must not create a lock: %v", err)
	}
	before := readScaffoldFile(t, root, "src/main.fy")
	err = Create(parent, spec)
	var failure *Error
	if !errors.As(err, &failure) || failure.Reason != ReasonTargetExists || readScaffoldFile(t, root, "src/main.fy") != before {
		t.Fatal("existing starter was not preserved")
	}
}

func TestWin32IdentityAndTemplateValidateBeforeFilesystem(t *testing.T) {
	for _, spec := range []Spec{
		{Name: "tool", UID: "example.tool", Template: "unknown"},
		{Name: strings.Repeat("a", 128), UID: "example.tool", Template: "win32"},
		{Name: "tool", UID: "example." + strings.Repeat("a", 256), Template: "win32"},
	} {
		err := Create(filepath.Join(t.TempDir(), "missing"), spec)
		var failure *Error
		if !errors.As(err, &failure) || (failure.Reason != ReasonInvalidTemplate && failure.Reason != ReasonWin32Identity) {
			t.Fatalf("unexpected validation: %v", err)
		}
	}
}
