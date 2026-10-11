"""Bind entry-scoped C selection to cleanup execution and original source limits."""
import base64
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def main():
    native = json.loads((CACHE / "native-c-entry-parts.json").read_bytes())
    assert native["engine"] == "native" and native["profile"] == "release"
    assert native["count"] == native["passed"] == 2 and native["failed"] == 0
    assert all(test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True) for test in native["tests"])
    execution = json.loads((CACHE / "c-entry-parts-native-c/execution.json").read_bytes())
    assert execution["passed"] and len(execution["cases"]) == 2
    assert all(len(case["executions"]) == 2 for case in execution["cases"].values())
    entry = execution["cases"]["project_c_entry_partition_keeps_nested_result_cleanup_and_omits_unused_bodies"]
    assert entry["frames"] == 4 and entry["units"] == 6
    assert all(run["run"]["stdout"] == "99,7," for run in entry["executions"])
    prior_path = ROOT / "docs/ops/evidence/candidate-project-c-parts-20261011.json"
    prior = json.loads(prior_path.read_bytes())
    literal = "project_c_partition_preserves_literal_pointer_identity"
    assert execution["cases"][literal]["packetSHA256"] == prior["separateTranslationUnits"]["cases"][literal]["packetSHA256"]
    check = json.loads((CACHE / "check-c-entry-final.json").read_bytes())
    assert check["exit"] == 0 and "291 source file(s), 3653 MIR function(s), lock verified" in check["stdout"] and not check["stderr"]
    original = json.loads((CACHE / "run-main-c-entry-probe.json").read_bytes())
    assert original["exit"] == 2 and not original["timedOut"] and not original["stderr"]
    assert original["stdout"] == "compiled units 48 retries 25\nlinked frames 3558 work 3321980\nproject drops 6392 root drops 4\nroot CFG blocks 247 work 7650\npartition code 1002 module 0 at 0..0\n"
    inputs = prior["originalProject"]["sourceInputs"]
    assert len(inputs) == 236 and original["argv"][1:] == [source["path"] for source in inputs]
    assert all(identity(ROOT / source["path"]) == source for source in inputs)
    assert not (CACHE / "project-entry-packet.bin").exists()
    probe = CACHE / "main-c-entry-probe"
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert source.read_bytes() == (probe / "src" / source.name).read_bytes(), source
    receipt = dict(schema="fairyc.candidate-c-entry-reachability/v1", passed=True,
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in ("module_unit_c_reachability.fy", "module_unit_c_emit.fy")],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_link.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-project-c-parts-v0.md"),
        receiptTools=[identity(ROOT / "tools" / name) for name in ("verify-candidate-project-c-parts.py", "verify-candidate-c-entry-reachability.py")],
        sourceCheck=check, native=dict(report=identity(CACHE / "native-c-entry-parts.json"), passed=2, observation=identity(CACHE / "native-c-entry-parts-observation.json")),
        separateTranslationUnits=dict(report=identity(CACHE / "c-entry-parts-native-c/execution.json"), cases=2, executions=4, selectedFrames=4, originalFrames=6, cleanupTrace="99,7,"),
        literalCompatibility=dict(priorReceipt=identity(prior_path), packetSHA256=execution["cases"][literal]["packetSHA256"], exact=True),
        originalProject=dict(passed=False, sourceCount=236, exactInputIdentitiesMatchPriorReceipt=True, report=identity(CACHE / "run-main-c-entry-probe.json"), exit=original["exit"], seconds=original["seconds"], stdout=original["stdout"], stderr=original["stderr"], timedOut=False, wrapper=identity(probe / "src/main.fy"), image=identity(probe / "build/main-c-entry-probe.exe"), bundlePublished=False, blocker="Code 1002 module 0 span 0..0; preparation/project work versus output phase not yet isolated"),
        limits=dict(preparationWork=1048576, unitWork=1048576, outputBytesPerBuffer=2097152, projectWork=4194304),
        deferred=["Locate and divide the original preparation/project budget phase", "Original complete C generation and execution", "Native driver/publication", "Default timeout/performance", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-c-entry-reachability-20261011.json").write_text(json.dumps(receipt, indent=2) + "\n")
    print("PASS: 2 native cases, 4 C executions, unchanged literal packet; original budget failure retained")


if __name__ == "__main__":
    main()
