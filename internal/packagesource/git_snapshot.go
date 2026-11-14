package packagesource

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

type GitSnapshot struct {
	Schema     string `json:"schema"`
	Commit     string `json:"commit"`
	PackageUID string `json:"packageUid"`
	Version    string `json:"version"`
	Language   string `json:"language"`
	SHA256     string `json:"sha256"`
	Files      int    `json:"files"`
	Archive    []byte `json:"-"`
}
type gitTreeBlob struct{ name, object string }
type boundedGitBuffer struct {
	data  bytes.Buffer
	limit int
}

func (b *boundedGitBuffer) Bytes() []byte { return b.data.Bytes() }

func (b *boundedGitBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.data.Len() {
		return 0, fmt.Errorf("Git object output byte bound")
	}
	return b.data.Write(p)
}

func configuredGitObjectCommand(ctx context.Context, repository string, input []byte, args ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-C", repository}, args...)...)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(strings.ToUpper(key), "GIT_") {
			command.Env = append(command.Env, entry)
		}
	}
	command.Env = append(command.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1", "GIT_TERMINAL_PROMPT=0")
	command.Stdin = bytes.NewReader(input)
	command.WaitDelay = time.Second
	return command
}

func gitObjectCommand(ctx context.Context, repository string, input []byte, limit int, args ...string) ([]byte, error) {
	command := configuredGitObjectCommand(ctx, repository, input, args...)
	out := &boundedGitBuffer{limit: limit}
	stderr := &boundedGitBuffer{limit: 4096}
	command.Stdout = out
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("local Git object command failed: %w", err)
	}
	return out.Bytes(), nil
}

// PrepareGitSnapshot reads committed object bytes, never a checkout or archive
// filter. No repository code, hooks, submodules or network command is executed.
func PrepareGitSnapshot(ctx context.Context, repository, commit string) (GitSnapshot, error) {
	if len(commit) != 40 && len(commit) != 64 {
		return GitSnapshot{}, fmt.Errorf("full immutable Git commit required")
	}
	if _, err := canonicalHex(commit, len(commit)/2); err != nil {
		return GitSnapshot{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	absolute, err := filepath.Abs(repository)
	if err != nil {
		return GitSnapshot{}, err
	}
	info, err := os.Lstat(absolute)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return GitSnapshot{}, fmt.Errorf("local repository root must be a directory")
	}
	root, err := gitObjectCommand(ctx, absolute, nil, 4096, "rev-parse", "--show-toplevel")
	if err != nil {
		return GitSnapshot{}, err
	}
	rootInfo, err := os.Stat(strings.TrimSpace(string(root)))
	if err != nil || !os.SameFile(info, rootInfo) {
		return GitSnapshot{}, fmt.Errorf("repository argument must name the worktree root")
	}
	resolved, err := gitObjectCommand(ctx, absolute, nil, 128, "rev-parse", "--verify", "--end-of-options", commit+"^{commit}")
	if err != nil || strings.TrimSpace(string(resolved)) != commit {
		return GitSnapshot{}, fmt.Errorf("immutable commit object required")
	}
	tree, err := gitObjectCommand(ctx, absolute, nil, MaxFiles*(maxArchivePathBytes+128), "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return GitSnapshot{}, err
	}
	var entries []gitTreeBlob
	var input strings.Builder
	for len(tree) > 0 {
		end := bytes.IndexByte(tree, 0)
		if end < 0 {
			return GitSnapshot{}, fmt.Errorf("unterminated Git tree record")
		}
		record := string(tree[:end])
		tree = tree[end+1:]
		header, name, ok := strings.Cut(record, "\t")
		parts := strings.Split(header, " ")
		if !ok || len(parts) != 3 || (parts[0] != "100644" && parts[0] != "100755") || parts[1] != "blob" || !portableEntry(name) || len(parts[2]) != len(commit) {
			return GitSnapshot{}, fmt.Errorf("Git tree contains unsupported path, symlink, submodule or mode")
		}
		if _, err := canonicalHex(parts[2], len(commit)/2); err != nil {
			return GitSnapshot{}, err
		}
		entries = append(entries, gitTreeBlob{name: name, object: parts[2]})
		if len(entries) > MaxFiles {
			return GitSnapshot{}, fmt.Errorf("Git snapshot file bound")
		}
		input.WriteString(parts[2] + "\n")
	}
	if len(entries) == 0 {
		return GitSnapshot{}, fmt.Errorf("empty Git snapshot")
	}
	archive := &boundedGitBuffer{limit: MaxArchiveBytes}
	writer := zip.NewWriter(archive)
	var manifestBytes []byte
	err = streamGitObjectCommand(ctx, absolute, []byte(input.String()), func(reader io.Reader) error {
		var readErr error
		manifestBytes, readErr = writeGitSnapshotBlobs(ctx, reader, entries, writer)
		return readErr
	}, "cat-file", "--batch")
	if err != nil {
		return GitSnapshot{}, err
	}
	if err = writer.Close(); err != nil {
		return GitSnapshot{}, err
	}
	manifest, err := projectmanifest.Parse(manifestBytes)
	if err != nil {
		return GitSnapshot{}, fmt.Errorf("Git package manifest: %w", err)
	}
	data := archive.Bytes()
	pin := Pin{PackageUID: manifest.Package.UID, Version: manifest.Package.Version}
	if _, err = unpack(ctx, data, pin, manifest.Package.Language); err != nil {
		return GitSnapshot{}, err
	}
	return GitSnapshot{Schema: "fairyc.git-snapshot/v1", Commit: commit, PackageUID: pin.PackageUID, Version: pin.Version, Language: manifest.Package.Language, SHA256: hash(data), Files: len(entries), Archive: data}, nil
}
