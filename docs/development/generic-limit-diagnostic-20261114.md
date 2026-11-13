# Source generic-limit diagnostic — 2026-11-14

The previous remaining-work review missed the later
`candidate-flat-result-forward-call-20261108.md` diagnosis. The original full
helper fixture has 68 generic specializations, exceeding the documented limit
of 64. Neither the limit nor Result lowering needs changing for that input.

Current-source replay of `TestCandidateFlatResultForwardCall` passed in 259.5s:
both low and optimized Fairy-authored producers accepted the minimal and full
baseline inputs; their C products returned 42 under both native profiles.
Both producers refused the unchanged full-helper input with exactly
`phase=3 stage=3 code=4503 start=638`, with no emitted packet. Evidence is in
`.cache/result-forward-current-20261114.log` and
`.cache/flat-result-forward-337870760/source-inputs.json` (409 authored inputs).

The actual CLI defect is in `project_check_status`: every raw 4503 was treated
as an internal invariant before source diagnostic rendering could use its
already-defined E4503 message. Status classification now belongs to the pure
`project_diagnostic_codes.compiler_status` helper. Only generic discovery's
phase 3 / stage 3 / 4503 becomes a source error (exit 3). The same raw code in
linking, publication and other stages retains exit 4. Limits and artifact
formats are unchanged.

The complete `tests/project_diagnostic_codes.fy` file passed as four interpreter
tests and four native Windows tests in an isolated workspace copied from the
current authored files, using `.cache/windows-jobs-20261113/fairyc.exe`.
No desktop interaction or dependency download occurred. This focused check
does not establish whole authored CLI/cache acceptance, which is the next slice.
