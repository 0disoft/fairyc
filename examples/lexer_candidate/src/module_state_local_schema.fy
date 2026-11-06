module module_state_local_schema;
import module_state_layout as layout;
import module_state_local_codec as local;
import std::bytes as bytes;
import std::vec as v;

struct Cursor{usize at;}
struct Span{usize start;usize end;}
fn bool charge(ref mut local::Work work,usize n){if(work.structure>16777216 || n>16777216-work.structure){return false;}work.structure+=n;return true;}
fn bool byte_charge(ref mut local::Work work,usize n){if(work.bytes>1073741824 || n>1073741824-work.bytes){return false;}work.bytes+=n;return true;}
fn bool take(ref u8[] raw,ref mut Cursor cursor,ref u8[] expected){
    let start=cursor.at;if(start>raw.len || expected.len>raw.len-start){return false;}
    if(!bytes::equal(raw[start..start+expected.len],expected)){return false;}cursor.at=start+expected.len;return true;
}
// Each numeric position has a small bound. Reject before multiplying, and
// require the caller's exact delimiter after the last decimal digit.
fn Result<usize,u32> integer(ref u8[] raw,ref mut Cursor cursor,usize max,ref mut local::Work work){
    if(!charge(work,1)){return .err(1002);}let start=cursor.at;
    if(start>=raw.len || raw[start]<48 || raw[start]>57){return .err(1001);}var usize n=0;
    while(cursor.at<raw.len && raw[cursor.at]>=48 && raw[cursor.at]<=57){
        let digit=(raw[cursor.at] as usize)-48;if(digit>max || n>(max-digit)/10){return .err(1001);}n=n*10+digit;cursor.at+=1;
    }
    if(cursor.at-start>1 && raw[start]==48){return .err(1001);}return .ok(n);
}
// The only strings in this grammar are ASCII identities and known codec IDs.
// Escapes/control/non-ASCII bytes are never canonical for those strings.
fn Result<Span,u32> quoted(ref u8[] raw,ref mut Cursor cursor,usize max,ref mut local::Work work){
    if(!charge(work,1)){return .err(1002);}if(!take(raw,cursor,b"\"")){return .err(1001);}let start=cursor.at;
    while(cursor.at<raw.len && raw[cursor.at]!=34){
        let c=raw[cursor.at];if(cursor.at-start>=max || c<32 || c>126 || c==92){return .err(1001);}cursor.at+=1;
    }
    let end=cursor.at;if(!take(raw,cursor,b"\"")){return .err(1001);}return .ok(Span{.start,.end});
}
fn Result<usize,u32> copy_name(ref u8[] raw,Span span,bool uid,ref mut bytes::Buffer names,ref mut local::Work work){
    let text=raw[span.start..span.end];if(!layout::valid_name(text,uid)){return .err(1001);}
    if(bytes::len(names)>1048576 || text.len>1048576-bytes::len(names) || !byte_charge(work,text.len)){return .err(1002);}
    if(!bytes::append(names,text).is_ok){return .err(1003);}return .ok(bytes::len(names));
}
fn Result<u32,u32> codec(ref u8[] raw,ref mut Cursor cursor,ref mut local::Work work){
    let span=quoted(raw,cursor,10,work)?;let text=raw[span.start..span.end];var u32 id=0;
    while(id<=7){if(bytes::equal(text,layout::codec(id))){return .ok(id);}id+=1;}return .err(1001);
}
fn Result<u32[8],u32> read_key(ref u8[] raw,ref mut Cursor cursor,ref mut local::Work work){
    if(!take(raw,cursor,b"[")){return .err(1001);}var u32[8] words=[0;8];var usize i=0;
    while(i<32){
        if(i!=0 && !take(raw,cursor,b",")){return .err(1001);}let n=integer(raw,cursor,255,work)?;
        let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";
        let octet=(digits[n/16]<<4)|digits[n%16];words[i/4]=(words[i/4]<<8)|(octet as u32);i+=1;
    }
    if(!take(raw,cursor,b"]")){return .err(1001);}return .ok(words);
}
fn Result<layout::Field,u32> field(ref u8[] raw,ref mut Cursor cursor,usize index,ref mut bytes::Buffer names,ref mut local::Work work){
    if(!charge(work,1)){return .err(1002);}if(!take(raw,cursor,b"{\"Index\":")){return .err(1001);}
    if(integer(raw,cursor,1023,work)?!=index || !take(raw,cursor,b",\"Name\":")){return .err(1001);}
    let span=quoted(raw,cursor,127,work)?;let name_start=bytes::len(names);let name_end=copy_name(raw,span,false,names,work)?;
    if(!take(raw,cursor,b",\"Kind\":")){return .err(1001);}let kind=integer(raw,cursor,2,work)?;
    if(kind==0 || !take(raw,cursor,b",\"KeyTypeID\":")){return .err(1001);}let key=codec(raw,cursor,work)?;
    if(!take(raw,cursor,b",\"ValueTypeID\":")){return .err(1001);}let value=codec(raw,cursor,work)?;
    if(value==0 || kind==1 && key!=0 || kind==2 && key==0 || !take(raw,cursor,b",\"Prefix\":")){return .err(1001);}
    let prefix=read_key(raw,cursor,work)?;if(!take(raw,cursor,b"}")){return .err(1001);}
    return .ok(layout::Field{.kind=if(kind==1){1}else{2},.key,.value,.name_start,.name_end,.start=0,.end=0,.prefix});
}

// STRUCTURAL HEADER ONLY. This reconstructs even a foreign layout, without
// trusting a caller-selected schema. Immediately call local::decode(plan,0,
// encoded,limits,work) to admit the checksum and complete canonical record.
// Both operations share the caller's ledger; no history is reset here.
pub fn Result<layout::Plan,u32> read(ref u8[] encoded,local::Limits limits,ref mut local::Work work){
    if(!local::valid(limits)){return .err(1001);}
    if(!charge(work,1) || !byte_charge(work,0)){return .err(1002);}
    if(encoded.len<32 || encoded.len>limits.bytes || !byte_charge(work,encoded.len)){return .err(1002);}
    let raw=encoded[..encoded.len-32];var cursor=Cursor{.at=0};
    if(!take(raw,ref mut cursor,b"{\"schema\":\"fairyc.local-state/v1\",\"contractUid\":")){return .err(1001);}
    let uid=quoted(raw,ref mut cursor,255,work)?;var names=bytes::empty();let uid_end=copy_name(raw,uid,true,ref mut names,work)?;
    if(!take(raw,ref mut cursor,b",\"layoutHash\":")){return .err(1001);}let hash=read_key(raw,ref mut cursor,work)?;
    if(!take(raw,ref mut cursor,b",\"fields\":")){return .err(1001);}var fields=v::empty<layout::Field>();
    if(!take(raw,ref mut cursor,b"null")){
        if(!take(raw,ref mut cursor,b"[")){return .err(1001);}
        while(true){
            let index=v::len<layout::Field>(ref fields);if(index>=1024){return .err(1002);}
            if(index!=0 && !take(raw,ref mut cursor,b",")){return .err(1001);}
            let item=field(raw,ref mut cursor,index,ref mut names,work)?;
            if(!v::push<layout::Field>(ref mut fields,item).is_ok){return .err(1003);}
            if(take(raw,ref mut cursor,b"]")){break;}
        }
    }
    if(!take(raw,ref mut cursor,b",\"version\":")){return .err(1001);}
    if(!charge(work,1)){return .err(1002);}var rows=v::empty<layout::Layout>();
    if(!v::push<layout::Layout>(ref mut rows,layout::Layout{.owner=0,.module_index=0,.declaration=1,.start=0,.end=encoded.len,.uid_start=0,.uid_end,.field_start=0,.field_count=v::len<layout::Field>(ref fields),.hash}).is_ok){return .err(1003);}
    let plan=layout::assemble(names,rows,fields);
    // validate_graph has its own 1MiB local ledger. Prepay the whole bounded
    // operation, then use a fresh LOCAL counter, never a fresh caller Work.
    if(!charge(work,1048576)){return .err(1002);}var usize validation_work=0;
    let code=layout::validate_graph(ref plan,ref mut validation_work);
    if(code!=0){return .err(if(code==10){1002}else if(code==11){1003}else{1001});}return .ok(plan);
}
