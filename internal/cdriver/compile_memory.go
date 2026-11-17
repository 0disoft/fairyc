package cdriver

import (
	"context"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
)

func linuxCompileWorkers(cpus int, available uint64) int {
	const reserve, perWorker = 256 << 20, 512 << 20
	if available <= reserve {
		return 1
	}
	return max(1, min(2, cpus, int(min(uint64(2), (available-reserve)/perWorker))))
}

// Prefer currently available memory, including a container's remaining budget.
// Unknown limits choose one worker. Never infer parallelism from host RAM alone
// when the current process is constrained by a lower cgroup allowance.
func availableCompileMemory() uint64 {
	return availableCompileMemoryWithOptions()
}

// Nil fields retain Background and os.ReadFile. A canceled observation returns
// the existing unknown-budget value (zero), which admits only one worker.
// Cancellation is cooperative between reads, not syscall preemption.
type compileMemoryOptions struct {
	Context  context.Context
	ReadFile func(string) ([]byte, error)
}

func availableCompileMemoryWithOptions(options ...compileMemoryOptions) uint64 {
	if len(options) > 1 {
		return 0
	}
	var policy compileMemoryOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if policy.ReadFile == nil {
		policy.ReadFile = os.ReadFile
	}
	readFile := func(name string) ([]byte, error) {
		if err := policy.Context.Err(); err != nil {
			return nil, err
		}
		data, err := policy.ReadFile(name)
		if stop := policy.Context.Err(); stop != nil {
			return nil, stop
		}
		return data, err
	}
	var available uint64
	known := false
	if data, err := readFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "MemAvailable:" {
				n, parseErr := strconv.ParseUint(fields[1], 10, 64)
				// Zero is a valid observation of exhausted host memory. Only
				// a parse/conversion failure makes this budget unknown.
				if parseErr == nil && n <= ^uint64(0)/1024 {
					available = n * 1024
					known = true
				}
				break
			}
		}
	}
	pairs := [][2]string{{"/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory.current"}, {"/sys/fs/cgroup/memory/memory.limit_in_bytes", "/sys/fs/cgroup/memory/memory.usage_in_bytes"}}
	if data, err := readFile("/proc/self/cgroup"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.SplitN(line, ":", 3)
			if len(fields) != 3 {
				continue
			}
			var root, limitName, usedName string
			hierarchy, hierarchyErr := strconv.ParseUint(fields[0], 10, 64)
			switch {
			case fields[0] == "0" && fields[1] == "":
				root, limitName, usedName = "/sys/fs/cgroup", "memory.max", "memory.current"
			case hierarchyErr == nil && hierarchy > 0 && slices.Contains(strings.Split(fields[1], ","), "memory"):
				root, limitName, usedName = "/sys/fs/cgroup/memory", "memory.limit_in_bytes", "memory.usage_in_bytes"
			default:
				continue
			}
			// A subtree-mounted container may report paths with '..'; its mounted
			// root is already covered above. Never follow them outside the mount.
			if strings.Contains(fields[2], "..") {
				continue
			}
			// These are Linux virtual paths even in host-independent tests.
			for dir := path.Join(root, strings.TrimPrefix(fields[2], "/")); dir != root; dir = path.Dir(dir) {
				pairs = append(pairs, [2]string{path.Join(dir, limitName), path.Join(dir, usedName)})
			}
		}
	}
	for _, pair := range pairs {
		if policy.Context.Err() != nil {
			return 0
		}
		limitBytes, e1 := readFile(pair[0])
		usedBytes, e2 := readFile(pair[1])
		if policy.Context.Err() != nil {
			return 0
		}
		if e1 != nil || e2 != nil {
			continue
		}
		limit, e1 := strconv.ParseUint(strings.TrimSpace(string(limitBytes)), 10, 64)
		used, e2 := strconv.ParseUint(strings.TrimSpace(string(usedBytes)), 10, 64)
		if e1 != nil || e2 != nil {
			continue
		}
		remaining := uint64(0)
		if limit > used {
			remaining = limit - used
		}
		if !known || remaining < available {
			available = remaining
			known = true
		}
	}
	if policy.Context.Err() != nil {
		return 0
	}
	return available
}
