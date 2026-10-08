"""Observe selected native Fairy tests; all authentication/type logic is Fairy."""
import argparse
import hashlib
import json
import os
import signal
from pathlib import Path
import subprocess

PROJECT = Path(__file__).resolve().parent
REPO = PROJECT.parents[1]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--compiler", required=True)
    ap.add_argument("--receipt", required=True)
    ap.add_argument("--exact", action="append", required=True)
    ap.add_argument("--profile", choices=("both", "development", "release"), default="both")
    ap.add_argument("--append", action="store_true")
    ap.add_argument("--input", action="append", type=Path, default=[])
    args = ap.parse_args()
    compiler = Path(args.compiler).resolve()
    receipt = Path(args.receipt).resolve()
    rows = []
    passed = False

    def identity(path):
        return dict(path=str(path), sha256=hashlib.sha256(path.read_bytes()).hexdigest())

    if args.append:
        prior = json.loads(receipt.read_text(encoding="utf-8"))
        assert prior["compiler"] == identity(compiler)
        assert prior["lock"] == identity(PROJECT / "fairy.lock")
        assert prior["sources"] == [identity(p) for p in sorted((PROJECT / "src").glob("*.fy"))]
        assert prior["tests"] == [identity(p) for p in sorted((PROJECT / "tests").glob("*.fy"))]
        rows = prior["executions"]
        assert all(r["exit"] == 0 and not r["stderr"] for r in rows)

    def save():
        receipt.parent.mkdir(parents=True, exist_ok=True)
        receipt.write_text(json.dumps(dict(schema="fairyc.candidate-sdk-native/v1",
            passed=passed, compiler=identity(compiler), executions=rows,
            sources=[identity(p) for p in sorted((PROJECT / "src").glob("*.fy"))],
            tests=[identity(p) for p in sorted((PROJECT / "tests").glob("*.fy"))],
            lock=identity(PROJECT / "fairy.lock"), observer=identity(Path(__file__)),
            linuxValidated=False), ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    try:
        profiles = (False, True) if args.profile == "both" else (args.profile == "release",)
        for release in profiles:
            argv = [str(compiler), "test", "--native", "--locked", "--json"]
            for name in args.exact:
                argv.extend(["--exact", name])
            if release:
                argv.append("--release")
            argv.extend(["--", str(REPO / "internal/sdkstdlib/sources/optional/std/vec.fy")])
            argv.extend(str(p.resolve()) for p in args.input)
            # Each exact test has its own C compilation. Bound the whole selection
            # separately and retain raw failure evidence before asserting success.
            timeout = max(360, 240 * len(args.exact) + 90)
            p = subprocess.Popen(argv, cwd=PROJECT, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                 encoding="utf-8", start_new_session=os.name != "nt")
            try:
                stdout, stderr = p.communicate(timeout=timeout)
            except subprocess.TimeoutExpired:
                if os.name == "nt":
                    subprocess.run(["taskkill", "/PID", str(p.pid), "/T", "/F"],
                                   capture_output=True, timeout=30)
                else:
                    os.killpg(p.pid, signal.SIGKILL)
                stdout, stderr = p.communicate(timeout=30)
                rows.append(dict(argv=argv, cwd=str(PROJECT), exit=p.returncode,
                                 stdout=stdout, stderr=stderr, timedOut=True, timeoutSeconds=timeout))
                save()
                raise RuntimeError("selected native tests timed out")
            rows.append(dict(argv=argv, cwd=str(PROJECT), exit=p.returncode, stdout=stdout, stderr=stderr))
            save()
            assert p.returncode == 0 and not stderr, rows[-1]
            report = json.loads(stdout)
            assert report["status"] == "ok" and report["passed"] == report["count"] == len(args.exact), report
        passed = True
    finally:
        save()
    print(f"PASS: {len(args.exact)} Fairy native tests; profile={args.profile}")


if __name__ == "__main__":
    main()
