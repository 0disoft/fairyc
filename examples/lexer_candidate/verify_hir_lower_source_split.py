"""Observe unchanged HIR lowering function bodies and self-source parsing."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

REPO = Path(__file__).resolve().parents[2]
SRC = REPO / "examples/lexer_candidate/src"


def identity(path):
    return dict(path=path.relative_to(REPO).as_posix(), bytes=path.stat().st_size,
                sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def functions(text):
    matches = list(re.finditer(r"^(?:pub )?fn [^\n]+", text, re.M))
    return {re.search(r"\b(\w+)\(", m[0])[1]: text[m.start():matches[i+1].start() if i+1<len(matches) else len(text)]
            for i, m in enumerate(matches)}


def normalized(text):
    text = re.sub(r"\b(?:lowering|expr_lower|stmt_lower|bindings_lower)::", "", text)
    return text.replace("pub fn ", "fn ").strip()


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="46893b9a")
    ap.add_argument("--parser", type=Path, required=True)
    ap.add_argument("--receipt", type=Path, required=True)
    args = ap.parse_args()
    original = functions(subprocess.check_output(
        ["git", "show", args.base + ":examples/lexer_candidate/src/hir_lower.fy"],
        cwd=REPO).decode("utf-8"))
    observations = []
    preserved = []
    for module in ["hir_lower", "hir_bindings_lower", "hir_expression_lower", "hir_statement_lower"]:
        path = SRC / (module + ".fy")
        for name, body in functions(path.read_text(encoding="utf-8")).items():
            assert normalized(body) == normalized(original[name]), name
            preserved.append(name)
        r = subprocess.run([str(args.parser.resolve()), "--declarations-file", str(path)],
                           cwd=REPO, capture_output=True, timeout=60)
        row = dict(**identity(path), exit=r.returncode,
                   stdoutBytes=len(r.stdout), stderr=r.stderr.decode("utf-8"))
        observations.append(row)
        assert r.returncode == 0 and not r.stderr, row
    assert sorted(preserved) == sorted(original), preserved
    for name in ["candidate_limits.fy", "parse_tokens.fy", "syntax.fy", "hir_state.fy"]:
        before = subprocess.check_output(["git", "show", args.base + ":examples/lexer_candidate/src/" + name], cwd=REPO)
        assert before.replace(b"\r\n", b"\n") == (SRC / name).read_bytes().replace(b"\r\n", b"\n"), name
    args.receipt.resolve().write_text(json.dumps(dict(
        schema="fairyc.candidate-hir-lower-source-split/v1", passed=True,
        originalCommit=args.base, functionsIdentical=preserved,
        sourceDepthRetained=True, limitsChanged=False, parserImplementationUnchanged=True,
        observer=identity(Path(__file__)), parser=identity(args.parser.resolve()),
        parseObservations=observations), indent=2) + "\n", encoding="utf-8")
    print(f"PASS: {len(preserved)} preserved functions, {len(observations)} parser-admitted files")


if __name__ == "__main__":
    main()
