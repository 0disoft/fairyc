"""Build and verify the example with a caller-supplied Fairy C compiler."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

ROOT = Path(__file__).resolve().parent
DEMO = "price = 4000000000 + fee; other = 7;"
USAGE = "usage: token-page-cli [ASCII-source]\n"
HELP = USAGE + "       token-page-cli --eval <ASCII-source>\n"
MAX_I64 = (1 << 63) - 1
LEXEME = re.compile(r"[A-Za-z_][A-Za-z0-9_]*|[0-9]+|[+=;]|[ \t\r\n]+")


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def oracle(args):
    evaluation = len(args) == 2 and args[0] == "--eval"
    if len(args) > 1 and not evaluation or args == ["--eval"]:
        return 2, "", USAGE
    source = args[1] if evaluation else args[0] if args else DEMO
    if len(source.encode("utf-8")) > 4095:
        return 2, "", "error: input limit (4095 UTF-8 bytes)\n"
    if args == ["--help"]:
        return 0, HELP, ""
    tokens, position = [], 0
    for match in LEXEME.finditer(source):
        if match.start() != position:
            return 2, "", "error: token limit (128)\n" if len(tokens) == 128 else "error: invalid ASCII input\n"
        text, position = match.group(), match.end()
        if text.isspace():
            continue
        if len(tokens) == 128:
            return 2, "", "error: token limit (128)\n"
        kind = "number" if text.isdecimal() else "identifier" if text[0].isalpha() or text[0] == "_" else text
        value = int(text) if kind == "number" else 0
        if value > MAX_I64:
            return 2, "", "error: integer overflow\n"
        tokens.append((kind, value, text))
    if position != len(source):
        return 2, "", "error: token limit (128)\n" if len(tokens) == 128 else "error: invalid ASCII input\n"
    if evaluation:
        return evaluation_oracle(tokens)
    if sum(value for _, value, _ in tokens) > MAX_I64:
        return 2, "", "error: sum overflow\n"
    first_end = next((i + 1 for i, token in enumerate(tokens) if token[0] == ";"), len(tokens))
    rows = []
    for prefix, segment in [("", tokens), ("first_", tokens[:first_end])]:
        rows.extend([(prefix + "tokens", len(segment)),
                     (prefix + "identifiers", sum(kind == "identifier" for kind, _, _ in segment)),
                     (prefix + "numbers", sum(kind == "number" for kind, _, _ in segment)),
                     (prefix + "sum", sum(value for _, value, _ in segment))])
    return 0, "".join(f"{key}\t{value}\n" for key, value in rows), ""


def evaluation_oracle(tokens):
    # Python arbitrary-precision values and insertion-ordered bindings are the oracle.
    remaining = iter(tokens)
    environment = {}

    def failure(message):
        return 2, "", f"error: {message}\n"

    while (name := next(remaining, None)) is not None:
        if name[0] != "identifier" or (equal := next(remaining, None)) is None or equal[0] != "=":
            return failure("invalid evaluation syntax")
        total = 0
        while True:
            term = next(remaining, None)
            if term is None or term[0] not in ("identifier", "number"):
                return failure("invalid evaluation syntax")
            if term[0] == "identifier" and term[2] not in environment:
                return failure("undefined variable")
            total += term[1] if term[0] == "number" else environment[term[2]]
            if total > MAX_I64:
                return failure("evaluation overflow")
            operator = next(remaining, None)
            if operator is None or operator[0] not in ("+", ";"):
                return failure("invalid evaluation syntax")
            if operator[0] == ";":
                break
        if name[2] not in environment and len(environment) == 16:
            return failure("variable limit (16)")
        environment[name[2]] = total
    return 0, "".join(f"{key}\t{value}\n" for key, value in environment.items()), ""


def cases():
    return [
        ("demo", []), ("help", ["--help"]), ("empty", [""]),
        ("whitespace", [" \t\r\n"]), ("leading-space", [" \t" + DEMO]),
        ("line-endings", ["a=12;\r\n b=34;\n"]), ("no-semicolon", ["abc=12+34"]),
        ("punctuation", ["+=;;"]), ("empty-first", [";abc"]), ("zero", ["x=0;"]),
        ("leading-zero", ["0" * 64 + "1"]), ("i64-max", [str(MAX_I64)]),
        ("sum-max", [f"{MAX_I64 - 7} 7"]), ("integer-overflow", [str(MAX_I64 + 1)]),
        ("sum-overflow", [f"{MAX_I64} 1"]), ("late-sum-overflow", [f"1;{MAX_I64}"]),
        ("invalid-character", ["a@"]), ("late-invalid", ["a;@"]),
        ("non-ASCII", ["한글"]), ("negative", ["-1"]), ("vertical-tab", ["\v"]),
        ("capacity", [" ".join(["a"] * 128)]), ("capacity-numbers", [" ".join(["0"] * 128)]),
        ("capacity-trailing-whitespace", [" ".join(["a"] * 128) + " \r\n"]),
        ("capacity-overflow", [" ".join(["a"] * 129)]),
        ("capacity-late-invalid", [" ".join(["a"] * 128) + " @"]),
        ("input-max", ["a" * 4095]), ("input-overflow", ["a" * 4096]),
        ("UTF8-byte-limit", ["가" * 1366]), ("extra-argument", ["a", "b"]),
        ("extra-argument-long-source", ["a" * 4096, "b"]),
    ] + evaluation_cases()


def evaluation_cases():
    fixtures = [
        ("demo", "price = 4000000000 + 7; total = price + 3;"),
        ("empty", ""), ("whitespace", " \t\r\n"),
        ("zero", "x=0;"), ("large", "x=4000000000;"),
        ("i64-max", f"x={MAX_I64};"),
        ("sum-max", f"x={MAX_I64 - 7}+7;"),
        ("separate-max-values", f"x={MAX_I64};y={MAX_I64};"),
        ("reassignment", "x=1; x=x+2; y=x+3; x=0;"),
        ("repeated-reference", "x=4;y=x+x+1;"),
        ("case-sensitive", "A=1;a=2;B=A+a;"),
        ("prefix-names", "ab=2;a=1;abc=ab+a;"),
        ("identifier-characters", "_x1=1; x1=_x1+2;"),
        ("first-definition-order", "z=1;a=2;z=a+1;"),
        ("offsets-and-lines", " \tfirst=1;\r\n second = first + 2;\n"),
        ("leading-zero", "x=" + "0" * 64 + "1;"),
        ("integer-overflow", f"x={MAX_I64 + 1};"),
        ("addition-overflow", f"x={MAX_I64}+1;"),
        ("reference-overflow", f"x={MAX_I64};y=x+1;"),
        ("late-overflow", f"x=1;y={MAX_I64};x=x+y;"),
        ("undefined", "x=missing;"), ("forward-reference", "x=y;y=1;"),
        ("new-self-reference", "x=x+1;"), ("late-undefined", "x=1;y=missing;"),
        ("missing-semicolon", "x=1"), ("late-missing-semicolon", "x=1;y=2"),
        ("missing-equal", "x 1;"), ("missing-term", "x=;"),
        ("trailing-plus", "x=1+;"), ("double-plus", "x=1++2;"),
        ("bare-number", "1;"), ("extra-term", "x=1 2;"),
        ("empty-statement", ";"), ("extra-semicolon", "x=1;;"),
        ("missing-statement", "x=1;y"), ("double-equal", "x==1;"),
        ("negative", "x=-1;"), ("parentheses", "x=(1);"),
        ("late-invalid", "x=1;@"), ("non-ASCII", "한글=1;"),
        ("variable-limit", "".join(f"v{i}=0;" for i in range(16))),
        ("variable-overflow", "".join(f"v{i}=0;" for i in range(17))),
        ("update-at-limit", "".join(f"v{i}=0;" for i in range(16)) + "v0=v15+1;"),
        ("token-limit", "x=" + "+".join("0" for _ in range(63)) + ";"),
        ("token-overflow", "x=" + "+".join("0" for _ in range(64)) + ";"),
        ("input-max", "x=1;" + " " * 4091),
        ("input-overflow", "x=1;" + " " * 4092),
        ("long-name", "x" * 4092 + "=0;"),
        ("help-as-source", "--help"),
    ]
    return [("eval-" + name, ["--eval", source]) for name, source in fixtures] + [
        ("eval-missing-source", ["--eval"]),
        ("eval-extra-argument", ["--eval", "x=1;", "extra"]),
    ]


def save(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8", newline="\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--compiler", required=True, type=Path)
    parser.add_argument("--receipt", type=Path, help="optional JSON receipt, profile logs and retained images")
    options = parser.parse_args()
    compiler = options.compiler.expanduser().resolve(strict=True)
    receipt = options.receipt.expanduser().resolve() if options.receipt else None
    common, profiles = [], []

    def run(argv, rows, name):
        process = subprocess.run([str(arg) for arg in argv], cwd=ROOT, stdin=subprocess.DEVNULL,
                                 capture_output=True, encoding="utf-8", errors="strict", timeout=60)
        row = dict(case=name, argv=[str(arg) for arg in argv], cwd=str(ROOT),
                   exit=process.returncode, stdout=process.stdout, stderr=process.stderr)
        rows.append(row)
        return row

    for args in [["--version"], ["check"], ["check", "--locked"], ["test", "--json", "--locked"]]:
        row = run([compiler, *args], common, "compiler " + " ".join(args))
        if row["exit"]:
            raise RuntimeError(row)
        if args[0] == "test":
            report = json.loads(row["stdout"])
            if report["count"] != 2 or report["passed"] != 2 or report["failed"] != 0:
                raise RuntimeError(row)
    for profile, flags in [("dev", []), ("release", ["--release"])]:
        rows = []
        row = run([compiler, "build", *flags, "--locked"], rows, "build")
        if row["exit"]:
            raise RuntimeError(row)
        row = run([compiler, "test", "--json", "--native", *flags, "--locked"], rows, "native source tests")
        report = json.loads(row["stdout"]) if row["exit"] == 0 else {}
        if row["exit"] or report["count"] != 2 or report["passed"] != 2 or report["failed"] != 0:
            raise RuntimeError(row)
        image = ROOT / "build" / ("token-page-cli.exe" if os.name == "nt" else "token-page-cli")
        image_hash = sha(image)
        for name, args in cases():
            row = run([image, *args], rows, name)
            expected = oracle(args)
            if (row["exit"], row["stdout"], row["stderr"]) != expected:
                raise RuntimeError(dict(profile=profile, case=name, expected=expected,
                                        actual=(row["exit"], row["stdout"], row["stderr"])))
        record = dict(profile=profile, imageSHA256=image_hash, cases=len(cases()), passed=True)
        if receipt:
            saved = receipt.parent / (profile + image.suffix)
            saved.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(image, saved)
            record["imagePath"] = str(saved)
            log = receipt.with_name(receipt.stem + "." + profile + ".json")
            save(log, dict(profile=profile, runs=rows))
            record.update(raw=str(log), rawSHA256=sha(log))
        profiles.append(record)
    if receipt:
        sources = [ROOT / "fairy.toml", ROOT / "fairy.lock", ROOT / "verify.py",
                   *sorted((ROOT / "src").glob("*.fy")), *sorted((ROOT / "tests").glob("*.fy"))]
        save(receipt, dict(schema="fairyc.token-page-cli-verification/v1", cwd=str(ROOT),
                           compiler=dict(path=str(compiler), sha256=sha(compiler)), common=common,
                           sources=[dict(path=str(p.relative_to(ROOT)), sha256=sha(p)) for p in sources],
                           profiles=profiles, executions=2 * len(cases())))
    print(f"PASS: {len(cases())} cases per profile; {2 * len(cases())} native executions")


if __name__ == "__main__":
    main()
