module module_state_graph_response_evidence;
import module_state_graph_response_common as common;
import module_state_graph_program as program;
import module_state_graph_model as m;
import module_state_layout as state;
import sha256_stream as sha;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

struct Limits{usize writes;usize bytes;usize value;usize key;}
struct Snapshot{u32[8] key;usize start;usize end;}
// Snapshot spans are offsets, never retained caller borrows. Only the sealed
// input owner supplies request views while admitting an event. All observations
// and overlay values are independently owned, including replaced write values.
// The first four fields match the existing std::bytes Buffer ABI. Mutations
// borrow the entire resource, avoiding mutable resource-field projections.
// Tables and values share one private allocation and retain offsets only.
// Reserve runs before any view is made; every view ends before another reserve.
@link_name("fy_std_buffer_reserve_v0") unsafe extern C fn u32 reserve_raw(ptr mut void out,usize capacity);
@link_name("fy_std_buffer_append_v0") unsafe extern C fn u32 append_raw(ptr mut void out,ptr u8 data,usize size);
@link_name("fy_std_buffer_resize_zeroed_v0") unsafe extern C fn u32 resize_raw(ptr mut void out,usize size);
@link_name("fy_std_buffer_drop_v0") unsafe extern C fn u32 drop_raw(ptr mut void out);
pub resource Evidence{ptr mut void data;usize capacity;usize alignment;usize length;usize request_size;Limits limits;v::Vec<Snapshot> snapshot;usize read_rows;usize read_size;usize read_capacity;usize write_rows;usize write_size;usize write_capacity;usize value_bytes;usize write_bytes;u32 error;
    cleanup{discard unsafe{drop_raw(ref mut this as ptr mut void)};}}
fn ref u8[] content(ref Evidence e){return unsafe{byte_slice::from_raw_parts(e,e.data,e.length)};}
fn ref mut u8[] content_mut(ref mut Evidence e){return unsafe{byte_slice::from_raw_parts_mut(e,e.data,e.length)};}
pub fn usize read_count(ref Evidence e){return if(e.error==0){e.read_size}else{0};}
pub fn common::Read read_at(ref Evidence e,usize index){assert(index<read_count(e));let raw=content(e);let at=e.read_rows+index*40;return common::Read{.operation=number(raw,at,8),.found=number(raw,at+8,8)!=0,.codec=raw[at+23] as u32,.start=number(raw,at+24,8),.end=number(raw,at+32,8)};}
pub fn ref u8[] read_value(ref Evidence e,usize index){let row=read_at(e,index);return content(e)[row.start..row.end];}
pub fn usize write_count(ref Evidence e){return if(e.error==0){e.write_size}else{0};}
pub fn common::Write write_at(ref Evidence e,usize index){assert(index<write_count(e));let raw=content(e);let at=e.write_rows+index*56;return common::Write{.key=words(raw[at..at+32]),.deleted=number(raw,at+32,8)!=0,.start=number(raw,at+40,8),.end=number(raw,at+48,8)};}
pub fn ref u8[] write_value(ref Evidence e,usize index){let row=write_at(e,index);return content(e)[row.start..row.end];}
pub fn usize write_bytes(ref Evidence e){return if(e.error==0){e.write_bytes}else{0};}
fn bool enter(ref mut common::Work work){return common::charge(work,1) && common::byte_charge(work,0);}
fn usize number(ref u8[] raw,usize at,usize width){var usize out=0;var usize i=0;while(i<width){out=out*256+raw[at+i] as usize;i+=1;}return out;}
fn u32[8] words(ref u8[] raw){var u32[8] out=[0;8];var usize i=0;while(i<8){let at=i*4;out[i]=(raw[at] as u32)<<24|(raw[at+1] as u32)<<16|(raw[at+2] as u32)<<8|(raw[at+3] as u32);i+=1;}return out;}
fn i32 compare(u32[8] a,u32[8] b){var usize i=0;while(i<8){if(a[i]<b[i]){return -1;}if(a[i]>b[i]){return 1;}i+=1;}return 0;}
fn u8 octet(usize n){let d=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (d[(n/16)%16]<<4)|d[n%16];}
fn u32 same(ref u8[] a,ref u8[] b,ref mut common::Work work){if(!common::byte_charge(work,a.len+b.len)){return 1002;}return if(bytes::equal(a,b)){0}else{1001};}
// Declaration codec order, intentionally independent of the native wire ABI.
// Runtime bytes/text do not inherit the source-literal 1MiB ceiling.
fn u32 canonical(u32 codec,ref u8[] raw,ref mut common::Work work){
    if(!common::charge(work,1)){return 1002;}
    if(codec==1){if(!common::byte_charge(work,1)){return 1002;}return if(raw.len==1 && raw[0]<=1){0}else{1001};}
    if(codec==2){return if(raw.len==4){0}else{1001};}if(codec==3){return if(raw.len==8){0}else{1001};}
    if(codec==4){return if(raw.len==20){0}else{1001};}if(codec==5){return if(raw.len==32){0}else{1001};}
    if(codec==6){return 0;}if(codec==7){if(!common::byte_charge(work,raw.len)){return 1002;}return if(text::validate_utf8(raw).is_ok){0}else{1001};}return 1001;
}
fn u32 absent(u32 codec,ref u8[] raw,ref mut common::Work work){
    if(!common::charge(work,1) || !common::byte_charge(work,raw.len)){return 1002;}
    let usize width=if(codec==1){1}else if(codec==2){4}else if(codec==3){8}else if(codec==4){20}else if(codec==5){32}else{0};
    if(codec<1 || codec>7 || raw.len!=width){return 1001;}var usize i=0;while(i<raw.len){if(raw[i]!=0){return 1001;}i+=1;}return 0;
}
fn u32 reserve(ref mut Evidence e,usize needed,ref mut common::Work work){
    if(needed>268435456){return 1002;}if(needed<=e.capacity){return 0;}
    var usize capacity=e.capacity;if(capacity==0){capacity=64;}
    while(capacity<needed){if(!common::charge(work,1)){return 1002;}capacity*=2;}
    if(!common::byte_charge(work,e.length)){return 1002;}return if(unsafe{reserve_raw(e as ptr mut void,capacity)}==0){0}else{1003};
}
fn u32 append(ref mut Evidence e,ref u8[] raw,ref mut common::Work work){
    if(raw.len>67108864-e.value_bytes){return 1002;}
    if(!common::charge(work,1) || !common::byte_charge(work,raw.len)){return 1002;}
    let needed=e.length+raw.len;let code=reserve(e,needed,work);if(code!=0){return code;}
    if(unsafe{append_raw(e as ptr mut void,raw as ptr u8,raw.len)}!=0){return 1003;}e.value_bytes+=raw.len;return 0;
}
// Geometric packed index tables preserve O(log N) lookup. Old table bytes
// become inert padding; values never move relative to the owning allocation.
fn u32 table(ref mut Evidence e,bool reads,ref mut common::Work work){
    let usize count=if(reads){e.read_size}else{e.write_size};let usize capacity=if(reads){e.read_capacity}else{e.write_capacity};if(count<capacity){return 0;}
    let usize width=if(reads){40}else{56};let usize grown=if(capacity==0){8}else{capacity*2};let start=e.length;let usize old=if(reads){e.read_rows}else{e.write_rows};
    if(grown>268435456/width || start>268435456-grown*width){return 1002;}
    if(!common::charge(work,1) || !common::byte_charge(work,grown*width+count*width)){return 1002;}
    let code=reserve(e,start+grown*width,work);if(code!=0){return code;}
    if(unsafe{resize_raw(e as ptr mut void,start+grown*width)}!=0){return 1003;}
    {let raw=content_mut(e);var usize i=0;while(i<count*width){raw[start+i]=raw[old+i];i+=1;}}
    if(reads){e.read_rows=start;e.read_capacity=grown;}else{e.write_rows=start;e.write_capacity=grown;}return 0;
}
fn void put(ref mut u8[] raw,usize at,usize n,usize width){var usize value=n;var usize i=0;while(i<width){raw[at+width-1-i]=octet(value%256);value/=256;i+=1;}}
fn void put_key(ref mut u8[] raw,u32[8] key){var usize i=0;while(i<8){put(raw,i*4,key[i] as usize,4);i+=1;}}
pub fn Result<Evidence,u32> begin(ref u8[] request,ref mut common::Work work){
    if(!enter(work)){return .err(1002);}if(request.len<152){return .err(1001);}if(request.len-152>16777216){return .err(1002);}
    if(!common::byte_charge(work,152)){return .err(1002);}if(!bytes::equal(request[..8],b"FYGRIN01")){return .err(1001);}
    let limits=Limits{.writes=number(request,112,8),.bytes=number(request,120,8),.value=number(request,128,8),.key=number(request,136,8)};
    if(limits.writes==0 || limits.writes>9223372036854775807 || limits.bytes==0 || limits.value==0 || limits.value>9223372036854775807 || limits.key==0 || limits.key>9223372036854775807){return .err(1001);}
    let count=number(request,144,4);let payload=number(request,148,4);
    if(payload!=request.len-152 || count>payload/36){return .err(1001);}if(count>65536){return .err(1002);}
    if(!common::charge(work,count+1)){return .err(1002);}var rows=v::empty<Snapshot>();if(!v::reserve<Snapshot>(ref mut rows,count).is_ok){return .err(1003);}
    var usize at=152;var usize i=0;var u32[8] previous=[0;8];
    while(i<count){
        if(!common::charge(work,10)){return .err(1002);}if(request.len-at<36){return .err(1001);}if(!common::byte_charge(work,36)){return .err(1002);}
        let key=words(request[at..at+32]);let size=number(request,at+32,4);at+=36;
        if(size>request.len-at || i!=0 && compare(previous,key)>=0){return .err(1001);}
        if(!v::push<Snapshot>(ref mut rows,Snapshot{.key,.start=at,.end=at+size}).is_ok){return .err(1003);}at+=size;previous=key;i+=1;
    }
    if(at!=request.len){return .err(1001);}
    return .ok(Evidence{.data=null,.capacity=0,.alignment=0,.length=0,.request_size=request.len,.limits,.snapshot=rows,.read_rows=0,.read_size=0,.read_capacity=0,.write_rows=0,.write_size=0,.write_capacity=0,.value_bytes=0,.write_bytes=0,.error=0});
}
fn usize snapshot_count(ref Evidence e){return v::len<Snapshot>(ref e.snapshot);}
fn Snapshot snapshot_at(ref Evidence e,usize index){return *v::at<Snapshot>(ref e.snapshot,index);}
fn Result<usize,u32> snapshot_index(ref Evidence e,u32[8] key,ref mut common::Work work){
    var usize lo=0;var usize hi=v::len<Snapshot>(ref e.snapshot);
    while(lo<hi){if(!common::charge(work,9)){return .err(1002);}let mid=lo+(hi-lo)/2;let order=compare(v::at<Snapshot>(ref e.snapshot,mid).key,key);if(order<0){lo=mid+1;}else{hi=mid;}}return .ok(lo);
}
fn Result<usize,u32> write_index(ref Evidence e,u32[8] key,ref mut common::Work work){
    var usize lo=0;var usize hi=write_count(e);
    while(lo<hi){if(!common::charge(work,9)){return .err(1002);}let mid=lo+(hi-lo)/2;let order=compare(write_at(e,mid).key,key);if(order<0){lo=mid+1;}else{hi=mid;}}return .ok(lo);
}
fn Result<u32[8],u32> physical(state::Field field,ref u8[] key,ref mut common::Work work){
    if(!common::charge(work,72) || !common::byte_charge(work,key.len+232)){return .err(1002);}
    var u8[40] suffix=[0;40];var usize i=0;var u32 shift=24;
    while(i<32){suffix[i]=octet(((field.prefix[i/4]>>shift)&255) as usize);i+=1;if(i%4==0){shift=24;}else{shift-=8;}}
    var usize size=key.len;i=0;while(i<8){suffix[39-i]=octet(size%256);size/=256;i+=1;}
    var hash=sha::empty();if(!sha::update(ref mut hash,b"fairyc-contract-state-map-entry/v1") || !sha::update(ref mut hash,ref suffix) || !sha::update(ref mut hash,key)){return .err(1002);}return .ok(sha::finish(ref hash));
}
fn u32 literal(m::Operand operand,ref m::Graph graph,ref u8[] value,ref mut common::Work work){
    if(operand.input!=0){return 0;}let data=m::data(graph);if(operand.start>operand.end || operand.end>data.len){return 1001;}return same(data[operand.start..operand.end],value,work);
}
fn u32 observe(ref mut Evidence e,u32[8] key,u32 codec,bool borrow,ref u8[] request,ref u8[] value,u8 found,usize operations,ref mut common::Work work){
    if(operations==0){return 1001;}
    let wi=switch(write_index(e,key,work)){case .ok(i):{i}case .err(c):{return c;}};
    var bool present=false;var u32 code=0;
    if(wi<write_count(e) && compare(write_at(e,wi).key,key)==0){
        let row=write_at(e,wi);present=!row.deleted;
        if(present || !borrow){code=same(content(e)[row.start..row.end],value,work);}
    }else{
        let si=switch(snapshot_index(e,key,work)){case .ok(i):{i}case .err(c):{return c;}};
        if(si<snapshot_count(e) && compare(snapshot_at(e,si).key,key)==0){
            let row=snapshot_at(e,si);if(row.start>row.end || row.end>request.len){return 1001;}present=true;code=same(request[row.start..row.end],value,work);
        }else if(!borrow && value.len!=0){return 1001;}
    }
    if(code!=0){return code;}if(present!=(found==1)){return 1001;}
    if(!present && borrow){code=absent(codec,value,work);}else if(present){code=canonical(codec,value,work);}if(code!=0){return code;}
    let count=read_count(e);if(count>=1677721){return 1002;}
    code=table(e,true,work);if(code!=0){return code;}let start=e.length;code=append(e,value,work);if(code!=0){return code;}
    if(!common::charge(work,40) || !common::byte_charge(work,40)){return 1002;}let at=e.read_rows+count*40;
    {let raw=content_mut(e);put(raw,at,operations-1,8);put(raw,at+8,if(present){1}else{0},8);put(raw,at+16,codec as usize,8);put(raw,at+24,start,8);put(raw,at+32,start+value.len,8);}
    e.read_size+=1;return 0;
}
fn u32 store(ref mut Evidence e,u32[8] key,bool deleted,ref u8[] value,ref mut common::Work work){
    let at=switch(write_index(e,key,work)){case .ok(i):{i}case .err(c):{return c;}};
    let count=write_count(e);let replaced=at<count && compare(write_at(e,at).key,key)==0;
    var usize total=e.write_bytes;if(replaced){let old=write_at(e,at);total-=33+old.end-old.start;}total+=33+value.len;
    if(count+if(replaced){0}else{1}>e.limits.writes || total>e.limits.bytes){return 1001;}
    let usize shifted=if(replaced){0}else{(count-at)*56};if(!common::charge(work,56) || !common::byte_charge(work,shifted+56)){return 1002;}
    if(!replaced){let code=table(e,false,work);if(code!=0){return code;}}
    let start=e.length;let copied=append(e,value,work);if(copied!=0){return copied;}let base=e.write_rows;
    {let raw=content_mut(e);var usize j=count*56;if(!replaced){while(j>at*56){raw[base+j+55]=raw[base+j-1];j-=1;}}
        put_key(raw[base+at*56..base+at*56+32],key);put(raw,base+at*56+32,if(deleted){1}else{0},8);put(raw,base+at*56+40,start,8);put(raw,base+at*56+48,start+value.len,8);}
    if(!replaced){e.write_size+=1;}e.write_bytes=total;return 0;
}
fn u32 admission(ref mut Evidence e,ref program::Verified proof,common::Step step,ref u8[] request,ref u8[] key_bytes,ref u8[] value,u8 found,usize operations,ref mut common::Work work){
    if(request.len!=e.request_size || request.len<152){return 1001;}if(key_bytes.len>67108864 || value.len>67108864){return 1002;}
    if(step.body>program::helper_count(proof)){return 1001;}
    let graph=if(step.body==0){program::root(proof)}else{program::helper(proof,step.body-1)};
    if(!step.operation){return if(step.index<v::len<m::Block>(m::blocks(graph)) && key_bytes.len==0 && value.len==0 && found==0){0}else{1001};}
    if(step.index>=v::len<m::Operation>(m::operations(graph))){return 1001;}let op=*v::at<m::Operation>(m::operations(graph),step.index);
    if(op.kind>=9 && op.kind<=11){return if(key_bytes.len==0 && value.len==0 && found==0){0}else{1001};}
    if(op.kind<1 || op.kind>8 || found>1){return 1001;}
    let layouts=program::layouts(proof);let layout=state::at(layouts,m::header(graph).layout);if(op.field>=layout.field_count){return 1001;}let field=state::field_at(layouts,layout.field_start+op.field);
    var u32[8] key=field.prefix;
    if(field.kind==2){
        if(key_bytes.len>e.limits.key){return 1001;}let code=canonical(field.key,key_bytes,work);if(code!=0){return code;}
        let matched=literal(op.key,graph,key_bytes,work);if(matched!=0){return matched;}
        key=switch(physical(field,key_bytes,work)){case .ok(k):{k}case .err(c):{return c;}};
    }else if(key_bytes.len!=0){return 1001;}
    if(op.kind==1 || op.kind==4 || op.kind==7 || op.kind==8){return observe(e,key,field.value,op.kind==7 || op.kind==8,request,value,found,operations,work);}
    let deleted=op.kind==3 || op.kind==6;
    if(deleted){if(found!=0 || value.len!=0){return 1001;}}
    else{if(found!=1 || value.len>e.limits.value){return 1001;}let code=canonical(field.value,value,work);if(code!=0){return code;}let matched=literal(op.value,graph,value,work);if(matched!=0){return matched;}}
    return store(e,key,deleted,value,work);
}
pub fn u32 admit(ref mut Evidence evidence,ref program::Verified proof,common::Step step,ref u8[] request,ref u8[] key_bytes,ref u8[] value,u8 found,usize operations,ref mut common::Work work){
    let entered=enter(work);if(evidence.error!=0){return evidence.error;}if(!entered){evidence.error=1002;return 1002;}
    if(!common::charge(work,32)){evidence.error=1002;return 1002;}
    let code=admission(evidence,proof,step,request,key_bytes,value,found,operations,work);if(code!=0){evidence.error=code;}return code;
}
// The sealed caller consumes this return code before publishing any Result.
// On rollback the wire must have no writes; the private overlay stays available
// for admission only, and Result accessors must gate it by the commit outcome.
pub fn u32 finish(ref Evidence evidence,ref u8[] output,ref v::Vec<common::Write> writes,usize claimed_bytes,bool commit,ref mut common::Work work){
    let entered=enter(work);if(evidence.error!=0){return evidence.error;}if(!entered){return 1002;}if(output.len>67108864){return 1002;}
    let count=v::len<common::Write>(writes);if(!commit){return if(count==0 && claimed_bytes==0){0}else{1001};}
    if(count!=write_count(evidence) || claimed_bytes!=evidence.write_bytes){return 1001;}
    var usize i=0;var usize total=0;
    while(i<count){
        if(!common::charge(work,18)){return 1002;}let received=*v::at<common::Write>(writes,i);let expected=write_at(evidence,i);
        if(received.start>received.end || received.end>output.len || compare(received.key,expected.key)!=0 || received.deleted!=expected.deleted || received.deleted && received.start!=received.end){return 1001;}
        if(i!=0 && compare(v::at<common::Write>(writes,i-1).key,received.key)>=0){return 1001;}
        let code=same(output[received.start..received.end],content(evidence)[expected.start..expected.end],work);if(code!=0){return code;}
        let cost=33+received.end-received.start;if(cost>claimed_bytes || total>claimed_bytes-cost){return 1001;}total+=cost;i+=1;
    }return if(total==claimed_bytes){0}else{1001};
}
