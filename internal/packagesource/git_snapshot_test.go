package packagesource

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const gitSnapshotFixtureManifest = `schema="fairyc.manifest/v1"
[package]
uid="example.math"
name="math"
version="1.0.0"
language="fairyc-language/20"
source="src"
`

func snapshotGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitObjectCommand(t.Context(), dir, nil, 4096, args...)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
func snapshotCommit(t *testing.T, dir string) string {
	t.Helper()
	snapshotGit(t, dir, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath="+filepath.Join(dir, ".git", "empty-hooks"), "commit", "-m", "fixture")
	return snapshotGit(t, dir, "rev-parse", "HEAD")
}
func snapshotFixture(t *testing.T) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git unavailable")
	}
	dir := t.TempDir()
	snapshotGit(t, dir, "init")
	if err := os.Mkdir(filepath.Join(dir, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"fairy.toml": gitSnapshotFixtureManifest, "src/main.fy": "fn main() -> i32 { return 1; }\n", ".gitattributes": "src/main.fy export-ignore\n"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	snapshotGit(t, dir, "add", ".")
	return dir, snapshotCommit(t, dir)
}

func TestLocalGitSnapshotCommittedBytes(t *testing.T) {
	dir, commit := snapshotFixture(t)
	t.Setenv("GIT_DIR", filepath.Join(dir, "missing-ambient-repository"))
	t.Setenv("GIT_OBJECT_DIRECTORY", filepath.Join(dir, "missing-ambient-objects"))
	if err := os.WriteFile(filepath.Join(dir, "src", "main.fy"), []byte("uncommitted bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := PrepareGitSnapshot(t.Context(), dir, commit)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PrepareGitSnapshot(t.Context(), dir, commit)
	if err != nil {
		t.Fatal(err)
	}
	if first.Commit != commit || first.PackageUID != "example.math" || first.Version != "1.0.0" || first.Files != 3 || first.SHA256 != hash(first.Archive) || !bytes.Equal(first.Archive, second.Archive) {
		t.Fatalf("snapshot receipt or determinism: %+v", first)
	}
	b, err := unpack(t.Context(), first.Archive, Pin{PackageUID: first.PackageUID, Version: first.Version}, first.Language)
	if err != nil {
		t.Fatal(err)
	}
	if string(b.files["src/main.fy"]) != "fn main() -> i32 { return 1; }\n" {
		t.Fatal("working tree or export-ignore affected object snapshot")
	}
	for _, bad := range []string{"HEAD", commit[:12], strings.ToUpper(commit), strings.Repeat("f", 40)} {
		if _, err := PrepareGitSnapshot(t.Context(), dir, bad); err == nil {
			t.Fatalf("mutable/invalid/missing commit accepted: %s", bad)
		}
	}
	if _, err := PrepareGitSnapshot(t.Context(), filepath.Join(dir, "src"), commit); err == nil {
		t.Fatal("implicit parent repository accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := PrepareGitSnapshot(ctx, dir, commit); err == nil {
		t.Fatal("canceled snapshot accepted")
	}
}

func TestLocalGitSnapshotRejectsLinksAndSubmodules(t *testing.T) {
	for _, mode := range []string{"120000", "160000"} {
		t.Run(mode, func(t *testing.T) {
			dir, commit := snapshotFixture(t)
			object := commit
			if mode == "120000" {
				object = snapshotGit(t, dir, "rev-parse", commit+":src/main.fy")
			}
			snapshotGit(t, dir, "update-index", "--add", "--cacheinfo", mode+","+object+",nested")
			next := snapshotCommit(t, dir)
			if _, err := PrepareGitSnapshot(t.Context(), dir, next); err == nil {
				t.Fatal("symlink/submodule admitted")
			}
		})
	}
}

func TestGitObjectWriterBoundCannotBypassThroughReaderFrom(t *testing.T) {
	buffer := &boundedGitBuffer{limit: 8}
	// LimitReader exposes no WriterTo; an embedded bytes.Buffer.ReadFrom would
	// silently bypass the checked Write method used by os/exec's pipe copier.
	if _, err := io.Copy(buffer, io.LimitReader(strings.NewReader(strings.Repeat("x", 32)), 32)); err == nil || len(buffer.Bytes()) > 8 {
		t.Fatal("Git object output bound bypassed")
	}
}
