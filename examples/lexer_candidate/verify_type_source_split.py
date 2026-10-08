"""Observe body preservation and bounded parser admission for the type split."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

REPO = Path(__file__).resolve().parents[2]
SRC = REPO / "examples/lexer_candidate/src"
REGIONS = {
    "type_expr_tagged": [(44, 72), (87, 134)],
    "type_expr_scalar": [(72, 87), (134, 190)],
    "type_expr_call": [(190, 259)],
    "type_expr_access": [(259, 339)],
    "type_expr_record": [(339, 395)],
}


def identity(path):
    return dict(path=str(path.relative_to(REPO)), bytes=path.stat().st_size,
                sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def normalized(text):
    text = re.sub(r"\b(?:checking|statements|declarations)::", "", text)
    return text.replace("pub fn ", "fn ").strip()


def declarations(text):
    return text[text.index("pub fn "):]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="823eaf6c")
    ap.add_argument("--parser", type=Path, required=True)
    ap.add_argument("--receipt", type=Path, required=True)
    ap.add_argument("--images", type=Path, nargs="+", default=[])
    args = ap.parse_args()
    original = subprocess.check_output(
        ["git", "show", args.base + ":examples/lexer_candidate/src/type_check.fy"],
        cwd=REPO).decode("utf-8")
    lines = original.splitlines(keepends=True)
    proofs = []
    for module, ranges in REGIONS.items():
        expected = (lines[37] + "".join(lines[39:41]) + lines[42] +
                    "".join("".join(lines[a-1:b-1]) for a, b in ranges) +
                    lines[394] + "}")
        actual = declarations((SRC / (module + ".fy")).read_text(encoding="utf-8"))
        assert normalized(actual) == normalized(expected), module
        proofs.append(module)
    for module, first, last in [("type_statement", 397, 515), ("type_collect", 527, 574)]:
        actual = declarations((SRC / (module + ".fy")).read_text(encoding="utf-8"))
        assert normalized(actual) == normalized("".join(lines[first-1:last-1])), module
        proofs.append(module)
    coordinator = (SRC / "type_check.fy").read_text(encoding="utf-8")
    assert normalized(coordinator[coordinator.index("pub fn state::Value value("):coordinator.index("pub fn state::Value expression(")]) == normalized("".join(lines[29:37]))
    assert normalized(coordinator[coordinator.index("pub fn state::Value block("):]) == normalized("".join(lines[514:526]) + "".join(lines[573:]))
    # Only the dispatcher changed. Its id/depth guards and source-depth arguments remain.
    assert "".join(lines[38:42]) in coordinator
    assert coordinator.count("resolved,tables,problem,id,expected,result_type,depth,loop_depth") == 5
    assert coordinator.index("tagged_expr::expression(") < coordinator.index("call_expr::expression(")
    assert "node.kind==6 && syntax::expression_at(parsed,node.left).kind==28" in coordinator
    observations = []
    for module in ["type_check", *REGIONS, "type_statement", "type_collect"]:
        path = SRC / (module + ".fy")
        row = subprocess.run([str(args.parser.resolve()), "--declarations-file", str(path)],
                             cwd=REPO, capture_output=True, timeout=60)
        observations.append(dict(**identity(path), exit=row.returncode,
                                 stdoutBytes=len(row.stdout), stderr=row.stderr.decode("utf-8")))
        assert row.returncode == 0 and not row.stderr, observations[-1]
    # No source/AST/work/depth limits or parser implementation were edited by the split.
    guarded = ["candidate_limits.fy", "parse_tokens.fy", "syntax.fy", "type_state.fy"]
    for name in guarded:
        before = subprocess.check_output(["git", "show", args.base + ":examples/lexer_candidate/src/" + name], cwd=REPO)
        assert before.replace(b"\r\n", b"\n") == (SRC / name).read_bytes().replace(b"\r\n", b"\n"), name
    rejections = []
    if args.images:
        output = REPO / ".cache/candidate-tagged/type-source-split"
        output.mkdir(parents=True, exist_ok=True)
        cases = {
            "binding-type": "fn i32 main(){let bool x=1;return 7;}",
            "return-type": "fn i32 main(){return true;}",
            "condition-type": "fn i32 main(){if(1){return 7;}return 7;}",
            "immutable-target": "fn i32 main(){let i32 x=1;x=2;return 7;}",
            "call-arity": "fn i32 f(i32 n){return n;}fn i32 main(){return f();}",
            "array-length": "fn i32 main(){let u8[2] x=[1,2,3];return 7;}",
            "tagged-payload": "fn i32 main(){let Option<i32> x=.some(true);return 7;}",
            "loop-control": "fn i32 main(){break;return 7;}",
        }
        for name, body in cases.items():
            source = output / (name + ".fy")
            source.write_text("module main;" + body, encoding="utf-8")
            manifest = output / (name + ".txt")
            manifest.write_text(source.relative_to(REPO).as_posix() + "\n", encoding="utf-8")
            results = []
            for image in args.images:
                r = subprocess.run([str(image.resolve()), "--c-modules-file", str(manifest)],
                                   cwd=REPO, capture_output=True, timeout=60)
                row = dict(image=identity(image.resolve()), exit=r.returncode,
                           stdoutBytes=len(r.stdout), stderr=r.stderr.decode("utf-8"))
                assert r.returncode == 2 and not r.stdout and r.stderr, row
                results.append(row)
            assert len({(r["exit"], r["stdoutBytes"], r["stderr"]) for r in results}) == 1, results
            rejections.append(dict(name=name, input=identity(source), results=results))
    args.receipt.resolve().write_text(json.dumps(dict(
        schema="fairyc.candidate-type-source-split/v1", passed=True,
        originalCommit=args.base, branchRegionsIdentical=proofs,
        coordinatorHelpersIdentical=True, sourceDepthRetained=True,
        limitsChanged=False, parserImplementationUnchanged=True,
        observer=identity(Path(__file__)), parser=identity(args.parser.resolve()),
        parseObservations=observations, exactRejections=rejections), indent=2) + "\n", encoding="utf-8")
    print(f"PASS: {len(proofs)} preserved regions, {len(observations)} parser-admitted files")


if __name__ == "__main__":
    main()
