module module_public_interface_write;
import module_public_interface_model as model;
import module_state_layout as work;
import std::bytes as bytes;

pub fn u32 raw(ref mut bytes::Buffer out,ref u8[] data,ref mut usize meter){if(*meter>1048576 || data.len>=1048576 || data.len+1>1048576-*meter){return 10;}return if(work::append(out,data,meter)){0}else{11};}
fn u8 octet(usize n){var usize remaining=n&255;var u8 result=0;var usize bit=128;var u8 mask=128;while(bit!=0){if(remaining>=bit){result+=mask;remaining-=bit;}bit/=2;mask/=2;}return result;}
pub fn u32 number(ref mut bytes::Buffer out,usize n,ref mut usize meter){var usize remaining=n;var u8[8] data=[0;8];var usize i=0;while(i<8){data[7-i]=octet(remaining);remaining/=256;i+=1;}if(remaining!=0){return 9;}let ref u8[] view=ref data;return raw(out,view,meter);}
pub fn u32 flag(ref mut bytes::Buffer out,bool value,ref mut usize meter){var u8[1] data=[0;1];if(value){data[0]=1;}return raw(out,ref data,meter);}
pub fn u32 text(ref model::Descriptor p,model::Text t,ref mut bytes::Buffer out,ref mut usize meter){if(!model::valid_text(p,t)){return 9;}let n=number(out,t.end-t.start,meter);return if(n!=0){n}else{raw(out,model::text(p,t),meter)};}
pub fn u32 strings(ref model::Descriptor p,model::Range r,ref mut bytes::Buffer out,ref mut usize meter){if(!model::valid_range(r,model::list_count(p))){return 9;}let n=number(out,r.count,meter);if(n!=0){return n;}var usize i=0;while(i<r.count){if(!work::charge(meter,1)){return 10;}let code=text(p,model::list_at(p,r.first+i),out,meter);if(code!=0){return code;}i+=1;}return 0;}
pub fn u32 fields(ref model::Descriptor p,model::Range r,ref mut bytes::Buffer out,ref mut usize meter){if(!model::valid_range(r,model::field_count(p))){return 9;}let n=number(out,r.count,meter);if(n!=0){return n;}var usize i=0;while(i<r.count){if(!work::charge(meter,1)){return 10;}let f=model::field_at(p,r.first+i);let name=text(p,f.name,out,meter);if(name!=0){return name;}let type=text(p,f.type,out,meter);if(type!=0){return type;}i+=1;}return 0;}
pub fn u32 variants(ref model::Descriptor p,model::Range r,ref mut bytes::Buffer out,ref mut usize meter){if(!model::valid_range(r,model::variant_count(p))){return 9;}let n=number(out,r.count,meter);if(n!=0){return n;}var usize i=0;while(i<r.count){if(!work::charge(meter,1)){return 10;}let v=model::variant_at(p,r.first+i);let name=text(p,v.name,out,meter);if(name!=0){return name;}let code=fields(p,v.fields,out,meter);if(code!=0){return code;}i+=1;}return 0;}
