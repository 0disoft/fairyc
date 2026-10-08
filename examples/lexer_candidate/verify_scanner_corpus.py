"""Feed the checked-in Go lexer corpus to candidate-emitted native scanners."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

from verify_scanner_sdk import observer


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--images", type=Path, nargs="+", required=True)
    ap.add_argument("--receipt", type=Path, required=True)
    args = ap.parse_args()
    repo = observer.REPO
    output = repo / ".cache/candidate-scanner-corpus"
    output.mkdir(parents=True, exist_ok=True)
    rows, profiles = [], []
    passed = False

    def identity(path):
        return dict(path=str(path), sha256=hashlib.sha256(path.read_bytes()).hexdigest())

    def run(argv, env=None):
        start = time.monotonic()
        process = subprocess.Popen([str(x) for x in argv], cwd=repo, env=env,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE, encoding="utf-8",
                                   start_new_session=os.name != "nt")
        try:
            stdout, stderr = process.communicate(timeout=360)
        except subprocess.TimeoutExpired:
            if os.name == "nt":
                subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"], capture_output=True, timeout=30)
            else:
                import signal
                os.killpg(process.pid, signal.SIGKILL)
            stdout, stderr = process.communicate(timeout=30)
            rows.append(dict(argv=[str(x) for x in argv], exit=process.returncode, timedOut=True,
                             stdout=stdout, stderr=stderr))
            raise RuntimeError("scanner corpus process timed out")
        row = dict(argv=[str(x) for x in argv], exit=process.returncode, stdout=stdout, stderr=stderr,
                   seconds=round(time.monotonic() - start, 2))
        rows.append(row)
        return row

    inputs = [repo / "examples/lexer_candidate/fixtures/scanner_protocol.fy",
              observer.SDK, *observer.DEPENDENCIES]
    images = [p.resolve(strict=True) for p in args.images]
    manifest = output / "sources.txt"
    manifest.write_text("".join(p.relative_to(repo).as_posix() + "\n" for p in inputs), encoding="utf-8")
    hashes = [identity(p) for p in inputs]
    native_sources = [repo / "internal/cli/fairy_lexer_corpus_test.go",
                      repo / "internal/cli/fairy_lexer_emitted_corpus_test.go",
                      *sorted((repo / "internal/lexer").glob("*.go")),
                      repo / "internal/source/budget.go"]
    native_hashes = [identity(p) for p in native_sources]
    try:
        generated = []
        for index, exe in enumerate(images):
            row = run([exe, "--c-modules-file", manifest])
            assert row["exit"] == 0 and not row["stderr"] and row["stdout"], row
            unit = output / f"scanner-{index}.c"
            unit.write_text(row["stdout"], encoding="utf-8", newline="\n")
            row["generatedC"] = identity(unit)
            row["stdoutBytes"] = len(row["stdout"].encode("utf-8"))
            row["stdout"] = ""
            generated.append(row["generatedC"]["sha256"])
            if index:
                assert generated[index] == generated[0], "candidate profiles emit different scanner C"
            native_receipt = output / f"native-{index}.json"
            env = dict(os.environ, FAIRYC_CANDIDATE_SCANNER_C=str(unit),
                       FAIRYC_CANDIDATE_SCANNER_RECEIPT=str(native_receipt))
            row = run(["bun", "tools/run-go.ts", "test", "./internal/cli", "-run",
                       "^TestWindowsCandidateEmittedLexerCorpus$", "-count=1", "-v", "-timeout=5m"], env)
            assert row["exit"] == 0 and not row["stderr"] and "--- PASS:" in row["stdout"], row
            native = json.loads(native_receipt.read_text(encoding="utf-8"))
            assert native["passed"] and len(native["profiles"]) == 2 and native["framedCases"] > 0, native
            assert all(p["passed"] and p["batches"] > 0 and p["malformedFrames"] == 23 for p in native["profiles"]), native
            profiles.append(dict(image=identity(exe), generatedC=identity(unit), native=native))
            print(f"PASS: candidate {index}: {native['framedCases']} framed cases in 2 native profiles", flush=True)
        assert hashes == [identity(p) for p in inputs], "scanner inputs changed during verification"
        assert native_hashes == [identity(p) for p in native_sources], "Go observer/reference changed during verification"
        passed = True
    finally:
        receipt = args.receipt.resolve()
        receipt.parent.mkdir(parents=True, exist_ok=True)
        receipt.write_text(json.dumps(dict(schema="fairyc.candidate-scanner-corpus/v1", passed=passed,
            images=[identity(p) for p in images], inputs=hashes, observer=identity(Path(__file__)),
            nativeObserverSources=native_hashes,
            profiles=profiles, executions=rows, linuxValidated=False), ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main()
