"""Record independent SDK vector layout and executable-specialization budgets."""
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


def main():
    path = CACHE / "native-unit-vector-layout-cap.json"
    data = json.loads(path.read_bytes())
    assert data["engine"] == "native" and data["profile"] == "release" and (data["count"], data["passed"], data["failed"]) == (2, 1, 1)
    cases = {case["identity"]: case for case in data["tests"]}
    layout_case = "tests::module_unit_vector_layout_capacity::sixty_five_vector_layouts_do_not_consume_executable_specializations"
    generic_case = "tests::module_project_split::generic_capacity_splits_without_losing_source_coverage"
    assert set(cases) == {layout_case, generic_case}
    assert cases[layout_case]["outcome"] == "failed"
    retry_path = CACHE / "native-unit-vector-layout-cap-final.json"
    retry = json.loads(retry_path.read_bytes())
    assert retry["engine"] == "native" and retry["profile"] == "development" and (retry["count"], retry["passed"], retry["failed"]) == (1, 1, 0)
    assert retry["tests"][0]["identity"] == layout_case
    cases[layout_case] = retry["tests"][0]
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    assert base64.b64decode(cases[layout_case]["stdoutBase64"], validate=True).decode() == "vector layouts 65 generic functions 0\n"
    before = ROOT / "docs/ops/evidence/candidate-project-independent-move-budget-20261011.json"
    assert json.loads(before.read_bytes())["original"]["stdout"] == "stage 3 code 4503 module 213 at 0..0 range 136..137\n"
    full_path = CACHE / "native-main-unit-vector-layout-cap.json"
    full = json.loads(full_path.read_bytes())
    assert full["engine"] == "native" and full["profile"] == "release" and (full["count"], full["passed"], full["failed"]) == (1, 0, 1)
    stdout = base64.b64decode(full["tests"][0]["stdoutBase64"], validate=True).decode()
    assert stdout == "stage 3 code 10 module 113 at 275..421 range 136..137\n"
    check = CACHE / "check-unit-vector-layout-cap.txt"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    evidence = dict(schema="fairyc.candidate-unit-vector-layout-cap/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_unit_interfaces.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"),
        test=identity(ROOT / "examples/lexer_candidate/tests/module_unit_vector_layout_capacity.fy"), receiptTool=identity(Path(__file__)),
        policy=dict(vectorLayouts=256, sharedLimit="sdk_vector_plan::limit", genericFunctionsPerUnit=64, genericFunctionsPerDeclaration=16,
                    metadataFields=1024, work=1048576, note="Independent table and traversal caps can reject before 256 layouts"),
        native=dict(report=identity(path), retry=identity(retry_path), passedCases=2, vectorLayouts=65, executableGenericInstances=0,
                    fixtureCorrection="Initial fixture exceeded the unchanged 128-declaration per-source catalog cap. Split the 65 nominal declarations and 65 function prototypes into separate sources. Also corrected the initial SDK field-count assertion and counted generic instances separately from retained ordinary functions. Unchanged passing generic-capacity case is reused.",
                    diagnostic=identity(CACHE / "native-unit-vector-layout-cap-diagnostic.json"),
                    coverage=["65 distinct nominal-element Vec layouts admitted without executable specializations", "80 executable generic instances still split under the 64-per-unit limit, preserving all 83 frames"]),
        check=identity(check), originalCompilationPassed=False,
        original=dict(before=identity(before), report=identity(full_path), stdout=stdout,
                      unitSource="examples/lexer_candidate/src/module_unit_origin_check.fy", attributedDeclaration="module_unit_drops::Node",
                      nextStep="Inspect imported shape/field capacity for the same original unit"),
        deferred=["Remaining original units", "Original dispatcher linking and C execution", "Linux and Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-unit-vector-layout-cap-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=2, vectorLayouts=65, genericFunctions=0, originalCompilationPassed=False)))


if __name__ == "__main__":
    main()
