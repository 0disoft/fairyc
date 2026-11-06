module module_state_local_codec;
import module_state_layout as layout;
import module_state_graph_request as request;
import module_state_local_base64 as base64;
import project_identity_bytes as identities;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

// Existing fairyc.local-state/v1 limits, independent of transaction policy.
pub struct Limits{usize bytes;usize entries;usize value;}
pub struct Work{usize structure;usize bytes;}
pub fn Work empty_work(){return Work{.structure=0,.bytes=0};}
pub fn bool valid(Limits limits){return limits.bytes>0 && limits.bytes<=16777216 && limits.entries>0 && limits.entries<=65536 && limits.value>0 && limits.value<=limits.bytes;}
struct Entry{u32[8] key;usize start;usize end;bool was_null;}
struct Identity{u32[8] hash;usize version;}
// Only checked canonical record decoding or an admitted layout creates State.
// Null and empty bytes remain distinct in the on-disk Go JSON representation.
pub resource State{bytes::Buffer prefix;Identity identity;bytes::Buffer data;v::Vec<Entry> entries;cleanup{}}
pub fn usize version(ref State state){return state.identity.version;}
pub fn u32[8] layout_hash(ref State state){return state.identity.hash;}
pub fn usize count(ref State state){return v::len<Entry>(ref state.entries);}
pub fn u32[8] key(ref State state,usize index){return v::at<Entry>(ref state.entries,index).key;}
pub fn bool is_null(ref State state,usize index){return v::at<Entry>(ref state.entries,index).was_null;}
pub fn ref u8[] value(ref State state,usize index){let e=*v::at<Entry>(ref state.entries,index);return bytes::view(ref state.data)[e.start..e.end];}
fn bool charge(ref mut Work work,usize n){if(work.structure>16777216 || n>16777216-work.structure){return false;}work.structure+=n;return true;}
fn bool byte_charge(ref mut Work work,usize n){if(work.bytes>1073741824 || n>1073741824-work.bytes){return false;}work.bytes+=n;return true;}
fn u32 ready(Limits limits,ref mut Work work){if(!valid(limits)){return 1001;}return if(charge(work,1) && byte_charge(work,0)){0}else{1002};}
fn Result<usize,u32> put(ref mut bytes::Buffer out,ref u8[] raw,Limits limits,ref mut Work work){
    if(limits.bytes<32 || bytes::len(out)>limits.bytes-32 || raw.len>limits.bytes-32-bytes::len(out) || !byte_charge(work,raw.len)){return .err(1002);}
    return switch(bytes::append(out,raw)){case .ok(n):{.ok(n)}case .err(_):{.err(1003)}};
}
fn Result<usize,u32> number(ref mut bytes::Buffer out,usize n,Limits limits,ref mut Work work){
    if(!charge(work,1)){return .err(1002);}var usize width=1;var usize rest=n;while(rest>=10){width+=1;rest/=10;}
    if(limits.bytes<32 || bytes::len(out)>limits.bytes-32 || width>limits.bytes-32-bytes::len(out) || !byte_charge(work,width)){return .err(1002);}
    return switch(bytes::append_usize(out,n)){case .ok(n):{.ok(n)}case .err(_):{.err(1003)}};
}
fn Result<usize,u32> quoted(ref mut bytes::Buffer out,ref u8[] text,Limits limits,ref mut Work work){
    discard put(out,b"\"",limits,work)?;discard put(out,text,limits,work)?;return put(out,b"\"",limits,work);
}
fn Result<usize,u32> words(ref mut bytes::Buffer out,u32[8] hash,Limits limits,ref mut Work work){
    discard put(out,b"[",limits,work)?;var usize i=0;var u32 shift=24;while(i<32){if(i!=0){discard put(out,b",",limits,work)?;}discard number(out,((hash[i/4]>>shift)&255) as usize,limits,work)?;i+=1;if(i%4==0){shift=24;}else{shift-=8;}}return put(out,b"]",limits,work);
}
fn Result<bytes::Buffer,u32> prefix(ref layout::Plan schemas,usize selected,Limits limits,ref mut Work work){
    if(selected>=layout::count(schemas)){return .err(1001);}
    // The existing layout validator owns a 1MiB local ceiling. Reserve that
    // complete bounded operation before invoking it; caller history never resets.
    if(!charge(work,1048576)){return .err(1002);}var usize validation_work=0;let validation=layout::validate_graph(schemas,ref mut validation_work);if(validation!=0){return .err(if(validation==10){1002}else if(validation==11){1003}else{1001});}
    let owner=layout::at(schemas,selected);var out=bytes::empty();discard put(ref mut out,b"{\"schema\":\"fairyc.local-state/v1\",\"contractUid\":",limits,work)?;
    discard quoted(ref mut out,layout::uid(schemas,selected),limits,work)?;discard put(ref mut out,b",\"layoutHash\":",limits,work)?;discard words(ref mut out,owner.hash,limits,work)?;
    discard put(ref mut out,b",\"fields\":",limits,work)?;if(owner.field_count==0){discard put(ref mut out,b"null",limits,work)?;}else{discard put(ref mut out,b"[",limits,work)?;}
    var usize i=0;while(i<owner.field_count){let field=layout::field_at(schemas,owner.field_start+i);if(i!=0){discard put(ref mut out,b",",limits,work)?;}
        discard put(ref mut out,b"{\"Index\":",limits,work)?;discard number(ref mut out,i,limits,work)?;discard put(ref mut out,b",\"Name\":",limits,work)?;discard quoted(ref mut out,layout::name(schemas,owner.field_start+i),limits,work)?;
        discard put(ref mut out,b",\"Kind\":",limits,work)?;discard number(ref mut out,field.kind as usize,limits,work)?;discard put(ref mut out,b",\"KeyTypeID\":",limits,work)?;discard quoted(ref mut out,layout::codec(field.key),limits,work)?;
        discard put(ref mut out,b",\"ValueTypeID\":",limits,work)?;discard quoted(ref mut out,layout::codec(field.value),limits,work)?;discard put(ref mut out,b",\"Prefix\":",limits,work)?;discard words(ref mut out,field.prefix,limits,work)?;discard put(ref mut out,b"}",limits,work)?;i+=1;
    }if(owner.field_count!=0){discard put(ref mut out,b"]",limits,work)?;}discard put(ref mut out,b",\"version\":",limits,work)?;return .ok(out);
}
pub fn Result<State,u32> initial(ref layout::Plan schemas,usize selected,Limits limits,ref mut Work work){
    let code=ready(limits,work);if(code!=0){return .err(code);}let head=prefix(schemas,selected,limits,work)?;
    return .ok(State{.prefix=head,.identity=Identity{.hash=layout::at(schemas,selected).hash,.version=0},.data=bytes::empty(),.entries=v::empty<Entry>()});
}
fn bool take(ref u8[] raw,ref mut usize at,ref u8[] expected){let start=*at;if(start>raw.len || expected.len>raw.len-start || !bytes::equal(raw[start..start+expected.len],expected)){return false;};*at=start+expected.len;return true;}
fn Result<usize,u32> integer(ref u8[] raw,ref mut usize at,ref mut Work work){
    if(!charge(work,1)){return .err(1002);}if(*at>=raw.len || raw[*at]<48 || raw[*at]>57){return .err(1001);}let begin=*at;var usize n=0;
    while(*at<raw.len && raw[*at]>=48 && raw[*at]<=57){let d=(raw[*at] as usize)-48;if(n>(18446744073709551615-d)/10){return .err(1001);}n=n*10+d;*at=*at+1;}
    if(*at-begin>1 && raw[begin]==48){return .err(1001);}return .ok(n);
}
fn Result<u32[8],u32> read_key(ref u8[] raw,ref mut usize at,ref mut Work work){
    if(!take(raw,at,b"[")){return .err(1001);}var u32[8] result=[0;8];var usize i=0;
    while(i<32){if(i!=0 && !take(raw,at,b",")){return .err(1001);}let n=integer(raw,at,work)?;if(n>255){return .err(1001);}let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";let octet=(digits[n/16]<<4)|digits[n%16];result[i/4]=(result[i/4]<<8)|(octet as u32);i+=1;}
    if(!take(raw,at,b"]")){return .err(1001);}return .ok(result);
}
fn i32 compare(u32[8] a,u32[8] b){var usize i=0;while(i<8){if(a[i]<b[i]){return -1;}if(a[i]>b[i]){return 1;}i+=1;}return 0;}
fn Result<bytes::Buffer,u32> checksum(ref u8[] raw,ref mut Work work){
    if(!byte_charge(work,raw.len+32)){return .err(1002);}var state=sha::empty();if(!sha::update(ref mut state,raw)){return .err(1002);}
    return switch(identities::words(sha::finish(ref state),ref mut work.structure)){case .ok(p):{.ok(p)}case .err(c):{.err(if(c==11){1003}else{1002})}};
}
pub fn Result<State,u32> decode(ref layout::Plan schemas,usize selected,ref u8[] encoded,Limits limits,ref mut Work work){
    let code=ready(limits,work);if(code!=0){return .err(code);}if(encoded.len<32 || encoded.len>limits.bytes || !byte_charge(work,encoded.len)){return .err(1002);}
    let raw=encoded[..encoded.len-32];let digest=checksum(raw,work)?;if(!bytes::equal(bytes::view(ref digest),encoded[raw.len..])){return .err(1001);}
    let head=prefix(schemas,selected,limits,work)?;var usize at=0;if(!take(raw,ref mut at,bytes::view(ref head))){return .err(1001);}let version=integer(raw,ref mut at,work)?;
    if(!take(raw,ref mut at,b",\"entries\":[")){return .err(1001);}var data=bytes::empty();var entries=v::empty<Entry>();
    if(!take(raw,ref mut at,b"]")){while(true){
        if(v::len<Entry>(ref entries)>=limits.entries || !charge(work,1)){return .err(1002);}if(v::len<Entry>(ref entries)!=0 && !take(raw,ref mut at,b",")){return .err(1001);}
        if(!take(raw,ref mut at,b"{\"Key\":")){return .err(1001);}let key=read_key(raw,ref mut at,work)?;if(v::len<Entry>(ref entries)!=0 && compare(v::at<Entry>(ref entries,v::len<Entry>(ref entries)-1).key,key)>=0){return .err(1001);}
        if(!take(raw,ref mut at,b",\"Value\":")){return .err(1001);}let nil=take(raw,ref mut at,b"null");let begin=bytes::len(ref data);
        if(!nil){if(!take(raw,ref mut at,b"\"")){return .err(1001);}let start=at;while(at<raw.len && raw[at]!=34){at+=1;}if(at==raw.len){return .err(1001);}
            let decoded=base64::decode(raw[start..at],limits.value,ref mut work.bytes)?;at+=1;
            if(!byte_charge(work,bytes::len(ref decoded))){return .err(1002);}if(!bytes::append(ref mut data,bytes::view(ref decoded)).is_ok){return .err(1003);}
        }
        if(!take(raw,ref mut at,b"}")){return .err(1001);}if(!v::push<Entry>(ref mut entries,Entry{.key,.start=begin,.end=bytes::len(ref data),.was_null=nil}).is_ok){return .err(1003);}
        if(take(raw,ref mut at,b"]")){break;}
    }}
    if(!take(raw,ref mut at,b"}") || at!=raw.len){return .err(1001);}
    return .ok(State{.prefix=head,.identity=Identity{.hash=layout::at(schemas,selected).hash,.version},.data,.entries});
}
pub fn Result<bytes::Buffer,u32> encode(ref State state,Limits limits,ref mut Work work){
    let code=ready(limits,work);if(code!=0){return .err(code);}if(count(state)>limits.entries){return .err(1002);}var out=bytes::empty();
    discard put(ref mut out,bytes::view(ref state.prefix),limits,work)?;discard number(ref mut out,state.identity.version,limits,work)?;discard put(ref mut out,b",\"entries\":[",limits,work)?;
    var usize i=0;while(i<count(state)){if(!charge(work,1)){return .err(1002);}let entry=*v::at<Entry>(ref state.entries,i);let raw=value(state,i);if(raw.len>limits.value){return .err(1002);}if(i!=0){discard put(ref mut out,b",",limits,work)?;}
        discard put(ref mut out,b"{\"Key\":",limits,work)?;discard words(ref mut out,entry.key,limits,work)?;discard put(ref mut out,b",\"Value\":",limits,work)?;
        if(entry.was_null){discard put(ref mut out,b"null",limits,work)?;}else{let text=base64::encode(raw,ref mut work.bytes)?;discard quoted(ref mut out,bytes::view(ref text),limits,work)?;}
        discard put(ref mut out,b"}",limits,work)?;i+=1;
    }
    discard put(ref mut out,b"]}",limits,work)?;let digest=checksum(bytes::view(ref out),work)?;
    if(!bytes::append(ref mut out,bytes::view(ref digest)).is_ok){return .err(1003);}return .ok(out);
}
pub fn Result<request::Snapshot,u32> snapshot(ref State state,ref mut Work work){
    if(!charge(work,count(state)+1) || !byte_charge(work,bytes::len(ref state.data))){return .err(1002);}let copied=switch(bytes::copy(bytes::view(ref state.data))){case .ok(p):{p}case .err(_):{return .err(1003);}};
    var entries=v::empty<request::Entry>();var usize i=0;while(i<count(state)){let e=*v::at<Entry>(ref state.entries,i);if(!v::push<request::Entry>(ref mut entries,request::Entry{.key=e.key,.start=e.start,.end=e.end}).is_ok){return .err(1003);}i+=1;}
    return .ok(request::snapshot_of(state.identity.hash,state.identity.version,copied,entries));
}
