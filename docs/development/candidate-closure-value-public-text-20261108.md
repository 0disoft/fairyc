# ADR0551 closure leaf public text

Scope: public type spelling only, on detached `329ebcf5` plus the parent-owned
source-set/interface changes. No executable closure admission or artifact format
change is supplied by this slice.

`module_public_type_text::walk` deliberately renders graph kind 23 as exactly
`fn i32(i32)` only at recursion depth zero and only when `scalar`, `module_index`,
`declaration`, `first` and `second` are all zero. A forged payload or a closure
visited through Option, either Result arm, pointer, borrow, array or slice returns
9. Opaque generic identities retain their existing name-table admission gates;
this change does not admit generic closure transport.

The leaf uses the existing node charge and byte writer: 1 + 11 + 1 = **13** work.
An exact remaining allowance succeeds; one fewer fails with 10 after the node
charge, without returning a partial buffer. Payload validation precedes writing;
an exhausted node allowance still fails with 10 before inspecting its payload.
The existing depth-64, owner, graph/table count and source-count checks remain.
Every feature-free rendering branch and its byte/work accounting is unchanged.

All existing public renderer APIs reach this branch. `render_identified` still
authenticates the complete name table and sources before walking; stale sources
return 12, and mismatched rows return 9. Nominal source authentication is
unchanged. Raw/prepared spelling remains text-only: it does not turn an
unauthenticated source or record into executable/cache/publication authority.

The new authored fixture builds real signatures and owned interfaces from
identity function sources, including direct closure results, parameters and
ordinary struct fields. The Go oracle uses the original frontend, semantic HIR
and `moduleinterface.Extract`; it transports exact authored source bytes with
comments, CRLF and Korean trivia. Three closure spellings and fourteen
feature-free signatures are checked at raw, raw-current, ordinary, prepared and
identified renderer entry points. Positive graphs are never fabricated and
candidate source bodies/imports are never replaced by a test model.

The same real source/interface records also pass through ordinary, identified
and both prepared-source modes of `module_public_interface_extract`. Each full
descriptor is encoded and hashed with the existing public v5 APIs and compared
byte-for-byte with original Go `moduleinterface.Encode/Compute`, including the
closure-bearing struct field. All four extraction entry paths have exact-work
success/one-short failure checks; ordinary/identified extraction still rejects
stale sources with 12. No descriptor is manufactured as a positive input.

Negative probes mutate only detached input records or name facts: all five leaf
payload words, Option, both Result arms, shared/mutable borrow, pointer, array,
nominal slices, Vec's missing identity gate, tuple, nested transport, invalid IDs,
owner/source/table/graph bounds, exact shared work and the 63/64-edge boundary.
These detached probes do not publish or execute the forged inputs.

Validation command:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValue(PublicText(Oracle)?|GraphOracle)$' -count=1 -v -timeout=12m
```

The native harness copies the current authored transitive import closure into a
temporary workspace, checks it, then runs Windows AMD64 low and optimized
(`--release --locked`) executables against the original Go oracle. Its lockfile
and outputs are temporary; the shared candidate lockfile is untouched.

Executed against the frozen parent source-set changes: all three selected Go
tests passed; the authored workspace source check passed; Windows low and
optimized runs both exited 0 with empty stdout/stderr (native test: 40.60 s).
Scoped tracked/untracked whitespace checks also passed. No remaining public-text
or descriptor-extraction consumer gate was observed.

Linux,
whole-candidate self-build, runtime admission, distribution and CI remain batched
or parent-owned gates. No bound increase, API signature change, database change,
runner change, commit, branch operation, CI, push or deployment belongs to this
slice.
