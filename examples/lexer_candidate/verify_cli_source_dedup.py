"""Compare real CLI streams/status after sharing frontend stages and writes."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

REPO = Path(__file__).resolve().parents[2]


def identity(path):
    return dict(path=str(path.relative_to(REPO)), sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--images", type=Path, nargs=2, required=True)
    ap.add_argument("--receipt", type=Path, required=True)
    args = ap.parse_args()
    images = [p.resolve() for p in args.images]
    output = REPO / ".cache/candidate-tagged/source-dedup/cli"
    output.mkdir(parents=True, exist_ok=True)
    sources = {
        "valid": "module main;fn i32 main(){return 7;}",
        "parser-error": "module main;fn i32 main(){return 7}",
        "name-error": "module main;fn i32 main(){return missing;}",
        "type-error": "module main;fn i32 main(){return true;}",
        "hir-error": "module main;fn ref u8[] leak(){let u8[1] a=[7];return ref a;}fn i32 main(){return 7;}",
        "payload-and-slice": 'module main;enum E{Some{i32 n;};None;}fn E make(){return .Some{.n=7};}fn i32 main(){let u8[1] a=[7];let ref u8[] b=ref a[..];return switch(make()){case .Some{.n=x}:{x}case .None:{b[0] as i32}};}',
    }
    cases = [["--help"], ["--unknown"], ["--check-types-file"], ["--tokens-file", str(output / "missing.fy")]]
    modes = ["--declarations-file", "--check-names-file", "--bindings-file", "--check-types-file", "--hir-file", "--mir-file", "--c-file"]
    inputs = []
    for name, text in sources.items():
        path = output / (name + ".fy")
        path.write_text(text, encoding="utf-8")
        inputs.append(identity(path))
        cases.extend([mode, str(path)] for mode in modes)
    observations = []
    for argv in cases:
        results = []
        streams = []
        for image in images:
            r = subprocess.run([str(image), *argv], cwd=REPO, capture_output=True, timeout=60)
            streams.append((r.returncode, r.stdout, r.stderr))
            results.append(dict(image=identity(image), exit=r.returncode,
                                stdoutBytes=len(r.stdout), stdoutSHA256=hashlib.sha256(r.stdout).hexdigest(),
                                stderr=r.stderr.decode("utf-8")))
        assert streams[0] == streams[1], (argv, results)
        assert streams[0][0] in (0, 2), (argv, results)
        observations.append(dict(args=argv, results=results))
    # Ensure the HIR/MIR diagnostic branches were exercised, not just early errors.
    hir_errors = [r["results"][0]["stderr"] for r in observations if r["args"][-1].endswith("hir-error.fy") and r["args"][0] in ("--hir-file", "--mir-file", "--c-file")]
    assert len(hir_errors) == 3 and all("HIR bytes " in s and "E2102" in s for s in hir_errors), hir_errors
    args.receipt.resolve().write_text(json.dumps(dict(
        schema="fairyc.candidate-cli-source-dedup/v1", passed=True,
        cases=len(observations), exactStreamsAndStatus=True,
        images=[identity(p) for p in images], inputs=inputs,
        observer=identity(Path(__file__)), observations=observations), indent=2) + "\n", encoding="utf-8")
    print(f"PASS: {len(observations)} CLI cases retain exact stdout, stderr and status")


if __name__ == "__main__":
    main()
