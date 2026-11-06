# Candidate state source operands

Private producer support in
`examples/lexer_candidate/src/module_state_graph_source_operands.fy`. This module
constructs raw facts from the parent's owned parser arena. It does not publish a
graph proof, Complete, artifact reuse permission or execution authority.

## Stable API

Aliases: `syntax` is `syntax`, `m` is `module_state_graph_model`, `state` is
`module_state_layout`, `context` is `module_state_graph_source_context`, `bytes`
is `std::bytes`, and `v` is `std::vec`.

```fairy
pub fn Result<bytes::Buffer,u32> string(ref syntax::Parsed p,usize expr,u32 literalKind,ref mut usize work)
pub fn u32 codec(ref syntax::Parsed p,usize typeRoot)
pub fn Result<usize,u32> u32_literal(ref syntax::Parsed p,usize expr,ref mut usize work)
pub fn Result<usize,u32> ungroup(ref syntax::Parsed p,usize expr,ref mut usize work)
pub fn Result<usize,u32> lookup(ref syntax::Parsed p,ref v::Vec<context::Binding> env,ref u8[] name,ref mut usize work)
pub fn Result<m::Value,u32> reference(ref syntax::Parsed p,usize expr,ref v::Vec<context::Binding> env,ref mut usize work)
pub fn Result<m::Operand,u32> operand(ref syntax::Parsed p,usize expr,u32 codec,ref v::Vec<context::Binding> env,ref mut bytes::Buffer data,ref mut usize work)
pub fn Result<usize,u32> field(ref syntax::Parsed p,usize expr,ref state::Plan layouts,ref mut usize work)
pub fn Result<bytes::Buffer,u32> call_name(ref syntax::Parsed p,usize expr,ref mut usize work)
pub fn Result<m::Operation,u32> access(ref syntax::Parsed p,usize expr,ref state::Plan layouts,ref v::Vec<context::Binding> env,ref mut context::Counters ids,ref mut bytes::Buffer data,ref mut usize work)
```

`lookup` returns a one-based environment row, or zero for a missing name.
`field` returns a zero-based field index in layout zero. `reference` returns the
value ID and codec, rather than a transport literal. Dynamic `operand` results
use `m::input`, with no literal codec/range metadata. Read access IDs are allocated
only through `context::value`; the environment and parser remain unchanged.

## Policy

Source behavior follows `internal/compiler/state_graph_operands.go` and decimal
view bounds in `internal/compiler/state_graph_borrows.go`:

- The seven codecs are bool, u32, u64, bytes20, bytes32, bytes and text.
  `codec` accepts only a plain named type and returns zero for other forms.
- Decimal u32/u64 literals multiply into four/eight canonical big-endian bytes,
  rejecting carry past the width. This includes `4294967295` and
  `18446744073709551615` without narrowing through host integers. Signs, radix
  prefixes, separators, floating tokens and grouped literals are rejected.
- Ordinary available references may be grouped. `storage::found` requires one
  raw local path argument; `storage::found((local))` remains closed. Presence is
  obtained from the binding's existing found ID, with bool codec.
- Literal strings use scanner-cooked decoding. Text requires the text literal
  token; bytes and fixed bytes require a byte-string token. Canonical validation
  checks fixed widths, bool representation and UTF-8, including empty dynamic
  values and embedded binary bytes.
- `call_name` accepts only raw call kind 6 with raw path callee kind 2. Grouped
  callees, generic specializations and labeled arguments are rejected. Names
  are normalized by `module_registry`. Argument entries are checked in one
  charged traversal, with no arbitrary argument-count limit. Callers validate
  each operation's argument count.
- `access` constructs the six cell/map read/write/delete forms. A map key and
  write value may refer to a local only when its codec matches the field.
  A read receives separate value/found IDs, capability from `Counters`, and
  cost one. Borrow/helper construction belongs to other producer modules.
- `lookup` preserves Go's first-match ordering. Environment names are spans in
  the parent's retained source. Parsed paths are normalized before comparing
  against those identifier spans, so path trivia does not change identity.

All work is charged to the existing 16 MiB phase ledger. Errors remain private:
`1001` unsupported or malformed shape, `1002` byte/work/ID bounds, and `1003`
allocation failure. Literal storage is bounded by the graph's 1 MiB byte limit.
Integer charge multiplication checks its bound before multiplication. Inputs
are the source producer's trusted, parser-created arenas and layout; these
helpers are not an independent validator for arbitrary forged parser arenas.

The parent abandons its raw construction on error. The mutable output buffer,
counters and work ledger are not rollback transactions; a failed access may
have consumed work, allocated an earlier read ID, or appended a valid key before
a later operand failed. No partially constructed proof is returned here.

## Previous worker validation receipt

An isolated scratch workspace under `.cache/state-source-operands-worker`
contains the actual operand module and its authored import closure, plus a
minimal main. Neither the full compiler workspace nor shared source files were
edited for this check.

- Existing Go bootstrap workspace `check`: passed, 66 source files and
  939 MIR functions. The initial check found that `ungroup` needed an explicit
  fallback return after `while(true)`; this was corrected before the passing run.
- Current Go source check via a temporary `cli.RunWithInput` runner:
  `go run -mod=readonly .cache/state-source-operands-worker/check.go` passed,
  66 source files, 939 MIR functions, lock unchanged. Log:
  `.cache/state-source-operands-worker/check-current-go.log`.
- Direct `experimental check-syntax`/`check-semantics` invocations using a
  repository-relative filename failed module/source-root path matching. These
  are not counted as passed; workspace checks above used the correct manifest
  source root.

The preceding isolated checks are historical worker results. Current Windows
low/optimized execution and independent Go comparison are recorded below. Linux
execution and allocation-failure injection remain deferred. No new Go tests,
shared-context edits, source-of-truth contract changes, public product API, DB,
dependencies, runners, stage, commit or push are part of this worker's scope.

## Integrated authored-source validation, 2026-11-07

The parent independently compared the actual source producer and restored
root/helper program with Go: 68 authored cases / 136 Windows low/optimized
processes, package 24.792s. Retained source and artifact regressions bring this
slice to 374 native processes and eight ownership refusals. Fresh bootstrap
`check` and `check --locked` pass 716 files / 8,917 MIR functions; scoped vet
and source hygiene pass. The separate operation-limit suite is not included.

Exact commands, work/ownership controls, failed-attempt accounting and full
integration records are in
[the source receipt](candidate-state-graph-source-20261107.md#integrated-authored-source-validation-2026-11-07).
The earlier module-specific checks above remain historical evidence.
Linux execution and allocator-failure injection remain deferred; ordinary
Complete/package/cache/command/execution admission remains separate.
