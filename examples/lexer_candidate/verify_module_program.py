"""Observe native Fairy module API tests; does not implement compiler logic."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

PROJECT = Path(__file__).resolve().parent
ROOT = PROJECT.parents[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--compiler", type=Path, required=True)
    parser.add_argument("--phase", choices=("join", "program"), required=True)
    args = parser.parse_args()
    compiler = args.compiler.resolve()
    output = ROOT / ".cache" / ("candidate-module-" + args.phase)
    output.mkdir(parents=True, exist_ok=True)
    rows = []

    def run(argv):
        result = subprocess.run(list(map(str, argv)), cwd=PROJECT,
                                capture_output=True, encoding="utf-8", timeout=480)
        row = dict(argv=list(map(str, argv)), cwd=str(PROJECT), exit=result.returncode,
                   stdout=result.stdout, stderr=result.stderr)
        rows.append(row)
        (output / "validation.json").write_text(json.dumps(rows, ensure_ascii=False,
                                                indent=2) + "\n", encoding="utf-8")
        print(result.stdout[-1600:], result.stderr[-1800:], flush=True)
        if result.returncode:
            raise RuntimeError(row)
        return result

    run([compiler, "check"])
    test = ("tests::module_ast_join::joined_arenas_remap_ids_spans_and_tokens"
            if args.phase == "join" else
            "tests::module_program::nominal_module_consumers_keep_identity_and_privacy")
    for release in (False, True):
        argv = [compiler, "test", "--native", "--locked", "--exact", test, "--json"]
        if release:
            argv.append("--release")
        report = json.loads(run(argv).stdout)
        assert report["count"] == 1 and report["passed"] == 1 and report["status"] == "ok"

    def digest(path):
        return dict(path=str(path.relative_to(ROOT)),
                    sha256=hashlib.sha256(path.read_bytes()).hexdigest())

    evidence = dict(schema="fairyc.candidate-module-" + args.phase + "/v1",
                    passed=True, bootstrap=digest(compiler), validation=rows,
                    sources=[digest(p) for p in sorted((PROJECT / "src").glob("*.fy"))],
                    tests=digest(PROJECT / "tests" / ("module_ast_join.fy" if
                                 args.phase == "join" else "module_program.fy")),
                    lock=digest(PROJECT / "fairy.lock"),
                    limits=["bounded private candidate source-set API",
                            "SDK and generic import admission pending", "Linux deferred"])
    path = ROOT / "docs/ops/evidence" / ("candidate-module-" + args.phase + "-20261008.json")
    path.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
