# Complete-project field capacity

A diagnostic-only emitter splits metadata capacity failures into layout, field,
callable, parameter and work classes. The full dispatcher replay reports 900002
at `project_toml_values.fy`, proving the former 1002 is the merged field-table
bound, rather than a work or callable failure. Diagnostic edits exist only in
ignored scratch sources, not in the admitted compiler.

The merged field-table limit is now 8192; the independently checked unit limit
remains 2048. Type, layout, callable, work, source and output bounds are unchanged.
The maintained boundary testcase links sixteen checked units containing 8192
fields, validates both body and cleanup field tables, then adds a seventeenth
unit and rejects 8193 fields without a program. Go-native and candidate C
development/release pass with identical complete output. Root check and locked
check pass 450 source files / 5822 MIR functions.

Receipt: [evidence](evidence/candidate-project-fields-20261014.json).
Next: replay the whole dispatcher with the current field-aware emitter and
connect the prepared identified build/run path. This boundary test does not
claim complete C emission, native CLI execution or fixed-point promotion.
Cache, product stack/deadlines, Linux/full suite and same-head CI remain open.
Public CLI/SDK ABI, Go, DB and CI are unchanged; no push.
