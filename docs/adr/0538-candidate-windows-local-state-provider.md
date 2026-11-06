# ADR 0538: Fairy Windows local-state persistence provider

- Status: Accepted for the existing local-store and Fairy graph layout family
- Date: 2026-11-07

`module_state_local_windows` composes the existing rooted file API, ADR 0536
authority/leases, ADR 0535 record codec and ADR 0537 mutation/schema admission.
Store is sealed and retains the actual directory, stable authority and explicit
full-state policy. Creation and reopen remain separate; neither collision nor
missing/corrupt existing state triggers reset or repair. Invalid layout, policy
or work refuses before root/authority I/O.

Every snapshot, compare-and-publish and layout publication takes an independent
nonblocking one-byte OS lease and rereads the complete bounded current record.
Header reconstruction is immediately followed by complete checksum/content
admission. Snapshot compares expected layout and returns owned state. Mutation
validation/sorting precedes the lease. Layout/version comparison and complete
candidate publication share the same lease. No caller callback, transaction
body, snapshot retry or conflict retry occurs inside the authority boundary.

The provider implements the three port authorities: snapshot, complete overlay
CAS and identical/append-only layout CAS. No-op and identical-layout requests
still compare the current version/layout while preserving version and file
bytes. Nonempty writes and genuine layout appends advance exactly once;
version exhaustion, incompatible schema and all bounds refuse publication.
Observed layout/version remain available with compare/backend errors, together
with the original rooted operation and OS code.

Publication exclusively creates a same-root `.tmp-<time>-<slot>` file, writes
and syncs its complete contents, then performs the existing handle-relative
rename as the commit decision. This intentionally retains the synced staging
handle through rename instead of reopening a closed pathname. Staging cleanup
owns only its created object; failed publication cannot delete a pre-existing
collision. The bounded per-time collision loop is not a CAS retry. There is
no fallible operation after successful rename that can conceal known success;
the published handle closes through resource cleanup. Abandoned crash stages
are ignored and never implicitly erased.

The trusted existing local-directory/ancestor model and process-restart claim
remain those of ADR 0500. Power-loss durability, hostile directory owners,
hard-link/rollback attacks and network filesystems are not certified. Windows
may prevent directory rename while the stable child authority is open; that
exclusion preserves the original authority. All allowed operations continue
to use the retained root and fixed flat authority names.

This closes actual Fairy local storage/CAS for the admitted family. Compiled
graph snapshot/run/admission/CAS orchestration, cancellation, ordinary command
selection, Linux and final acceptance remain separately required.
Evidence: `docs/development/candidate-windows-local-state-provider-20261107.md`.
