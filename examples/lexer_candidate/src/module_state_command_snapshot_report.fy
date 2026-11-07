module module_state_command_snapshot_report;
import module_state_command_options as command;
import module_state_graph_transport as transport;
import module_state_graph_native as native;
import module_state_local_codec as codec;
import module_state_local_base64 as base64;
import module_state_graph_windows_persistent as persistent;
import module_state_graph_response as response;
import module_state_layout as layout;
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
// Outcome remains authoritative even when an acknowledged receipt disagrees.
pub fn i32 outcome_exit(ref persistent::Outcome outcome){
    let h=response::header(persistent::admitted(outcome));let p=persistent::publication(outcome);
    if(h.status!=0 || !p.consistent){return 4;}
    if(h.exit==0){return if(p.acknowledged){0}else{4};}
    if(p.acknowledged){return 4;}return if(h.exit==1){6}else if(h.exit==2){8}else{4};
}
fn bool same_hash(u32[8] words,ref u8[] raw){
    if(raw.len!=32){return false;}var usize i=0;var u32 shift=24;while(i<32){if(((words[i/4]>>shift)&255)!=(raw[i] as u32)){return false;}i+=1;if(i%4==0){shift=24;}else{shift-=8;}}return true;
}
fn Result<usize,u32> receipt(ref mut bytes::Buffer out,u32[8] hash,usize version,usize writes,usize write_bytes,bool commit,ref mut usize work){
    discard put(out,b"{\"LayoutHash\":",work)?;discard words(out,hash,work)?;discard put(out,b",\"Version\":",work)?;discard number(out,version,work)?;
    discard put(out,b",\"Writes\":",work)?;discard number(out,writes,work)?;
    if(commit){discard put(out,b",\"WriteBytes\":",work)?;discard number(out,write_bytes,work)?;}return put(out,b"}",work);
}
pub fn Result<bytes::Buffer,u32> execute(command::Options options,ref transport::Prepared prepared,ref persistent::Outcome outcome,ref mut usize work){
    if(options.action!=2 || !persistent::matches(outcome,prepared)){return .err(1001);}
    let admitted=persistent::admitted(outcome);let h=response::header(admitted);let p=persistent::publication(outcome);
    let entry=transport::entry(prepared);let hash=native::layout_hash(entry);let body=native::body_hash(entry);let program=transport::program_hash(prepared);
    if(hash.len!=32 || body.len!=32 || program.len!=32 || h.status!=0 || h.exit>2){return .err(1001);}
    if(h.exit==0){
        if(!p.acknowledged || response::write_count(admitted)!=h.writes || h.writes!=0 && response::version(admitted)==18446744073709551615){return .err(1001);}
        let next=if(h.writes==0){response::version(admitted)}else{response::version(admitted)+1};
        let consistent=same_hash(p.receipt.layout,hash) && p.receipt.version==next && p.receipt.writes==h.writes;
        if(consistent!=p.consistent){return .err(1001);}
    }else{
        if(p.acknowledged || !p.consistent || !layout::equal(p.receipt.layout,[0;8]) || p.receipt.version!=0 || p.receipt.writes!=0 || h.writes!=0 || h.write_bytes!=0){return .err(1001);}
    }
    if(h.exit==2 && response::trap_category(admitted).len==0){return .err(1001);}
    if(!charge(work,response::read_count(admitted)+1)){return .err(1002);}
    var out=bytes::empty();discard put(ref mut out,b"{\"schema\":\"fairyc.local-state-run/v2\",\"action\":\"execute\",\"layoutHash\":\"fycs-layout1-sha256-",work)?;discard hex(ref mut out,hash,work)?;
    if(h.exit==2){
        // Admission owns canonical [a-z0-9-] trap categories, so no escaping.
        discard put(ref mut out,b"\",\"trapCategory\":\"",work)?;discard put(ref mut out,response::trap_category(admitted),work)?;discard put(ref mut out,b"\"",work)?;
    }else{
        discard put(ref mut out,b"\",\"outcome\":{\"Exit\":\"",work)?;discard put(ref mut out,if(h.exit==0){b"commit"}else{b"reject"},work)?;discard put(ref mut out,b"\",\"Receipt\":",work)?;
        discard receipt(ref mut out,p.receipt.layout,p.receipt.version,p.receipt.writes,if(h.exit==0){h.write_bytes}else{0},true,work)?;
        discard put(ref mut out,b",\"Reads\":",work)?;
        if(response::read_count(admitted)==0){discard put(ref mut out,b"null",work)?;}else{
            discard put(ref mut out,b"[",work)?;var usize i=0;while(i<response::read_count(admitted)){
                let row=response::read_at(admitted,i);if(row.codec<1 || row.codec>7){return .err(1001);}
                if(i!=0){discard put(ref mut out,b",",work)?;}discard put(ref mut out,b"{\"Operation\":",work)?;discard number(ref mut out,row.operation,work)?;
                discard put(ref mut out,b",\"Found\":",work)?;discard put(ref mut out,if(row.found){b"true"}else{b"false"},work)?;
                discard put(ref mut out,b",\"Value\":{\"TypeID\":\"",work)?;discard put(ref mut out,layout::codec(row.codec),work)?;discard put(ref mut out,b"\",\"Encoded\":\"",work)?;
                let encoded=base64::encode(response::read_value(admitted,i),work)?;discard put(ref mut out,bytes::view(ref encoded),work)?;discard put(ref mut out,b"\"}}",work)?;i+=1;
            }discard put(ref mut out,b"]",work)?;
        }discard put(ref mut out,b",\"Cost\":",work)?;discard number(ref mut out,h.cost,work)?;discard put(ref mut out,b"}",work)?;
    }
    discard put(ref mut out,b",\"engine\":\"native-graph\",\"profile\":\"",work)?;discard put(ref mut out,if(options.release){b"optimized"}else{b"low"},work)?;
    discard put(ref mut out,b"\",\"bodyHash\":\"fycsbody1-sha256-",work)?;discard hex(ref mut out,body,work)?;discard put(ref mut out,b"\",\"programHash\":\"",work)?;discard hex(ref mut out,program,work)?;discard put(ref mut out,b"\"",work)?;
    if(p.acknowledged && !p.consistent){
        discard put(ref mut out,b",\"publication\":",work)?;discard receipt(ref mut out,p.receipt.layout,p.receipt.version,p.receipt.writes,0,false,work)?;
        discard put(ref mut out,b",\"failure\":\"native state publication was acknowledged with an inconsistent receipt; inspect stored state before retrying\"",work)?;
    }discard put(ref mut out,b"}\n",work)?;return .ok(out);
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
