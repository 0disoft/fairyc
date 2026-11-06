module linux_package_directory;
import linux_rooted_file as rooted;
import linux_command as command;
import std::bytes as bytes;

@effects(filesystem) @link_name("fairyc_private_linux_package_directory_create_v1") unsafe extern C fn u32 native_create(u32 root,ptr u8 leaf,usize size,ptr mut u32 child,ptr mut u32 parent,ptr mut u32 cleanup_code);
@effects(filesystem) @link_name("fairyc_private_linux_package_directory_edit_v1") unsafe extern C fn u32 native_edit(u32 root,ptr u8 leaf,usize size,ptr mut u32 child,ptr mut u32 parent,ptr mut u32 cleanup_code);
@effects(filesystem) @link_name("fairyc_private_linux_package_directory_publish_v1") unsafe extern C fn u32 native_publish(u32 child,u32 parent,ptr u8 from,usize from_size,u32 root,ptr u8 leaf,usize size);
@effects(filesystem) @link_name("fairyc_private_linux_package_directory_erase_v1") unsafe extern C fn u32 native_erase(u32 child,u32 parent,ptr u8 leaf,usize size);
@effects(filesystem) @link_name("fairyc_private_linux_package_directory_clone_v1") unsafe extern C fn u32 native_clone(u32 root,ptr mut u32 output,ptr mut u32 cleanup_code);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_close_v1") unsafe extern C fn u32 native_close(u32 raw);

// This resource owns both descriptors. The child capability is a nonowning
// view, borrowed through directory(), so consuming close can report both errors
// without moving a resource field or leaving a second automatic close owner.
@effects(filesystem) pub resource Directory {
    rooted::Directory child;u32 raw;u32 parent;bytes::Buffer leaf;bool named;
    cleanup { unsafe { discard native_close(this.parent);discard native_close(this.raw); }; }
}
// 1 input, 2 allocation, 3 native, 5 ABI, 6 phase.
pub struct Problem { u32 kind;u32 os_code;u32 cleanup_code; }
fn Problem fail(u32 kind,u32 os_code,u32 cleanup_code){return Problem{.kind,.os_code,.cleanup_code};}
fn bool handle(u32 raw){return raw>0 && raw<=2147483648;}
fn Result<bytes::Buffer,Problem> path(ref u8[] leaf){
    if(!rooted::valid_leaf(leaf)){return .err(fail(1,0,0));}
    return switch(command::path(leaf)){case .ok(p):{.ok(p)}case .err(p):{.err(fail(if(p.kind==3){2}else{1},0,0))}};
}
// Only unpublished named children carry namespace edit authority. Keeping the
// failed native lease permits bounded tree cleanup after a publication failure.
@effects(filesystem) fn Result<Directory,Problem> acquire(ref rooted::Directory root,ref u8[] leaf,bool create){
    let encoded=switch(path(leaf)){case .ok(p):{p}case .err(p):{return .err(p);}};
    var u32 child=0;var u32 parent=0;var u32 cleanup_code=0;
    let u32 code=unsafe{if(create){native_create(rooted::directory_handle(root),bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref mut child as ptr mut u32,ref mut parent as ptr mut u32,ref mut cleanup_code as ptr mut u32)}else{native_edit(rooted::directory_handle(root),bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref mut child as ptr mut u32,ref mut parent as ptr mut u32,ref mut cleanup_code as ptr mut u32)}};
    let borrowed=rooted::directory_handle(root);
    if(code!=0 || cleanup_code!=0 || !handle(child) || !handle(parent) || child==parent || child==borrowed || parent==borrowed){
        // Refuse malformed native success and reclaim each plausible handle at
        // most once, including partial-error output and duplicate-handle output.
        if(handle(child) && child!=borrowed){let closed=unsafe{native_close(child)};if(cleanup_code==0){cleanup_code=closed;}}
        if(handle(parent) && parent!=child && parent!=borrowed){let closed=unsafe{native_close(parent)};if(cleanup_code==0){cleanup_code=closed;}}
        return .err(fail(if(code!=0){3}else{5},code,cleanup_code));
    }
    let view=unsafe{rooted::from_borrowed_directory_handle(child)};
    return .ok(Directory{.child=view,.raw=child,.parent,.leaf=encoded,.named=true});
}
@effects(filesystem) pub fn Result<Directory,Problem> create(ref rooted::Directory root,ref u8[] leaf){return acquire(root,leaf,true);}
@effects(filesystem) pub fn Result<Directory,Problem> edit(ref rooted::Directory root,ref u8[] leaf){return acquire(root,leaf,false);}
// Borrowed access carries the containing resource's lifetime. It grants no
// ownership of the child, parent descriptor, retained name or phase flag.
pub fn ref rooted::Directory directory(ref Directory value){return ref value.child;}
// Transfer a separate descriptor for the same held object, with no pathname
// reopen and no permission to erase the original lease's retained name.
@effects(filesystem) pub fn Result<rooted::Directory,Problem> duplicate(ref Directory value){
    let borrowed=rooted::directory_handle(ref value.child);var u32 raw=0;var u32 cleanup_code=0;
    let code=unsafe{native_clone(borrowed,ref mut raw as ptr mut u32,ref mut cleanup_code as ptr mut u32)};
    if(code!=0 || cleanup_code!=0 || !handle(raw) || raw==borrowed || raw==value.parent){
        if(handle(raw) && raw!=borrowed && raw!=value.parent){let closed=unsafe{native_close(raw)};if(cleanup_code==0){cleanup_code=closed;}}
        return .err(fail(if(code!=0){3}else{5},code,cleanup_code));
    }
    return .ok(unsafe{rooted::from_owned_directory_handle(raw)});
}
@effects(filesystem) fn u32 publish_native(ref Directory value,ref rooted::Directory destination,ref u8[] leaf){
    return unsafe{native_publish(rooted::directory_handle(ref value.child),value.parent,bytes::view(ref value.leaf) as ptr u8,bytes::len(ref value.leaf),rooted::directory_handle(destination),leaf as ptr u8,leaf.len)};
}
@effects(filesystem) fn u32 erase_native(ref Directory value){
    return unsafe{native_erase(rooted::directory_handle(ref value.child),value.parent,bytes::view(ref value.leaf) as ptr u8,bytes::len(ref value.leaf))};
}
@effects(filesystem) pub fn Result<bool,Problem> publish(ref mut Directory value,ref rooted::Directory destination,ref u8[] leaf){
    if(!value.named){return .err(fail(6,0,0));}
    let encoded=switch(path(leaf)){case .ok(p):{p}case .err(p):{return .err(p);}};
    let code=publish_native(value,destination,bytes::view(ref encoded));
    if(code!=0){return .err(fail(3,code,0));}value.named=false;return .ok(true);
}
@effects(filesystem) pub fn Result<bool,Problem> erase(ref mut Directory value){
    if(!value.named){return .err(fail(6,0,0));}
    let code=erase_native(value);
    if(code!=0){return .err(fail(3,code,0));}value.named=false;return .ok(true);
}
// Close never removes a name or retries an uncertain close. The parent closes
// first, preserving automatic cleanup order; a child error is retained too.
@effects(filesystem) pub fn Result<bool,Problem> close(Directory input){
    var Directory value=input;let parent=value.parent;let child=value.raw;value.parent=0;value.raw=0;
    let parent_code=unsafe{native_close(parent)};let child_code=unsafe{native_close(child)};
    if(parent_code!=0){return .err(fail(3,parent_code,child_code));}
    return if(child_code!=0){.err(fail(3,child_code,0))}else{.ok(true)};
}
