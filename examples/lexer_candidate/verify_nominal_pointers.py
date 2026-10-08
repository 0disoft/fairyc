"""Observe exact struct-pointer identity across source sets and C objects."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "nominal-pointers"
observer.SDK = Path(__file__).parent / "samples/nominal-pointer-provider.fy"
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import pointer_provider as p;"
observer.RUNTIME_NAMES = ()
observer.PREFIX = """
#include <stdint.h>
struct Packet { int32_t value; };
const struct Packet *fixture_packet(const struct Packet *p) { return p; }
int32_t fixture_read_packet(const struct Packet *p) { return p ? p->value : 7; }
void fixture_write_packet(struct Packet *p) { p->value = 7; }
void fixture_slot(void **p) { static int32_t value = 7; *p = &value; }
"""
observer.SUFFIX = ""
observer.CASES = {
    "opaque-output-regression": """
unsafe extern C fn void fixture_slot(ptr mut ptr mut void x);
fn i32 main(){var ptr mut void x=null;unsafe{fixture_slot(ref mut x as ptr mut ptr mut void);};assert(x!=null);return 7;}
""",
    "imported-shared-pointer": """
unsafe extern C fn i32 fixture_read_packet(ptr p::Packet x);
fn i32 main(){let p::Packet x=p::Packet{.value=7};return unsafe{fixture_read_packet(ref x as ptr p::Packet)};}
""",
    "imported-mutable-pointer": """
unsafe extern C fn void fixture_write_packet(ptr mut p::Packet x);
fn i32 main(){var p::Packet x=p::Packet{.value=1};unsafe{fixture_write_packet(ref mut x as ptr mut p::Packet);};return x.value;}
""",
    "imported-pointer-result": """
unsafe extern C fn ptr p::Packet fixture_packet(ptr p::Packet x);
unsafe extern C fn i32 fixture_read_packet(ptr p::Packet x);
fn i32 main(){let p::Packet x=p::Packet{.value=7};let ptr p::Packet a=unsafe{ref x as ptr p::Packet};let ptr p::Packet b=unsafe{fixture_packet(a)};assert(a==b && p::identity(b)==a);return unsafe{fixture_read_packet(b)};}
""",
    "typed-null": """
unsafe extern C fn i32 fixture_read_packet(ptr p::Packet x);
fn i32 main(){let ptr p::Packet x=null;assert(x==null);return unsafe{fixture_read_packet(x)};}
""",
}
observer.TRAPS = {}
observer.REJECT = {
    "wrong-pointee": ("fn i32 main(){let p::Packet x=p::Packet{.value=7};let ptr p::Other a=unsafe{ref x as ptr p::Other};return 7;}", "code 52"),
    "strengthened-borrow": ("fn i32 main(){let p::Packet x=p::Packet{.value=7};let ptr mut p::Packet a=unsafe{ref x as ptr mut p::Packet};return 7;}", "code 52"),
    "missing-unsafe": ("fn i32 main(){let p::Packet x=p::Packet{.value=7};let ptr p::Packet a=ref x as ptr p::Packet;return 7;}", "code 5101"),
    "ordinary-struct-extern": ("struct Local{i32 value;}unsafe extern C fn i32 fixture_read_packet(ptr Local x);fn i32 main(){return 7;}", "code 5102"),
    "resource-pointer": ("resource R{i32 value;cleanup{}}fn i32 main(){let ptr R x=null;return 7;}", "phase 4 code 3"),
}

if __name__ == "__main__":
    observer.main()
