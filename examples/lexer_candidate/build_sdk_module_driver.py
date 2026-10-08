"""Build the opt-in Fairy SDK compiler driver; no compiler logic lives here."""
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
    ap.add_argument("--receipt", type=Path, required=True)
    ap.add_argument("--profiles", nargs="+", choices=("development", "release"), default=["development", "release"])
    args = ap.parse_args()
    compiler = args.compiler.resolve()
    output = REPO / ".cache/candidate-tagged/sdk-module-driver"
    output.mkdir(parents=True, exist_ok=True)
    rows, images = [], []
    passed = False
    # The temporary checkout is contained here before its recursive cleanup.
    workspace = Path(tempfile.mkdtemp(prefix="workspace-", dir=output)).resolve()
    assert workspace.is_relative_to(output.resolve())
    def run(argv):
        p = subprocess.Popen([str(x) for x in argv], cwd=workspace,
                             stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
                             start_new_session=os.name != "nt")
        try:
            stdout, stderr = p.communicate(timeout=300)
        except subprocess.TimeoutExpired:
            if os.name == "nt":
                subprocess.run(["taskkill", "/PID", str(p.pid), "/T", "/F"], capture_output=True, timeout=30)
            else:
                os.killpg(p.pid, signal.SIGKILL)
            stdout, stderr = p.communicate(timeout=30)
            rows.append(dict(argv=[str(x) for x in argv], exit=p.returncode,
                             stdout=stdout, stderr=stderr, timedOut=True))
            raise RuntimeError("SDK observation driver build timed out")
        rows.append(dict(argv=[str(x) for x in argv], exit=p.returncode, stdout=stdout, stderr=stderr))
        assert p.returncode == 0 and not stderr, rows[-1]
        print(stdout, end="", flush=True)
    try:
        (workspace / "src").mkdir()
        for source in PROJECT.joinpath("src").glob("*.fy"):
            shutil.copyfile(source, workspace / "src" / source.name)
        shutil.copyfile(PROJECT / "probes/sdk_module_driver.fy", workspace / "src/main.fy")
        shutil.copyfile(PROJECT / "fairy.toml", workspace / "fairy.toml")
        run([compiler, "check"])
        for profile in args.profiles:
            flags = ["--release"] if profile == "release" else []
            run([compiler, "build", *flags, "--locked"])
            image = output / ("driver-dev.exe" if not flags else "driver-release.exe")
            shutil.copyfile(workspace / "build/lexer-candidate.exe", image)
            images.append(dict(profile=profile, **identity(image)))
        passed = True
    finally:
        receipt = dict(schema="fairyc.candidate-sdk-module-driver-build/v1", passed=passed,
                       compiler=identity(compiler), entry=identity(PROJECT / "probes/sdk_module_driver.fy"),
                       manifest=identity(PROJECT / "fairy.toml"), observer=identity(Path(__file__)),
                       sources=[identity(p) for p in sorted(PROJECT.joinpath("src").glob("*.fy")) if p.name != "main.fy"],
                       commands=rows, images=images,
                       scope="Opt-in SDK compaction API; original main/file CLI dispatcher is unchanged")
        args.receipt.resolve().write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
        shutil.rmtree(workspace)

if __name__ == "__main__":
    main()
