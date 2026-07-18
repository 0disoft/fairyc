# Native C fixture runtime

`runtime.c` is a test-only implementation of the internal `fy_runtime_trap(uint32_t)` transport
symbol. The native differential runner supplies a reserved process-status base and maps each fixed
Stage 0 trap code to one status without unwinding, running cleanup, or emitting host output.

This file is not the Fairy C runtime, a stable runtime ABI, a public C header, or a supported-target
artifact. It exists only so generated scalar C can be linked and its canonical trap category can be
observed by the Linux fixture runner.
