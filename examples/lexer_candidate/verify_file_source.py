"""Compile the actual candidate file loader and observe its native ownership."""
from pathlib import Path
import verify_sdk_vector as observer

heap_prefix = observer.PREFIX
import verify_sdk_file as file_runtime

observer.SUITE = "file-source"
observer.SDK = observer.REPO / "examples/lexer_candidate/src/file_source.fy"
observer.DEPENDENCIES = (
    observer.REPO / "examples/lexer_candidate/src/candidate_limits.fy",
    observer.REPO / "internal/sdkstdlib/sources/std/fs.fy",
    observer.REPO / "internal/sdkstdlib/sources/std/text.fy",
    observer.REPO / "internal/sdkstdlib/sources/std/bytes.fy",
)
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import file_source as files;import std::bytes as b;"
observer.RUNTIME_NAMES += ("stage0_owned_text.c",)
observer.PREFIX = heap_prefix + r'''
#include <windows.h>
static size_t fixture_open_files;
static HANDLE fixture_create_file(LPCWSTR path,DWORD access,DWORD share,LPSECURITY_ATTRIBUTES security,DWORD mode,DWORD flags,HANDLE template_file){
    HANDLE file=CreateFileW(path,access,share,security,mode,flags,template_file);
    if(file!=INVALID_HANDLE_VALUE)fixture_open_files++;
    return file;
}
static BOOL fixture_close_handle(HANDLE file){
    BOOL closed=CloseHandle(file);
    if(closed){if(!fixture_open_files)abort();fixture_open_files--;}
    return closed;
}
size_t fixture_open_count(void){return fixture_open_files;}
#define CreateFileW fixture_create_file
#define CloseHandle fixture_close_handle
'''
observer.SUFFIX = ""
root = observer.REPO / ".cache/candidate-tagged/file-source"
paths = {name: (root / name).as_posix() for name in ("정상.fy", "empty.fy", "boundary.fy", "overflow.fy", "missing.fy")}
body = observer.FOREIGN + r'''
unsafe extern C fn usize fixture_open_count();
fn bool error(Result<b::Buffer,u32> result,u32 expected){return switch(result){case .ok(_):{false}case .err(code):{code==expected}};}
@effects(filesystem) fn i32 main(){
    {
        let b::Buffer input=switch(files::load(b"SMALL")){case .ok(x):{x}case .err(_):{assert(false);return 1;}};
        assert(b::equal(b::view(ref input),b"module main;\nfn i32 main(){return 7;}\n"));
        assert(unsafe{fixture_open_count()}==0 && unsafe{fixture_live()}==1);
    }
    {
        let b::Buffer input=switch(files::load(b"EMPTY")){case .ok(x):{x}case .err(_):{assert(false);return 1;}};
        assert(b::len(ref input)==0 && unsafe{fixture_open_count()}==0);
    }
    {
        let b::Buffer input=switch(files::load(b"BOUNDARY")){case .ok(x):{x}case .err(_):{assert(false);return 1;}};
        assert(b::len(ref input)==65536 && b::view(ref input)[0]==65 && b::view(ref input)[65535]==90);
    }
    assert(error(files::load(b"OVERFLOW"),4));
    assert(error(files::load(b"MISSING"),1));
    assert(error(files::load(b"\xff"),1));
    assert(error(files::load(b""),1));
    unsafe{fixture_fail_next();}assert(error(files::load(b"SMALL"),3));
    assert(unsafe{fixture_open_count()}==0 && unsafe{fixture_live()}==0);
    {
        let b::Buffer input=switch(files::manifest(b"SMALL")){case .ok(x):{x}case .err(_):{assert(false);return 1;}};
        assert(b::len(ref input)==38);
    }
    assert(error(files::manifest(b"OVERFLOW"),4));
    var u8[4096] long_path=[65;4096];assert(error(files::manifest(ref long_path),1));
    assert(unsafe{fixture_open_count()}==0 && unsafe{fixture_live()}==0);
    return 7;
}
'''
for key, name in (("SMALL","정상.fy"),("EMPTY","empty.fy"),("BOUNDARY","boundary.fy"),("OVERFLOW","overflow.fy"),("MISSING","missing.fy")):
    body = body.replace(key, paths[name])
observer.CASES = {"bounded-inputs-and-cleanup": body}
observer.REJECT = {}
if __name__ == "__main__":
    root.mkdir(parents=True, exist_ok=True)
    (root / "정상.fy").write_bytes(b"module main;\nfn i32 main(){return 7;}\n")
    (root / "empty.fy").write_bytes(b"")
    (root / "boundary.fy").write_bytes(b"A" + b"Z" * 65535)
    (root / "overflow.fy").write_bytes(b"A" * 65537)
    assert not (root / "missing.fy").exists()
    observer.main()
