"""Observe selected native Fairy tests; all authentication/type logic is Fairy."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

PROJECT = Path(__file__).resolve().parent
REPO = PROJECT.parents[1]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--compiler", required=True)
    ap.add_argument("--receipt", required=True)
    ap.add_argument("--exact", action="append", required=True)
    args = ap.parse_args()
    compiler = Path(args.compiler).resolve()
    receipt = Path(args.receipt).resolve()
    rows = []
    passed = False

    def identity(path):
        return dict(path=str(path), sha256=hashlib.sha256(path.read_bytes()).hexdigest())

    def save():
        receipt.parent.mkdir(parents=True, exist_ok=True)
        receipt.write_text(json.dumps(dict(schema="fairyc.candidate-sdk-native/v1",
            passed=passed, compiler=identity(compiler), executions=rows,
            sources=[identity(p) for p in sorted((PROJECT / "src").glob("*.fy"))],
            tests=[identity(p) for p in sorted((PROJECT / "tests").glob("*.fy"))],
            lock=identity(PROJECT / "fairy.lock"), observer=identity(Path(__file__)),
            linuxValidated=False), ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    try:
        for release in (False, True):
            argv = [str(compiler), "test", "--native", "--locked", "--json"]
            for name in args.exact:
                argv.extend(["--exact", name])
            if release:
                argv.append("--release")
            argv.extend(["--", str(REPO / "internal/sdkstdlib/sources/optional/std/vec.fy")])
            p = subprocess.run(argv, cwd=PROJECT, capture_output=True, encoding="utf-8", timeout=360)
            rows.append(dict(argv=argv, cwd=str(PROJECT), exit=p.returncode, stdout=p.stdout, stderr=p.stderr))
            save()
            assert p.returncode == 0 and not p.stderr, rows[-1]
            report = json.loads(p.stdout)
            assert report["status"] == "ok" and report["passed"] == report["count"] == len(args.exact), report
        passed = True
    finally:
        save()
    print(f"PASS: {len(args.exact)} Fairy native tests in development and release")


if __name__ == "__main__":
    main()
