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

## Native stdin capture

`linux_test_capture` opens a captured process with fd 3 trap control and one of
three stdin policies: mode 0 inherits caller input without reading it, mode 1
provides an already-closed pipe (EOF), and mode 2 opens the absolute private replay
file afresh. Replay opens use CLOEXEC/NOFOLLOW/NONBLOCK, require a regular file no
larger than 65,536 bytes, and move low descriptors above fd 3 before spawn actions.
Input is duplicated to fd 0 before stdout, stderr and control redirection, then
all remaining descriptors from 4 upward close in the child. Parent input handles
close on success and failure. Source file admission remains with the Fairy caller.

The wrapper shares `linux_run_capture.collect_owned`, its receipt validation,
process-group cleanup and 65,536-byte per-stream capture. Test deadlines admit
1..60,000 ms; the shared collector still serves ordinary run unchanged.
Windows low/optimized POSIX substitutes check all three modes, closed standard
descriptors, open/stat/type/size failures and descriptor cleanup. Both-target Fairy
emission and host wrapper execution check mode/path transport and failed-open
ownership. Native binary stdin/replay/EOF and symlink probes are authored but
skipped on Windows. This adapter alone does not connect test command dispatch.
