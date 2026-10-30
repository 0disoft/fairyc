package cdriver

import (
	"os"
	"path/filepath"
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
	var available uint64
	known := false
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "MemAvailable:" {
				n, _ := strconv.ParseUint(fields[1], 10, 64)
				available = n * 1024
				known = n != 0
				break
			}
		}
	}
	pairs := [][2]string{{"/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory.current"}, {"/sys/fs/cgroup/memory/memory.limit_in_bytes", "/sys/fs/cgroup/memory/memory.usage_in_bytes"}}
	if data, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.SplitN(line, ":", 3)
			if len(fields) != 3 || fields[0] != "0" || fields[1] != "" {
				continue
			}
			root := "/sys/fs/cgroup"
			// A subtree-mounted container may report paths with '..'; its mounted
			// root is already covered above. Never follow them outside the mount.
			if strings.Contains(fields[2], "..") {
				continue
			}
			for dir := filepath.Join(root, strings.TrimPrefix(fields[2], "/")); dir != root; dir = filepath.Dir(dir) {
				pairs = append(pairs, [2]string{filepath.Join(dir, "memory.max"), filepath.Join(dir, "memory.current")})
			}
		}
	}
	for _, pair := range pairs {
		limitBytes, e1 := os.ReadFile(pair[0])
		usedBytes, e2 := os.ReadFile(pair[1])
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
	return available
}
