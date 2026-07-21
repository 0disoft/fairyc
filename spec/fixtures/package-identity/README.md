# PackageInstanceId fixtures

`manifest-v1.json` freezes the canonical byte encoding, full SHA-256 text identity, component
boundaries, and feature-set rejection behavior defined by `docs/package/package-instance-id-v1.md`.
The fixture inputs are already-resolved opaque identity components. They are not public package
coordinates, dependency requirements, source-download instructions, or a lockfile.

`internal/packageidentityfixture` decodes the manifest strictly and compares every passing vector
against exact bytes and the complete lowercase digest. Reject cases assert stable validation reasons
rather than compiler diagnostic prose. The ordinary package-graph runner remains responsible for
identity equality, visibility, imports, and source roots.
