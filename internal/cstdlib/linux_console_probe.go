package cstdlib

import _ "embed"

//go:embed stage0_linux_console_failure_probe.c
var stage0LinuxConsoleFailureProbe string

// LinuxConsoleFailureProbeSource returns deterministic console syscall failure scripts.
func LinuxConsoleFailureProbeSource() (string, error) {
	return canonicalSource("Linux console failure probe", stage0LinuxConsoleFailureProbe)
}

// LinuxConsoleFailureProbeImplementationSource redirects only production console syscalls.
func LinuxConsoleFailureProbeImplementationSource() (string, error) {
	source, err := LinuxConsoleSource()
	if err != nil {
		return "", err
	}
	const prefix = `#define _POSIX_C_SOURCE 200809L
#include <errno.h>
#include <limits.h>
#include <stddef.h>
#include <stdint.h>
#include <unistd.h>
ssize_t fy_std_probe_write_v0(int, const void *, size_t);
ssize_t fy_std_probe_read_v0(int, void *, size_t);
#define write(descriptor, data, length) fy_std_probe_write_v0(descriptor, data, length)
#define read(descriptor, data, length) fy_std_probe_read_v0(descriptor, data, length)
`
	return canonicalSource("Linux console failure probe implementation", prefix+source+"\n#undef read\n#undef write\n")
}
