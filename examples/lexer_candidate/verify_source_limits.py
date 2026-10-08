"""Observe file/source-set bounds and execute Fairy-emitted C; no compiler logic."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

PROJECT = Path(__file__).resolve().parent
REPO = PROJECT.parents[1]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--images", nargs=2, required=True, type=Path)
    ap.add_argument("--native-runner", required=True, type=Path)
    ap.add_argument("--receipt", required=True, type=Path)
    args = ap.parse_args()
    images = [p.resolve() for p in args.images]
    output = REPO / ".cache/candidate-source-limits-cli"
    output.mkdir(parents=True, exist_ok=True)
    rows, inputs, jobs = [], [], []

    def sha(p):
        return hashlib.sha256(p.read_bytes()).hexdigest()

    def save(p, value):
        p.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    def run(argv, expected=0):
        r = subprocess.run(list(map(str, argv)), cwd=PROJECT, capture_output=True,
                           encoding="utf-8", timeout=120)
        row = dict(argv=list(map(str, argv)), cwd=str(PROJECT), exit=r.returncode,
                   stdout=r.stdout, stderr=r.stderr)
        rows.append(row)
        save(output / "raw.json", rows)
        assert r.returncode == expected, row
        return row

    def fixture(name, sources):
        paths = []
        for i, data in enumerate(sources):
            p = output / f"{name}-{i}.fy"
            p.write_bytes(data)
            paths.append(p)
        p = output / (name + ".list")
        p.write_text("\n".join(os.path.relpath(q, PROJECT) for q in paths) + "\n", encoding="utf-8")
        inputs.extend(dict(path=str(q), sha256=sha(q), bytes=q.stat().st_size) for q in paths + [p])
        return p

    def padded(name):
        prefix = f"module {name};/*".encode()
        return prefix + b" " * (65536 - len(prefix) - 2) + b"*/"

    cases = (
        ("256-files", [f"module m{i};".encode() for i in range(255)] +
         [b"module main;fn i32 main(){return 7;}"]),
        ("large-spans", [padded("pad"), b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=7};}fn i32 main(){let R r=make();return r.n;}"]),
    )
    for name, sources in cases:
        path = fixture(name, sources)
        emitted = [run([image, "--c-modules-file", path]) for image in images]
        assert emitted[0]["stdout"] == emitted[1]["stdout"] and all(not r["stderr"] for r in emitted)
        p = output / (name + ".observed.c")
        p.write_text('#define FY_CANDIDATE_NO_MAIN\n#include <stdio.h>\n#include <inttypes.h>\n' + emitted[0]["stdout"] +
                     'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t n=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",n);return 0;}\n', encoding="utf-8")
        jobs.append(dict(name=name, source=str(p), observe=True))
    failures = (
        ("file-over", [padded("pad") + b" "], "source limit (65536 bytes)"),
        ("files-over", [b"module m;"] * 257, "module limit (256 files)"),
        ("program-over", [padded(f"p{i}") for i in range(17)], "phase 1 code 10"),
        ("separator-over", [padded(f"p{i}") for i in range(16)], "phase 2 code 10"),
    )
    for name, sources, message in failures:
        path = fixture(name, sources)
        for image in images:
            row = run([image, "--c-modules-file", path], 2)
            assert not row["stdout"] and message in row["stderr"], row
    path = output / "manifest-over.list"
    path.write_bytes(b"a" * 65537)
    for image in images:
        row = run([image, "--c-modules-file", path], 2)
        assert not row["stdout"] and "manifest limit (65536 bytes)" in row["stderr"]
    assert all(sha(Path(row["path"])) == row["sha256"] for row in inputs)
    request, result = output / "request.json", output / "native.json"
    save(request, dict(schema="fairyc.candidate-c-native-request/v1", output=str(output / "native"), jobs=jobs))
    run([args.native_runner.resolve(), request, result])
    native = json.loads(result.read_text(encoding="utf-8"))
    assert len(native) == 4, native
    for row in native:
        assert not row.get("error") and row["trap"] == 0
        assert (row["exit"], row["stdout"], row["value"]) == (0, "", 7), row
    receipt = args.receipt.resolve()
    save(receipt, dict(schema="fairyc.candidate-source-limits-cli/v1", passed=True,
         images=[dict(path=str(p), sha256=sha(p)) for p in images],
         raw=dict(path=str(output / "raw.json"), sha256=sha(output / "raw.json")),
         sourceInputs=inputs, native=native, linuxValidated=False))
    print("PASS: two source-set programs, five bounded errors, four native executions")


if __name__ == "__main__":
    main()
