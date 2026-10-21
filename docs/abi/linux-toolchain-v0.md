# Linux toolchain v0

The prototype reference environment is native Ubuntu 24.04 amd64, glibc >= 2.39,
and GCC >= 13 or Clang >= 18, compiling ISO C11. Windows cross-compilers and WSL are not required.
[ADR 0474](../adr/0474-linux-toolchain-baseline-and-provisioning.md) accepts this policy.

These floors follow [run 39471634965](https://github.com/0disoft/fairyc/actions/runs/39471634965)
at `24dbbae016c45f50c168a1ff6840aa9bfa6636c1`: glibc 2.39, GCC 13.3, Clang 18.1.3.
They describe the verified reference, not a technical claim that older compilers cannot compile C11.
Same-major package updates remain unpinned. Newer compiler majors may pass preflight without being
independently tested or target-supported; reference major upgrades require the existing CI lane.

## Provisioning

On Ubuntu 24.04 amd64, install the compiler, development headers/startup objects, linker tools,
and Clang sanitizer runtime, then expose versioned compilers in a temporary PATH directory:

```sh
sudo apt-get update
sudo apt-get install gcc-13 clang-18 libc6-dev binutils libclang-rt-18-dev
toolchain_bin="$(mktemp -d)"
ln -s /usr/bin/gcc-13 "$toolchain_bin/gcc"
ln -s /usr/bin/clang-18 "$toolchain_bin/clang"
export PATH="$toolchain_bin:$PATH"
```

The compiler driver selects its linker, startup objects, and default native sysroot; no custom
sysroot or cross-distribution compatibility is promised. The CI provisioning script
`.github/scripts/setup_linux_toolchain.sh` verifies Ubuntu 24.04/x86_64, installs only missing
packages above, and exposes compilers through task-local symlinks. Its optional first argument
selects the output bin directory; ordinary shells must export PATH rather than rely on GITHUB_PATH.
The existing Frontend `native-c` job uses `ubuntu-24.04`; no new job or matrix is needed.

## Preflight and evidence

Preflight probes `getconf GNU_LIBC_VERSION`, GCC `-dumpfullversion -dumpversion` or Clang `--version`,
and `-dumpmachine` matching `x86_64-*-linux-gnu`. Reject old, malformed, or wrong-target results
with installation hints; automatic selection tries valid GCC, then Clang.
Version strings alone do not prove C11 headers, linking, startup, or sanitizer readiness.
[Linux job 119130288187](https://github.com/0disoft/fairyc/actions/runs/39571134661/job/119130288187)
at `e2aba615a2063202398eb5b295bfdd24d3aad4e4` passed reference-toolchain provisioning and the full
native fixture lane on Ubuntu 24.04 amd64: glibc 2.39, GCC 13.3.0, Clang 18.1.3, report v36.
This fulfills `toolchain-baseline` and `toolchain-provisioning` for this bounded native profile.
Complete runtime and standard-library evidence remain missing; `emittable`, `conformant`, and
`supported` remain false. The hosted Linux job supplies execution evidence unavailable on Windows.

Package sources: [GCC 13](https://packages.ubuntu.com/noble/gcc-13),
[Clang 18](https://packages.ubuntu.com/noble/clang-18), [libc development](https://packages.ubuntu.com/noble/libc6-dev).
[Runner image inventory](https://github.com/actions/runner-images/blob/main/images/ubuntu/Ubuntu2404-Readme.md).
