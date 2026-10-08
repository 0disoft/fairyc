"""Observe bounded array admission and actual Fairy-generated native code."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

PROJECT = Path(__file__).resolve().parent
REPO = PROJECT.parents[1]
CASES = (
    ("state-fields", "struct Token{i32 n;}struct State{u8[4096] bytes;usize[1153] slots;Token[128] tokens;}fn i32 main(){var State s=State{.bytes=[0;4096],.slots=[0;1153],.tokens=[Token{.n=7};128]};s.bytes[4095]=9;s.slots[1152]=5;let State copy=s;assert(copy.bytes[4095]==9 && copy.slots[1152]==5 && copy.tokens.len==128);return copy.tokens[127].n;}", 0),
    ("i64-copy", "fn i32 main(){let i64[1024] a=[4000000000;1024];let b=a;assert(b[1023]==4000000000 && b.len==1024);return 7;}", 0),
    ("mutable-byte-subview", "fn i32 main(){var u8[4096] a=[0;4096];let ref mut u8[] view=ref mut a;view[4095]=7;let child=view[4000..4096];assert(child.len==96);return child[95] as i32;}", 0),
    ("large-index-trap", "fn i32 main(){let i32[4096] a=[7;4096];return a[4096];}", 6),
)
ERRORS = (
    ("over", "fn i32 main(){let i32[4097] a=[0;4097];return 0;}"),
    ("zero", "struct A{u8[0] bytes;}fn i32 main(){return 0;}"),
    ("decimal-overflow", "struct A{u8[999999999999999999999999999999999999] bytes;}fn i32 main(){return 0;}"),
    ("wrong-length", "fn i32 main(){let usize[1153] a=[0;1152];return 0;}"),
    ("resource-element", "resource R{i32 n;cleanup{}}fn i32 main(){let R[65] a=[R{.n=7};65];return 0;}"),
    ("large-rank-two", "fn i32 main(){let i32[65][1] a=[[7;65];1];return 0;}"),
)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--images", nargs=2, required=True, type=Path)
    ap.add_argument("--native-runner", required=True, type=Path)
    ap.add_argument("--receipt", required=True, type=Path)
    ap.add_argument("--legacy", type=Path)
    args = ap.parse_args()
    images = [p.resolve() for p in args.images]
    folder = REPO / ".cache/candidate-large-arrays-cli"
    folder.mkdir(exist_ok=True)
    rows, fixtures, jobs = [], [], []
    digest = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()

    def save(p, v):
        p.write_text(json.dumps(v, ensure_ascii=False, indent=2) + "\n", encoding="utf-8", newline="\n")

    def run(argv, expected=0):
        r = subprocess.run(list(map(str, argv)), cwd=PROJECT, capture_output=True, encoding="utf-8", timeout=120)
        row = dict(argv=list(map(str, argv)), cwd=str(PROJECT), exit=r.returncode, stdout=r.stdout, stderr=r.stderr)
        rows.append(row)
        save(folder / "raw.json", rows)
        assert r.returncode == expected, row
        return row

    for name, body, trap in CASES:
        p = folder / (name + ".fy")
        p.write_text("module main;" + body, encoding="utf-8", newline="\n")
        fixtures.append(dict(name=name, path=str(p), sha256=digest(p), trap=trap))
        emitted = [run([image, "--c-file", p]) for image in images]
        assert emitted[0]["stdout"] == emitted[1]["stdout"] and all(not r["stderr"] for r in emitted)
        q = folder / (name + ".observed.c")
        q.write_text('#define FY_CANDIDATE_NO_MAIN\n#include <stdio.h>\n#include <inttypes.h>\n' + emitted[0]["stdout"] +
                     'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t n=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",n);return 0;}\n', encoding="utf-8", newline="\n")
        jobs.append(dict(name=name, source=str(q), observe=True))
    for name, body in ERRORS:
        p = folder / (name + ".fy")
        p.write_text("module main;" + body, encoding="utf-8", newline="\n")
        fixtures.append(dict(name=name, path=str(p), sha256=digest(p), rejected=True))
        for image in images:
            row = run([image, "--c-file", p], 2)
            assert not row["stdout"] and row["stderr"], row
    if args.legacy:
        p = folder / "legacy.fy"
        p.write_text("module main;fn i32 main(){let i32[2] a=[3,4];let i32[2][1] b=[a];return b[0][0]+b[0][1];}", encoding="utf-8")
        old = run([args.legacy.resolve(), "--c-file", p])
        assert all(run([image, "--c-file", p])["stdout"] == old["stdout"] for image in images)
    request, report = folder / "request.json", folder / "native.json"
    save(request, dict(schema="fairyc.candidate-c-native-request/v1", output=str(folder / "native"), jobs=jobs))
    run([args.native_runner.resolve(), request, report])
    native = json.loads(report.read_text(encoding="utf-8"))
    expected = {n: t for n, _, t in CASES}
    assert len(native) == 8
    for row in native:
        assert not row.get("error") and row["trap"] == expected[row["name"]], row
        if row["trap"] == 0:
            assert (row["exit"], row["value"], row["stdout"]) == (0, 7, ""), row
    assert all(digest(Path(f["path"])) == f["sha256"] for f in fixtures)
    save(args.receipt.resolve(), dict(schema="fairyc.candidate-large-arrays-cli/v1", passed=True,
         images=[dict(path=str(p), sha256=digest(p)) for p in images], fixtures=fixtures, native=native,
         raw=dict(path=str(folder / "raw.json"), sha256=digest(folder / "raw.json")),
         sources=[dict(path=str(p.relative_to(REPO)), sha256=digest(p)) for p in sorted((PROJECT / "src").glob("*.fy"))],
         lockSHA256=digest(PROJECT / "fairy.lock"), observerSHA256=digest(Path(__file__)), linuxValidated=False))
    print("PASS: four array programs, six errors, eight native runs and legacy C equality")


if __name__ == "__main__":
    main()
