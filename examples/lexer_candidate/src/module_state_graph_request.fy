module module_state_graph_request;
import module_state_graph_transport_plan as plan;
import project_cache_encoding as budget;
import sha256_stream as sha;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

pub struct Entry{u32[8] key;usize start;usize end;}
// Construction moves storage; admission belongs exclusively to encode.
struct Identity{u32[8] layout;usize version;}
pub resource Snapshot{Identity identity;bytes::Buffer data;v::Vec<Entry> entries;cleanup{}}
pub fn Snapshot snapshot_of(u32[8] layout,usize version,bytes::Buffer data,v::Vec<Entry> entries){return Snapshot{.identity=Identity{.layout,.version},.data,.entries};}
// Immutable observations also let the local-store bridge verify detached
// snapshot lifetime; no mutable projection or execution authority is exposed.
pub fn usize snapshot_version(ref Snapshot p){return p.identity.version;}
pub fn u32[8] snapshot_layout(ref Snapshot p){return p.identity.layout;}
pub fn usize snapshot_count(ref Snapshot p){return v::len<Entry>(ref p.entries);}
pub fn u32[8] snapshot_key(ref Snapshot p,usize index){return v::at<Entry>(ref p.entries,index).key;}
pub fn ref u8[] snapshot_value(ref Snapshot p,usize index){let row=*v::at<Entry>(ref p.entries,index);return bytes::view(ref p.data)[row.start..row.end];}
pub struct Limits{usize writes;usize bytes;usize value;usize key;}
pub struct Work{usize structure;usize bytes;}
pub fn Work empty_work(){return Work{.structure=0,.bytes=0};}
// Structural packet only. Neither constructor nor mutable views are exported.
pub resource Packet{bytes::Buffer data;bytes::Buffer hash;cleanup{}}
pub fn ref u8[] content(ref Packet packet){return bytes::view(ref packet.data);}
pub fn ref u8[] digest(ref Packet packet){return bytes::view(ref packet.hash);}
fn bool byte_charge(ref mut Work work,usize cost){if(work.bytes>1073741824 || cost>1073741824-work.bytes){return false;}work.bytes+=cost;return true;}
fn u8 octet(usize value){let d=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (d[(value/16)%16]<<4)|d[value%16];}
fn i32 compare(u32[8] a,u32[8] b){var usize i=0;while(i<8){if(a[i]<b[i]){return -1;}if(a[i]>b[i]){return 1;}i+=1;}return 0;}
fn u32[8] words(ref u8[] raw){var u32[8] out=[0;8];var usize i=0;while(i<8){let at=i*4;out[i]=(raw[at] as u32)<<24|(raw[at+1] as u32)<<16|(raw[at+2] as u32)<<8|(raw[at+3] as u32);i+=1;}return out;}
fn bool hash_valid(ref u8[] raw){if(raw.len!=32){return false;}var usize i=0;while(i<32){if(raw[i]!=0){return true;}i+=1;}return false;}
fn Result<usize,u32> codec(ref plan::Plan owner,u32[8] key,ref mut Work work){
    var usize lo=0;var usize hi=plan::cell_count(owner);
    while(lo<hi){if(!budget::charge(ref mut work.structure,9)){return .err(1002);}let mid=lo+(hi-lo)/2;let row=plan::cell_at(owner,mid);let order=compare(key,row.key);if(order==0){return .ok(row.codec);}if(order<0){hi=mid;}else{lo=mid+1;}}
    return .ok(0);
}
fn u32 canonical(usize codec,ref u8[] raw,ref mut Work work){
    if(codec==0 || codec==4){return 0;}
    if(codec==1){return if(raw.len==1 && raw[0]<=1){0}else{1001};}
    if(codec==2){return if(raw.len==4){0}else{1001};}
    if(codec==3){return if(raw.len==8){0}else{1001};}
    if(codec==5){if(!byte_charge(work,raw.len)){return 1002;}return if(text::validate_utf8(raw).is_ok){0}else{1001};}
    if(codec==6){return if(raw.len==20){0}else{1001};}
    if(codec==7){return if(raw.len==32){0}else{1001};}return 1001;
}
fn bool number(ref mut bytes::Buffer out,usize value,usize width){var u8[8] raw=[0;8];var usize i=0;var usize rest=value;while(i<width){raw[width-i-1]=octet(rest%256);rest/=256;i+=1;}let ref u8[] data=ref raw;return rest==0 && bytes::append(out,data[..width]).is_ok;}
fn bool key_bytes(ref mut bytes::Buffer out,u32[8] key){var u8[32] raw=[0;32];var usize i=0;var u32 shift=24;while(i<32){raw[i]=octet(((key[i/4]>>shift)&255) as usize);i+=1;if(i%4==0){shift=24;}else{shift-=8;}}return bytes::append(out,ref raw).is_ok;}
pub fn Result<Packet,u32> encode(ref plan::Plan owner,ref u8[] layout,ref u8[] body,ref u8[] program,ref Snapshot snapshot,Limits limits,ref mut Work work){
    // Check overdrawn ledgers even for an empty snapshot, without subtraction wrap.
    if(!budget::charge(ref mut work.structure,1) || !byte_charge(work,0)){return .err(1002);}
    if(!budget::charge(ref mut work.structure,105)){return .err(1002);}
    if(!hash_valid(layout) || !hash_valid(body) || !hash_valid(program)){return .err(1001);}
    if(compare(words(layout),snapshot.identity.layout)!=0){return .err(1001);}
    // Go int-backed limits must fit signed host int; MaxWriteBytes is uint64.
    if(limits.writes==0 || limits.writes>9223372036854775807 || limits.bytes==0 || limits.value==0 || limits.value>9223372036854775807 || limits.key==0 || limits.key>9223372036854775807){return .err(1001);}
    let count=v::len<Entry>(ref snapshot.entries);if(count>65536){return .err(1002);}var usize payload=0;var usize i=0;
    while(i<count){
        if(!budget::charge(ref mut work.structure,10)){return .err(1002);}let row=*v::at<Entry>(ref snapshot.entries,i);
        if(row.start>row.end || row.end>bytes::len(ref snapshot.data)){return .err(1001);}
        if(i!=0 && compare(v::at<Entry>(ref snapshot.entries,i-1).key,row.key)>=0){return .err(1001);}
        let size=row.end-row.start;if(size>16777216-36 || payload>16777216-36-size){return .err(1002);}payload+=36+size;
        let kind=switch(codec(owner,row.key,work)){case .ok(c):{c}case .err(c):{return .err(c);}};
        let code=canonical(kind,bytes::view(ref snapshot.data)[row.start..row.end],work);if(code!=0){return .err(code);}i+=1;
    }
    let size=152+payload;
    // Prepay every output copy and SHA input byte, final blocks and digest.
    // Reserve once: appending a 16MiB value never repeatedly copies its prefix.
    if(!budget::charge(ref mut work.structure,count*64+161) || !byte_charge(work,size*2+160)){return .err(1002);}
    var output=switch(bytes::with_capacity(size)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    if(!bytes::append(ref mut output,b"FYGRIN01").is_ok || !bytes::append(ref mut output,layout).is_ok || !bytes::append(ref mut output,body).is_ok || !bytes::append(ref mut output,program).is_ok || !number(ref mut output,snapshot.identity.version,8) || !number(ref mut output,limits.writes,8) || !number(ref mut output,limits.bytes,8) || !number(ref mut output,limits.value,8) || !number(ref mut output,limits.key,8) || !number(ref mut output,count,4) || !number(ref mut output,payload,4)){return .err(1003);}
    i=0;while(i<count){let row=*v::at<Entry>(ref snapshot.entries,i);if(!key_bytes(ref mut output,row.key) || !number(ref mut output,row.end-row.start,4) || !bytes::append(ref mut output,bytes::view(ref snapshot.data)[row.start..row.end]).is_ok){return .err(1003);}i+=1;}
    var state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref output))){return .err(1002);}let hash=sha::finish(ref state);var u8[32] raw=[0;32];i=0;var u32 shift=24;while(i<32){raw[i]=octet(((hash[i/4]>>shift)&255) as usize);i+=1;if(i%4==0){shift=24;}else{shift-=8;}}
    let digest=switch(bytes::copy(ref raw)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    return .ok(Packet{.data=output,.hash=digest});
}
