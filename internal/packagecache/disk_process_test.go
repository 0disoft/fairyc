package packagecache

import (
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
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			t.Fatalf("expired subprocess temporary still exists: %s", entry.Name())
		}
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
	default:
		t.Fatalf("unknown disk subprocess helper mode %q", mode)
	}
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
