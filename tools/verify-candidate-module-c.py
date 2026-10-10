"""Compile and execute C captured by the private owned-body emitter tests.

The input must be the JSON output of fairyc test --native --capture-output
--file tests/module_unit_c_emit.fy. This checks generated C, not stage1/stage2.
It does not authenticate when that report's compiler sources were built.
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
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--vcvars", required=True, type=Path)
    args = parser.parse_args()
    report_bytes = args.report.read_bytes()
    report = json.loads(report_bytes)
    assert report["engine"] == "native" and report["count"] == 6
    assert report["passed"] == 6 and report["failed"] == 0
    cases = {}
    for test in report["tests"]:
        assert test["outcome"] == "passed" and not test["stderrBase64"]
        name = test["identity"].removeprefix("tests::module_unit_c_emit::")
        assert name not in cases
        cases[name] = base64.b64decode(test["stdoutBase64"], validate=True)
    assert set(cases) == set(EXPECTED) | {REJECTION}
    assert not cases[REJECTION]
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
    shim.write_text("#include <stdint.h>\n#include <stdlib.h>\n"
                    "_Noreturn void fy_runtime_trap(uint32_t category)"
                    "{exit(100+(int)category);}\n", encoding="utf-8")
    executions = []
    for name, expected in EXPECTED.items():
        source = output / f"{name}.c"
        assert cases[name].startswith(b"/* fairyc.candidate-module-scalar-c/v1 */\n")
        source.write_bytes(cases[name])
        for profile, flag in (("low", "/Od"), ("optimized", "/O2")):
            binary = output / f"{name}-{profile}.exe"
            build = run([str(compiler), "/nologo", "/std:c11", "/TC", flag,
                         str(source), str(shim), f"/Fe:{binary}", "/link",
                         "/INCREMENTAL:NO"], output, environment)
            assert build["exit"] == 0, build
            result = run([str(binary)], output)
            assert result["exit"] == expected, result
            assert not result["stdout"] and not result["stderr"], result
            executions.append(dict(name=name, profile=profile, expectedExit=expected,
                                   sourceSHA256=hashlib.sha256(cases[name]).hexdigest(),
                                   build=build, run=result))
            print(f"PASS {profile}: {name} (exit {expected})", flush=True)
    receipt = dict(schema="fairyc.candidate-module-c-execution/v1", passed=True,
                   nativeReport=dict(path=str(args.report.resolve()),
                                     sha256=hashlib.sha256(report_bytes).hexdigest()),
                   compiler=str(compiler), executions=executions,
                   trapShim="Standalone test shim maps runtime category to exit 100+category",
                   limitation="Report source/compiler provenance must be supplied by the caller")
    (output / "execution.json").write_text(json.dumps(receipt, indent=2) + "\n",
                                           encoding="utf-8")


if __name__ == "__main__":
    main()
