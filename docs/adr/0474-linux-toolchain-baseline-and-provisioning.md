# ADR 0474: Linux toolchain baseline and provisioning

- Status: Accepted (prototype policy, not a maturity promotion)
- Date: 2026-10-21
- Contract: [Linux toolchain v0](../abi/linux-toolchain-v0.md)
- Implementation owners: `internal/cdriver/`, `.github/scripts/setup_linux_toolchain.sh`, existing Frontend `native-c` job

## Context

[Run 39471634965](https://github.com/0disoft/fairyc/actions/runs/39471634965) at
`24dbbae016c45f50c168a1ff6840aa9bfa6636c1` verified glibc 2.39, GCC 13.3, and Clang 18.1.3.
Use those observations to define a reproducible reference, not to rule out older C11 toolchains.

## Decision

Choose native Ubuntu 24.04 amd64, glibc >= 2.39, GCC >= 13 or Clang >= 18, and ISO C11.
Keep same-major updates unpinned; preflight acceptance of a newer major does not establish independent
testing or support. Changing reference compiler majors requires the existing native CI lane.
Probe libc version, compiler version, and GNU/Linux x86_64 target; reject invalid results with
installation hints. Automatic compiler selection tries valid GCC before Clang.
Provision only missing GCC 13, Clang 18, libc development, binutils, and Clang 18 sanitizer packages
on checked Ubuntu 24.04/x86_64; expose task-local compiler symlinks through PATH.
Pin the existing `native-c` runner to `ubuntu-24.04`, without extra jobs or matrix entries.
The compiler driver owns the linker/startup/default native sysroot; no Windows cross-compiler,
WSL requirement, custom sysroot, or broader distribution compatibility is introduced.

## Evidence boundary

Only the full native fixture lane proves header, linker, startup, and sanitizer readiness.
[Linux job 119130288187](https://github.com/0disoft/fairyc/actions/runs/39571134661/job/119130288187)
at `e2aba615a2063202398eb5b295bfdd24d3aad4e4` passed provisioning and native fixtures with glibc 2.39,
GCC 13.3.0, and Clang 18.1.3 (report v36). This fulfills the two toolchain evidence markers for the
bounded Ubuntu 24.04 native profile. Complete runtime and standard-library evidence remain missing;
all three Linux maturity flags stay false. This receipt adds no distribution or support claim.
Package and runner sources, manual installation, and PATH setup are linked in the contract.
