module linux_starter;
import project_starter as starter;
import project_manifest_values as values;
import linux_rooted_file as rooted;
import linux_command as command;
import std::bytes as bytes;
@effects(filesystem) @link_name("fairyc_private_linux_starter_create_v1")
unsafe extern C fn u32 create_raw(u32 parent,ptr u8 name,usize name_size,ptr u8 data,usize size,ptr void lengths,ptr mut u32 slot,ptr mut u32 rollback);
@repr(C) struct Sizes{usize a;usize b;usize c;usize d;usize e;}
// Same product failure categories as the existing starter: parent/existing/permission/I/O/rollback/internal.
pub struct Problem{u32 kind;usize slot;}
fn Problem fail(u32 kind,usize slot){return Problem{.kind,.slot};}
pub fn Problem classify(u32 code,u32 slot,u32 rollback){return fail(if(rollback!=0){5}else if(slot==0 && code==17){2}else if(code==1 || code==13){3}else{4},slot as usize);}
@effects(filesystem) pub fn Result<bool,Problem> create(ref u8[] name,ref u8[] uid,ref u8[] language){
    if(!values::name(name) || !values::segment(name) || !values::uid(uid)){return .err(fail(8,0));}
    let encoded=switch(command::path(name)){case .ok(p):{p}case .err(_):{return .err(fail(8,0));}};
    var bytes::Buffer data=bytes::empty();var usize[5] lengths=[0;5];var usize i=0;
    while(i<5){let body=switch(starter::body(i,name,uid,language)){case .ok(p):{p}case .err(_):{return .err(fail(8,i+3));}};
        let content=bytes::view(ref body);if(content.len>65536-bytes::len(ref data) || !bytes::append(ref mut data,content).is_ok){return .err(fail(8,i+3));}lengths[i]=content.len;i+=1;
    }
    let parent=switch(rooted::open(b".")){case .ok(p):{p}case .err(_):{return .err(fail(1,0));}};
    var u32 slot=0;var u32 rollback=0;let sizes=Sizes{.a=lengths[0],.b=lengths[1],.c=lengths[2],.d=lengths[3],.e=lengths[4]};
    let code=unsafe{create_raw(rooted::directory_handle(ref parent),bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),bytes::view(ref data) as ptr u8,bytes::len(ref data),ref sizes as ptr void,ref mut slot as ptr mut u32,ref mut rollback as ptr mut u32)};
    let closed=rooted::close_directory(parent).is_ok;
    if(slot>7 || rollback>1 || code==0 && rollback!=0){return .err(fail(8,0));}
    if(code!=0){return .err(classify(code,slot,rollback));}
    if(!closed){return .err(fail(4,0));}return .ok(true);
}
