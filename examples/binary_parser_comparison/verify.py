import argparse
import hashlib
import json
import shutil
import statistics
import subprocess
import tempfile
from pathlib import Path

EXAMPLE = Path(__file__).resolve().parent
PACKAGE = "example.binary-parser-comparison"


def execute(args, cwd, expected=0, timeout=180):
    result = subprocess.run(args, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    if result.returncode != expected:
        raise RuntimeError(
            f"{args}: exit {result.returncode}, expected {expected}\n"
            f"{result.stdout}{result.stderr}"
        )
    return result


def change(source, old, new):
    if source.count(old) != 1:
        raise RuntimeError(f"Expected exactly one match for {old!r}")
    return source.replace(old, new, 1)


def setup(root, source):
    root.mkdir()
    (root / "src").mkdir()
    for name in ("fairy.toml", "fairy.windows.json", "clock.c"):
        shutil.copy2(EXAMPLE / name, root / name)
    (root / "src/main.fy").write_text(source, encoding="utf-8")


def build_zig(zig, root, name, source, compile_only=False):
    path = root / f"{name}.zig"
    path.write_text(source, encoding="utf-8")
    output = root / f"{name}.exe"
    command = [
        zig, "build-exe", path, root / "clock.c", "-lc", "-lkernel32",
        "-target", "x86_64-windows-msvc", "-mcpu", "baseline", "-O", "ReleaseSafe",
        "-fno-emit-bin" if compile_only else f"-femit-bin={output}",
    ]
    if compile_only:
        command.append(f"-femit-llvm-ir={root / (name + '.ll')}")
    execute(command, root, timeout=1200)
    if compile_only and not (root / f"{name}.ll").stat().st_size:
        raise RuntimeError(f"No compiled IR for {name}")
    return output


def verify_guards(fairyc, zig, root, before, fairy_source, zig_source):
    move_at = "    let ref u8[] input = bytes::view(ref buffer);"
    moved = change(
        fairy_source,
        move_at,
        move_at + "\n    release(buffer);",
    ) + "\nfn void release(bytes::Buffer buffer) {}\n"
    borrowed = root / "borrowed"
    setup(borrowed, moved)
    borrow = execute([fairyc, "check"], borrowed, expected=3)
    if "E2105" not in borrow.stderr:
        raise RuntimeError(f"Expected E2105: {borrow.stderr}")

    stale = change(zig_source, "    defer allocator.free(buffer);\n", "")
    stale = change(
        stale, "    const input: []const u8 = buffer;",
        "    const input: []const u8 = buffer;\n    allocator.free(buffer);",
    )
    build_zig(zig, before, "zig-lifetime-check", stale, compile_only=True)

    signature = "fn Result<usize, i32> decode(ref u8[] input) {"
    read = """
    switch (fs::open_read_utf8("audit.bin")) {
        case .ok(file): { discard fs::close(file); }
        case .err(_): {}
    };"""
    undeclared = root / "undeclared"
    setup(undeclared, change(fairy_source, signature, signature + read))
    effect = execute([fairyc, "check"], undeclared, expected=3)
    if "E7102" not in effect.stderr:
        raise RuntimeError(f"Expected E7102: {effect.stderr}")

    after = root / "after"
    setup(after, change(fairy_source, signature, "@effects(filesystem)\n" + signature + read))
    execute([fairyc, "check"], after)
    diff = execute([
        fairyc, "experimental", "atlas-diff", "--baseline", before,
        "--package-uid", PACKAGE, "--max-declarations", "1000",
        "--max-output-bytes", "262144",
        "--symbol", "main::decode", "--deny-new-effect", "filesystem",
    ], after, expected=2)
    if diff.stdout or 'denied new effect "filesystem" on "main::decode"' not in diff.stderr:
        raise RuntimeError(f"Expected Atlas effect denial: {diff.stdout}{diff.stderr}")

    zig_effect = change(
        zig_source,
        "    var offset: usize = 0;",
        '    if (fopen("audit.bin", "rb")) |file| _ = fclose(file);\n    var offset: usize = 0;',
    )
    build_zig(zig, before, "zig-effect-check", zig_effect, compile_only=True)
    return {
        "fairy_borrow": borrow.stderr.strip(),
        "fairy_undeclared_effect": effect.stderr.strip(),
        "fairy_atlas_gate": diff.stderr.splitlines()[0],
        "zig_lifetime_and_file_effect": "compiled to LLVM IR; not executed",
    }


def parse(binary, cwd, data_path, expected_code, expected_sum):
    result = execute([binary], cwd, expected_code)
    if result.stderr:
        raise RuntimeError(f"{binary}: unexpected stderr {result.stderr!r}")
    if expected_code:
        if result.stdout:
            raise RuntimeError(f"{binary}: unexpected output on failure {result.stdout!r}")
        return None
    checksum, ticks, frequency = map(int, result.stdout.split())
    if checksum != expected_sum or ticks < 0 or frequency <= 0:
        raise RuntimeError(f"{binary}: invalid result for {data_path}: {result.stdout!r}")
    return ticks * 1000 / frequency


def compare(fairyc, zig, root):
    fairy_source = (EXAMPLE / "src/main.fy").read_text(encoding="utf-8")
    zig_source = (EXAMPLE / "parser.zig").read_text(encoding="utf-8")
    before = root / "baseline"
    setup(before, fairy_source)
    execute([fairyc, "check"], before)
    execute([fairyc, "build", "--release", "--locked"], before)
    binaries = {
        "Fairy C": before / "build/binary-parser.exe",
        "Zig": build_zig(zig, before, "zig-parser", zig_source),
    }
    guards = verify_guards(fairyc, zig, root, before, fairy_source, zig_source)
    input_path = before / "input.bin"
    cases = {
        "empty": (b"", 0, 0),
        "empty record": (b"\x00\x00", 0, 0),
        "complete records": (b"\x02\x00\x00\xff\x00\x00\x01\x00\x2a", 0, 297),
        "truncated header": (b"\x00\x00\x01", 2, None),
        "truncated payload": (b"\x02\x00\xff", 3, None),
    }
    for data, code, checksum in cases.values():
        input_path.write_bytes(data)
        for binary in binaries.values():
            parse(binary, before, input_path, code, checksum)

    payloads = [bytes((i * 17 + length) % 256 for i in range(length))
                for length in (0, 1, 31, 255, 512)]
    block = b"".join(len(payload).to_bytes(2, "little") + payload for payload in payloads)
    repetitions = (16 << 20) // len(block)
    data = block * repetitions
    checksum = sum(sum(payload) for payload in payloads) * repetitions
    input_path.write_bytes(data)
    samples = {name: [] for name in binaries}
    for trial in range(10):
        order = list(binaries) if trial % 2 == 0 else reversed(binaries)
        for name in order:
            ms = parse(binaries[name], before, input_path, 0, checksum)
            if trial:
                if ms <= 0:
                    raise RuntimeError(f"{name}: nonpositive timed sample")
                samples[name].append(ms)
    samples_summary = {}
    for name, values in samples.items():
        median = statistics.median(values)
        samples_summary[name] = {
            "samples_ms": values, "median_ms": median,
            "MiB/s": len(data) / (1 << 20) * 1000 / median,
        }
    return {
        "fairyc_sha256": hashlib.sha256(fairyc.read_bytes()).hexdigest(),
        "zig_version": execute([zig, "version"], root).stdout.strip(),
        "cases": {name: {"exit": code, "checksum": checksum}
                  for name, (_, code, checksum) in cases.items()},
        "guards": guards,
        "benchmark": {
            "bytes": len(data), "records": len(payloads) * repetitions,
            "checksum": checksum, "input_sha256": hashlib.sha256(data).hexdigest(),
            "samples": samples_summary,
        },
    }


def main():
    parser = argparse.ArgumentParser(description="Compare Fairy C and Zig binary parsers on Windows AMD64.")
    parser.add_argument("--fairyc", type=Path, required=True)
    parser.add_argument("--zig", type=Path, required=True)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="fairyc-zig-parser-") as scratch:
        results = compare(args.fairyc.resolve(strict=True), args.zig.resolve(strict=True), Path(scratch))
    print(json.dumps(results, indent=2))


if __name__ == "__main__":
    main()
