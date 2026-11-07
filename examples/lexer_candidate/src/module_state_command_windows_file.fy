module module_state_command_windows_file;
import windows_rooted_file as host;
import windows_cancel as cancel;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::text as text;

// Stable stages: 1 path/options/path work, 2 cancellation observation,
// 3 retained parent open, 4 retained file open, 5 metadata/consistency,
// 6 data budget/allocation/read/size, 7 explicit file close, 8 parent close.
// Codes: 1001 invalid input/changed metadata, 1002 work/size, 1003 allocation,
// 1009 canceled, 1010 token observation error, 0 original native I/O failure.
pub struct Failure{u32 stage;u32 code;u32 os_code;}
fn Failure fail(u32 stage,u32 code,u32 os_code){return Failure{.stage,.code,.os_code};}
fn Failure native(u32 stage,host::Problem p){return fail(stage,if(p.stage==1){1001}else if(p.stage==2){1003}else{0},p.os_code);}
@effects(filesystem) fn Result<bool,Failure> checkpoint(ref cancel::Token token){return switch(cancel::requested(token)){case .ok(stop):{if(stop){.err(fail(2,1009,0))}else{.ok(false)}}case .err(p):{.err(fail(2,1010,p.os_code))}};}
fn bool separator(u8 c){return c==47 || c==92;}
// Split without normalization. C:\leaf retains C:\, /leaf retains /, and
// leaf uses parent '.'. Drive-relative C:leaf is not a rooted leaf. Other
// Windows parent spellings pass unchanged to the existing rooted contract;
// this module adds no UNC/device/ancestor-reparse resolution guarantee.
fn Result<usize,u32> leaf_start(ref u8[] path){
    if(path.len==0 || path.len>131064 || !text::validate_utf8(path).is_ok){return .err(1001);}
    var usize start=0;var usize words=0;var usize leaf_words=0;var usize i=0;
    while(i<path.len){let c=path[i];if(c==0){return .err(1001);}if(c<128 || c>=192){let usize width=if(c>=240){2}else{1};words+=width;leaf_words+=width;}if(separator(c)){start=i+1;leaf_words=0;}i+=1;}
    if(words>32766 || leaf_words>255 || !host::valid_leaf(path[start..])){return .err(1001);}return .ok(start);
}
@effects(filesystem) pub fn Result<bytes::Buffer,Failure> load(ref u8[] path,usize maximum,ref cancel::Token token,ref mut usize work){
    if(maximum==0 || maximum>16777216){return .err(fail(1,1001,0));}
    let start=switch(leaf_start(path)){case .ok(p):{p}case .err(code):{return .err(fail(1,code,0));}};
    let path_cost=path.len*2+4;if(*work>16777216 || path_cost>16777216-*work){return .err(fail(1,1002,0));}
    discard checkpoint(token)?;
    // Admission and a pre-signaled token precede any path allocation or charge.
    assert(budget::charge(work,path_cost));
    let usize parent_end=if(start==0){0}else if(start==1 || start==3 && path[1]==58){start}else{start-1};
    let parent=if(start==0){b"."}else{path[..parent_end]};
    let root=switch(host::open(parent)){case .ok(p):{p}case .err(p):{return .err(native(3,p));}};
    var file=switch(host::open_read(ref root,path[start..])){case .ok(p):{p}case .err(p):{return .err(native(4,p));}};
    let before=switch(host::observe(ref file)){case .ok(p):{p}case .err(p):{return .err(native(5,p));}};
    let size=switch(host::size(ref file)){case .ok(p):{p}case .err(p):{return .err(native(5,p));}};
    if(!host::regular(before) || size!=before.size){return .err(fail(5,1001,0));}
    if(size>maximum){return .err(fail(6,1002,0));}
    discard checkpoint(token)?;
    // Charge the full owned payload and EOF sentinel before
    // allocating. All later reads use this one retained handle, at most size+1.
    if(!budget::charge(work,size+1)){return .err(fail(6,1002,0));}
    var out=bytes::empty();if(!bytes::resize_zeroed(ref mut out,size+1).is_ok){return .err(fail(6,1003,0));}
    var usize used=0;while(used<=size){
        discard checkpoint(token)?;
        let remaining=size+1-used;let usize count=if(remaining>65536){65536}else{remaining};
        let read=switch(host::read(ref mut file,bytes::view_mut(ref mut out)[used..used+count])){case .ok(p):{p}case .err(p):{return .err(native(6,p));}};
        if(read==0){break;}
        used+=read;if(used>size){return .err(fail(6,if(used>maximum){1002}else{1001},0));}
    }
    let after=switch(host::observe(ref file)){case .ok(p):{p}case .err(p):{return .err(native(5,p));}};
    if(before.identity!=after.identity || before.modified!=after.modified || before.size!=after.size || before.attributes!=after.attributes || used!=size){return .err(fail(5,1001,0));}
    assert(bytes::truncate(ref mut out,used).is_ok);
    discard checkpoint(token)?;
    switch(host::close(file)){case .ok(_):{}case .err(p):{return .err(native(7,p));}};
    switch(host::close_directory(root)){case .ok(_):{}case .err(p):{return .err(native(8,p));}};
    discard checkpoint(token)?;return .ok(out);
}
