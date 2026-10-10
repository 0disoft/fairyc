"""Execute private owned SDK C against the repository's real Windows runtime."""
import argparse
import base64
import hashlib
import importlib.util
import json
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("candidate_c", ROOT / "tools/verify-candidate-module-c.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)

CASES = {
    "sdk_c_executes_process_arguments_through_exact_result_bridge":
        (["stage0_windows_process_args.c"], ["abc", "λ"], ""),
    "sdk_c_executes_arena_factory_switch_and_owned_cleanup":
        (["stage0_system_heap.c", "stage0_arena.c"], [], ""),
    "sdk_c_executes_console_numbers_bytes_and_record_result_bridges":
        (["stage0_windows_console.c"], [], "42|-7|9|-9223372036854775808|λ\n"),
    "sdk_c_executes_stdin_and_stderr_byte_results":
        (["stage0_windows_console.c"], [], "A\0λZ"),
}
PROOF = "sdk_c_plan_respects_work_budget_before_authority_or_output"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", type=Path, action="append", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--vcvars", type=Path, required=True)
    parser.add_argument("--case", choices=CASES, action="append")
    args = parser.parse_args()
    outcomes = {}
    reports = []
    for path in args.report:
        raw = path.read_bytes()
        report = json.loads(raw)
        assert report["engine"] == "native" and report["target"] == "x86_64-windows-msvc"
        for test in report["tests"]:
            prefix = "tests::module_unit_c_sdk::"
            if test["identity"].startswith(prefix):
                outcomes[test["identity"].removeprefix(prefix)] = test
        reports.append(dict(path=str(path), sha256=hashlib.sha256(raw).hexdigest()))
    assert set(outcomes) == set(CASES) | {PROOF}
    assert all(test["outcome"] == "passed" for test in outcomes.values()), outcomes
    assert not base64.b64decode(outcomes[PROOF]["stdoutBase64"], validate=True)
    vcvars = str(args.vcvars.resolve())
    assert args.vcvars.is_file() and not any(c in vcvars for c in '\r\n"&|<>^%!')
    setup = driver.run(f'cmd.exe /d /s /c ""{vcvars}" x64 >nul && set"', ROOT)
    assert setup["exit"] == 0, setup
    environment = {key.upper(): value for key, value in os.environ.items()}
    for line in setup["stdout"].splitlines():
        if "=" in line and not line.startswith("="):
            key, value = line.split("=", 1)
            environment[key.upper()] = value
    compiler = next(Path(folder) / "cl.exe" for folder in environment["PATH"].split(";")
                    if (Path(folder) / "cl.exe").is_file())
    args.output.mkdir(parents=True, exist_ok=True)
    output = args.output.resolve()
    trap = output / "trap.c"
    trap.write_text("#include <stdint.h>\n#include <stdlib.h>\n"
                    "_Noreturn void fy_runtime_trap(uint32_t category){exit(100+(int)category);}\n",
                    encoding="utf-8")
    executions = []
    for name in dict.fromkeys(args.case or CASES):
        components, argv, expected = CASES[name]
        source = base64.b64decode(outcomes[name]["stdoutBase64"], validate=True)
        assert source.startswith(b"/* fairyc.candidate-module-owned-c/v1 */\n")
        assert not base64.b64decode(outcomes[name]["stderrBase64"], validate=True)
        generated = output / f"{name}.c"
        generated.write_bytes(source)
        inputs = [ROOT / "internal/cstdlib" / component for component in components]
        # Arena shares the existing SystemHeap component and its status/type
        # definitions in one translation unit, as the normal runtime builder does.
        runtime = output / f"{name}-runtime.c"
        arena = "stage0_arena.c" in components
        instrumentation = b""
        if arena:
            instrumentation = (
                b"#undef fy_std_arena_initialize_v0\n#undef fy_std_arena_drop_v0\n"
                b"static unsigned initialized,dropped,invalid_status;\n"
                b"uint32_t fy_std_arena_initialize_v0(struct fy_std_arena_v0 *arena){"
                b"uint32_t status=fy_test_actual_arena_initialize_v0(arena);"
                b"++initialized;if(status)++invalid_status;return status;}\n"
                b"uint32_t fy_std_arena_drop_v0(struct fy_std_arena_v0 *arena){"
                b"uint32_t status=fy_test_actual_arena_drop_v0(arena);"
                b"++dropped;if(status)++invalid_status;return status;}\n"
                b"extern int32_t fy_candidate_entry(void);\n"
                b"int main(void){int32_t result=fy_candidate_entry();"
                b"return result?result:(initialized==2&&dropped==2&&!invalid_status?0:90);}\n")
        runtime.write_bytes(b"#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)\n" +
                            (b"#define fy_std_arena_initialize_v0 fy_test_actual_arena_initialize_v0\n"
                             b"#define fy_std_arena_drop_v0 fy_test_actual_arena_drop_v0\n" if arena else b"") +
                            b"\n".join(path.read_bytes() for path in inputs) + b"\n" + instrumentation)
        for profile, flag in (("low", "/Od"), ("optimized", "/O2")):
            binary = output / f"{name}-{profile}.exe"
            build = driver.run([str(compiler), "/nologo", "/std:c11", "/TC", flag,
                                *(["/DFY_CANDIDATE_NO_MAIN"] if arena else []),
                                str(generated), str(runtime), str(trap), f"/Fe:{binary}",
                                "/link", "/INCREMENTAL:NO"], output, environment)
            assert build["exit"] == 0, build
            reading = name == "sdk_c_executes_stdin_and_stderr_byte_results"
            stdin = b"A\0\xce\xbbZ" if reading else None
            expected_stderr = "42|-7|9|-9223372036854775808|true|λ\n" if reading else ""
            run = driver.run([str(binary)] + argv, output, stdin=stdin)
            assert run["exit"] == 0 and run["stdout"] == expected and run["stderr"] == expected_stderr, run
            executions.append(dict(name=name, profile=profile, sourceSHA256=hashlib.sha256(source).hexdigest(),
                                   stdinBase64=base64.b64encode(stdin).decode('ascii') if stdin is not None else None,
                                   arenaCounts={"initialize": 2, "drop": 2, "nonzeroStatus": 0} if arena else None,
                                   runtimeInputs=[dict(path=str(path.relative_to(ROOT)),
                                                      sha256=hashlib.sha256(path.read_bytes()).hexdigest())
                                                  for path in inputs], build=build, run=run))
    receipt = dict(schema="fairyc.candidate-sdk-c-execution/v1", passed=True,
                   reports=reports, executions=executions, compiler=str(compiler),
                   source="Private Fairy emitter; actual repository C runtime components, no SDK stand-ins")
    (output / "receipt.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"SDK C execution passed: {len(executions)} runs")


if __name__ == "__main__":
    main()
