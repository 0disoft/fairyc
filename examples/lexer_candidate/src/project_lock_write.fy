module project_lock_write;
import project_lock_reader as reader;
import project_lock_json as json;
import project_path as work;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;

// Stages: 1 strict validation, 2 canonical marshal, 3 noncanonical bytes,
// 4 path/allocation/work, 5 exclusive create/namespace bound, 6 write,
// 7 sync, 8 close, 9 publication. Secondary close/cleanup failures never
// overwrite the primary failure. Only the name this call created is removed.
pub struct Problem{u32 stage;u32 code;u32 os_code;usize completed;u32 close_code;u32 close_os_code;u32 cleanup_code;u32 cleanup_os_code;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code,.os_code=0,.completed=0,.close_code=0,.close_os_code=0,.cleanup_code=0,.cleanup_os_code=0};}
fn Problem io(u32 stage,fs::FileError error){var Problem p=fail(stage,error.status);p.os_code=error.os_code;p.completed=error.completed;return p;}
fn Result<text::CString,Problem> path(ref u8[] root,bool temporary,usize slot,ref mut usize meter){
    if(root.len==0 || root.len>4095 || !work::charge(meter,root.len+64)){return .err(fail(4,1));}
    var bytes::Buffer output=bytes::empty();if(!bytes::append(ref mut output,root).is_ok){return .err(fail(4,2));}
    if(root[root.len-1]!=47 && root[root.len-1]!=92 && !bytes::append_byte(ref mut output,47).is_ok){return .err(fail(4,2));}
    if(temporary){if(!bytes::append(ref mut output,b".fairy-lock-").is_ok || !bytes::append_usize(ref mut output,slot).is_ok || !bytes::append(ref mut output,b".tmp").is_ok){return .err(fail(4,2));}}
    else if(!bytes::append(ref mut output,b"fairy.lock").is_ok){return .err(fail(4,2));}
    if(bytes::len(ref output)>4095){return .err(fail(4,1));}let value=switch(text::validate_utf8(bytes::view(ref output))){case .ok(value):{value}case .err(_):{return .err(fail(4,3));}};
    return switch(text::cstring_from_text(value)){case .ok(value):{.ok(value)}case .err(error):{.err(fail(4,if(error.allocation_status!=0){2}else{3}))}};
}
@effects(filesystem)
fn Result<bool,Problem> finish(fs::File input,ref CStr temporary,ref CStr destination,ref u8[] payload){var fs::File file=input;var Problem problem=fail(0,0);
    switch(fs::write_all(ref mut file,payload)){case .ok(count):{if(count!=payload.len){problem=fail(6,4);problem.completed=count;}}case .err(error):{problem=io(6,error);}};
    if(problem.stage==0){switch(fs::sync(ref mut file)){case .ok(_):{}case .err(error):{problem=io(7,error);}};}
    switch(fs::close(file)){case .ok(_):{}case .err(error):{if(problem.stage==0){problem=io(8,error);}else{problem.close_code=error.status;problem.close_os_code=error.os_code;}}};
    if(problem.stage==0){switch(fs::replace_file(temporary,destination)){case .ok(_):{return .ok(true);}case .err(error):{problem=io(9,error);}};}
    switch(fs::remove_file(temporary)){case .ok(_):{}case .err(error):{problem.cleanup_code=error.status;problem.cleanup_os_code=error.os_code;}};return .err(problem);
}
@effects(filesystem)
pub fn Result<bool,Problem> write(ref u8[] root,bytes::Buffer input,ref mut usize meter){
    if(bytes::len(ref input)>8388608){return .err(fail(1,3));}
    let copy=switch(bytes::copy(bytes::view(ref input))){case .ok(value):{value}case .err(_):{return .err(fail(4,2));}};
    let document=switch(reader::parse(copy,meter)){case .ok(value):{value}case .err(p):{return .err(fail(1,p.code));}};
    let canonical=switch(json::marshal(ref document,meter)){case .ok(value):{value}case .err(code):{return .err(fail(2,code));}};
    if(!work::charge(meter,bytes::len(ref input)+bytes::len(ref canonical)+1)){return .err(fail(4,1));}if(!bytes::equal(bytes::view(ref input),bytes::view(ref canonical))){return .err(fail(3,1));}
    let destination=switch(path(root,false,0,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};
    let root_text=switch(text::validate_utf8(root)){case .ok(value):{value}case .err(_):{return .err(fail(4,3));}};
    let directory=switch(fs::read_dir_utf8(root_text)){case .ok(value):{value}case .err(error):{return .err(io(4,error));}};
    switch(fs::close_dir(directory)){case .ok(_):{}case .err(error):{return .err(io(4,error));}};
    // Reserve bounded preparation/publication work before exclusive creation.
    // Collision attempts charge their paths separately, not unwritten payloads.
    // Cleanup must run even after later errors.
    if(!work::charge(meter,bytes::len(ref canonical)+8)){return .err(fail(4,1));}
    var usize slot=0;while(slot<4096){let temporary=switch(path(root,true,slot,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};
        let file=switch(fs::create_new(text::cstring_as_cstr(ref temporary))){case .ok(value):{value}case .err(error):{
            // A successful path observation only selects a different name; it
            // never authorizes opening, replacing or deleting the colliding one.
            let bool collision=switch(fs::path_kind(text::cstring_as_cstr(ref temporary))){case .ok(_):{true}case .err(_):{false}};
            if(collision){slot+=1;continue;}return .err(io(5,error));
        }};
        return finish(file,text::cstring_as_cstr(ref temporary),text::cstring_as_cstr(ref destination),bytes::view(ref canonical));
    }return .err(fail(5,3));
}
