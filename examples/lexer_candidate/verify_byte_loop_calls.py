"""Observe induction provenance and call-bounded loans in authored byte loops."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "byte-loop-calls"
observer.SDK = observer.REPO / "examples/lexer_candidate/src/candidate_limits.fy"
observer.DEPENDENCIES = ()
observer.RUNTIME_NAMES = ()
observer.PREAMBLE = "module main;"
observer.PREFIX = "#include <stddef.h>\n"
observer.SUFFIX = ""
observer.CASE_OBSERVER = Path(__file__)
head = "struct Buffer{usize size;}fn bool append(ref mut Buffer out,ref u8[] value){out.size+=value.len;return true;}"
loop = 'fn bool hex(ref mut Buffer out,ref u8[] value){let ref u8[] digits=b"0123456789abcdef";for(let u8 byte in value){if(!append(out,digits[..1]) || !append(out,digits[1..2])){return false;}}return true;}'
observer.CASES = {
    "consecutive-loop-calls": head + loop + 'fn i32 main(){var Buffer b=Buffer{.size=0};assert(hex(ref mut b,b"ABC"));assert(b.size==6);return 7;}',
    "explicit-loop-reborrows": head + loop.replace("append(out,", "append(ref mut *out,") + 'fn i32 main(){var Buffer b=Buffer{.size=0};assert(hex(ref mut b,b"AB"));assert(b.size==4);return 7;}',
}
observer.REJECT = {
    "held-through-later-argument": (head + 'fn bool bad(ref mut Buffer out,ref u8[] value){for(let u8 byte in value){if(!append(out,{out.size+=1;b"x"})){return false;}}return true;}fn i32 main(){var Buffer b=Buffer{.size=0};assert(bad(ref mut b,b"x"));return 7;}', "code 2101"),
}
if __name__ == "__main__":
    observer.main()
