package cdriver

import (
	"context"
	"path"
	"strconv"
	"strings"
	"testing"
)

func TestCompileMemoryUsesProcessV1Hierarchy(t *testing.T) {
	for _, controllers := range []string{"memory", "cpu,memory", "memory,cpu"} {
		t.Run(controllers, func(t *testing.T) {
			files := map[string]string{
				"/proc/meminfo":     "MemAvailable: 4194304 kB\n",
				"/proc/self/cgroup": "5:" + controllers + ":/jobs/build\n",
				path.Join("/sys/fs/cgroup/memory", "jobs", "build", "memory.limit_in_bytes"): strconv.FormatUint(768<<20, 10),
				path.Join("/sys/fs/cgroup/memory", "jobs", "build", "memory.usage_in_bytes"): strconv.FormatUint(256<<20, 10),
			}
			got := availableCompileMemoryWithOptions(compileMemoryOptions{ReadFile: compileMemoryFixtureReader(files)})
			if got != 512<<20 || linuxCompileWorkers(8, got) != 1 {
				t.Fatalf("process limit ignored: available=%d workers=%d, want %d/1", got, linuxCompileWorkers(8, got), 512<<20)
			}
		})
	}
}

func TestCompileMemoryProcessHierarchyKeepsTightestBudget(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		root, limitName, usedName, membership := "/sys/fs/cgroup", "memory.max", "memory.current", "0::/jobs/build\n"
		if version == "v1" {
			root, limitName, usedName, membership = "/sys/fs/cgroup/memory", "memory.limit_in_bytes", "memory.usage_in_bytes", "5:memory:/jobs/build\n"
		}
		for _, test := range []struct {
			name                    string
			leafLimit, leafUsed     uint64
			parentLimit, parentUsed uint64
			want                    uint64
			workers                 int
		}{
			{"parent-tightest", 2 << 30, 0, 768 << 20, 256 << 20, 512 << 20, 1},
			{"exhausted-leaf", 256 << 20, 256 << 20, 4 << 30, 0, 0, 1},
			{"overused-leaf", 256 << 20, 512 << 20, 4 << 30, 0, 0, 1},
			{"parallel-threshold", 1280 << 20, 0, 4 << 30, 0, 1280 << 20, 2},
		} {
			t.Run(version+"/"+test.name, func(t *testing.T) {
				files := map[string]string{
					"/proc/meminfo": "MemAvailable: 4194304 kB\n", "/proc/self/cgroup": membership,
					path.Join(root, "jobs", "build", limitName): strconv.FormatUint(test.leafLimit, 10),
					path.Join(root, "jobs", "build", usedName):  strconv.FormatUint(test.leafUsed, 10),
					path.Join(root, "jobs", limitName):          strconv.FormatUint(test.parentLimit, 10),
					path.Join(root, "jobs", usedName):           strconv.FormatUint(test.parentUsed, 10),
				}
				got := availableCompileMemoryWithOptions(compileMemoryOptions{ReadFile: compileMemoryFixtureReader(files)})
				if got != test.want || linuxCompileWorkers(8, got) != test.workers {
					t.Fatalf("hierarchical budget: available=%d workers=%d, want %d/%d", got, linuxCompileWorkers(8, got), test.want, test.workers)
				}
			})
		}
	}
}

func TestCompileMemoryProcessHierarchyRejectsInvalidMembership(t *testing.T) {
	for _, membership := range []string{
		"5:memoryx:/jobs/build", "5:cpu,nomemory:/jobs/build", "bad:memory:/jobs/build",
		"18446744073709551616:memory:/jobs/build", "-1:memory:/jobs/build",
		"0:memory:/jobs/build", "5:memory:/../outside", "0::/../outside", "malformed",
	} {
		t.Run(membership, func(t *testing.T) {
			read := compileMemoryFixtureReader(map[string]string{"/proc/meminfo": "MemAvailable: 4194304 kB\n", "/proc/self/cgroup": membership})
			forbidden := false
			got := availableCompileMemoryWithOptions(compileMemoryOptions{ReadFile: func(name string) ([]byte, error) {
				forbidden = forbidden || strings.Contains(name, "/jobs/") || strings.Contains(name, "outside")
				return read(name)
			}})
			if got != 4<<30 || forbidden {
				t.Fatalf("invalid membership used: available=%d forbidden=%t", got, forbidden)
			}
		})
	}
}

func TestCompileMemoryCancelsDuringProcessHierarchyRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	read := compileMemoryFixtureReader(map[string]string{"/proc/meminfo": "MemAvailable: 4194304 kB\n", "/proc/self/cgroup": "5:memory:/jobs/build\n"})
	readAfterCancel := false
	got := availableCompileMemoryWithOptions(compileMemoryOptions{Context: ctx, ReadFile: func(name string) ([]byte, error) {
		readAfterCancel = readAfterCancel || ctx.Err() != nil
		if name == "/sys/fs/cgroup/memory/jobs/build/memory.limit_in_bytes" {
			cancel()
		}
		return read(name)
	}})
	if got != 0 || ctx.Err() == nil || readAfterCancel {
		t.Fatalf("canceled hierarchy observation survived: available=%d canceled=%v readAfterCancel=%t", got, ctx.Err(), readAfterCancel)
	}
}
