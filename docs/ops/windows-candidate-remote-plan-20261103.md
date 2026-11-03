# Windows Fairy remote source plans, 2026-11-03

Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231. Native Linux remains deferred
at the maintainer's request; this laptop has no usable WSL.

## Source plan parsing

`project_remote_plan.fy` owns validated source pins, with decoded strings held in an owned JSON
tree. It requires exact root/package fields, valid UTF-8, the source schema, 1-32 entries and
at most 65,536 input bytes. Duplicate decoded keys, paths and UIDs reject. Existing manifest
predicates validate portable vendor names, UIDs and exact SemVer; SHA-256 is lowercase hex.
Pins are returned in byte-lexicographic path order. The existing shared work meter still applies.

The separate `project_source_url.fy` checks absolute HTTPS, no userinfo/query/fragment,
nonempty host/path, ports 1-65535, valid percent escapes and IPv6 literals/zones. URL spelling
is retained, including accepted uppercase schemes and encoded path bytes. This is a private
preparation-policy validator, not a new general URL SDK API.

`TestCandidateSourcePlanParity` compiles the real authored import closure into partitioned C
and links/runs MSVC low and optimized images. For each source document, it compares reject/accept
and all five decoded, sorted fields with `packagesource.Parse`. Inputs include malformed,
duplicate/escaped/unknown/missing keys, invalid UTF-8, surrogate replacement, invalid pin
coordinates, duplicate paths/UIDs, 32/33 entries, 65,536/65,537 bytes, HTTPS policy examples,
all 256 host characters and all 256 percent escapes in both host and path positions.

Executed and passed:

```powershell
go test -mod=readonly -count=1 -timeout 5m ./internal/cli -run '^TestCandidateSourcePlanParity$'
```

The native comparison passed in 57.784 seconds. Initial local compile errors from C-style
numeric cast syntax were corrected to Fairy `as` expressions before the successful run.

Parser, URL validator and oracle: `4e31287f`.

## Workspace binding

`project_remote_bind.fy` consumes a parsed plan and the existing validated Fairy manifest.
It requires each pin to be an explicit workspace member with a non-root UID, checks exact
versions for dependencies on that path, rejects directory overlap with root source/other
members, and reserves `build/package-sources` against root source and every member. Portable
manifest paths are ASCII; case folding matches Go `Bind` for this admitted domain. Similar
prefixes without a directory boundary remain distinct. No member or cache files are opened.

`TestCandidateRemoteSourceBinding` feeds the same 24 TOML roots through Go and Fairy manifest
parsers and compares binding with `packagesource.Bind`. Both low and optimized pass 7 accepted
and 17 rejected roots, plus 24 already-exhausted shared-work-meter controls. Controls include
missing/empty membership, optional dependencies, unrelated dependency versions, exact version
mismatch, root UID, source/other-member/cache ancestry, case collisions and harmless prefixes.
This is metadata parity within the existing bounded work policy, not filesystem admission.

Executed and passed in 11.347 seconds:

```powershell
go test -mod=readonly -count=1 -timeout 3m -v ./internal/cli -run '^TestCandidateRemoteSourceBinding$'
```

Also passed: `go test -mod=readonly -count=1 ./internal/packagesource`, scoped
`go vet -mod=readonly ./internal/packagesource ./internal/cli`, Go formatting and Git diff checks.

Skipped: Linux native execution (batched), full compiler CLI acceptance (no command is wired
yet), network/cache/archive restoration (later work), full repository/release suites (outside
this private parser slice). URL comparison uses the installed Go toolchain; no Linux/toolchain
matrix is implied. Public CLI/API, SDK ABI, DB, dependencies, runners and repository hygiene
are unchanged. No push or deployment was performed.
