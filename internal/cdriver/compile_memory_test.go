package cdriver

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
)

func compileMemoryFixtureReader(files map[string]string) func(string) ([]byte, error) {
	return func(name string) ([]byte, error) {
		if data, found := files[name]; found {
			return []byte(data), nil
		}
		return nil, os.ErrNotExist
	}
}

func TestCompileMemoryPreservesKnownZeroHostBudget(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"/proc/meminfo":                 "MemAvailable: 0 kB\n",
		"/sys/fs/cgroup/memory.max":     "2147483648\n",
		"/sys/fs/cgroup/memory.current": "0\n",
	}
	got := availableCompileMemoryWithOptions(compileMemoryOptions{ReadFile: func(name string) ([]byte, error) {
		if data, found := files[name]; found {
			return []byte(data), nil
		}
		return nil, os.ErrNotExist
	}})
	if got != 0 || linuxCompileWorkers(8, got) != 1 {
		t.Fatalf("exhausted host budget became available=%d workers=%d", got, linuxCompileWorkers(8, got))
	}
}

func TestCompileMemoryHostNumericBoundaries(t *testing.T) {
	t.Parallel()
	maxKiB := ^uint64(0) / 1024
	for _, test := range []struct {
		name, value string
		want        uint64
	}{
		{"zero", "0", 0},
		{"one-KiB", "1", 1024},
		{"below-conversion-limit", strconv.FormatUint(maxKiB-1, 10), (maxKiB - 1) * 1024},
		{"at-conversion-limit", strconv.FormatUint(maxKiB, 10), maxKiB * 1024},
		{"above-conversion-limit", strconv.FormatUint(maxKiB+1, 10), 0},
		{"above-conversion-limit-plus-one", strconv.FormatUint(maxKiB+2, 10), 0},
		{"parse-overflow", "18446744073709551616", 0},
		{"negative", "-1", 0},
		{"malformed", "unknown", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			readFile := compileMemoryFixtureReader(map[string]string{"/proc/meminfo": "MemAvailable: " + test.value + " kB\n"})
			if got := availableCompileMemoryWithOptions(compileMemoryOptions{ReadFile: readFile}); got != test.want {
				t.Fatalf("host memory %q: got=%d want=%d", test.value, got, test.want)
			}
		})
	}
}

func TestCompileMemoryEffectiveBudgetBoundaries(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, hostKiB     string
		limit, used, want uint64
		v1                bool
	}{
		{"zero-host-v2", "0", 2 << 30, 0, 0, false},
		{"zero-host-v1", "0", 2 << 30, 0, 0, true},
		{"small-host", "1", 2 << 30, 0, 1024, false},
		{"smaller-container", "4194304", 2 << 30, 1 << 30, 1 << 30, false},
		{"usage-below-limit", "4194304", 2 << 30, (2 << 30) - 1, 1, false},
		{"usage-at-limit", "4194304", 2 << 30, 2 << 30, 0, false},
		{"usage-above-limit", "4194304", 2 << 30, (2 << 30) + 1, 0, false},
		{"unknown-host-keeps-container-fallback", "invalid", 2 << 30, 1 << 30, 1 << 30, false},
		{"overflow-host-keeps-container-fallback", strconv.FormatUint(^uint64(0)/1024+1, 10), 2 << 30, 1 << 30, 1 << 30, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			limitName, usedName := "/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory.current"
			if test.v1 {
				limitName, usedName = "/sys/fs/cgroup/memory/memory.limit_in_bytes", "/sys/fs/cgroup/memory/memory.usage_in_bytes"
			}
			files := map[string]string{
				"/proc/meminfo": "MemAvailable: " + test.hostKiB + " kB\n",
				limitName:       strconv.FormatUint(test.limit, 10),
				usedName:        strconv.FormatUint(test.used, 10),
			}
			if got := availableCompileMemoryWithOptions(compileMemoryOptions{ReadFile: compileMemoryFixtureReader(files)}); got != test.want {
				t.Fatalf("effective budget: got=%d want=%d", got, test.want)
			}
		})
	}
}

func TestCompileMemoryObservationControlsWorkerThreshold(t *testing.T) {
	t.Parallel()
	const threshold = 1280 << 20
	for _, memory := range []uint64{threshold - 1, threshold, threshold + 1} {
		t.Run(fmt.Sprint(memory), func(t *testing.T) {
			files := map[string]string{
				"/proc/meminfo":                 "MemAvailable: 4194304 kB\n",
				"/sys/fs/cgroup/memory.max":     strconv.FormatUint(memory, 10),
				"/sys/fs/cgroup/memory.current": "0",
			}
			got := availableCompileMemoryWithOptions(compileMemoryOptions{ReadFile: compileMemoryFixtureReader(files)})
			wantWorkers := 2
			if memory < threshold {
				wantWorkers = 1
			}
			if got != memory || linuxCompileWorkers(8, got) != wantWorkers {
				t.Fatalf("available=%d workers=%d want %d/%d", got, linuxCompileWorkers(8, got), memory, wantWorkers)
			}
		})
	}
}

func TestCompileMemoryCancellationDropsPartialObservation(t *testing.T) {
	t.Parallel()
	for _, cancelAt := range []string{"before-read", "/proc/meminfo", "/proc/self/cgroup", "/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory.current", "/sys/fs/cgroup/memory/memory.usage_in_bytes"} {
		t.Run(cancelAt, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reads, readAfterCancel := 0, false
			policy := compileMemoryOptions{Context: ctx, ReadFile: func(name string) ([]byte, error) {
				reads++
				readAfterCancel = readAfterCancel || ctx.Err() != nil
				if name == cancelAt {
					cancel()
				}
				switch name {
				case "/proc/meminfo":
					return []byte("MemAvailable: 4194304 kB\n"), nil
				case "/proc/self/cgroup":
					return nil, nil
				case "/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory/memory.limit_in_bytes":
					return []byte("4294967296"), nil
				default:
					return []byte("0"), nil
				}
			}}
			if cancelAt == "before-read" {
				cancel()
			}
			if got := availableCompileMemoryWithOptions(policy); got != 0 || readAfterCancel || cancelAt == "before-read" && reads != 0 {
				t.Fatalf("canceled observation: got=%d reads=%d readAfterCancel=%t", got, reads, readAfterCancel)
			}
			if cancelAt != "before-read" && ctx.Err() == nil {
				t.Fatal("cancellation boundary was not reached")
			}
		})
	}
}

func TestCompileMemoryMissingInputsAndInvalidOptions(t *testing.T) {
	t.Parallel()
	missing := compileMemoryOptions{ReadFile: func(string) ([]byte, error) { return nil, os.ErrPermission }}
	if got := availableCompileMemoryWithOptions(missing); got != 0 {
		t.Fatalf("unreadable memory inputs admitted %d bytes", got)
	}
	called := false
	policy := compileMemoryOptions{ReadFile: func(string) ([]byte, error) { called = true; return nil, nil }}
	if got := availableCompileMemoryWithOptions(policy, policy); got != 0 || called {
		t.Fatalf("invalid options performed I/O: got=%d called=%t", got, called)
	}
}
