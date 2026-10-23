package packagecache

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/moduleinterface"
)

const (
	diskProcessModeEnv    = "FAIRYC_TEST_DISK_PROCESS_MODE"
	diskProcessRootEnv    = "FAIRYC_TEST_DISK_PROCESS_ROOT"
	diskProcessPayloadEnv = "FAIRYC_TEST_DISK_PROCESS_PAYLOAD"
)

var diskProcessLimits = DiskLimits{
	MaxEntries: 8, MaxBytes: 1 << 20, MaxRecordBytes: 256 << 10, TempMaxAge: time.Second,
}

func TestDiskStoreConvergesAndRejectsConflictsAcrossProcesses(t *testing.T) {
	root := t.TempDir()
	first := startDiskProcessHelper(t, root, "put", "same")
	second := startDiskProcessHelper(t, root, "put", "same")
	waitDiskProcessHelper(t, first)
	waitDiskProcessHelper(t, second)
	store, err := NewDiskStore(root, diskProcessLimits)
	if err != nil {
		t.Fatal(err)
	}
	key := Key{1}
	if got, found := store.Get(key); !found || string(got.Artifacts[0].Data) != "same" {
		t.Fatalf("identical subprocess writers did not converge: %+v, %v", got, found)
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Entries != 1 {
		t.Fatalf("identical subprocess writers retained %d generations", stats.Entries)
	}

	first = startDiskProcessHelper(t, root, "put", "left")
	second = startDiskProcessHelper(t, root, "put", "right")
	waitDiskProcessHelper(t, first)
	waitDiskProcessHelper(t, second)
	if _, found := store.Get(key); found {
		t.Fatal("conflicting subprocess writers produced a cache hit")
	}
	stats, err = store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Entries != 0 || stats.Rejected < 3 || stats.Misses == 0 {
		t.Fatalf("conflicting subprocess state = %+v", stats)
	}
}

func TestDiskStoreRecoversTemporaryFileLeftByExitedProcess(t *testing.T) {
	root := t.TempDir()
	process := startDiskProcessHelper(t, root, "abandon-temp", "")
	waitDiskProcessHelper(t, process)
	store, err := NewDiskStore(root, diskProcessLimits)
	if err != nil {
		t.Fatal(err)
	}
	keyDirectory := store.keyDirectory(Key{1})
	entries, err := os.ReadDir(keyDirectory)
	if err != nil {
		t.Fatal(err)
	}
	foundTemp := false
	old := time.Now().Add(-2 * diskProcessLimits.TempMaxAge)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			foundTemp = true
			if err := os.Chtimes(filepath.Join(keyDirectory, entry.Name()), old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !foundTemp {
		t.Fatal("exited subprocess left no recovery temporary")
	}
	if err := store.Sweep(); err != nil {
		t.Fatal(err)
	}
	entries, err = os.ReadDir(keyDirectory)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			t.Fatalf("expired subprocess temporary still exists: %s", entry.Name())
		}
	}
}

func TestDiskStoreProcessLossAtPublicationBoundariesYieldsOnlyMissOrVerifiedHit(t *testing.T) {
	beforeRoot := t.TempDir()
	killDiskProcessAtCheckpoint(t, beforeRoot, "pause-after-temp-close")
	beforeStore, err := NewDiskStore(beforeRoot, diskProcessLimits)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := beforeStore.Get(Key{1}); found {
		t.Fatal("process loss before publication produced a cache hit")
	}
	beforeStats, err := beforeStore.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if beforeStats.Entries != 0 || beforeStats.Misses != 1 {
		t.Fatalf("pre-publication crash state = %+v", beforeStats)
	}

	afterRoot := t.TempDir()
	killDiskProcessAtCheckpoint(t, afterRoot, "pause-after-publish")
	afterStore, err := NewDiskStore(afterRoot, diskProcessLimits)
	if err != nil {
		t.Fatal(err)
	}
	got, found := afterStore.Get(Key{1})
	if !found || len(got.Artifacts) != 1 || string(got.Artifacts[0].Data) != "checkpoint" {
		t.Fatalf("post-publication crash state = %+v, %v", got, found)
	}
	afterStats, err := afterStore.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if afterStats.Entries != 1 || afterStats.Hits != 1 || afterStats.Rejected != 0 {
		t.Fatalf("post-publication crash stats = %+v", afterStats)
	}
}

func TestDiskStoreProcessHelper(t *testing.T) {
	mode := os.Getenv(diskProcessModeEnv)
	if mode == "" {
		t.Skip("subprocess helper")
	}
	root := os.Getenv(diskProcessRootEnv)
	store, err := NewDiskStore(root, diskProcessLimits)
	if err != nil {
		t.Fatal(err)
	}
	switch mode {
	case "put":
		record := Record{
			Key: Key{1}, PublicInterface: moduleinterface.Hash{1},
			Artifacts: []Artifact{{Kind: ArtifactMIR, Data: []byte(os.Getenv(diskProcessPayloadEnv))}},
		}
		if err := store.Put(record); err != nil {
			t.Fatal(err)
		}
	case "abandon-temp":
		keyDirectory := store.keyDirectory(Key{1})
		if err := os.MkdirAll(keyDirectory, 0o700); err != nil {
			t.Fatal(err)
		}
		file, err := os.CreateTemp(keyDirectory, ".tmp-")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte("interrupted publication")); err != nil {
			t.Fatal(err)
		}
		// The process exit closes the handle without invoking DiskStore.Put cleanup.
	case "pause-after-temp-close", "pause-after-publish":
		record := Record{
			Key: Key{1}, PublicInterface: moduleinterface.Hash{1},
			Artifacts: []Artifact{{Kind: ArtifactMIR, Data: []byte("checkpoint")}},
		}
		hooks := diskPutHooks{}
		if mode == "pause-after-temp-close" {
			hooks.afterTemporaryClosed = func(string) { parkDiskProcessAtCheckpoint(mode) }
		} else {
			hooks.afterPublished = func(string) { parkDiskProcessAtCheckpoint(mode) }
		}
		if err := store.put(record, hooks); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown disk subprocess helper mode %q", mode)
	}
}

func parkDiskProcessAtCheckpoint(name string) {
	_, _ = fmt.Fprintln(os.Stdout, name)
	_, _ = io.Copy(io.Discard, os.Stdin)
}

type diskProcess struct {
	command *exec.Cmd
	output  *strings.Builder
}

func startDiskProcessHelper(t *testing.T, root, mode, payload string) diskProcess {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	output := &strings.Builder{}
	command := exec.Command(executable, "-test.run=^TestDiskStoreProcessHelper$", "-test.v=false")
	command.Env = append(os.Environ(),
		diskProcessModeEnv+"="+mode,
		diskProcessRootEnv+"="+root,
		diskProcessPayloadEnv+"="+payload,
	)
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	return diskProcess{command: command, output: output}
}

func waitDiskProcessHelper(t *testing.T, process diskProcess) {
	t.Helper()
	if err := process.command.Wait(); err != nil {
		t.Fatalf("disk subprocess failed: %v\n%s", err, process.output.String())
	}
}

func killDiskProcessAtCheckpoint(t *testing.T, root, mode string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestDiskStoreProcessHelper$", "-test.v=false")
	command.Env = append(os.Environ(),
		diskProcessModeEnv+"="+mode,
		diskProcessRootEnv+"="+root,
	)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	output := &strings.Builder{}
	command.Stderr = output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("read disk subprocess checkpoint: %v; stderr %s", err, output.String())
	}
	if strings.TrimSpace(line) != mode {
		t.Fatalf("disk subprocess checkpoint = %q, want %q", line, mode)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		waited = true
		t.Fatal("killed disk subprocess exited successfully")
	}
	waited = true
}
