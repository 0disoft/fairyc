"""Bind private literal constants to focused native/C and whole-project observations."""
import base64
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROBE = CACHE / "main-link-literal-constants-final-probe"
IMPLEMENTATION = ("module_unit_constants.fy", "module_unit_value_model.fy", "module_unit_value_collect.fy", "module_unit_values.fy", "module_unit_body_ir_lower.fy", "module_unit_body_ir_verify.fy")


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def read(name):
    return json.loads((CACHE / name).read_bytes())


def main():
    native = read("native-literal-constants.json")
    guard = read("native-literal-constants-prefix-guard.json")
    assert native["engine"] == guard["engine"] == "native"
    assert native["profile"] == guard["profile"] == "release"
    assert (native["count"], native["passed"], native["failed"]) == (3, 3, 0)
    assert (guard["count"], guard["passed"], guard["failed"]) == (1, 1, 0)
    for report in (native, guard):
        for case in report["tests"]:
            assert case["outcome"] == "passed"
            assert not base64.b64decode(case["stderrBase64"], validate=True)
    assert guard["tests"][0]["identity"].endswith("::private_literals_keep_declared_type_value_and_reject_unused_invalid_items")
    generated = next(case for case in native["tests"] if case["identity"].endswith("::folded_literals_link_and_execute_as_scalar_operands"))
    generated_hash = hashlib.sha256(base64.b64decode(generated["stdoutBase64"], validate=True)).hexdigest()
    executed = read("literal-constants-c/execution.json")
    assert executed["passed"] and len(executed["executions"]) == 2
    for run in executed["executions"]:
        assert run["sourceSHA256"] == generated_hash
        assert run["build"]["exit"] == run["run"]["exit"] == 0
        assert not run["run"]["stdout"] and not run["run"]["stderr"]
    check = read("check-literal-constants-final.json")
    assert check["exit"] == 0 and "lock verified" in check["stdout"] and not check["stderr"]
    for name in ("check", "build"):
        result = read(name + "-main-link-literal-constants-final-probe.json")
        assert result["exit"] == 0 and not result["stderr"]
    original = read("run-main-link-literal-constants-final-probe.json")
    assert not original["stderr"]
    assert "compiled units 48 retries 25\n" in original["stdout"]
    assert "linked frames 3558 work 3321980\n" in original["stdout"]
    lines = original["stdout"].splitlines()
    emitted = next(line for line in lines if line.startswith("emit code "))
    emit_code = int(emitted.split()[2])
    assert original["exit"] == (0 if emit_code == 0 else 2)
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert source.read_bytes() == (PROBE / "src" / source.name).read_bytes(), source
    paths = read("main-literal-constants-inputs.json")
    assert len(paths) == 236 and len(set(paths)) == 236
    evidence = dict(schema="fairyc.candidate-literal-constants/v1", passed=True,
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in IMPLEMENTATION],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_unit_constants.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"),
        receiptTool=identity(Path(__file__)), cExecutionTool=identity(ROOT / "tools/verify-candidate-module-c.py"),
        sourceCheck=identity(CACHE / "check-literal-constants-final.json"),
        bootstrap=identity(ROOT / ".cache/candidate-tagged/compiler-build-120s.exe"),
        native=dict(report=identity(CACHE / "native-literal-constants.json"), guardRerun=identity(CACHE / "native-literal-constants-prefix-guard.json"),
            distinctPassed=3, totalObservations=4,
            reuse="The final source adds rejection of nested negation. The literal validation test was rerun; the unchanged C and mismatch fixtures contain no such expression.",
            beforeGuardSource=identity(CACHE / "main-link-literal-constants-probe/src/module_unit_constants.fy")),
        generatedC=dict(report=identity(CACHE / "literal-constants-c/execution.json"), sha256=generated_hash, nativeExecutions=2, profiles=["/Od", "/O2"]),
        originalProject=dict(inputs=[identity(ROOT / path) for path in paths], count=236, units=48, retries=25, frames=3558, work=3321980,
            fullLinkPassed=True, cGenerationPassed=emit_code == 0, finalObservation=emitted, seconds=original["seconds"],
            wrapper=identity(PROBE / "src/main.fy"), image=identity(PROBE / "build/main-link-literal-constants-final-probe.exe"),
            check=identity(CACHE / "check-main-link-literal-constants-final-probe.json"), build=identity(CACHE / "build-main-link-literal-constants-final-probe.json"), run=identity(CACHE / "run-main-link-literal-constants-final-probe.json")),
        limits=["Partial ADR 0220: private bool/u8/i32/u32 literals, groups and scalar prefixes only; negation requires a literal operand through groups.",
                "General constant expressions, dependency/cycle checks, conditionals/casts and const-eval calls remain closed.",
                "The full-link observer has an explicit 120-second process deadline; stable native tests retain their default 60-second process deadline.",
                "Host native builds use the existing validation-only 120-second bootstrap compile deadline.",
                "Original full C execution, native driver, default timeout/performance, standalone candidate profiles, Linux, full suite and Stage1/Stage2 remain."],
        pushed=False)
    output = ROOT / "docs/ops/evidence/candidate-literal-constants-20261011.json"
    output.write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, nativeCExecutions=2, fullOriginalLink=True, originalC=emit_code == 0, nextObservation=emitted)))


if __name__ == "__main__":
    main()
