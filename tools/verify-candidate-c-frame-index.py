"""Bind bounded frame lookup to native, unchanged C and original-project evidence."""
import base64
import hashlib
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROBE = CACHE / "main-c-frame-ranges-probe"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def read(name):
    return json.loads((CACHE / name).read_bytes())


def main():
    native = read("native-c-frame-ranges.json")
    assert (native["count"], native["passed"], native["failed"]) == (4, 4, 0)
    assert native["engine"] == "native" and native["profile"] == "release"
    cases = {}
    for test in native["tests"]:
        assert test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True)
        cases[test["identity"].split("::")[-1]] = base64.b64decode(test["stdoutBase64"], validate=True)
    stress = cases["ordinary_project_frames_above_one_thousand_keep_distinct_coordinates"]
    assert re.fullmatch(rb"frames 1089 work \d+ callables 1080\n", stress)
    reused = []
    for name in ("project-link-c/execution.json", "c-layout-index-sdk-c/receipt.json"):
        report = read(name)
        assert report["passed"]
        selected = [run for run in report["executions"] if run["name"] in cases]
        assert len(selected) == (4 if name.startswith("project-link-c/") else 2)
        for run in selected:
            assert run["sourceSHA256"] == hashlib.sha256(cases[run["name"]]).hexdigest()
            assert run["build"]["exit"] == run["run"]["exit"] == 0 and not run["run"]["stderr"]
            for source in run.get("runtimeInputs", []):
                assert identity(ROOT / source["path"])["sha256"] == source["sha256"]
        reused.append(dict(report=identity(CACHE / name), executions=len(selected)))
    check = read("check-c-frame-ranges-final.json")
    assert check["exit"] == 0 and "lock verified" in check["stdout"] and not check["stderr"]
    before = read("run-main-c-callable-index-probe.json")
    assert before["exit"] == 2 and "emit code 725012 module 213" in before["stdout"]
    original = read("run-main-c-frame-ranges-probe.json")
    assert not original["timedOut"] and not original["stderr"]
    assert "compiled units 48 retries 25\nlinked frames 3558 work 3321980\n" in original["stdout"]
    emitted = re.search(r"emit code (\d+) module (\d+) at (\d+)\.\.(\d+) C bytes (\d+)\n$", original["stdout"])
    assert emitted
    code, module, start, end, output = map(int, emitted.groups())
    assert original["exit"] == (0 if code == 0 else 2)
    assert (code, module, start, end, output) == (728013, 2, 1689, 1847, 0)
    phases = read("c-frame-ranges-phase-codes.json")
    stage = phases.get(str(code), "Success" if code == 0 else "Non-budget error; inspect original source span")
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name not in ("main.fy", "module_unit_c_emit.fy"):
            assert source.read_bytes() == (PROBE / "src" / source.name).read_bytes(), source
    paths = read("main-cfg-drop-index-closed-inputs.json")
    assert len(paths) == len(set(paths)) == 236 and original["argv"][1:] == paths
    evidence = dict(schema="fairyc.candidate-c-frame-index/v1", passed=True,
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in
            ("module_unit_c_frame_index.fy", "module_unit_c_emit.fy")],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_frames_capacity.fy"),
        receiptTool=identity(Path(__file__)), contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"),
        sourceCheck=check, native=dict(report=identity(CACHE / "native-c-frame-ranges.json"), passed=4,
            coverage=["1089 original frame row identities with bounded preparation/queries", "Missing specialized identity and exhausted preparation/query work", "Distinct generic units", "Shared cleanup foreign prototype", "Authentic SDK Vec signatures"]),
        reusedC=dict(reports=reused, executions=6, reason="New emitted C is byte-identical to previously compiled and executed /Od and /O2 artifacts; SDK runtime inputs also match"),
        originalProject=dict(inputList=identity(CACHE / "main-cfg-drop-index-closed-inputs.json"), inputBytesSHA256=hashlib.sha256(b"".join(path.encode() + b"\0" + (ROOT / path).read_bytes() + b"\0" for path in paths)).hexdigest(), count=236, units=48, retries=25, frames=3558,
            before=identity(CACHE / "run-main-c-callable-index-probe.json"), after=identity(CACHE / "run-main-c-frame-ranges-probe.json"),
            phaseMap=identity(CACHE / "c-frame-ranges-phase-codes.json"),
            diagnosticEmitter=identity(PROBE / "src/module_unit_c_emit.fy"), cGenerationPassed=code == 0,
            result=dict(code=code, module=module, start=start, end=end, publicOutputBytes=output, phase=stage)),
        limits=dict(work=1048576, outputBytes=2097152, callableCount=4096, raised=False),
        abandonedHashAttempt=dict(native=identity(CACHE / "native-c-frame-hash-attempt.json"), original=identity(CACHE / "run-main-c-frame-index-probe.json"), reason="Full hash preparation exhausted existing C work before body emission; replaced with verified ranges, context identity and module-local lists"),
        deferred=["Full original C generation/execution", "Native driver", "Default timeout/performance", "Standalone candidate profiles", "Linux", "Full suite", "Stage1/Stage2"],
        stableCLIChanged=False, sdkABIChanged=False, GoChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    meter_path = CACHE / "run-main-c-frame-budget-probe.json"
    if meter_path.exists():
        meter = json.loads(meter_path.read_bytes())
        assert meter["exit"] == 2 and not meter["timedOut"] and not meter["stderr"]
        measured = re.search(r"emit code 729013 module (\d+) at (\d+)\.\.(\d+) C bytes 0\n$", meter["stdout"])
        assert measured
        state, work, temporary_bytes = map(int, measured.groups())
        evidence["originalProject"]["nextBudgetDiagnostic"] = dict(report=identity(meter_path), stateCode=state, work=work, temporaryBytes=temporary_bytes)
    (ROOT / "docs/ops/evidence/candidate-c-frame-index-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=4, reusedCExecutions=6, originalCode=code, originalC=code == 0)))


if __name__ == "__main__":
    main()
