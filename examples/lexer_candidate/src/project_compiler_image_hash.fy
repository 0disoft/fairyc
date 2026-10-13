module project_compiler_image_hash;
import sha256_stream as sha;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;

// Exact file bytes, with bounded streaming storage and no version fallback.
// A file digest alone does not establish that this is the running compiler.
pub resource Identity{bytes::Buffer text;usize size;cleanup{}}
pub struct Problem{u32 stage;u32 status;u32 os_code;usize completed;}
// 1 path/open, 2 read/size observation, 3 close, 4 empty/non-disk/oversize or
// changed length, 5 work, 6 allocation, 7 reserved for host image-path errors.
fn Problem problem(u32 stage,usize completed){return Problem{.stage,.status=0,.os_code=0,.completed};}
fn Problem io_problem(u32 stage,fs::FileError e,usize completed){return Problem{.stage,.status=fs::error_status(e),.os_code=fs::error_os_code(e),.completed};}
pub fn ref u8[] value(ref Identity p){return bytes::view(ref p.text);}
pub fn usize size(ref Identity p){return p.size;}
pub fn usize maximum(){return 100663296;}
pub fn ref u8[] prefix(){return b"fairyc.compiler-executable-sha256/v1:";}
@effects(filesystem)
pub fn Result<Identity,Problem> read_path(ref u8[] path_bytes,ref mut usize work){
    if(!budget::charge(work,path_bytes.len+1)){return .err(problem(5,0));}let path=switch(text::validate_utf8(path_bytes)){case .ok(p):{p}case .err(_):{return .err(problem(1,0));}};
    var fs::File file=switch(fs::open_read_utf8(path)){case .ok(p):{p}case .err(e):{return .err(io_problem(1,e,0));}};
    if(!budget::charge(work,2)){return .err(problem(5,0));}let observed=switch(fs::size(ref file)){case .ok(n):{n}case .err(e):{return .err(io_problem(2,e,0));}};let kind=switch(fs::kind(ref file)){case .ok(k):{k}case .err(e):{return .err(io_problem(2,e,0));}};
    if(!fs::kind_is_disk(kind) || observed==0 || observed>maximum()){return .err(problem(4,0));}var bytes::Buffer chunk=bytes::empty();if(!budget::charge(work,32769)){return .err(problem(5,0));}if(!bytes::resize_zeroed(ref mut chunk,32768).is_ok){return .err(problem(6,0));}
    var sha::State hash=sha::empty();var usize used=0;while(true){if(!budget::charge(work,1)){return .err(problem(5,used));}let n=switch(fs::read(ref mut file,bytes::view_mut(ref mut chunk))){case .ok(n):{n}case .err(e):{return .err(io_problem(2,e,used));}};
        if(n>32768 || n>maximum()-used){return .err(problem(4,used));}if(n==0){break;}if(!budget::charge(work,(n+63)/64)){return .err(problem(5,used));}if(!sha::update(ref mut hash,bytes::view(ref chunk)[..n])){return .err(problem(4,used));}used+=n;
    }
    if(!budget::charge(work,1)){return .err(problem(5,used));}let final_size=switch(fs::size(ref file)){case .ok(n):{n}case .err(e):{return .err(io_problem(2,e,used));}};
    if(used!=observed || final_size!=observed){return .err(problem(4,used));}switch(fs::close(file)){case .ok(_):{}case .err(e):{return .err(io_problem(3,e,used));}};
    if(!budget::charge(work,prefix().len+65)){return .err(problem(5,used));}var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,prefix()).is_ok || !sha::append_hex(ref hash,ref mut out)){return .err(problem(6,used));}return .ok(Identity{.text=out,.size=used});
}
