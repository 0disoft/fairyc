"""Verify indexed C layout lookup without changing work or output limits."""
import base64
import hashlib
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROBE = CACHE / "main-c-layout-index-probe"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def read(name):
    return json.loads((CACHE / name).read_bytes())


def main():
    native = read("native-c-layout-index.json")
    assert (native["count"], native["passed"], native["failed"]) == (4, 2, 2)
    assert native["engine"] == "native" and native["profile"] == "release"
    rerun = read("native-c-layout-index-rerun.json")
    assert (rerun["count"], rerun["passed"], rerun["failed"]) == (2, 2, 0)
    assert rerun["engine"] == "native" and rerun["profile"] == "release"
    latest = {test["identity"]: test for report in (native, rerun) for test in report["tests"]}
    assert len(latest) == 4
    cases = {}
    for test in latest.values():
        assert test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True)
        cases[test["identity"].split("::")[-1]] = base64.b64decode(test["stdoutBase64"], validate=True)
    stress = cases["aggregate_c_indexes_layouts_and_respects_budget"]
    measured = re.fullmatch(rb"layouts 200 work (\d+)\n", stress)
    assert measured and int(measured[1]) < 4000
    assert not cases["sdk_c_plan_respects_work_budget_before_authority_or_output"]
    c_runs = []
    for name in ("c-layout-index-owned-c/execution.json", "c-layout-index-sdk-c/receipt.json"):
        report = read(name)
        assert report["passed"] and len(report["executions"]) == 2
        for run in report["executions"]:
            assert run["sourceSHA256"] == hashlib.sha256(cases[run["name"]]).hexdigest()
            assert run["build"]["exit"] == run["run"]["exit"] == 0
            assert not run["run"]["stderr"]
        c_runs.append(identity(CACHE / name))
    check = read("check-c-layout-index-final.json")
    assert check["exit"] == 0 and "lock verified" in check["stdout"] and not check["stderr"]
    before = read("run-main-c-literal-shared-budget-probe.json")
    assert before["exit"] == 2 and "emit code 721007 module 729542" in before["stdout"]
    after = read("run-main-c-layout-index-probe.json")
    assert after["exit"] == 2 and not after["timedOut"] and not after["stderr"]
    assert "compiled units 48 retries 25\nlinked frames 3558 work 3321980\n" in after["stdout"]
    assert after["stdout"].endswith("emit code 721008 module 0 at 0..0 C bytes 0\n")
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name not in ("main.fy", "module_unit_c_emit.fy"):
            assert source.read_bytes() == (PROBE / "src" / source.name).read_bytes(), source
    paths = read("main-cfg-drop-index-closed-inputs.json")
    assert len(paths) == len(set(paths)) == 236 and after["argv"][1:] == paths
    invalid = read("native-c-layout-index-invalid-visibility-observation.json")
    assert invalid["exit"] == 3 and "item is not visible" in invalid["stderr"]
    evidence = dict(schema="fairyc.candidate-c-layout-index/v1", passed=True,
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in
            ("module_unit_c_layout_index.fy", "module_unit_c_aggregate.fy", "module_unit_c_sdk.fy")],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_unit_c_owned.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"),
        receiptTool=identity(Path(__file__)), sourceCheck=check,
        native=dict(report=identity(CACHE / "native-c-layout-index.json"), rerun=identity(CACHE / "native-c-layout-index-rerun.json"), distinctPassed=4, observations=6,
            stress=dict(layouts=200, work=int(measured[1]), outOfRangeAndExhaustedWorkReject=True)),
        generatedC=dict(reports=c_runs, executions=4, profiles=["/Od", "/O2"],
            coverage=["Nested resource and tagged cleanup order", "Real SDK Vec record slices and view mutation"]),
        originalProject=dict(inputs=[identity(ROOT / path) for path in paths], count=236, units=48, retries=25, frames=3558,
            literalStageNowPassed=True, cGenerationPassed=False,
            before=identity(CACHE / "run-main-c-literal-shared-budget-probe.json"),
            after=identity(CACHE / "run-main-c-layout-index-probe.json"), nextPhase="SDK/lifecycle definitions or foreign signatures"),
        invalidHarnessAttempt=dict(observation=invalid, reason="A test attempted to mutate private IR fields; removed without opening visibility or changing product authority"),
        initialFixtureFailures=["200 declarations exceeded the existing per-module 128 declaration limit; corrected fixture splits them across two modules", "Vec SDK source was passed from the non-existent std/vec.fy path; corrected argument uses optional/std/vec.fy"],
        limits=dict(work=1048576, outputBytes=2097152, raised=False),
        deferred=["Full original C generation/execution", "Native driver", "Default timeout/performance", "Standalone candidate profiles", "Linux", "Full suite", "Stage1/Stage2"],
        stableCLIChanged=False, sdkABIChanged=False, GoChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    next_path = CACHE / "run-main-c-sdk-stage-probe.json"
    if next_path.exists():
        next_ = json.loads(next_path.read_bytes())
        assert next_["exit"] == 2 and not next_["timedOut"] and not next_["stderr"]
        assert next_["stdout"].endswith("emit code 722003 module 979985 at 1048576..351827 C bytes 0\n")
        evidence["originalProject"]["nextPhase"] = "module_unit_c_emit::foreign_signatures"
        evidence["originalProject"]["nextDiagnostic"] = dict(report=identity(next_path), stdout=next_["stdout"],
            initialWork=979985, exhaustedWork=1048576, temporaryOutputBytes=351827, publicOutputBytes=0,
            emitter=identity(CACHE / "main-c-sdk-stage-probe/src/module_unit_c_emit.fy"))
    (ROOT / "docs/ops/evidence/candidate-c-layout-index-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=4, generatedCExecutions=4, aggregateWork=int(measured[1]), originalC=False)))


if __name__ == "__main__":
    main()
