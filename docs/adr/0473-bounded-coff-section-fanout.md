# ADR 0473: Bounded COFF section fanout for the compiler candidate

- Status: Accepted
- Date: 2026-10-07

The optimized Fairy compiler candidate now produces an ordinary AMD64 COFF object
with 4,135 sections. The inspector rejects it at the old 4,096-section policy
limit before linking, although that count fits the admitted wire format.

Raise the shared ordinary/bigobj inspection budget to 16,384 sections. This stays
below the ordinary signed 16-bit section-number limit. Retain the 64 MiB input,
262,144-symbol and relocation budgets, range checks, primary-symbol validation
and COMDAT rules. This changes internal admission policy, not public output or
the exact versioned generated-object fixture bytes. ADR 0243's original bound is
superseded only for this count.

Check acceptance at 154, 4,096, the observed 4,135, and 16,384 sections, plus
rejection at 16,385 and existing malformed ordinary/bigobj tests. Validate the
actual candidate's dev and release builds and native borrow-flow cases as unit 40.
No performance improvement is claimed. Package compilation and artifact cost
remain separate work; source semantics must not be removed to fit this old cap.
