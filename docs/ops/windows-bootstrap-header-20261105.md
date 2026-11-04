# Bounded Go bootstrap header, 2026-11-05

The complete authored CLI gained structural edit preview/application. Its Go
bootstrap C header reached 2,110,183 bytes, exceeding the existing 2 MiB MSVC
project-input limit by 13,031 bytes. Both profiles refused before invoking C.
This was an input-bound failure, not a compiler timeout or native CLI result.

Partitioned private function prototypes now omit definition-local parameter
names. C types, calling signatures, function definitions and metadata are
unchanged. Zero-argument prototypes retain `void`. Single-unit emission keeps its
existing bytes. The shared header is 1,919,958 bytes: 190,225 fewer bytes (9.01%),
with 177,194 bytes below the same 2 MiB limit. Runtime is 153,115 bytes, with 23
source units and a largest unit of 2,097,062 bytes. No transport limit or timeout
was increased; no runtime-speed claim follows from the size observation.

Validation:

- `go test -mod=readonly -count=1 -timeout 8m ./internal/cbackend -run
  'Test(Partitioned|SmallBuild|EmissionTransport|EmitDeduplicatesCompatibleForeign)'`
  passed in 35.849s. The partitioned fixtures execute both MSVC profiles, including
  callbacks and shared inline definitions. Metadata, deterministic output and
  small/single-unit identity checks pass.
- `go vet -mod=readonly ./internal/cbackend` and `git diff --check` passed.
- Full current-CLI input preparation reports the sizes above. The invocation
  selected no native subtests, so its passing preparation is not full CLI proof.
  Complete native replay remains a separate gate.

The Fairy emitter and eventual consecutive self-build equality remain separate
owners. Native Linux stays batched. Public language/SDK ABI, dependencies, DB,
repository hygiene, CI and runner settings did not change. No push or release.
