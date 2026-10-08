"""Observe storage and arena bounds with the authored Fairy probe."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile

REPO = Path(__file__).resolve().parents[2]
PROJECT = Path(__file__).resolve().parent


def identity(path):
    return dict(path=str(path), sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--compiler", type=Path, required=True)
    ap.add_argument("--manifest", type=Path, required=True)
    ap.add_argument("--native-test-report", type=Path, required=True)
    ap.add_argument("--receipt", type=Path, required=True)
    ap.add_argument("--probe", choices=("joined_scanner_storage", "partitioned_module_join"), default="joined_scanner_storage")
    args = ap.parse_args()
    compiler, manifest = args.compiler.resolve(), args.manifest.resolve()
    probe = PROJECT / "probes" / (args.probe + ".fy")
    output = REPO / ".cache/candidate-tagged/scanner-storage-observation"
    output.mkdir(parents=True, exist_ok=True)
    workspace = Path(tempfile.mkdtemp(prefix="workspace-", dir=output)).resolve()
    assert workspace.is_relative_to(output.resolve())
    commands, passed = [], False
    tests = json.loads(args.native_test_report.read_text(encoding="utf-8-sig"))
    assert tests["failed"] == 0 and tests["status"] == "ok"
    sources = sorted(PROJECT.joinpath("src").glob("*.fy"))

    def run(argv, cwd):
        p = subprocess.Popen([str(x) for x in argv], cwd=cwd, stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE, encoding="utf-8",
                             start_new_session=os.name != "nt")
        timed_out = False
        try:
            stdout, stderr = p.communicate(timeout=300)
        except subprocess.TimeoutExpired:
            timed_out = True
            if os.name == "nt":
                subprocess.run(["taskkill", "/PID", str(p.pid), "/T", "/F"], capture_output=True, timeout=30)
            else:
                os.killpg(p.pid, signal.SIGKILL)
            stdout, stderr = p.communicate(timeout=30)
        commands.append(dict(argv=[str(x) for x in argv], exit=p.returncode,
                             stdout=stdout, stderr=stderr, timedOut=timed_out))
        print(stdout + stderr, end="", flush=True)
        assert not timed_out and p.returncode == 0 and not stderr, commands[-1]

    try:
        (workspace / "src").mkdir()
        for path in sources:
            shutil.copyfile(path, workspace / "src" / path.name)
        shutil.copyfile(probe, workspace / "src/main.fy")
        shutil.copyfile(PROJECT / "fairy.toml", workspace / "fairy.toml")
        run([compiler, "check"], workspace)
        run([compiler, "build", "--locked"], workspace)
        run([workspace / "build/lexer-candidate.exe", manifest], REPO)
        passed = True
    finally:
        receipt = dict(schema="fairyc.candidate-cooked-storage/v1" if args.probe == "joined_scanner_storage" else "fairyc.candidate-module-units/v1", passed=passed,
                       compiler=identity(compiler), probe=identity(probe), observer=identity(Path(__file__)),
                       manifest=identity(manifest), sources=[identity(p) for p in sources],
                       nativeTests=tests, commands=commands,
                       scope="Development storage and compacted arena observations; full dispatcher compilation remains incomplete")
        args.receipt.resolve().write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
        shutil.rmtree(workspace)


if __name__ == "__main__":
    main()
