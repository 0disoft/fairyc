module linux_workspace_fs;
import linux_rooted_file as rooted;
import linux_directory as directory;
import linux_host_paths as paths;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Private compatibility facade for workspace readers, not a replacement SDK.
pub struct WorkspaceFileError { u32 status; u32 os_code; usize completed; }
pub enum WorkspacePathKind { File; Directory; Other; }
pub enum WorkspaceFileKind { Disk; }
pub struct WorkspaceFileId { usize device; usize identity; }
@effects(filesystem) @link_name("fairyc_private_linux_root_file_close_v1") unsafe extern C fn u32 close_raw(u32 file);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_info_v1") unsafe extern C fn u32 info_raw(u32 file,ptr mut void output);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_read_v1") unsafe extern C fn u32 read_raw(u32 file,ptr mut u8 output,usize capacity,ptr mut usize count);
@effects(filesystem) pub resource WorkspaceFile { u32 raw; cleanup{unsafe { discard close_raw(this.raw); };} }
struct Row { usize start; usize length; usize size; u32 mode; }
pub resource WorkspaceReadDir { bytes::Buffer names; v::Vec<Row> rows; usize index; cleanup{} }
pub struct WorkspaceDirEntry { bool has_entry; usize name_length; WorkspacePathKind kind; usize size; bool is_reparse; }
fn WorkspaceFileError fail(u32 status,u32 os_code) { return WorkspaceFileError{.status,.os_code,.completed=0}; }
fn WorkspaceFileError problem(rooted::Problem p) { return fail(if(p.stage==2){3}else{1},p.os_code); }
pub fn bool path_kind_is_file(WorkspacePathKind p) { return switch(p){case .File:{true} case .Directory:{false} case .Other:{false}}; }
pub fn bool path_kind_is_directory(WorkspacePathKind p) { return switch(p){case .File:{false} case .Directory:{true} case .Other:{false}}; }
pub fn bool kind_is_disk(WorkspaceFileKind p) { return true; }
pub fn bool file_id_equal(ref WorkspaceFileId a,ref WorkspaceFileId b) { return a.device==b.device && a.identity==b.identity; }
fn WorkspacePathKind classify(u32 mode) { return if((mode&61440)==32768){.File}else if((mode&61440)==16384){.Directory}else{.Other}; }

// Pin every directory prefix; subsequent opens never traverse an unchecked prefix.
@effects(filesystem) pub fn Result<rooted::Directory,WorkspaceFileError> open_directory(ref u8[] path) {
    if(!paths::decode(path).is_ok) { return .err(fail(2,0)); }
    var current=switch(rooted::open(if(path[0]==47){b"/"}else{b"."})) { case .ok(p):{p} case .err(e):{return .err(problem(e));} };
    var usize start=if(path[0]==47){1}else{0};
    while(start<path.len) {
        var usize end=start; while(end<path.len && path[end]!=47) { end+=1; }
        let leaf=path[start..end];
        if(leaf.len!=0 && !bytes::equal(leaf,b".")) {
            let child=switch(rooted::directory(ref current,leaf)) { case .ok(p):{p} case .err(e):{return .err(problem(e));} };
            switch(rooted::close_directory(current)) { case .ok(_):{} case .err(e):{return .err(problem(e));} };
            current=child;
        }
        start=end+1;
    }
    return .ok(current);
}
@effects(filesystem) pub fn Result<WorkspaceReadDir,WorkspaceFileError> read_dir_utf8(ref str path) {
    let root=open_directory(text::as_bytes(path))?; var usize work=0;
    let entries=switch(directory::scan(ref root,4096,ref mut work)) {
        case .ok(p):{p} case .err(e):{return .err(fail(if(e.kind==7){6}else if(e.kind==2){3}else if(e.kind==8){5}else{1},e.os_code));}
    };
    switch(rooted::close_directory(root)) { case .ok(_):{} case .err(e):{return .err(problem(e));} };
    var bytes::Buffer names=bytes::empty(); var v::Vec<Row> rows=v::empty<Row>(); var usize i=0;
    while(i<directory::count(ref entries)) {
        let name=directory::name(ref entries,i); let info=directory::metadata(ref entries,i); let start=bytes::len(ref names);
        if(!bytes::append(ref mut names,name).is_ok || !v::push<Row>(ref mut rows,Row{.start,.length=name.len,.size=info.size,.mode=info.mode}).is_ok) { return .err(fail(3,0)); }
        i+=1;
    }
    return .ok(WorkspaceReadDir{.names,.rows,.index=0});
}
fn usize row_count(ref WorkspaceReadDir stream) { return v::len<Row>(ref stream.rows); }
fn Row row_at(ref WorkspaceReadDir stream,usize index) { return *v::at<Row>(ref stream.rows,index); }
fn ref u8[] row_name(ref WorkspaceReadDir stream,Row row) { return bytes::view(ref stream.names)[row.start..row.start+row.length]; }
pub fn Result<WorkspaceDirEntry,WorkspaceFileError> next_dir(ref mut WorkspaceReadDir stream,ref mut u8[] output) {
    if(stream.index==row_count(stream)) { return .ok(WorkspaceDirEntry{.has_entry=false,.name_length=0,.kind=.Other,.size=0,.is_reparse=false}); }
    let row=row_at(stream,stream.index);
    if(row.length>output.len) { return .err(fail(6,0)); }
    stream.index+=1; let name=row_name(stream,row);
    var usize i=0; while(i<name.len) { output[i]=name[i]; i+=1; }
    return .ok(WorkspaceDirEntry{.has_entry=true,.name_length=name.len,.kind=classify(row.mode),.size=row.size,.is_reparse=(row.mode&61440)==40960});
}
pub fn Result<i32,WorkspaceFileError> close_dir(WorkspaceReadDir directory) { return .ok(0); }
@effects(filesystem) pub fn Result<WorkspaceFile,WorkspaceFileError> open_read_utf8(ref str spelling) {
    let path=text::as_bytes(spelling); if(!paths::decode(path).is_ok) { return .err(fail(2,0)); }
    var usize leaf=0; var usize i=0; while(i<path.len) { if(path[i]==47) { leaf=i+1; } i+=1; }
    if(!rooted::valid_leaf(path[leaf..])) { return .err(fail(2,0)); }
    let parent=open_directory(if(leaf==0){b"."}else if(leaf==1){b"/"}else{path[..leaf-1]})?;
    let file=switch(rooted::open_read(ref parent,path[leaf..])) { case .ok(p):{p} case .err(e):{return .err(problem(e));} };
    switch(rooted::close_directory(parent)) { case .ok(_):{} case .err(e):{return .err(problem(e));} };
    return .ok(WorkspaceFile{.raw=rooted::into_handle(file)});
}
@effects(filesystem) fn Result<rooted::Metadata,WorkspaceFileError> observe(ref WorkspaceFile file) {
    var info=rooted::Metadata{.device=0,.identity=0,.size=0,.modified_seconds=0,.modified_nanos=0,.mode=0};
    let code=unsafe { info_raw(file.raw,ref mut info as ptr mut void) };
    return if(code==0 && rooted::regular(info) && info.modified_nanos<1000000000){.ok(info)}else{.err(fail(1,code))};
}
@effects(filesystem) pub fn Result<WorkspaceFileKind,WorkspaceFileError> kind(ref WorkspaceFile file) { discard observe(file)?; return .ok(.Disk); }
@effects(filesystem) pub fn Result<usize,WorkspaceFileError> size(ref WorkspaceFile file) { let info=observe(file)?; return .ok(info.size); }
@effects(filesystem) pub fn Result<WorkspaceFileId,WorkspaceFileError> file_id(ref WorkspaceFile file) { let info=observe(file)?; return .ok(WorkspaceFileId{.device=info.device,.identity=info.identity}); }
@effects(filesystem) pub fn Result<usize,WorkspaceFileError> read(ref mut WorkspaceFile file,ref mut u8[] output) {
    if(output.len>65536) { return .err(fail(2,0)); } var usize count=0;
    let code=unsafe { read_raw(file.raw,output as ptr mut u8,output.len,ref mut count as ptr mut usize) };
    return if(code==0 && count<=output.len){.ok(count)}else{.err(fail(1,code))};
}
@effects(filesystem) pub fn Result<i32,WorkspaceFileError> close(WorkspaceFile input) {
    var WorkspaceFile file=input;
    let raw=file.raw; file.raw=0; let code=unsafe { close_raw(raw) };
    return if(code==0){.ok(0)}else{.err(fail(1,code))};
}
@effects(filesystem) @link_name("fairyc_private_linux_workspace_path_kind_v1")
unsafe extern C fn u32 path_kind_raw(ptr u8 path,ptr mut u32 kind);
@effects(filesystem) pub fn Result<WorkspacePathKind,WorkspaceFileError> path_kind(ref CStr path) {
    var u32 mode=0; let code=unsafe { path_kind_raw(path as ptr u8,ref mut mode as ptr mut u32) };
    return if(code==0){.ok(classify(mode))}else{.err(fail(1,code))};
}
