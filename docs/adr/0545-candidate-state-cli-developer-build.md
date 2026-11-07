# ADR 0545: Repeatable private Windows state developer build

- Status: Accepted for preparation and the existing explicit native-graph command
- Date: 2026-11-07

`tools/prepare-candidate-state-cli.py` prepares a fresh isolated workspace under
repository `.cache` from exact authored sources and `tools/dev/fairyc-state/main.fy`.
It preserves the example manifest and embedded-SDK selection, copies no lock,
rewrites no source and generates the two existing pinned graph packs next to the
intended executable. It records explicit bootstrap/input/copy/pack hashes and
alternative low/optimized build argv/cwd. Preparation never executes a compiler.
Existing outputs, path escape/traversal and observed links/reparse points are
refused; input changes prevent success publication. Partial output is retained
for inspection rather than deleted or reused. Independent profile images use
different fresh workspaces.

Actual native integration exposed a missing Windows connection in the existing
`nativesupport.Options.CandidateHost` path: Linux private adapters were selected,
but Windows private names reached the closed standard-library resolver. Windows
opt-in now separates only complete valid private symbol spellings, delegates
admission to the existing `candidatehost.WindowsSource` closed allowlist, and
appends that exact source after the existing standard/vector/trap runtime.
Unknown operations, bare prefixes, injected declaration text and wrong-target
symbols fail; there is no arbitrary system-import bypass or second allowlist.
Without opt-in, refusal remains. Ordinary support bytes and caller symbol
storage remain unchanged; Linux feature ordering/selection is unchanged.

The existing bootstrap workspace build can therefore compile the actual authored
developer entry with the real private host runtime. This is compilation tooling,
not a Go executor/provider substitute for product transactions. The resulting
Fairy binary owns source admission, native graph execution, reporting and local
one-attempt state publication. Neither profile implies public CLI promotion,
installed distribution support, clean-machine acceptance or a completed queue.

Evidence: `docs/development/candidate-state-cli-preparation-20261107.md`.
