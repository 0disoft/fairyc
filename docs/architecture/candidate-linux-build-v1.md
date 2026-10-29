# Private Linux candidate workspace build v1

The initial Linux backend is `tools/candidate/fairyc-candidate-linux-build`, a Go
bootstrap companion. It contains no Fairy frontend. It consumes a fresh bounded
`FYCPARTS1` packet, requires the `x86_64-linux-gnu` target marker, selects the closed
Linux runtime, compiles separate C objects, and publishes the final executable.
It does not invoke MSVC, another Fairy compiler, cached packets or source discovery.

`internal/candidatebuild/linux.go` owns packet/runtime admission and publication;
`internal/cdriver/linux_project.go` owns native GCC/Clang compilation. Existing Linux
toolchain admission applies: native Linux amd64/glibc, GCC >=13 or Clang >=18.
Windows runtime symbols and unknown SDK/vector/host symbols fail selection.

The caller supplies a new private staging directory containing only `packet.bin`.
Sources use exclusive creation; objects are bounded and inspected as ELF64 x86-64
relocatables. A response file lists only generated ASCII object basenames. The linked
ELF must be executable/PIE with a nonzero entry and executable load segment. Publication
uses existing rooted atomic replacement and executable mode. Failure before publication
preserves the prior image. C input/runtime/part/count/image bounds remain explicit.

Every compiler invocation and link has a one-minute deadline from the companion;
output is bounded to 64 KiB. The companion's caller owns the outer capture/process
group, overall deadline and staging cleanup. Compiler children inherit that group;
they must not create independent groups. GNU temporary files use the owned stage.
This is a bootstrap backend, not a claim that the Linux native driver is Fairy-owned.

Runtime selection, host/deadline admission and invalid ELF rejection are checked on
the Windows development host. Linux-native packet compilation/publication and
compile/link failure preservation tests are authored and skipped on Windows.
CLI connection and caller-owned staging are implemented in the following slice.
