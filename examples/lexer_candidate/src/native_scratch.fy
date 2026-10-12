module native_scratch;
import std::fs as fs;
import std::text as text;
import std::bytes as bytes;

// Only a successful exclusive directory creation confers cleanup ownership.
// The caller supplies the host temp root, never a workspace source directory.
@effects(filesystem) pub resource Directory{bytes::Buffer path;text::CString native;bool owned;cleanup{if(this.owned){discard fs::remove_dir_all(text::cstring_as_cstr(ref this.native));}}}
pub struct Problem{u32 stage;u32 status;u32 os_code;usize completed;}
fn Problem fail(u32 stage,u32 status,u32 os_code,usize completed){return Problem{.stage,.status,.os_code,.completed};}
pub fn ref u8[] path(ref Directory directory){return bytes::view(ref directory.path);}
fn Result<bytes::Buffer,Problem> name(ref u8[] root,usize slot){
    if(root.len==0 || root.len>4000){return .err(fail(1,1,0,0));}var bytes::Buffer output=bytes::empty();if(!bytes::append(ref mut output,root).is_ok){return .err(fail(1,2,0,0));}
    if(root[root.len-1]!=47 && root[root.len-1]!=92 && !bytes::append_byte(ref mut output,47).is_ok){return .err(fail(1,2,0,0));}
    if(!bytes::append(ref mut output,b"fairyc-native-").is_ok || !bytes::append_usize(ref mut output,slot).is_ok){return .err(fail(1,2,0,0));}return .ok(output);
}
@effects(filesystem) pub fn Result<Directory,Problem> create(ref u8[] root){
    var usize slot=0;while(slot<4096){let location=switch(name(root,slot)){case .ok(p):{p}case .err(e):{return .err(e);}};
        let value=switch(text::validate_utf8(bytes::view(ref location))){case .ok(p):{p}case .err(_):{return .err(fail(1,3,0,0));}};let native=switch(text::cstring_from_text(value)){case .ok(p):{p}case .err(e):{return .err(fail(1,if(e.allocation_status!=0){2}else{3},0,0));}};
        switch(fs::create_dir(text::cstring_as_cstr(ref native))){case .ok(_):{return .ok(Directory{.path=location,.native,.owned=true});}case .err(e):{
            // Observation chooses another name, never opens or removes a collision.
            if(fs::path_kind(text::cstring_as_cstr(ref native)).is_ok){slot+=1;continue;}return .err(fail(2,e.status,e.os_code,e.completed));
        }};
    }return .err(fail(2,3,0,0));
}
@effects(filesystem) pub fn Result<usize,Problem> close(Directory input){var Directory directory=input;directory.owned=false;return switch(fs::remove_dir_all(text::cstring_as_cstr(ref directory.native))){case .ok(n):{.ok(n)}case .err(e):{.err(fail(3,e.status,e.os_code,e.completed))}};}
