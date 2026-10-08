"""Execute original File SDK bodies with the original Windows file runtime."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "sdk-file"
observer.SDK = observer.REPO / "internal/sdkstdlib/sources/std/fs.fy"
observer.DEPENDENCIES = (observer.REPO / "internal/sdkstdlib/sources/std/bytes.fy",)
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import std::fs as f;"
observer.RUNTIME_NAMES = (
    "stage0_system_heap.c", "stage0_buffer.c", "stage0_windows_file.c",
    "stage0_windows_file_portable.c", "stage0_windows_file_metadata.c",
    "stage0_windows_file_link.c", "stage0_windows_file_remove_tree.c",
    "stage0_windows_file_buffer.c", "stage0_windows_directory.c", "stage0_text_view.c",
)
observer.RUNTIME_INCLUDES = True
observer.PREFIX = "#include <stddef.h>\n#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)\n"
observer.SUFFIX = ""
# Paths are owned by this observer's cache directory; the Fairy program removes
# its two files before successful exit, including after replacement consumes one.
target = (observer.REPO / ".cache/candidate-tagged/sdk-file/파일-target.tmp").as_posix()
replacement = (observer.REPO / ".cache/candidate-tagged/sdk-file/파일-replacement.tmp").as_posix()
observer.CASES = {
    "write-sync-replace-read-close": r'''
@effects(filesystem) fn i32 main(){
    let f::File original=switch(f::create_truncate_utf8("TARGET")){case .ok(x):{x}case .err(_):{assert(false);return 1;}};
    assert(f::close(original).is_ok);
    var f::File next=switch(f::create_truncate_utf8("REPLACEMENT")){case .ok(x):{x}case .err(_):{assert(false);return 1;}};
    let ref u8[] payload=b"한\0Z";
    assert(switch(f::write(ref mut next,payload[..3])){case .ok(n):{n==3}case .err(_):{false}});
    assert(switch(f::write_all(ref mut next,payload[3..])){case .ok(n):{n==payload.len-3}case .err(_):{false}});
    assert(f::sync(ref mut next).is_ok);assert(f::close(next).is_ok);
    assert(f::replace_file(c"REPLACEMENT",c"TARGET").is_ok);
    assert(switch(f::open_read_utf8("REPLACEMENT")){case .ok(_):{false}case .err(e):{e.status!=0}});
    var f::File file=switch(f::open_read_utf8("TARGET")){case .ok(x):{x}case .err(_):{assert(false);return 1;}};
    assert(switch(f::kind(ref file)){case .ok(k):{switch(k){case .Disk:{true}case .Character:{false}case .Pipe:{false}case .Unknown:{false}}}case .err(_):{false}});
    var u8[8] data=[99;8];let ref mut u8[] view=ref mut data;
    assert(switch(f::read(ref mut file,view)){case .ok(n):{n==payload.len}case .err(_):{false}});
    var usize i=0;while(i<payload.len){assert(view[i]==payload[i]);i+=1;}assert(view[5]==99);
    assert(switch(f::read(ref mut file,view)){case .ok(n):{n==0}case .err(_):{false}});
    assert(f::close(file).is_ok && f::remove_file(c"TARGET").is_ok);
    assert(switch(f::open_read_utf8("TARGET")){case .ok(_):{false}case .err(e):{e.status!=0}});
    assert(switch(f::open_read_utf8("bad\0path")){case .ok(_):{false}case .err(e):{e.status==2}});
    return 7;
}
'''.replace("TARGET",target).replace("REPLACEMENT",replacement),
}
observer.REJECT = {
    "missing-filesystem-effect": ('fn i32 main(){discard f::open_read_utf8("x");return 7;}', "code 7102"),
    "stolen-file-symbol": ('@effects(filesystem)@link_name("fy_std_file_open_utf8_v0")unsafe extern C fn usize stolen(i32 value);fn i32 main(){return 7;}', "code 5106"),
}
if __name__ == "__main__":
    observer.main()
