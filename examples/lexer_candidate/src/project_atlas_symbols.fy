module project_atlas_symbols;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_json_tree as json;
import project_path as work;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

fn u8 octet(usize n){let ref u8[] digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return digits[n/16]*16+digits[n%16];}
fn Result<bool,u32> add(ref mut bytes::Buffer out,ref u8[] data,ref mut usize meter){if(bytes::len(out)>1048576 || data.len>1048576-bytes::len(out) || !work::charge(meter,data.len+1)){return .err(3);}return if(bytes::append(out,data).is_ok){.ok(true)}else{.err(2)};}
fn Result<bool,u32> tag(ref mut bytes::Buffer out,u8 kind,ref mut usize meter){var u8[1] b=[kind];return add(out,ref b,meter);}
fn Result<bool,u32> number(ref mut bytes::Buffer out,usize n,ref mut usize meter){var u8[8] b=[0;8];var usize left=n;var usize i=8;while(i>0){i-=1;b[i]=octet(left%256);left/=256;}return add(out,ref b,meter);}
fn Result<bool,u32> text(ref mut bytes::Buffer out,u8 kind,ref u8[] value,ref mut usize meter){discard tag(out,kind,meter)?;discard number(out,value.len,meter)?;return add(out,value,meter);}
fn Result<bool,u32> boolean(ref mut bytes::Buffer out,u8 kind,bool value,ref mut usize meter){discard tag(out,kind,meter)?;return tag(out,if(value){1}else{0},meter);}
fn Result<bool,u32> strings(ref json::Tree t,usize at,ref mut bytes::Buffer out,u8 kind,ref mut usize meter){let rows=model::rows(t,at,4096,meter)?;discard tag(out,kind,meter)?;discard number(out,v::len<usize>(ref rows),meter)?;var usize i=0;while(i<v::len<usize>(ref rows)){discard text(out,115,json::value(t,*v::at<usize>(ref rows,i)),meter)?;i+=1;}return .ok(true);}
fn Result<bool,u32> type(ref json::Tree t,usize header,usize at,ref mut bytes::Buffer out,usize depth,ref mut usize meter){
 if(depth>=64){return .err(7);}discard tag(out,116,meter)?;discard text(out,107,wire::string(t,at,b"kind"),meter)?;discard text(out,110,wire::string(t,at,b"name"),meter)?;
 let coordinate=wire::string(t,at,b"package_coordinate");let refs=model::rows(t,wire::field(t,header,b"package_references"),33,meter)?;
 var usize ref_at=json::count(t);if(coordinate.len!=0){ref_at=model::find(t,ref refs,b"coordinate",coordinate,meter);if(ref_at==json::count(t)){return .err(7);}}
 if(ref_at<json::count(t)){discard text(out,117,wire::string(t,ref_at,b"uid"),meter)?;}else{discard text(out,117,b"",meter)?;}
 discard boolean(out,109,wire::boolean(t,at,b"mutable"),meter)?;discard tag(out,108,meter)?;discard number(out,wire::integer(t,at,b"length"),meter)?;
 let args=model::rows(t,wire::field(t,at,b"arguments"),4096,meter)?;discard tag(out,97,meter)?;discard number(out,v::len<usize>(ref args),meter)?;var usize i=0;
 while(i<v::len<usize>(ref args)){discard type(t,header,*v::at<usize>(ref args,i),out,depth+1,meter)?;i+=1;}return .ok(true);
}
fn Result<bool,u32> fields(ref json::Tree t,usize header,usize at,ref mut bytes::Buffer out,ref mut usize meter){let rows=model::rows(t,at,4096,meter)?;discard tag(out,100,meter)?;discard number(out,v::len<usize>(ref rows),meter)?;var usize i=0;while(i<v::len<usize>(ref rows)){let row=*v::at<usize>(ref rows,i);discard text(out,110,wire::string(t,row,b"name"),meter)?;discard type(t,header,wire::field(t,row,b"type"),out,0,meter)?;i+=1;}return .ok(true);}
fn Result<bytes::Buffer,u32> signature(ref json::Tree t,usize header,usize decl,ref mut usize meter){var bytes::Buffer out=bytes::empty();discard add(ref mut out,b"fairyc.symbol-signature\0\x01",meter)?;
 let key=wire::field(t,decl,b"key");discard text(ref mut out,107,wire::string(t,key,b"kind"),meter)?;
 discard boolean(ref mut out,117,wire::boolean(t,decl,b"unsafe"),meter)?;discard boolean(ref mut out,102,wire::boolean(t,decl,b"foreign"),meter)?;discard boolean(ref mut out,99,wire::boolean(t,decl,b"repr_c"),meter)?;
 discard text(ref mut out,97,wire::string(t,decl,b"abi"),meter)?;discard text(ref mut out,108,wire::string(t,decl,b"link_name"),meter)?;
 let owner=wire::field(t,decl,b"owner");let bool present=json::token(t,owner).kind!=11;discard boolean(ref mut out,111,present,meter)?;if(present){discard type(t,header,owner,ref mut out,0,meter)?;}
 discard text(ref mut out,114,wire::string(t,decl,b"receiver"),meter)?;discard type(t,header,wire::field(t,decl,b"type"),ref mut out,0,meter)?;
 let params=model::rows(t,wire::field(t,decl,b"parameters"),4096,meter)?;discard tag(ref mut out,112,meter)?;discard number(ref mut out,v::len<usize>(ref params),meter)?;var usize i=0;while(i<v::len<usize>(ref params)){discard type(t,header,wire::field(t,*v::at<usize>(ref params,i),b"type"),ref mut out,0,meter)?;i+=1;}
 discard strings(t,wire::field(t,decl,b"effects"),ref mut out,101,meter)?;discard fields(t,header,wire::field(t,decl,b"fields"),ref mut out,meter)?;
 let variants=model::rows(t,wire::field(t,decl,b"variants"),4096,meter)?;discard tag(ref mut out,118,meter)?;discard number(ref mut out,v::len<usize>(ref variants),meter)?;i=0;while(i<v::len<usize>(ref variants)){let row=*v::at<usize>(ref variants,i);discard text(ref mut out,110,wire::string(t,row,b"name"),meter)?;discard fields(t,header,wire::field(t,row,b"fields"),ref mut out,meter)?;i+=1;}
 discard text(ref mut out,120,wire::string(t,decl,b"drop_kind"),meter)?;discard strings(t,wire::field(t,decl,b"drop_effects"),ref mut out,121,meter)?;return .ok(out);
}
pub fn bool matches(ref json::Tree t,usize header,usize decl,usize symbol,ref mut usize meter){
 let sig=switch(signature(t,header,decl,meter)){case .ok(p):{p}case .err(_):{return false;}};var bytes::Buffer data=bytes::empty();
 if(!add(ref mut data,b"fairyc.symbol-identity\0\x01",meter).is_ok || !text(ref mut data,117,wire::string(t,header,b"package_uid"),meter).is_ok){return false;}
 let key=wire::field(t,decl,b"key");if(!text(ref mut data,109,wire::string(t,key,b"module"),meter).is_ok || !text(ref mut data,107,wire::string(t,key,b"kind"),meter).is_ok || !text(ref mut data,112,wire::string(t,key,b"name"),meter).is_ok || !tag(ref mut data,103,meter).is_ok || !number(ref mut data,0,meter).is_ok || !text(ref mut data,115,bytes::view(ref sig),meter).is_ok){return false;}
 let full=wire::string(t,symbol,b"full");let visible=wire::string(t,symbol,b"visible");if(!wire::hex(full,b"fysym1-sha256-256-",64,false) || !wire::hex(visible,b"fysym1-sha256-128-",32,false) || !bytes::equal(visible[18..],full[18..50])){return false;}
 var sha::State state=sha::empty();return sha::update(ref mut state,bytes::view(ref data)) && sha::matches(ref state,full[18..]);
}
