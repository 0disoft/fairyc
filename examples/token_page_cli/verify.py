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
HELP = "usage: token-page-cli [ASCII-source]\n"
MAX_I64 = (1 << 63) - 1
LEXEME = re.compile(r"[A-Za-z_][A-Za-z0-9_]*|[0-9]+|[+=;]|[ \t\r\n]+")


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def oracle(args):
    if len(args) > 1:
        return 2, "", HELP
    source = args[0] if args else DEMO
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
        tokens.append((kind, value))
    if position != len(source):
        return 2, "", "error: token limit (128)\n" if len(tokens) == 128 else "error: invalid ASCII input\n"
    if sum(value for _, value in tokens) > MAX_I64:
        return 2, "", "error: sum overflow\n"
    first_end = next((i + 1 for i, token in enumerate(tokens) if token[0] == ";"), len(tokens))
    rows = []
    for prefix, segment in [("", tokens), ("first_", tokens[:first_end])]:
        rows.extend([(prefix + "tokens", len(segment)),
                     (prefix + "identifiers", sum(kind == "identifier" for kind, _ in segment)),
                     (prefix + "numbers", sum(kind == "number" for kind, _ in segment)),
                     (prefix + "sum", sum(value for _, value in segment))])
    return 0, "".join(f"{key}\t{value}\n" for key, value in rows), ""


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
            if report["count"] != 1 or report["passed"] != 1 or report["failed"] != 0:
                raise RuntimeError(row)
    for profile, flags in [("dev", []), ("release", ["--release"])]:
        rows = []
        row = run([compiler, "build", *flags, "--locked"], rows, "build")
        if row["exit"]:
            raise RuntimeError(row)
        row = run([compiler, "test", "--json", "--native", *flags, "--locked"], rows, "native span test")
        report = json.loads(row["stdout"]) if row["exit"] == 0 else {}
        if row["exit"] or report["count"] != 1 or report["passed"] != 1 or report["failed"] != 0:
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
