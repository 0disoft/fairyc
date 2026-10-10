"""Bind reused permission lookup to focused mutations and original owned IR."""
import base64
import hashlib
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROBE = CACHE / "main-link-storage-permission-index-probe"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def main():
    report_path = CACHE / "native-ir-permission-index.json"
    report = json.loads(report_path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (4, 4, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    stress = "tests::module_project_frames_capacity::ordinary_project_frames_above_one_thousand_keep_distinct_coordinates"
    cleanup = "tests::module_project_link::project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"
    assert base64.b64decode(cases[stress]["stdoutBase64"], validate=True) == b"frames 1089 work 9573 callables 1080\n"
    sdk_path = CACHE / "native-cached-storage-permission.json"
    sdk = json.loads(sdk_path.read_bytes())
    assert (sdk["count"], sdk["passed"], sdk["failed"]) == (1, 1, 0)
    assert sdk["tests"][0]["identity"] == "tests::module_unit_c_sdk::sdk_c_rejects_forged_storage_metadata"
    for case in [*cases.values(), *sdk["tests"]]:
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    check_path = CACHE / "check-ir-permission-final.json"
    check = json.loads(check_path.read_bytes())
    assert check["exit"] == 0 and "check ok:" in check["stdout"] and not check["stderr"]
    build_path = CACHE / "build-main-link-storage-permission-index-probe.json"
    probe_check_path = CACHE / "check-main-link-storage-permission-index-probe.json"
    assert json.loads(build_path.read_bytes())["exit"] == json.loads(probe_check_path.read_bytes())["exit"] == 0
    run_path = CACHE / "run-main-link-storage-permission-index-probe.json"
    run = json.loads(run_path.read_bytes())
    assert run["exit"] == 2 and run["stdout"] == "compiled units 48 retries 25\nlink code 1001 module 187 at 4211..4256\n" and not run["stderr"]
    diagnostic_path = CACHE / "run-main-link-permission-contract-diagnostic.json"
    diagnostic = json.loads(diagnostic_path.read_bytes())
    assert diagnostic["exit"] == 2 and diagnostic["stdout"] == "compiled units 48 retries 25\nlink code 430003 module 187 at 4211..4256\n"
    maps_path = CACHE / "main-call-contract-diagnostic-codes.json"
    guard = json.loads(maps_path.read_bytes())["430003"]
    assert guard["module"] == "module_project_calls" and "!c.checked || c.kind!=1 && c.kind!=2" in guard["guard"]
    state_path = CACHE / "run-main-link-invocation-state-probe.json"
    state = json.loads(state_path.read_bytes())
    assert state["exit"] == 2 and not state["stderr"]
    assert "invocation kind 1 checked 0 args 5\n" in state["stdout"]
    assert "arg 4 expected 5 actual 0 checked 0 kind 2 state 0 text PAYLOAD_BODY\n" in state["stdout"]
    for ordinal in range(4):
        assert "checked 1 kind 2 state 1 text " in state["stdout"].split("arg " + str(ordinal) + " ", 1)[1].split("\n", 1)[0]
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert (PROBE / "src" / source.name).read_bytes() == source.read_bytes(), source
    generated_hash = hashlib.sha256(base64.b64decode(cases[cleanup]["stdoutBase64"], validate=True)).hexdigest()
    execution_path = CACHE / "project-link-c/execution.json"
    executions = json.loads(execution_path.read_bytes())
    runs = [row for row in executions["executions"] if row["name"] == cleanup.split("::")[-1]]
    assert executions["passed"] and len(runs) == 2
    for row in runs:
        assert row["sourceSHA256"] == generated_hash and row["build"]["exit"] == row["run"]["exit"] == 0
    paths = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    evidence = dict(schema="fairyc.candidate-ir-permission-index/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in ("module_unit_body_ir_verify.fy", "module_unit_body_ir_storage.fy", "module_project_calls.fy")],
        tests=[identity(ROOT / "examples/lexer_candidate/tests" / name) for name in ("module_project_frames_capacity.fy", "module_unit_c_sdk.fy", "module_unit_c_pointers.fy")],
        receiptTool=identity(Path(__file__)), contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"),
        policy=dict(contractSlots=8192, contractsKey="module/declaration", onePermissionDecisionPerBody=True,
            unsafeBlocksCollectedOncePerBody=True, sharedBy=["pointer casts", "raw parts", "SDK storage", "project calls"],
            unitPhaseWork=1048576, projectWork=4194304, permissionsAndCapacitiesUnchanged=True),
        native=dict(initialReport=identity(report_path), finalStorageReport=identity(sdk_path), passed=5,
            castsInOneBody=1000, linkedFrames=1089, callables=1080, calls=1080,
            coverage=["Indexed callable and unsafe-block handling in a large linked body", "Removed/changed cast authority rejects", "Raw owner/argument/unsafe metadata rejects", "Cached and standalone SDK storage checks agree on authentic input and seven mutations", "Exhausted storage work rejects in both paths"],
            reuse="The four initial cases have no SDK storage operations; cached storage was then added and independently tested on authentic Vec input"),
        sourceCheck=identity(check_path), reusedCExecution=dict(report=identity(execution_path), runs=2, sourceSHA256=generated_hash),
        originalProject=dict(inputs={path: identity(ROOT / path)["sha256"] for path in paths}, count=len(paths), units=48, retries=25,
            wrapper=identity(PROBE / "src/main.fy"), image=identity(PROBE / "build/main-link-storage-permission-index-probe.exe"),
            sourceCheck=identity(probe_check_path), build=identity(build_path), execution=identity(run_path), seconds=run["seconds"],
            ownedIRAssemblyAndRevalidationPassed=True, fullLinkPassed=False,
            nextFailure=dict(diagnostic=identity(diagnostic_path), codeMap=identity(maps_path), temporaryCode=430003,
                source="examples/lexer_candidate/src/records.fy", moduleIndex=187, start=4211, end=4256,
                guard="Invocation is unchecked or its kind is neither ordinary nor specialized",
                invocationState=identity(state_path), kind=1, checked=False, arguments=5,
                unresolvedArgument=dict(position=4, expectedType=5, actualType=0, state=0, sourceName="PAYLOAD_BODY"),
                otherArgumentsChecked=True, nextTask="Connect typed module constant references to owned value/IR/C lowering")),
        deferred=["Complete call validation", "Original C generation/execution", "Native driver", "Default timeout/performance", "Standalone candidate profiles", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-ir-permission-index-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=5, originalOwnedIRValidated=True, originalLinked=False)))


if __name__ == "__main__":
    main()
