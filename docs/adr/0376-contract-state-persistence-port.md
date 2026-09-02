# ADR 0376: Contract-state persistence compare-and-publish port

## Status

Accepted as the vendor-neutral persistence boundary for the contract-state oracle. No database, chain storage engine, replication system, migration product, or deployment provider is selected.

## Decision

A persistence adapter must implement three explicit authorities: snapshot one exact layout generation, compare-and-publish one complete overlay against an exact layout hash and store version, and publish an identical or append-only layout against the same version guard.

The port deliberately does not expose an arbitrary database transaction callback. The adapter must make the compare condition and atomic publication boundary visible so isolation drift cannot be hidden behind provider-specific transaction defaults.

A snapshot contains the admitted layout hash, committed store version, and deep-copied physical key/value entries. A write publication contains canonical unique physical-key mutations. Stale versions return `ErrConflict`; wrong layouts return `ErrLayoutMismatch`. Either result publishes nothing.

No-op publications preserve the version. A successful non-empty overlay advances the version exactly once regardless of write count. Accepted append-only layout publication also advances it exactly once, invalidating transactions opened on the prior generation.

## Minimum adapter primitive

A production adapter needs the semantic equivalent of one atomic conditional operation:

`publish(new_state) if layout_hash == expected_layout && store_version == expected_version`

The condition and all value mutations must share one durable commit decision. A serializable database transaction can implement this only if the adapter explicitly checks the version/layout row inside the same transaction and updates that authority in the same commit. A conditional batch/compare-and-swap store may implement it directly. We do not require a specific SQL isolation-level name because provider meanings differ; the observable conditional publish is the contract.

An adapter must not report a generic failure after it knows durable publication succeeded. If a provider can produce an ambiguous commit result, its adapter needs a provider-specific recovery/receipt protocol before it can satisfy this port.

## Reference adapter

`ReferencePersistence` is a deterministic process-local implementation used only as an executable oracle for the port. It copies state before mutation and swaps the complete candidate state only after every guard and injected pre-publication fault point passes.

Fault injection exists before compare and immediately before publish. Both prove that backend errors leave layout, version, and values unchanged. The reference adapter also proves canonical ordering, defensive copies, deletion, stale writers, layout mismatch, and append-only layout publication.

## Failure mapping

Caller rejection, language trap, verifier failure, and execution limits occur before this persistence authority is invoked and therefore have no durable write. Persistence conflict maps to `ErrConflict`. Layout drift maps to `ErrLayoutMismatch`. Adapter/backend errors remain explicit errors and must not be converted to successful or partial receipts.

Automatic retry is not part of this boundary. The caller-visible retry contract remains authoritative; a new attempt takes a fresh snapshot and submits a new compare-and-publish request.

## Separation

The compiler and Fairy source model do not name PostgreSQL, SQLite, cloud databases, chains, transaction isolation brands, or migration tools. Native runtime lowering may consume this contract later but cannot weaken it or bypass the same compare-and-publish authority.
