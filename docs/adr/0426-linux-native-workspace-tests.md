# ADR 0426: Linux native workspace tests

- Status: Accepted
- Date: 2026-10-03

## Decision

`fairyc test --native` selects the same host backend as ordinary build/run:
Windows AMD64 MSVC or Linux GNU amd64/glibc with GCC preferred over Clang.
The test iterator, canonical selection/order, isolated MIR harness, stdin
replay, timeout, stream capture and report writer are shared. Each OS adapter
validates its own private trap channel before returning a common outcome.
Tests run in a fresh temporary process/directory and retain no executable.

The existing v1-v4 report shapes are unchanged. Their target enums add
`x86_64-linux-gnu`; Windows report bytes and interpreter target/profile nulls
stay unchanged. Consumers that restrict target values must use the updated
schemas. SDK stage0.79, language/20 and runtime/0 do not change. This does not
promote target conformance or publish a Linux compiler distribution.

## Constant traps and non-returning calls

An actual Windows test exposed MSVC C4702 on the native harness's return after
`assert(false)`. The private C transport now omits impossible literal-bool edges
and propagates proven non-returning local callees to their callers. Those local
functions use private C11 `_Noreturn void` declarations; their normal continuations are
omitted. Foreign calls stay unknown and unwind edges remain conservative. No
MSVC warnings are disabled and no public Fairy no-return attribute is added.

## Validation

The shared host-native test covers development/release execution, a trapped test
followed by a passing SDK-vector test, file/directory/exact selection, source
locations, binary JSON streams, empty/UTF-8 arguments, direct stdin, per-test
file replay and bounded timeout. Existing Windows unit/native proofs remain;
the same test runs in the existing Ubuntu Go job without adding a runner.

This Windows host has no usable WSL distribution or container runtime. Windows
execution and Linux cross-compilation can be checked here; actual Linux CLI and
publication/save-failure execution must come from a Linux host. Subsequent
[Drimi Linux evidence](../compatibility/linux-drimi-20261003.md) records native
test success and the exact original/patched save matrix boundaries; newer
runtime-cache execution remains pending rather than inheriting those results.
