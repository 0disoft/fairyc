module linux_package_file;
import linux_rooted_file as rooted;
import linux_command as command;
import std::bytes as bytes;

@effects(filesystem) @link_name("fairyc_private_linux_package_file_info_v1") unsafe extern C fn u32 native_info(u32 file,ptr mut void output);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_erase_if_v1") unsafe extern C fn u32 native_erase(u32 root,ptr u8 leaf,usize size,ptr void expected,ptr mut u32 removed,ptr mut u32 closed);
// 1 input, 2 allocation, 3 native, 4 close, 5 ABI, 6 hard-link refusal.
pub struct Problem {u32 kind;u32 os_code;u32 close_code;bool removed;}
fn Problem fail(u32 kind,u32 os_code,u32 close_code,bool removed){return Problem{.kind,.os_code,.close_code,.removed};}
@effects(filesystem) pub fn Result<rooted::Metadata,Problem> observe(ref rooted::File file){
    var info=rooted::Metadata{.device=0,.identity=0,.size=0,.modified_seconds=0,.modified_nanos=0,.mode=0};
    let code=unsafe{native_info(rooted::file_handle(file),ref mut info as ptr mut void)};
    if(code!=0){return .err(fail(if(code==31){6}else{3},code,0,false));}
    if(!rooted::regular(info) || info.identity==0 || info.modified_nanos>=1000000000){return .err(fail(5,0,0,false));}
    return .ok(info);
}
// Private staged-tree cleanup only, under the workspace fetch lock. Preserve
// exact device/inode/size/time/mode and the native single-link condition.
@effects(filesystem) pub fn Result<bool,Problem> erase_if(ref rooted::Directory root,ref u8[] leaf,rooted::Metadata expected){
    if(!rooted::valid_leaf(leaf) || !rooted::regular(expected) || expected.identity==0 || expected.modified_nanos>=1000000000){return .err(fail(1,0,0,false));}
    let encoded=switch(command::path(leaf)){case .ok(p):{p}case .err(p):{return .err(fail(if(p.kind==3){2}else{1},0,0,false));}};
    var u32 removed=0;var u32 closed=0;
    let code=unsafe{native_erase(rooted::directory_handle(root),bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref expected as ptr void,ref mut removed as ptr mut u32,ref mut closed as ptr mut u32)};
    if(removed>1){return .err(fail(5,code,closed,false));}
    if(code!=0 || closed!=0){return .err(fail(if(code!=0){3}else{4},code,closed,removed==1));}
    return .ok(removed==1);
}
