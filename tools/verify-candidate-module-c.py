"""Compile and execute C captured by the private owned-body emitter tests.

The input must be the JSON output of fairyc test --native --json --capture-output,
selecting the matching C emitter tests. The arrays suite selects the five tests
in module_unit_c_arrays.fy plus the scalar and owned unsupported-type rejections;
use --exact identities to include those two tests without their full suites.
Select the matching --suite. This checks generated C, not stage1/stage2.
It does not authenticate when that report's compiler sources were built.
Repeat --report in execution order for focused reruns. The latest outcome for
each test must pass; callers must verify unchanged compiler inputs across reports.
"""

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess


EXPECTED = {
    "module_c_executes_calls_loops_and_lazy_branch_values": 0,
    "module_c_freezes_defer_arguments_before_reassignment_and_return": 0,
    "module_c_emits_closed_integer_widths_and_normalized_literals": 0,
    "module_c_emits_overflow_trap": 101,
    "module_c_emits_divide_by_zero_trap": 102,
}
REJECTION = "module_c_rejects_unsupported_types_and_bad_entry_without_partial_output"
OWNED = {
    "owned_c_replaces_owners_after_rhs_and_mutates_cleanup_receiver": "102,103,",
    "owned_c_drops_active_tagged_payloads_and_nested_fields_in_reverse_order": "5,4,3,100,2,1,",
    "owned_c_moves_returns_and_unwinds_completed_partial_values_on_propagation": "20,1,2,3,9,4,",
    "owned_c_invokes_loop_defer_suffix_before_drops_and_keeps_outer_owner": "20,10,0,21,11,1,100,",
    "owned_c_keeps_provider_nominal_identity_and_shared_foreign_linkage": "30,10,",
}
OWNED_REJECTION = "owned_c_rejects_pending_view_transport_without_partial_output"
REFERENCES = {
    "reference_c_snapshots_root_and_projected_values_before_rhs_mutation": "3,6,",
    "reference_c_mutates_borrowed_resource_without_dropping_the_reference": "10,3,",
    "reference_c_retains_shared_projection_and_generic_reference_returns": "",
    "reference_c_snapshots_indirect_compound_target_before_aliasing_rhs": "5,",
    "reference_c_allows_sequential_mutable_and_shared_scopes": "8,",
}
REFERENCE_PROOF = "reference_c_rejects_missing_duplicate_and_malformed_compound_snapshots"
ARRAYS = {
    "array_c_copies_values_across_calls_and_returns": (0, "1,9,3,6,"),
    "array_c_evaluates_repeated_values_once_and_orders_literal_effects": (0, "4,1,3,4,9,1,"),
    "array_c_preserves_indexed_storage_and_old_values_before_rhs": (0, "77,5,8,8,"),
    "array_c_traps_read_at_extent": (106, ""),
    "array_c_checks_target_bounds_before_rhs_effects": (106, "1,"),
}
VIEWS = {
    "view_c_mutates_owner_and_shares_call_arguments": (0, "7,4,5,"),
    "view_c_selects_original_plain_struct_field_storage": (0, "6,7,"),
    "view_c_executes_generic_scalar_view_specializations": (0, ""),
    "view_c_orders_bounds_and_retains_nested_empty_ranges": (0, "11,22,2,33,"),
    "view_c_traps_reversed_slice_without_unwinding": (106, "1,"),
    "view_c_traps_upper_bound_beyond_view_extent": (106, "2,"),
    "view_c_traps_index_into_empty_end_view": (106, ""),
}
CASTS = {
    "cast_c_preserves_all_admitted_integer_ranges": (0, ""),
    "cast_c_roundtrips_signed32_through_separate_c_functions": (0, "-2147483648,2147483647,42,"),
    "cast_c_projects_operands_and_evaluates_calls_once": (0, "1,7,9,"),
    "cast_c_widened_arithmetic_keeps_checked_overflow": (101, ""),
}
CAST_REJECTION = "cast_c_rejects_other_c_scalars_and_malformed_casts_transactionally"


def run(argv, cwd, env=None, timeout=60):
    process = subprocess.Popen(argv, cwd=cwd, env=env, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE)
    try:
        stdout, stderr = process.communicate(timeout=timeout)
    except subprocess.TimeoutExpired:
        subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"],
                       capture_output=True, timeout=30)
        process.communicate(timeout=30)
        raise RuntimeError(f"timed out: {argv[0]}") from None
    return dict(argv=argv, exit=process.returncode,
                stdout=stdout.decode("utf-8", errors="replace"),
                stderr=stderr.decode("utf-8", errors="replace"))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", required=True, type=Path, action="append")
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--vcvars", required=True, type=Path)
    parser.add_argument("--suite", choices=("scalar", "owned", "references", "arrays", "views", "casts", "all"), default="scalar")
    args = parser.parse_args()
    scalar_cases = {name: (exit_code, "") for name, exit_code in EXPECTED.items()}
    owned_cases = {name: (0, trace) for name, trace in OWNED.items()}
    reference_cases = {name: (0, trace) for name, trace in REFERENCES.items()}
    suites = {"scalar": (scalar_cases, {REJECTION}),
              "owned": (owned_cases, {OWNED_REJECTION}),
              "references": (reference_cases, {REFERENCE_PROOF}),
              "arrays": (ARRAYS, {REJECTION, OWNED_REJECTION}),
              "views": (VIEWS | {"array_c_evaluates_repeated_values_once_and_orders_literal_effects": ARRAYS["array_c_evaluates_repeated_values_once_and_orders_literal_effects"],
                                "owned_c_drops_active_tagged_payloads_and_nested_fields_in_reverse_order": owned_cases["owned_c_drops_active_tagged_payloads_and_nested_fields_in_reverse_order"]},
                        {REJECTION, OWNED_REJECTION}),
              "casts": (CASTS | {"module_c_emits_closed_integer_widths_and_normalized_literals": scalar_cases["module_c_emits_closed_integer_widths_and_normalized_literals"],
                                 "view_c_mutates_owner_and_shares_call_arguments": VIEWS["view_c_mutates_owner_and_shares_call_arguments"]}, {CAST_REJECTION}),
              "all": (scalar_cases | owned_cases | reference_cases | ARRAYS | VIEWS | CASTS,
                      {REJECTION, OWNED_REJECTION, REFERENCE_PROOF, CAST_REJECTION})}
    expected_cases, rejections = suites[args.suite]
    count = len(expected_cases) + len(rejections)
    reports = []
    outcomes = {}
    configuration = None
    for path in args.report:
        raw = path.read_bytes()
        report = json.loads(raw)
        assert report["engine"] == "native"
        assert report["count"] == len(report["tests"])
        assert report["count"] == report["passed"] + report["failed"]
        observed_passes = sum(test["outcome"] == "passed" for test in report["tests"])
        assert observed_passes == report["passed"]
        current = (report["target"], report["profile"])
        assert configuration is None or current == configuration
        configuration = current
        identities = set()
        for test in report["tests"]:
            assert test["identity"] not in identities
            identities.add(test["identity"])
            outcomes[test["identity"]] = test
        reports.append(dict(path=str(path.resolve()), sha256=hashlib.sha256(raw).hexdigest(),
                            count=report["count"], passed=report["passed"], failed=report["failed"]))
    assert len(outcomes) == count
    cases = {}
    for test in outcomes.values():
        assert test["outcome"] == "passed" and not test["stderrBase64"]
        identity = test["identity"]
        prefix = next((prefix for prefix in ("tests::module_unit_c_emit::",
                       "tests::module_unit_c_owned::", "tests::module_unit_c_references::",
                       "tests::module_unit_c_arrays::", "tests::module_unit_c_views::",
                       "tests::module_unit_c_casts::")
                       if identity.startswith(prefix)), None)
        assert prefix is not None
        name = identity.removeprefix(prefix)
        assert name not in cases
        cases[name] = base64.b64decode(test["stdoutBase64"], validate=True)
    assert set(cases) == set(expected_cases) | rejections
    assert all(not cases[name] for name in rejections)
    assert args.vcvars.is_file()
    # cmd has its own quote grammar; Python's list2cmdline escaping is not valid
    # for a nested quoted batch path. Only this fixed, validated setup path uses it.
    vcvars = str(args.vcvars.resolve())
    assert not any(character in vcvars for character in '\r\n"&|<>^%!')
    setup = run(f'cmd.exe /d /s /c ""{vcvars}" x64 >nul && set"', Path.cwd())
    assert setup["exit"] == 0, setup["stderr"]
    environment = {key.upper(): value for key, value in os.environ.items()}
    for line in setup["stdout"].splitlines():
        if "=" in line and not line.startswith("="):
            key, value = line.split("=", 1)
            environment[key.upper()] = value
    compiler = next((Path(folder) / "cl.exe" for folder in environment["PATH"].split(";")
                     if (Path(folder) / "cl.exe").is_file()), None)
    assert compiler is not None
    args.output.mkdir(parents=True, exist_ok=True)
    output = args.output.resolve()
    shim = output / "trap.c"
    shim.write_text("#include <stdint.h>\n#include <stdlib.h>\n#include <stdio.h>\n"
                    "void observe_cleanup(int32_t n){printf(\"%d,\",(int)n);}\n"
                    "int produce_c_int(int32_t n){return (int)n;}\n"
                    "int consume_c_int(int n){printf(\"%d,\",n);return n;}\n"
                    "_Noreturn void fy_runtime_trap(uint32_t category)"
                    "{exit(100+(int)category);}\n", encoding="utf-8")
    executions = []
    for name, (expected, trace) in expected_cases.items():
        source = output / f"{name}.c"
        assert cases[name].startswith((b"/* fairyc.candidate-module-scalar-c/v1 */\n", b"/* fairyc.candidate-module-owned-c/v1 */\n"))
        source.write_bytes(cases[name])
        for profile, flag in (("low", "/Od"), ("optimized", "/O2")):
            binary = output / f"{name}-{profile}.exe"
            build = run([str(compiler), "/nologo", "/std:c11", "/TC", flag,
                         str(source), str(shim), f"/Fe:{binary}", "/link",
                         "/INCREMENTAL:NO"], output, environment)
            assert build["exit"] == 0, build
            result = run([str(binary)], output)
            assert result["exit"] == expected, result
            assert result["stdout"] == trace and not result["stderr"], result
            executions.append(dict(name=name, profile=profile, expectedExit=expected, expectedStdout=trace,
                                   sourceSHA256=hashlib.sha256(cases[name]).hexdigest(),
                                   build=build, run=result))
            print(f"PASS {profile}: {name} (exit {expected})", flush=True)
    receipt = dict(schema="fairyc.candidate-module-c-execution/v1", passed=True, suite=args.suite,
                   nativeReport=reports[0], nativeReruns=reports[1:],
                   compiler=str(compiler), executions=executions,
                   trapShim="Standalone test shim maps runtime category to exit 100+category",
                   limitation="Report source/compiler provenance must be supplied by the caller")
    (output / "execution.json").write_text(json.dumps(receipt, indent=2) + "\n",
                                           encoding="utf-8")


if __name__ == "__main__":
    main()
