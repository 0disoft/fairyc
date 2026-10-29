# Private Linux candidate workspace test v1

## Portable result boundary

`native_test_result` retains only canonical trap category and binary stdout/stderr.
The host executor validates OS status and its private control channel before
constructing that result. `project_test_options.test_native_report` takes an
explicit target identity and shares the existing text/JSON v1-v4, base64, counts,
selection order, lock and source-location formatter. The Windows wrapper keeps
its existing target and output bytes; semantic reports retain null target/profile.
Windows test execution converts each validated capture before retaining it.

Both-target emission and low/optimized host execution check mixed pass/fail
outcomes, exact binary base64 streams and Windows/Linux JSON target identities.
This result boundary alone does not connect Linux test dispatch or native stdin.
The Linux host adapter and command integration are recorded below as they land.
