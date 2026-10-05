"""Build and verify the example with a caller-supplied Fairy C compiler."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent
DEMO = "price = 4000000000 + fee; other = 7;"
USAGE = "usage: token-page-cli [ASCII-source]\n"
HELP = USAGE + "       token-page-cli --eval <ASCII-source>\n"
HELP += "       token-page-cli --eval-file <path>\n"
MAX_I64 = (1 << 63) - 1
LEXEME = re.compile(r"[A-Za-z_][A-Za-z0-9_]*|[0-9]+|[+=;]|[ \t\r\n]+")
EVAL_LEXEME = re.compile(r"[A-Za-z_][A-Za-z0-9_]*|[0-9]+|[+=;*()]|[ \t\r\n]+")


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def byte_location(source, offset):
    # Count newlines independently of the product's byte-by-byte location loop.
    prefix = source[:offset]
    breaks = list(re.finditer(rb"\r\n|\r|\n", prefix))
    return len(breaks) + 1, len(prefix) - (breaks[-1].end() if breaks else 0) + 1


def located_failure(source, label, offset, message):
    line, column = byte_location(source, offset)
    return 2, "", f"{label}:{line}:{column}: error: {message}\n"


def oracle(args, label="<argument>", raw_source=None):
    evaluation = len(args) == 2 and args[0] == "--eval"
    if len(args) > 1 and not evaluation or args in (["--eval"], ["--eval-file"]):
        return 2, "", USAGE
    source = args[1] if evaluation else args[0] if args else DEMO
    source_bytes = source.encode("utf-8") if raw_source is None else raw_source
    if len(source_bytes) > 4095:
        return 2, "", "error: input limit (4095 UTF-8 bytes)\n"
    if args == ["--help"]:
        return 0, HELP, ""
    def failure(message, offset):
        if evaluation:
            byte_offset = len(source[:offset].encode("utf-8")) if raw_source is None else offset
            return located_failure(source_bytes, label, byte_offset, message)
        return 2, "", f"error: {message}\n"

    tokens, offsets, position = [], [], 0
    for match in (EVAL_LEXEME if evaluation else LEXEME).finditer(source):
        if match.start() != position:
            return failure("token limit (128)" if len(tokens) == 128 else "invalid ASCII input", position)
        text, position = match.group(), match.end()
        if text.isspace():
            continue
        if len(tokens) == 128:
            return failure("token limit (128)", match.start())
        kind = "number" if text.isdecimal() else "identifier" if text[0].isalpha() or text[0] == "_" else text
        value = int(text) if kind == "number" else 0
        if value > MAX_I64:
            overflow_digit = next(i for i in range(1, len(text) + 1) if int(text[:i]) > MAX_I64)
            return failure("integer overflow", match.start() + overflow_digit - 1)
        tokens.append((kind, value, text))
        offsets.append(match.start())
    if position != len(source):
        return failure("token limit (128)" if len(tokens) == 128 else "invalid ASCII input", position)
    if evaluation:
        return evaluation_oracle(tokens, offsets, source_bytes, label)
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


def evaluation_oracle(tokens, offsets, source, label):
    # Recursive grammar oracle, independent of the product's iterative stacks.
    # Python arbitrary-precision values make each i64 overflow check explicit.
    cursor, environment = 0, {}

    def failure(message, offset=None):
        if offset is None:
            offset = offsets[cursor] if cursor < len(tokens) else len(source)
        raise ValueError(message, offset)

    def peek():
        return tokens[cursor][0] if cursor < len(tokens) else None

    def take(kind):
        nonlocal cursor
        if peek() != kind:
            failure("invalid evaluation syntax")
        token = tokens[cursor]
        cursor += 1
        return token

    def checked(value, offset):
        if value > MAX_I64:
            failure("evaluation overflow", offset)
        return value

    def primary(depth):
        if peek() == "number":
            return take("number")[1]
        if peek() == "identifier":
            offset = offsets[cursor]
            name = take("identifier")[2]
            if name not in environment:
                failure("undefined variable", offset)
            return environment[name]
        if peek() == "(":
            if depth == 16:
                failure("parenthesis limit (16)")
            take("(")
            value = expression(depth + 1)
            take(")")
            return value
        failure("invalid evaluation syntax")

    def product(depth):
        value = primary(depth)
        while peek() == "*":
            offset = offsets[cursor]
            take("*")
            value = checked(value * primary(depth), offset)
        return value

    def expression(depth):
        value = product(depth)
        while peek() == "+":
            offset = offsets[cursor]
            take("+")
            value = checked(value + product(depth), offset)
        return value

    try:
        while cursor < len(tokens):
            name_offset = offsets[cursor]
            name = take("identifier")[2]
            take("=")
            value = expression(0)
            take(";")
            if name not in environment and len(environment) == 16:
                failure("variable limit (16)", name_offset)
            environment[name] = value
    except ValueError as error:
        return located_failure(source, label, error.args[1], error.args[0])
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
        ("analysis-rejects-multiply", ["x=2*3;"]),
        ("analysis-rejects-open-parenthesis", ["x=(1);"]),
        ("analysis-rejects-close-parenthesis", ["x=1);"]),
        ("file-missing-path", ["--eval-file"]),
        ("file-extra-argument", ["--eval-file", "source.calc", "extra"]),
    ] + evaluation_cases() + arithmetic_cases() + diagnostics_cases()


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


def arithmetic_cases():
    fixtures = [
        ("precedence", "a=2+3*4;b=(2+3)*4;"),
        ("multiple-products", "x=2*3+4*5+6;"),
        ("left-associated-products", "x=2*3*4;"),
        ("grouped-products", "x=(2+3)*(4+5);"),
        ("nested", "x=2*(3+(4*5))+1;"),
        ("reassignment", "a=2+3*4;b=(a+1)*2;a=(a+b)*2;"),
        ("whitespace", "x = ( 2 + 3 ) *\r\n 4 ;"),
        ("zero-left", f"x=0*{MAX_I64};"),
        ("zero-right", f"x={MAX_I64}*0;"),
        ("multiply-one", f"x={MAX_I64}*1;"),
        ("exact-product-max", f"x=7*{MAX_I64 // 7};"),
        ("near-product-max", "x=3037000499*3037000500;"),
        ("product-overflow", "x=3037000500*3037000500;"),
        ("late-product-overflow", f"a=1;b={MAX_I64}*2;"),
        ("reference-overflow", f"a={MAX_I64};b=a*2;"),
        ("zero-after-overflow", f"x={MAX_I64}*2*0;"),
        ("zero-times-overflow-group", f"x=0*({MAX_I64}+1);"),
        ("precedence-avoids-add-overflow", f"x={MAX_I64}+1*0;"),
        ("grouped-add-overflow", f"x=({MAX_I64}+1)*0;"),
        ("depth-limit", "x=" + "(" * 16 + "1" + ")" * 16 + ";"),
        ("depth-overflow", "x=" + "(" * 17 + "1" + ")" * 17 + ";"),
        ("depth-reset", "x=" + "+".join("(1)" for _ in range(17)) + ";"),
        ("stack-pressure", "x=" + "(1+2*" * 16 + "1" + ")" * 16 + ";"),
        ("multiply-token-limit", "x=" + "*".join("0" for _ in range(63)) + ";"),
        ("multiply-token-overflow", "x=" + "*".join("0" for _ in range(64)) + ";"),
        ("missing-close", "x=(1+2;"), ("extra-close", "x=1+2);"),
        ("empty-group", "x=();"), ("trailing-group-operator", "x=(1+);"),
        ("adjacent-groups", "x=(1)(2);"), ("implicit-multiply", "x=2(3);"),
        ("leading-multiply", "x=*1;"), ("double-multiply", "x=1**2;"),
        ("trailing-multiply", "x=1*;"), ("mixed-operators", "x=1+*2;"),
        ("group-undefined", "x=(missing+1)*0;"),
        ("late-syntax", "a=14;b=2*(3+);"),
        ("late-lexical", "a=14;b=2*(3+4);@"),
        ("division-rejected", "x=6/2;"), ("subtraction-rejected", "x=6-2;"),
    ]
    return [("arithmetic-" + name, ["--eval", source]) for name, source in fixtures]


def diagnostics_cases():
    fixtures = [
        ("syntax-LF", "price=10;\ntotal=price+;", 2, 13, "invalid evaluation syntax"),
        ("syntax-CRLF", "price=10;\r\ntotal=price+;", 2, 13, "invalid evaluation syntax"),
        ("syntax-CR", "price=10;\rtotal=price+;", 2, 13, "invalid evaluation syntax"),
        ("tabs", "x=1;\n\ty=missing;", 2, 4, "undefined variable"),
        ("EOF", "x=1;\r\ny=2", 2, 4, "invalid evaluation syntax"),
        ("EOF-after-newline", "x=1;\r\ny=2\r\n", 3, 1, "invalid evaluation syntax"),
        ("EOF-after-CR", "x=1;\ry=2\r", 3, 1, "invalid evaluation syntax"),
        ("EOF-open-group", "x=(1+2", 1, 7, "invalid evaluation syntax"),
        ("byte-invalid", "x=1;\r\n\t@", 2, 2, "invalid ASCII input"),
        ("UTF8-invalid", "x=1;\n\t한", 2, 2, "invalid ASCII input"),
        ("overflow-digit", "x=1;\ny=9223372036854775808;", 2, 21, "integer overflow"),
        ("overflow-product", "x=1;\ny=3037000500*3037000500;", 2, 13, "evaluation overflow"),
        ("overflow-earlier-plus", "x=9223372036854775807+1+0;", 1, 22, "evaluation overflow"),
        ("overflow-group", "x=0*(9223372036854775807+1);", 1, 25, "evaluation overflow"),
        ("mixed-newlines", "x=1;\r\n\ny=2;\r\tz=missing;", 4, 4, "undefined variable"),
        ("variable-limit", "".join(f"v{i}=0;\n" for i in range(17)), 17, 1, "variable limit (16)"),
        ("token-limit", "x=" + "+".join("0" for _ in range(63)) + ";\n@", 2, 1, "token limit (128)"),
        ("paren-limit", "x=\n" + "(" * 17 + "1" + ")" * 17 + ";", 2, 17, "parenthesis limit (16)"),
    ]
    result = []
    for name, source, line, column, message in fixtures:
        args = ["--eval", source]
        expected = (2, "", f"<argument>:{line}:{column}: error: {message}\n")
        assert oracle(args) == expected, (name, oracle(args), expected)
        result.append(("diagnostics-" + name, args))
    return result


def save(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8", newline="\n")


def file_cases(directory):
    fixtures = [
        ("empty", b""), ("whitespace", b" \t\r\n"),
        ("syntax-LF", b"price=10;\ntotal=price+;"),
        ("syntax-CRLF", b"price=10;\r\ntotal=price+;"),
        ("syntax-CR", b"price=10;\rtotal=price+;"),
        ("EOF-newline", b"x=1;\r\ny=2\r\n"),
        ("tabs-undefined", b"x=1;\n\ty=missing;"),
        ("invalid-byte-after-newline", b"x=1;\r\n\t\xff"),
        ("LF", b"price=4000000000+7;\ntotal=(price+3)*2;\n"),
        ("CRLF", b"price=4000000000+7;\r\ntotal=(price+3)*2;\r\n"),
        ("no-final-newline", b"a=2+3*4;b=(2+3)*4;"),
        ("korean-and-space-path", b"a=14;b=a*2;"),
        ("max-size", b"x=1;" + b" " * 4091),
        ("size-overflow", b"x=1;" + b" " * 4092),
        ("larger-file", b" " * 8192),
        ("invalid-byte", b"a=14;\xff"), ("UTF8-BOM", b"\xef\xbb\xbfx=1;"),
        ("non-ASCII", "한글=1;".encode("utf-8")),
        ("embedded-NUL", b"a=14;\x00"),
        ("late-syntax", b"a=14;b=(2+);"),
        ("undefined", b"a=14;b=missing;"),
        ("overflow", f"a=14;b={MAX_I64}*2;".encode("ascii")),
        ("token-overflow", ("x=" + "*".join("0" for _ in range(64)) + ";").encode("ascii")),
    ]
    result = []
    for name, content in fixtures:
        path = directory / ("한글 계산.calc" if name == "korean-and-space-path" else name + ".calc")
        path.write_bytes(content)
        argument = str(path.relative_to(ROOT)) if name == "korean-and-space-path" else str(path)
        expected = oracle(["--eval", content.decode("latin1")], label=argument, raw_source=content)
        result.append(("file-" + name, ["--eval-file", argument], expected, path))
    sample = ROOT / "samples" / "program.calc"
    result.append(("file-sample", ["--eval-file", "samples/program.calc"],
                   oracle(["--eval", sample.read_text(encoding="ascii")], label="samples/program.calc"), sample))
    failure = 2, "", "error: input file read failed\n"
    result.extend([
        ("file-not-found", ["--eval-file", str(directory / "missing.calc")], failure, None),
        ("file-directory", ["--eval-file", str(directory)], failure, None),
        ("file-empty-path", ["--eval-file", ""], failure, None),
        ("file-path-overflow", ["--eval-file", "a" * 4096],
         (2, "", "error: file path limit (4095 UTF-8 bytes)\n"), None),
    ])
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--compiler", required=True, type=Path)
    parser.add_argument("--receipt", type=Path, help="optional JSON receipt, profile logs and retained images")
    options = parser.parse_args()
    compiler = options.compiler.expanduser().resolve(strict=True)
    receipt = options.receipt.expanduser().resolve() if options.receipt else None
    common, profiles = [], []
    # Fixtures stay under this example's generated build directory and are removed
    # automatically. No project source or user input file is used for cleanup.
    build_directory = ROOT / "build"
    build_directory.mkdir(exist_ok=True)

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
            if report["count"] != 4 or report["passed"] != 4 or report["failed"] != 0:
                raise RuntimeError(row)
    with tempfile.TemporaryDirectory(prefix="file-input-", dir=build_directory) as temporary:
        fixture_directory = Path(temporary).resolve()
        assert fixture_directory.parent == build_directory.resolve()
        file_fixtures = file_cases(fixture_directory)
        verify_profiles(compiler, receipt, common, profiles, file_fixtures, run)


def verify_profiles(compiler, receipt, common, profiles, file_fixtures, run):
    total_cases = len(cases()) + len(file_fixtures)
    for profile, flags in [("dev", []), ("release", ["--release"])]:
        rows = []
        row = run([compiler, "build", *flags, "--locked"], rows, "build")
        if row["exit"]:
            raise RuntimeError(row)
        row = run([compiler, "test", "--json", "--native", *flags, "--locked"], rows, "native source tests")
        report = json.loads(row["stdout"]) if row["exit"] == 0 else {}
        if row["exit"] or report["count"] != 4 or report["passed"] != 4 or report["failed"] != 0:
            raise RuntimeError(row)
        image = ROOT / "build" / ("token-page-cli.exe" if os.name == "nt" else "token-page-cli")
        image_hash = sha(image)
        for name, args in cases():
            row = run([image, *args], rows, name)
            expected = oracle(args)
            if (row["exit"], row["stdout"], row["stderr"]) != expected:
                raise RuntimeError(dict(profile=profile, case=name, expected=expected,
                                        actual=(row["exit"], row["stdout"], row["stderr"])))
        for name, args, expected, source in file_fixtures:
            before = sha(source) if source else None
            row = run([image, *args], rows, name)
            if (row["exit"], row["stdout"], row["stderr"]) != expected:
                raise RuntimeError(dict(profile=profile, case=name, expected=expected,
                                        actual=(row["exit"], row["stdout"], row["stderr"])))
            if source:
                row["input"] = dict(path=str(source), bytes=source.stat().st_size,
                                    beforeSHA256=before, afterSHA256=sha(source))
                if row["input"]["afterSHA256"] != before:
                    raise RuntimeError(dict(profile=profile, case=name, input_changed=True))
        record = dict(profile=profile, imageSHA256=image_hash, cases=total_cases, fileCases=len(file_fixtures), passed=True)
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
                   *sorted((ROOT / "src").glob("*.fy")), *sorted((ROOT / "tests").glob("*.fy")),
                   *sorted((ROOT / "samples").glob("*.calc"))]
        save(receipt, dict(schema="fairyc.token-page-cli-verification/v1", cwd=str(ROOT),
                           compiler=dict(path=str(compiler), sha256=sha(compiler)), common=common,
                           sources=[dict(path=str(p.relative_to(ROOT)), sha256=sha(p)) for p in sources],
                           profiles=profiles, executions=2 * total_cases))
    print(f"PASS: {total_cases} cases per profile; {2 * total_cases} native executions")


if __name__ == "__main__":
    main()
