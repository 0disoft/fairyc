"""Observe the actual file-token CLI component against prior CLI streams."""
from pathlib import Path
import subprocess
import verify_sdk_vector as observer
heap_prefix = observer.PREFIX
import verify_sdk_file as file_runtime

observer.SUITE = "file-tokens"
observer.SDK = observer.REPO / "examples/lexer_candidate/src/file_tokens.fy"
observer.DEPENDENCIES = [observer.REPO / "examples/lexer_candidate/src" / f"{name}.fy" for name in
    ("file_source", "scanner", "candidate_limits", "numbers", "diagnostics", "utf8", "strings", "budget", "token_output")]
observer.DEPENDENCIES += [observer.REPO / "internal/sdkstdlib/sources" / name for name in
    ("std/fs.fy", "std/text.fy", "std/bytes.fy", "std/io.fy", "optional/std/vec.fy")]
observer.RUNTIME_NAMES += ("stage0_owned_text.c", "stage0_vector_storage.c", "stage0_windows_console.c")
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import file_tokens as tokens;"
observer.PREFIX = heap_prefix.replace("#define fy_private_vector_pop_slot_v0 fixture_pop_impl", "")
observer.SUFFIX = ""
root = observer.REPO / ".cache/candidate-tagged/file-tokens"
reference = observer.REPO / ".cache/candidate-fs/candidate-loader-dev.exe"
paths = {name: root / name for name in ("정상.fy", "malformed.fy", "overflow.fy", "missing.fy")}
observer.CASES = {}
observer.STREAMS = {}
observer.VALUES = {}
observer.REJECT = {}
if __name__ == "__main__":
    root.mkdir(parents=True, exist_ok=True)
    paths["정상.fy"].write_bytes('module main;\r\n/// doc\r\nfn i32 main(){let ref str s="한\\n";return 7;}\n'.encode())
    paths["malformed.fy"].write_bytes(b'module main;"unterminated')
    paths["overflow.fy"].write_bytes(b"A" * 65537)
    assert not paths["missing.fy"].exists()
    # No compiler logic runs in the observer. The prior Stage0-built authored CLI
    # supplies exact output/error/status expectations for identical path bytes.
    observer.CASE_CONTEXT = dict(reference=observer.identity(reference), referenceCases=[])
    for name, path in paths.items():
        expected = subprocess.run([str(reference), "--tokens-file", path.as_posix()], cwd=observer.REPO,
                                  capture_output=True, encoding="utf-8", timeout=30)
        key = "valid" if name=="정상.fy" else path.stem
        observer.CASES[key] = '@effects(filesystem,console) fn i32 main(){return tokens::run(b"' + path.as_posix() + '");}'
        observer.STREAMS[key] = ("", expected.stdout, expected.stderr)
        observer.VALUES[key] = expected.returncode
        observer.CASE_CONTEXT["referenceCases"].append(dict(name=key, path=str(path), exit=expected.returncode,
                                                         stdout=expected.stdout, stderr=expected.stderr))
    observer.REJECT = {
        "unused-invalid-user": (observer.CASES["valid"] + "fn i32 unused_invalid(){return true;}", "phase 5 code 1"),
        "private-sdk-target": ('import std::bytes as b;fn i32 main(){return b::signed_decimal_digit(1) as i32;}', "phase 3 code 6"),
        "missing-file-effect": ('fn i32 main(){return tokens::run(b"' + paths["정상.fy"].as_posix() + '");}', "code 7102"),
    }
    observer.main()
