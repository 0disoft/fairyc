"""Record the original unit's imported metadata and unchanged IR guards."""
import base64
import hashlib
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def report(name, passed, failed):
    path = CACHE / name
    data = json.loads(path.read_bytes())
    assert data["engine"] == "native" and data["target"] == "x86_64-windows-msvc"
    assert (data["count"], data["passed"], data["failed"]) == (passed + failed, passed, failed)
    for case in data["tests"]:
        assert not base64.b64decode(case["stderrBase64"], validate=True)
    return data, identity(path)


def main():
    focused, focused_id = report("native-origin-imported-metadata-fields.json", 1, 0)
    case = focused["tests"][0]
    assert case["outcome"] == "passed" and case["identity"] == "tests::module_unit_imported_metadata_capacity::original_origin_check_retains_all_imported_fields_above_the_source_arena_cap"
    stdout = base64.b64decode(case["stdoutBase64"], validate=True).decode()
    assert stdout == "fields 1231 shapes 179 vectors 75 work 478696\n"
    guards, guards_id = report("native-imported-metadata-guards.json", 1, 0)
    assert guards["tests"][0]["outcome"] == "passed" and guards["tests"][0]["identity"] == "tests::module_unit_body_ir::body_ir_rejects_malformed_owned_body_tables"
    original, original_id = report("native-main-imported-metadata-fields.json", 0, 1)
    failure = base64.b64decode(original["tests"][0]["stdoutBase64"], validate=True).decode()
    assert failure == "stage 6 code 1002 module 138 at 0..0 range 138..139\n"
    paths = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    assert paths[136].endswith("/module_unit_origin_check.fy") and paths[138].endswith("/module_unit_origin_flow.fy")
    check = CACHE / "check-imported-metadata-fields.txt"
    assert check.read_text(encoding="utf-8-sig").strip() == "check ok: 2 instance(s), 285 source file(s), 3550 MIR function(s), lock updated"
    evidence = dict(schema="fairyc.candidate-imported-metadata-fields/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        sources=[identity(ROOT / p) for p in paths],
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / p) for p in ("candidate_limits.fy", "module_unit_interfaces.fy", "module_unit_body_ir_verify.fy")],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_unit_imported_metadata_capacity.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"), receiptTool=identity(Path(__file__)),
        policy=dict(importedMetadataFields=2048, sourceAndJoinedASTFields=1024, shapes=512, vectorLayouts=256, genericFunctions=64, work=1048576),
        native=dict(focused=focused_id, guards=guards_id, passedCases=2, stdout=stdout), check=identity(check),
        originalCompilationPassed=False, original=dict(report=original_id, files=len(paths), stdout=failure, nextSource=paths[138],
            progress="Original origin_check unit passed; the next single-source origin_flow unit fails during owned IR"),
        deferred=["Remaining original units", "Dispatcher linking and execution", "Native driver", "Linux", "Standalone compiler profiles", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-imported-metadata-fields-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=2, fields=1231, originalCompilationPassed=False)))


if __name__ == "__main__":
    main()
