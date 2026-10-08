"""Focused native observations for shared CLI stages and HIR default factories."""
from pathlib import Path
import verify_mir_source_split
import verify_sdk_vector as observer

observer.SUITE = "source-dedup"
observer.CASE_OBSERVER = Path(__file__)
observer.CASES.update({
    "enum-payload-capture": 'enum E{Some{i32 n;};None;}fn E make(){return .Some{.n=7};}fn i32 main(){return switch(make()){case .Some{.n=v}:{v}case .None:{0}};}',
    "slice-defaults-and-byte-loop": 'fn i32 sum(ref u8[] data){var i32 n=0;for(let u8 item in data){n+=item as i32;}return n;}fn i32 main(){let u8[3] a=[2,3,2];let ref u8[] all=ref a[..];assert(sum(all)==7);let ref u8[] part=ref a[1..];assert(sum(part)==5);return sum(all);}',
})

if __name__ == "__main__":
    observer.main()
