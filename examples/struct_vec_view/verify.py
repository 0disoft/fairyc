"""Portable public-CLI smoke; Fairy source owns all token analysis."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

root = Path(__file__).resolve().parent
if len(sys.argv) != 2:
    raise SystemExit("usage: python verify.py <compiler-executable>")
compiler = str(Path(sys.argv[1]).resolve(strict=True))
rows = []


def run(args, code=0, stdout=None, stderr=""):
    result = subprocess.run(args, cwd=root, stdin=subprocess.DEVNULL,
                            capture_output=True, text=True, encoding="utf-8",
                            errors="strict", timeout=60)
    row = dict(argv=args, cwd=str(root), stdout=result.stdout,
               stderr=result.stderr, exit=result.returncode)
    rows.append(row)
    if result.returncode != code or (stdout is not None and result.stdout != stdout) or result.stderr != stderr:
        raise AssertionError(row)
    return row


def stats(tokens, ids, numbers, total, first, first_ids, first_numbers):
    return (f"tokens\t{tokens}\nidentifiers\t{ids}\nnumbers\t{numbers}\nsum\t{total}\n"
            f"first_tokens\t{first}\nfirst_identifiers\t{first_ids}\nfirst_numbers\t{first_numbers}\n")


run([compiler, "check"])
image = root / "build" / ("struct-vec-view.exe" if sys.platform == "win32" else "struct-vec-view")
for release in (False, True):
    build = run([compiler, "build", "--locked"] + (["--release"] if release else []))
    image_hash = hashlib.sha256(image.read_bytes()).hexdigest()
    build["imageSHA256"] = image_hash
    profile = "release" if release else "dev"
    run([compiler, "run", "--locked"] + (["--release"] if release else []), stdout=stats(10, 3, 2, 4000000007, 6, 2, 1))
    for args, expected in (([], stats(10, 3, 2, 4000000007, 6, 2, 1)),
                           ([""], stats(0, 0, 0, 0, 0, 0, 0)),
                           (["x=7;"], stats(4, 1, 1, 7, 4, 1, 1))):
        row = run([str(image)] + args, stdout=expected)
        row.update(profile=profile, imageSHA256=image_hash)
    for source in ("@", "9223372036854775808", "+" * 129):
        row = run([str(image), source], code=2, stdout="",
                  stderr="invalid ASCII input, integer overflow, token limit or allocation failure\n")
        row.update(profile=profile, imageSHA256=image_hash)
receipt = dict(schema="fairyc.struct-view-portable-smoke/v1",
               compilerSHA256=hashlib.sha256(Path(compiler).read_bytes()).hexdigest(), runs=rows)
(root / "build" / "verify-results.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("PASS: public CLI and 12 native outcomes; build/verify-results.json")
