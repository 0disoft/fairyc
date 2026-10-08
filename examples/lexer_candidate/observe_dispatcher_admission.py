"""Inventory imports as harness data, then observe the actual Fairy dispatcher input."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

REPO = Path(__file__).resolve().parents[2]


def identity(path):
    return dict(path=path.relative_to(REPO).as_posix(), bytes=path.stat().st_size,
                sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--image", type=Path, required=True)
    ap.add_argument("--receipt", type=Path, required=True)
    args = ap.parse_args()
    modules = {}
    for directory in [REPO / "examples/lexer_candidate/src", REPO / "internal/sdkstdlib/sources"]:
        for path in sorted(directory.rglob("*.fy")):
            match = re.search(r"^module (\S+);", path.read_text(encoding="utf-8"), re.M)
            if match:
                assert match[1] not in modules, match[1]
                modules[match[1]] = path
    paths = [REPO / "examples/lexer_candidate/src/main.fy"]
    seen = set(paths)
    for path in paths:
        for module in re.findall(r"^import (\S+) as \w+;", path.read_text(encoding="utf-8"), re.M):
            imported = modules[module]
            if imported not in seen:
                seen.add(imported)
                paths.append(imported)
    output = REPO / ".cache/candidate-tagged/dispatcher-admission"
    output.mkdir(parents=True, exist_ok=True)
    manifest = output / (args.receipt.stem + ".txt")
    manifest.write_text("".join(p.relative_to(REPO).as_posix() + "\n" for p in paths), encoding="utf-8")
    image = args.image.resolve()
    argv = [str(image), "--c-modules-file", str(manifest)]
    r = subprocess.run(argv, cwd=REPO, capture_output=True, timeout=240)
    total = sum(p.stat().st_size for p in paths)
    receipt = dict(schema="fairyc.candidate-dispatcher-admission/v1",
                   method="Lexical import inventory creates only the input manifest; actual Fairy driver parses and compiles it",
                   passed=r.returncode == 0 and not r.stderr and bool(r.stdout),
                   root=paths[0].relative_to(REPO).as_posix(), files=len(paths),
                   bytes=total, joinedBytes=total+len(paths)-1,
                   sources=[identity(p) for p in paths], observer=identity(Path(__file__)),
                   image=identity(image), argv=argv, exit=r.returncode,
                   stdoutBytes=len(r.stdout), stderr=r.stderr.decode("utf-8"))
    args.receipt.resolve().write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
    print(f"Observed: {len(paths)} files, {receipt['joinedBytes']} joined bytes, exit {r.returncode}: {receipt['stderr'].strip()}")


if __name__ == "__main__":
    main()
