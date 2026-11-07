module module_state_command_snapshot_report;
import module_state_command_options as command;
import module_state_graph_transport as transport;
import module_state_graph_native as native;
import module_state_local_codec as codec;
import module_state_local_base64 as base64;
import std::bytes as bytes;

// Pure private developer-command report, matching fairyc.local-state-run/v2.
// One cumulative 1GiB ledger charges entry visits, emitted bytes and the existing
// base64 helper's reads/initialization/writes. Errors retain spent work; only a
// complete owned buffer (including newline) escapes. No caller ledger resets.
pub fn usize output_limit(){return 33554432;}
pub fn usize work_limit(){return 1073741824;}
fn bool charge(ref mut usize work,usize n){if(*work>work_limit() || n>work_limit()-*work){return false;};*work=*work+n;return true;}
fn Result<usize,u32> put(ref mut bytes::Buffer out,ref u8[] raw,ref mut usize work){
    if(bytes::len(out)>output_limit() || raw.len>output_limit()-bytes::len(out) || !charge(work,raw.len)){return .err(1002);}
    return switch(bytes::append(out,raw)){case .ok(n):{.ok(n)}case .err(_):{.err(1003)}};
}
fn usize width(usize n){var usize rest=n;var usize size=1;while(rest>=10){rest/=10;size+=1;}return size;}
fn Result<usize,u32> number(ref mut bytes::Buffer out,usize n,ref mut usize work){
    let size=width(n);if(bytes::len(out)>output_limit() || size>output_limit()-bytes::len(out) || !charge(work,size)){return .err(1002);}
    return switch(bytes::append_usize(out,n)){case .ok(p):{.ok(p)}case .err(_):{.err(1003)}};
}
fn Result<usize,u32> hex(ref mut bytes::Buffer out,ref u8[] raw,ref mut usize work){
    let digits=b"0123456789abcdef";var usize i=0;while(i<raw.len){let at=raw[i] as usize;discard put(out,digits[at/16..at/16+1],work)?;discard put(out,digits[at%16..at%16+1],work)?;i+=1;}return .ok(raw.len);
}
fn Result<usize,u32> words(ref mut bytes::Buffer out,u32[8] hash,ref mut usize work){
    discard put(out,b"[",work)?;var usize i=0;var u32 shift=24;while(i<32){if(i!=0){discard put(out,b",",work)?;}discard number(out,((hash[i/4]>>shift)&255) as usize,work)?;i+=1;if(i%4==0){shift=24;}else{shift-=8;}}return put(out,b"]",work);
}
pub fn Result<bytes::Buffer,u32> snapshot(command::Options options,ref transport::Prepared prepared,ref codec::State state,ref mut usize work){
    if(options.action!=1 && options.action!=3){return .err(1001);}
    let entry=transport::entry(prepared);let hash=native::layout_hash(entry);let body=native::body_hash(entry);let program=transport::program_hash(prepared);
    if(hash.len!=32 || body.len!=32 || program.len!=32){return .err(1001);}
    let expected=codec::layout_hash(state);var usize i=0;var u32 shift=24;
    while(i<32){if(((expected[i/4]>>shift)&255)!=(hash[i] as u32)){return .err(1001);}i+=1;if(i%4==0){shift=24;}else{shift-=8;}}
    if(codec::count(state)>65536 || !charge(work,codec::count(state)+1)){return .err(1002);}
    var out=bytes::empty();discard put(ref mut out,b"{\"schema\":\"fairyc.local-state-run/v2\",\"action\":\"",work)?;
    discard put(ref mut out,if(options.action==1){b"init"}else{b"snapshot"},work)?;
    discard put(ref mut out,b"\",\"layoutHash\":\"fycs-layout1-sha256-",work)?;discard hex(ref mut out,hash,work)?;
    discard put(ref mut out,b"\",\"snapshot\":{\"LayoutHash\":",work)?;discard words(ref mut out,expected,work)?;
    discard put(ref mut out,b",\"Version\":",work)?;discard number(ref mut out,codec::version(state),work)?;
    discard put(ref mut out,b",\"Entries\":[",work)?;i=0;
    while(i<codec::count(state)){
        if(i!=0){discard put(ref mut out,b",",work)?;}discard put(ref mut out,b"{\"Key\":",work)?;discard words(ref mut out,codec::key(state,i),work)?;
        discard put(ref mut out,b",\"Value\":",work)?;
        if(codec::is_null(state,i)){discard put(ref mut out,b"null",work)?;}else{
            let value=base64::encode(codec::value(state,i),work)?;discard put(ref mut out,b"\"",work)?;discard put(ref mut out,bytes::view(ref value),work)?;discard put(ref mut out,b"\"",work)?;
        }discard put(ref mut out,b"}",work)?;i+=1;
    }
    discard put(ref mut out,b"]},\"engine\":\"native-graph\",\"profile\":\"",work)?;discard put(ref mut out,if(options.release){b"optimized"}else{b"low"},work)?;
    discard put(ref mut out,b"\",\"bodyHash\":\"fycsbody1-sha256-",work)?;discard hex(ref mut out,body,work)?;
    discard put(ref mut out,b"\",\"programHash\":\"",work)?;discard hex(ref mut out,program,work)?;discard put(ref mut out,b"\"}\n",work)?;return .ok(out);
}
