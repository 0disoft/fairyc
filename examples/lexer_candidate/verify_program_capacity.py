"""Verify bounded joined bodies, high-ID unsafe calls and global admission."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
from verify_sdk_vector import REPO, identity


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--images", type=Path, nargs="+", required=True)
    ap.add_argument("--native-runner", type=Path, required=True)
    ap.add_argument("--receipt", type=Path, required=True)
    args = ap.parse_args()
    images = [p.resolve() for p in args.images]
    runner = args.native_runner.resolve()
    output = REPO / ".cache/candidate-scanner/capacity"
    output.mkdir(parents=True, exist_ok=True)
    rows, jobs, inputs = [], [], []
    passed = False
    report = output / "native.json"

    def write(name, data):
        path = output / name
        path.write_text(data, encoding="utf-8", newline="\n")
        inputs.append(path)
        return path

    def run(argv):
        p = subprocess.run([str(x) for x in argv], cwd=REPO, capture_output=True, timeout=240)
        row = dict(argv=[str(x) for x in argv], exit=p.returncode, stdout=p.stdout.decode("utf-8"), stderr=p.stderr.decode("utf-8"))
        rows.append(row)
        return row

    def manifest(name, paths):
        return write(name + ".txt", "\n".join(p.relative_to(REPO).as_posix() for p in paths) + "\n")

    try:
        # Each file stays inside its parser budget; only the joined arena is wide.
        padding = [write(f"padding-{i}.fy", f"module padding{i};" + "".join(f"pub fn i32 value{j}(){{return {j};}}" for j in range(17))
                         + "fn void wide(){" + "discard 1+2;" * 200 + "}") for i in range(8)]
        tail = write("last.fy", "module last;unsafe extern C fn i32 fixture_value();fn void touch(ref mut u8[] data){if(data.len>0){data[0]=7;}}pub fn i32 run(){var u8[1] data=[0];let ref mut u8[] parent=ref mut data;touch(parent[..1]);assert(parent[0]==7);return unsafe{fixture_value()};}")
        main_source = write("main.fy", "module main;import last as last;fn i32 main(){return last::run();}")
        accepted = manifest("accepted", [main_source, *padding, tail])
        # The final module puts foreign calls and mutable reborrows beyond 4096 nodes.
        missing = write("missing.fy", "module last;unsafe extern C fn i32 fixture_value();pub fn i32 run(){return fixture_value();}")
        missing_manifest = manifest("missing-unsafe", [main_source, *padding, missing])
        conflict = write("conflict.fy", "module last;fn ref mut u8[] child(ref mut u8[] data){return data[..1];}pub fn i32 run(){var u8[1] data=[0];let ref mut u8[] parent=ref mut data;let ref mut u8[] held=child(parent);parent[0]=1;held[0]=2;return 7;}")
        conflict_manifest = manifest("borrow-conflict", [main_source, *padding, conflict])
        overflow = [write(f"overflow-{i}.fy", f"module overflow{i};fn void wide(){{" + "discard 1+2;" * 1000 + "}") for i in range(6)]
        overflow_main = write("overflow-main.fy", "module main;import last as last;fn i32 main(){return last::run();}")
        overflow_manifest = manifest("overflow", [overflow_main, *overflow, tail])
        # Single-file name admission keeps its earlier 128-global boundary.
        single = write("single.fy", "module main;" + "".join(f"fn i32 value{i}(){{return {i};}}" for i in range(129)) + "fn i32 main(){return 7;}")
        foreign = write("foreign.c", "#include <stdint.h>\nint32_t fixture_value(void){return 7;}\n")
        for index, image in enumerate(images):
            row = run([image, "--c-modules-file", accepted])
            assert row["exit"] == 0 and not row["stderr"] and row["stdout"], row
            c = write(f"accepted-{index}.c", row["stdout"])
            row["generatedC"] = identity(c)
            row["stdout"] = ""
            jobs.append(dict(name=f"wide-{index}", source=str(c), observe=False, foreignSource=str(foreign)))
            for case, diagnostic in [(missing_manifest, "code 5101"), (conflict_manifest,"code 2101"), (overflow_manifest, "phase 2 code 10")]:
                row = run([image, "--c-modules-file", case])
                assert row["exit"] == 2 and not row["stdout"] and diagnostic in row["stderr"], row
            row = run([image, "--c-file", single])
            assert row["exit"] == 2 and not row["stdout"] and "name-check limit" in row["stderr"], row
        request = write("native-request.json", json.dumps(dict(schema="fairyc.candidate-c-native-request/v1", output=str(output / "native"), jobs=jobs)))
        row = run([runner, request, report])
        assert row["exit"] == 0 and not row["stderr"], row
        native = json.loads(report.read_text(encoding="utf-8"))
        assert len(native) == 2 * len(images) and all(not r.get("error") and r["trap"] == 0 and r["value"] == 7 and not r["stdout"] and not r["stderr"] for r in native), native
        passed = True
    finally:
        args.receipt.resolve().write_text(json.dumps(dict(schema="fairyc.candidate-program-capacity/v1", passed=passed,
            images=[identity(p) for p in images], runner=identity(runner), observer=identity(Path(__file__)),
            sources=[identity(p) for p in sorted((REPO / "examples/lexer_candidate/src").glob("*.fy"))],
            inputs=[identity(p) for p in inputs], lock=identity(REPO / "examples/lexer_candidate/fairy.lock"), executions=rows,
            native=json.loads(report.read_text(encoding="utf-8")) if passed else [], linuxValidated=False), ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"PASS: {len(images)} wide joined programs, {4*len(images)} rejections, {len(native)} native runs")

if __name__ == "__main__":
    main()
