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
@effects(filesystem) pub fn Result<Entries,Problem> scan(ref rooted::Directory root,usize maximum,ref mut usize work) {
    if(maximum==0 || maximum>4096) { return .err(fail(1,0,0)); }
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
