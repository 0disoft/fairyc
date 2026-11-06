module linux_directory;
import linux_rooted_file as rooted;
import std::bytes as bytes;
import std::vec as v;

@effects(filesystem) @link_name("fairyc_private_linux_directory_scan_open_v1") unsafe extern C fn u32 native_open(u32 root,ptr mut usize output);
@effects(filesystem) @link_name("fairyc_private_linux_directory_scan_next_v1") unsafe extern C fn u32 native_next(usize cursor,ptr mut u8 name,usize capacity,ptr mut u32 length,ptr mut u32 state,ptr mut void metadata);
@effects(filesystem) @link_name("fairyc_private_linux_directory_scan_close_v1") unsafe extern C fn u32 native_close(usize cursor);

@effects(filesystem) resource Cursor { usize raw; cleanup { unsafe { discard native_close(this.raw); }; } }
// 1 input, 2 allocation, 3 open, 4 next, 5 close, 6 ABI, 7 work/items, 8 UTF-8/name.
pub struct Problem { u32 kind; u32 os_code; usize completed; u32 close_code; }
pub resource Entries { v::Vec<bytes::Buffer> names; v::Vec<rooted::Metadata> facts; cleanup {} }
fn Problem fail(u32 kind,u32 os_code,usize completed) { return Problem{.kind,.os_code,.completed,.close_code=0}; }
pub fn usize work_limit() { return 16777216; }
pub fn usize count(ref Entries p) { return v::len<bytes::Buffer>(ref p.names); }
pub fn ref u8[] name(ref Entries p,usize index) { return bytes::view(v::at<bytes::Buffer>(ref p.names,index)); }
pub fn rooted::Metadata metadata(ref Entries p,usize index) { return *v::at<rooted::Metadata>(ref p.facts,index); }
fn bool charge(ref mut usize work,usize cost) {
    if(*work>work_limit() || cost>work_limit()-*work) { return false; }; *work+=cost; return true;
}
fn rooted::Metadata empty_metadata() { return rooted::Metadata{.device=0,.identity=0,.size=0,.modified_seconds=0,.modified_nanos=0,.mode=0}; }

@effects(filesystem) fn Result<Entries,Problem> collect(ref mut Cursor cursor,usize maximum,ref mut usize work) {
    var v::Vec<bytes::Buffer> names=v::empty<bytes::Buffer>();
    var v::Vec<rooted::Metadata> facts=v::empty<rooted::Metadata>();
    var u8[256] raw=[0;256];
    while(true) {
        let completed=v::len<bytes::Buffer>(ref names);
        // Every OS record and the EOF probe cost work, including skipped names.
        if(!charge(work,297)) { return .err(fail(7,0,completed)); }
        var u32 length=0;var u32 state=0;var info=empty_metadata();
        let code={ let ref mut u8[] writable=ref mut raw; unsafe { native_next(cursor.raw,writable as ptr mut u8,256,ref mut length as ptr mut u32,ref mut state as ptr mut u32,ref mut info as ptr mut void) } };
        if(code!=0) { return .err(fail(4,code,completed)); }
        if(state==0) { if(length!=0) { return .err(fail(6,0,completed)); } return .ok(Entries{.names,.facts}); }
        if((state!=1 && state!=2) || length==0 || length>255) { return .err(fail(6,0,completed)); }
        let ref u8[] view=ref raw;let leaf=view[..(length as usize)];
        if(bytes::equal(leaf,b".") || bytes::equal(leaf,b"..")) { if(state!=2) { return .err(fail(6,0,completed)); } continue; }
        if(!rooted::valid_leaf(leaf)) { return .err(fail(8,0,completed)); }
        if(state==2) { continue; }
        let kind=info.mode&61440;
        if(info.modified_nanos>=1000000000 || !(kind==4096 || kind==8192 || kind==16384 || kind==24576 || kind==32768 || kind==40960 || kind==49152)) { return .err(fail(6,0,completed)); }
        if(completed>=maximum || !charge(work,(length as usize)+40)) { return .err(fail(7,0,completed)); }
        let owned=switch(bytes::copy(leaf)) { case .ok(p): { p } case .err(_): { return .err(fail(2,0,completed)); } };
        if(!v::push<bytes::Buffer>(ref mut names,owned).is_ok || !v::push<rooted::Metadata>(ref mut facts,info).is_ok) { return .err(fail(2,0,completed)); }
    } return .err(fail(6,0,0));
}

// Returned names and metadata own detached storage; an error never returns a prefix.
@effects(filesystem) fn Result<Entries,Problem> scan_bounded(ref rooted::Directory root,usize maximum,ref mut usize work) {
    if(!charge(work,1)) { return .err(fail(7,0,0)); }
    var usize raw=0;let opened=unsafe { native_open(rooted::directory_handle(root),ref mut raw as ptr mut usize) };
    if(opened!=0) { return .err(fail(3,opened,0)); } if(raw==0) { return .err(fail(6,0,0)); }
    var cursor=Cursor{.raw};let Result<Entries,Problem> result=collect(ref mut cursor,maximum,work);
    cursor.raw=0;let closed=unsafe { native_close(raw) };
    switch(result) {
        case .ok(entries): { if(closed!=0) { return .err(fail(5,closed,count(ref entries))); } return .ok(entries); }
        case .err(original): { var problem=original;problem.close_code=closed;return .err(problem); }
    };return .err(fail(6,0,0));
}

@effects(filesystem) pub fn Result<Entries,Problem> scan(ref rooted::Directory root,usize maximum,ref mut usize work) {
    if(maximum==0 || maximum>4096) { return .err(fail(1,0,0)); }
    return scan_bounded(root,maximum,work);
}
// Package ZIP source policy permits at most 8,192 explicit/implicit path rows.
// Other directory users retain the original 4,096-item admission above.
@effects(filesystem) pub fn Result<Entries,Problem> scan_packages(ref rooted::Directory root,ref mut usize work) {
    return scan_bounded(root,8192,work);
}

// Edit spelling checks need only one exact name, not an allocated package list.
// Share this budget across path components (or an entire edit snapshot).
pub struct EditBudget { usize entries; usize work; }
pub struct Lookup { bool found; rooted::Metadata info; }
pub fn usize edit_entry_limit(){return 100000;}
pub fn usize edit_work_limit(){return 67108864;}
fn bool edit_charge(ref mut EditBudget budget,usize cost){
    if(budget.entries>edit_entry_limit() || budget.work>edit_work_limit() || cost>edit_work_limit()-budget.work){return false;}
    budget.work+=cost;return true;
}
@effects(filesystem) fn Result<Lookup,Problem> find_edit(ref mut Cursor cursor,ref u8[] wanted,ref mut EditBudget budget){
    var u8[256] raw=[0;256];
    while(true){
        // Charge skipped records and EOF too, so malformed/vanishing streams
        // cannot evade the work limit. Dot entries do not spend the entry limit.
        if(!edit_charge(budget,297)){return .err(fail(7,0,budget.entries));}
        var u32 length=0;var u32 state=0;var info=empty_metadata();
        let code={let ref mut u8[] output=ref mut raw;unsafe{native_next(cursor.raw,output as ptr mut u8,256,ref mut length as ptr mut u32,ref mut state as ptr mut u32,ref mut info as ptr mut void)}};
        if(code!=0){return .err(fail(4,code,budget.entries));}
        if(state==0){if(length!=0){return .err(fail(6,0,budget.entries));}return .ok(Lookup{.found=false,.info=empty_metadata()});}
        if((state!=1 && state!=2) || length==0 || length>255){return .err(fail(6,0,budget.entries));}
        let ref u8[] view=ref raw;let leaf=view[..(length as usize)];
        if(view[length as usize]!=0){return .err(fail(6,0,budget.entries));}
        var usize i=0;while(i<leaf.len){if(leaf[i]==0 || leaf[i]==47){return .err(fail(6,0,budget.entries));}i+=1;}
        if(bytes::equal(leaf,b".") || bytes::equal(leaf,b"..")){if(state!=2){return .err(fail(6,0,budget.entries));}continue;}
        if(budget.entries==edit_entry_limit()){return .err(fail(7,0,budget.entries));}budget.entries+=1;
        if(state==2){continue;}
        // Unrelated names need not be portable or UTF-8, just as in Go's
        // exact spelling check. Regular hard links are ordinary read inputs.
        if(bytes::equal(leaf,wanted)){
            if(info.modified_nanos>=1000000000){return .err(fail(6,0,budget.entries));}
            return .ok(Lookup{.found=true,.info});
        }
    }return .err(fail(6,0,budget.entries));
}
@effects(filesystem) pub fn Result<Lookup,Problem> lookup_edit(ref rooted::Directory root,ref u8[] leaf,ref mut EditBudget budget){
    if(!rooted::valid_leaf(leaf)){return .err(fail(1,0,budget.entries));}
    if(!edit_charge(budget,1)){return .err(fail(7,0,budget.entries));}
    var usize raw=0;let opened=unsafe{native_open(rooted::directory_handle(root),ref mut raw as ptr mut usize)};
    if(opened!=0){return .err(fail(3,opened,budget.entries));}if(raw==0){return .err(fail(6,0,budget.entries));}
    var cursor=Cursor{.raw};let result=find_edit(ref mut cursor,leaf,budget);
    cursor.raw=0;let closed=unsafe{native_close(raw)};
    return switch(result){
        case .ok(value):{if(closed!=0){.err(fail(5,closed,budget.entries))}else{.ok(value)}}
        case .err(original):{var p=original;p.close_code=closed;.err(p)}
    };
}
