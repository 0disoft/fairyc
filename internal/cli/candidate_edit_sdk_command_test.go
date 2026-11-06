package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

const editSDKBody = `{let value=switch(b::copy(b"43")){case .ok(p):{p}case .err(_):{return 1;}};
return switch(n::parse_i64_decimal(b::view(ref value))){case .ok(p):{if(p==43){43}else{1}}case .err(_):{1}};}`

type candidateEditSDKFixture struct {
	source, body, invalid, module, name string
	provider, deniedUID                 string
	subject                             string
	verifyCurrent                       func(*testing.T, string, string, preparedAtlasWorkspace)
}

func TestCandidateEditSDKCommand(t *testing.T) {
	main := `module main;import project_edit_cli as edit;@effects(filesystem,environment,console)fn i32 main(){return edit::arguments();}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, verifyCandidateEditSDKCommand)
}

func verifyCandidateEditSDKCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateEditSDKCommandFixture(t, image, candidateEditSDKFixture{
		source:  atlasSDKCommandSource,
		body:    editSDKBody,
		invalid: "{return n::missing();}",
		module:  "std::num",
		name:    "parse_i64_decimal",
	})
}

func verifyCandidateEditSDKCommandFixture(t *testing.T, image string, fixture candidateEditSDKFixture) {
	t.Helper()
	write := func() string {
		root := writeWorkspaceSessionFixture(t, fixture.source)
		if fixture.provider != "" {
			writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", fixture.provider)
		}
		return root
	}
	root := write()
	prepare := func(root string) preparedAtlasWorkspace {
		t.Helper()
		var diagnostic bytes.Buffer
		p, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("SDK edit source oracle: %d %s", code, &diagnostic)
		}
		return p
	}
	prepared := prepare(root)
	subject := fixture.subject
	if subject == "" {
		subject = "leaf"
	}
	var symbol, sdkSymbol string
	for _, p := range prepared.partitions {
		for _, s := range p.Symbols() {
			if p.Header().PackageUID == "example.session.app" && s.Declaration.Name == subject {
				symbol = s.Visible
			} else if s.Declaration.Module == fixture.module && s.Declaration.Name == fixture.name {
				sdkSymbol = s.Visible
			}
		}
	}
	if symbol == "" || sdkSymbol == "" {
		t.Fatal("missing SDK/root edit selectors")
	}
	writeWorkspaceFile(t, root, "edits/body.txt", fixture.body)
	args := []string{"experimental", "atlas-edit", "replace-body", "--package-uid", "example.session.app", "--symbol", symbol, "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(fixture.source))}
	var reference, diagnostic bytes.Buffer
	if code := runAtlasEdit(args[2:], root, &reference, &diagnostic); code != 0 {
		t.Fatalf("SDK edit Go report: %d %s", code, &diagnostic)
	}
	var want sourceedit.Report
	if err := json.Unmarshal(reference.Bytes(), &want); err != nil || !want.Verified || want.Applied {
		t.Fatalf("SDK edit oracle report: %v %s", err, &reference)
	}
	apply := append(append([]string{}, args...), "--apply")
	oracleRoot := write()
	prepare(oracleRoot)
	writeWorkspaceFile(t, oracleRoot, "edits/body.txt", fixture.body)
	reference.Reset()
	diagnostic.Reset()
	if code := runAtlasEdit(apply[2:], oracleRoot, &reference, &diagnostic); code != 0 {
		t.Fatalf("SDK edit Go apply report: %d %s", code, &diagnostic)
	}
	var wantApply sourceedit.Report
	if err := json.Unmarshal(reference.Bytes(), &wantApply); err != nil || !wantApply.Verified || !wantApply.Applied {
		t.Fatalf("SDK edit apply oracle report: %v %s", err, &reference)
	}
	t.Setenv("PATH", "")
	before := remoteTree(t, root)
	code, out, stderr := candidateAtlasSDKCLIProcess(t, image, root, args)
	var got sourceedit.Report
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != want || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("SDK edit preview: %d %s %s, want %+v", code, out, stderr, want)
	}
	writeWorkspaceFile(t, root, "edits/body.txt", fixture.invalid)
	invalid := remoteTree(t, root)
	code, out, stderr = candidateAtlasSDKCLIProcess(t, image, root, apply)
	if code != 2 || out != "" || stderr == "" || !reflect.DeepEqual(invalid, remoteTree(t, root)) {
		t.Fatalf("invalid SDK overlay: %d %q %q", code, out, stderr)
	}
	writeWorkspaceFile(t, root, "edits/body.txt", fixture.body)
	deniedUID := fixture.deniedUID
	if deniedUID == "" {
		deniedUID = "fairyc.std"
	}
	denied := replaceWorkspaceSessionArgument(apply, "--package-uid", deniedUID)
	denied = replaceWorkspaceSessionArgument(denied, "--symbol", sdkSymbol)
	before = remoteTree(t, root)
	code, out, stderr = candidateAtlasSDKCLIProcess(t, image, root, denied)
	if code != 2 || out != "" || stderr == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("SDK declaration write: %d %q %q", code, out, stderr)
	}
	lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	// Apply verifies original and overlay both before and under the write lock.
	// Keep all four checks; this diagnostic cap is not a product timeout change.
	started := time.Now()
	code, out, stderr = candidateAtlasSDKCLIProcessLimit(t, image, root, apply, 6*time.Minute)
	t.Logf("SDK edit apply elapsed=%s", time.Since(started))
	if code != 0 || stderr != "" || json.Unmarshal([]byte(out), &got) != nil || got != wantApply {
		t.Fatalf("SDK edit apply: %d %s %s, want %+v", code, out, stderr, wantApply)
	}
	current, err := os.ReadFile(filepath.Join(root, "src/main.fy"))
	expected := fixture.source[:want.BodyStartByte] + fixture.body + fixture.source[want.BodyEndByte:]
	if err != nil || string(current) != expected {
		t.Fatalf("SDK edited source bytes: %v %s", err, current)
	}
	currentLock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil || !bytes.Equal(lock, currentLock) {
		t.Fatal("SDK edit changed lock", err)
	}
	before["src/main.fy"] = sourceedit.Digest(current)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("SDK edit left staging/lock files or changed unrelated bytes")
	}
	code, out, stderr = candidateAtlasSDKCLIProcess(t, image, root, args)
	if code != 7 || out != "" || stderr == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("SDK edit old lock: %d %q %q", code, out, stderr)
	}
	currentPrepared := prepare(root)
	before = remoteTree(t, root)
	code, out, stderr = candidateAtlasSDKCLIProcess(t, image, root, args)
	if code != 2 || out != "" || stderr == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("SDK edit old hash: %d %q %q", code, out, stderr)
	}
	if fixture.verifyCurrent != nil {
		fixture.verifyCurrent(t, image, root, currentPrepared)
	}
}
