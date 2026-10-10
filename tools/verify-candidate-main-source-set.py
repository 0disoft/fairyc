"""Bind native source-set admission reports to the complete original main import closure."""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def closure():
    modules = {}
    for folder in (ROOT / "examples/lexer_candidate/src", ROOT / "internal/sdkstdlib/sources"):
        for path in folder.rglob("*.fy"):
            name = re.search(r"^module\s+([\w:]+);", path.read_text()).group(1)
            assert name not in modules, name
            modules[name] = path
    seen = set()
    pending = ["main"]
    while pending:
        name = pending.pop()
        if name in seen:
            continue
        seen.add(name)
        pending.extend(re.findall(r"^import\s+([\w:]+)", modules[name].read_text(), re.M))
    return [modules["main"], *[modules[name] for name in sorted(seen - {"main"})]]


def identity(path):
    return dict(path=path.resolve().relative_to(ROOT).as_posix(), bytes=len(path.read_bytes()),
                sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def report(path, expected):
    value = json.loads(path.read_bytes())
    assert value["engine"] == "native" and value["profile"] == "release"
    assert value["target"] == "x86_64-windows-msvc"
    assert value["count"] == value["passed"] == expected and value["failed"] == 0
    for test in value["tests"]:
        assert test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True)
    return value


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--inputs-output", type=Path)
    for name in ("report", "baseline", "boundaries", "parser-boundary", "regression", "check", "output"):
        parser.add_argument("--" + name, type=Path)
    args = parser.parse_args()
    sources = closure()
    if args.inputs_output:
        args.inputs_output.write_text(json.dumps([path.relative_to(ROOT).as_posix() for path in sources], indent=2) + "\n", encoding="utf-8")
    if not args.output:
        return
    assert all(getattr(args, name) for name in ("report", "baseline", "boundaries", "parser_boundary", "regression", "check"))
    native = report(args.report, 1)
    test = native["tests"][0]
    assert test["identity"] == "tests::module_source_set::original_main_sources_fit_catalog_and_units"
    stdout = base64.b64decode(test["stdoutBase64"], validate=True).decode()
    proof = re.fullmatch(r"sources (\d+) bytes (\d+)\nunits (\d+) joined bytes (\d+)\n", stdout)
    assert proof
    byte_count = sum(len(path.read_bytes()) for path in sources)
    assert int(proof[1]) == len(sources) and int(proof[2]) == byte_count
    assert int(proof[4]) == byte_count + len(sources) - 1 and int(proof[3]) > 1
    before = json.loads(args.baseline.read_bytes())
    assert before["count"] == before["failed"] == 1 and before["passed"] == 0
    before_stdout = base64.b64decode(before["tests"][0]["stdoutBase64"], validate=True).decode()
    assert before_stdout == "parse code 4 module 113\n"
    assert sources[113].name == "module_unit_generics.fy"
    boundaries = json.loads(args.boundaries.read_bytes())
    assert boundaries["engine"] == "native" and boundaries["profile"] == "release"
    assert boundaries["count"] == 3 and boundaries["passed"] == 2 and boundaries["failed"] == 1
    parser_identity = "tests::source_limits::parser_expression_arena_uses_the_bounded_unit_limit"
    for case in boundaries["tests"]:
        if case["identity"] == parser_identity:
            assert case["outcome"] == "failed" and base64.b64decode(case["stderrBase64"]).decode() == "parser 4"
        else:
            assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"])
    parser_boundary = report(args.parser_boundary, 1)
    assert parser_boundary["tests"][0]["identity"] == parser_identity
    regression = report(args.regression, 2)
    checked = re.fullmatch(r"check ok: 2 instance\(s\), (\d+) source file\(s\), (\d+) MIR function\(s\), lock verified\n", args.check.read_text().replace("\r\n", "\n"))
    assert checked
    records = [identity(path) for path in sources]
    evidence = dict(schema="fairyc.candidate-main-source-set-admission/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        sourceSet=dict(files=len(sources), privateFiles=sum("lexer_candidate/src" in x["path"] for x in records),
                       rawBytes=byte_count, separatorInclusiveBytes=int(proof[4]), units=int(proof[3]),
                       method="Exact main.fy and transitive import declarations; native parser/catalog/unit planner consume every original file"),
        sourcePaths=[x["path"] for x in records], sourceBytes=[x["bytes"] for x in records],
        sourceSHA256=[x["sha256"] for x in records],
        priorFailure=dict(report=identity(args.baseline), stdout=before_stdout, module=sources[113].relative_to(ROOT).as_posix()),
        admission=dict(report=identity(args.report), stdout=stdout, sourceSnapshotsEqual=True,
                       allModulesGroupedExactlyOnce=True, unitArenaCapsVerified=True),
        boundaryTests=dict(report=identity(args.boundaries), identities=[x["identity"] for x in boundaries["tests"]],
                           parserRetry=identity(args.parser_boundary),
                           initialParserFixtureFailure="Initial 2340-statement fixture hit the unchanged 4096 statement-record cap; the corrected 1092-statement fixture isolates the expression cap. The two unchanged passing cases are reused.",
                           coverage=["256 files accepted, 257 rejected", "2 MiB catalog raw source accepted, overflow rejected",
                                     "2 MiB separator-inclusive grouping accepted, one byte over rejected",
                                     "1 MiB single joined owner retained", "16381 parsed expressions accepted, next fifteen rejected",
                                     "Owned HIR/MIR generation for source spans beyond 64 KiB"]),
        unitRegression=dict(report=identity(args.regression), identities=[x["identity"] for x in regression["tests"]]),
        sourceCheck=dict(report=identity(args.check), locked=True, sourceFiles=int(checked[1]), mirFunctions=int(checked[2])),
        limits=dict(sourceBytes=65536, sourceFiles=256, sourceSetBytes=2097152, joinedOwnerBytes=1048576,
                    parserExpressions=16384, joinedExpressions=16384, statementEntryTypeFieldDepthLimitsUnchanged=True),
        bootstrap=dict(image=identity(ROOT / ".cache/candidate-tagged/compiler-file-tokens-120s.exe"),
                        validationOnlyHostCompileTimeoutSeconds=120, nativeExecutionTimeoutSeconds=60,
                        provenance="docs/ops/evidence/candidate-original-file-tokens-c-20261011.json"),
        changedContracts=["docs/architecture/candidate-compilation-units-v0.md", "docs/architecture/candidate-source-join-v0.md"],
        unchanged=["Go compiler", "Stable CLI/API", "SDK ABI", "DB", "CI", "Repository hygiene"], pushed=False,
        deferred=["Whole dispatcher signatures/body closure/typing/C generation and execution", "Native driver integration",
                  "Default host compile timeout/performance", "Standalone candidate development/Release", "Linux", "Full suite", "Stage1/Stage2"])
    args.output.write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(f"Main source-set admission passed: {len(sources)} files, {byte_count} bytes, {proof[3]} units")


if __name__ == "__main__":
    main()
