module module_state_interface;
import module_state_layout as state;
import std::bytes as bytes;
import std::vec as v;

// The v5 header and state section precede the module/item stream. This is a
// prefix, not a complete public-interface hash or a reusable package cache key.
fn u8 octet(usize n){var usize left=n&255;var u8 result=0;var usize bit=128;var u8 mask=128;while(bit!=0){if(left>=bit){result+=mask;left-=bit;}bit/=2;mask/=2;}return result;}
fn u32 put(ref mut bytes::Buffer out,ref u8[] data,ref mut usize work){if(*work>1048576 || data.len>=1048576 || data.len+1>1048576-*work){return 10;}return if(state::append(out,data,work)){0}else{11};}
fn u32 number(ref mut bytes::Buffer out,usize n,usize width,ref mut usize work){var u8[8] data=[0;8];var usize remaining=n;var usize i=0;while(i<width){data[width-i-1]=octet(remaining);remaining/=256;i+=1;}if(remaining!=0){return 9;}let ref u8[] raw=ref data;return put(out,raw[..width],work);}

// Rebuild only the selected package layout into an independent owner. Invalid
// unrelated packages cannot change this package's public compatibility record.
fn Result<state::Plan,u32> selected(ref state::Plan p,usize id,ref mut usize work){
    let row=state::at(p,id);if(row.uid_start>row.uid_end || row.uid_end>state::name_bytes(p) || row.field_count>1024 || row.field_start>state::field_count(p) || row.field_count>state::field_count(p)-row.field_start){return .err(9);}
    var bytes::Buffer names=bytes::empty();let copied=put(ref mut names,state::uid(p,id),work);if(copied!=0){return .err(copied);}
    var v::Vec<state::Field> fields=v::empty<state::Field>();var usize i=0;
    while(i<row.field_count){if(!state::charge(work,1)){return .err(10);}var field=state::field_at(p,row.field_start+i);if(field.name_start>field.name_end || field.name_end>state::name_bytes(p)){return .err(9);}let start=bytes::len(ref names);let copied=put(ref mut names,state::name(p,row.field_start+i),work);if(copied!=0){return .err(copied);}field.name_start=start;field.name_end=bytes::len(ref names);if(!v::push<state::Field>(ref mut fields,field).is_ok){return .err(11);}i+=1;}
    var layout=row;layout.owner=0;layout.module_index=0;layout.declaration=1;layout.uid_start=0;layout.uid_end=row.uid_end-row.uid_start;layout.field_start=0;
    var v::Vec<state::Layout> layouts=v::empty<state::Layout>();if(!v::push<state::Layout>(ref mut layouts,layout).is_ok){return .err(11);}let plan=state::assemble(names,layouts,fields);
    // Validate in its bounded history, then fold its complete charge once into
    // the caller's existing ledger before publishing any compatibility bytes.
    var usize validation_work=0;let code=state::validate(ref plan,ref mut validation_work);if(!state::charge(work,validation_work)){return .err(10);}if(code!=0){return .err(code);}return .ok(plan);
}

pub fn Result<bytes::Buffer,u32> prefix(ref state::Plan p,usize owner,ref mut usize work){
    if(owner>=512 || state::count(p)>512){return .err(9);}var usize found=state::count(p);var usize i=0;
    while(i<state::count(p)){if(!state::charge(work,1)){return .err(10);}if(state::at(p,i).owner==owner){if(found!=state::count(p)){return .err(9);}found=i;}i+=1;}
    var bytes::Buffer output=bytes::empty();let header=put(ref mut output,b"fairyc.public-interface\x00\x05",work);if(header!=0){return .err(header);}let count=number(ref mut output,if(found==state::count(p)){0}else{1},8,work);if(count!=0){return .err(count);}
    if(found==state::count(p)){return .ok(output);}
    let layout=switch(selected(p,found,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let uid=state::uid(ref layout,0);
    let length=number(ref mut output,uid.len,8,work);if(length!=0){return .err(length);}let text=put(ref mut output,uid,work);if(text!=0){return .err(text);}let hash=state::at(ref layout,0).hash;i=0;while(i<8){let word=number(ref mut output,hash[i] as usize,4,work);if(word!=0){return .err(word);}i+=1;}
    return .ok(output);
}
