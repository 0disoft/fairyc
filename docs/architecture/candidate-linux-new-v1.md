# Private Linux candidate workspace new v1

## Shared starter bytes

`project_starter.body` owns the five deterministic starter files, with explicit
name, UID and bundled language inputs. The Windows `file_cli` delegates body
construction and preserves its own host creation/rollback logic. Paths remain
`.gitignore`, `README.md`, `fairy.toml`, `src/main.fy` and `tests/main.fy`.
No lock is created; the first ordinary workspace command owns lock publication.

`TestCandidateStarterTemplates` compares all five file bodies byte-for-byte with
the reference Go scaffold, emits both target profiles and executes the checks in
low/optimized host profiles. The candidate source workspace passes both `check`
and `check --locked`. The Linux adapter and dispatch are documented as they land.
