module project_mcp_json;
import project_json_tree as json;
import std::bytes as bytes;
import std::text as text;

fn u8 folded(u8 c){return if(c>=65 && c<=90){c+32}else{c};}
pub fn bool equal(ref u8[] a,ref u8[] b,bool fold){if(a.len!=b.len){return false;}var usize i=0;while(i<a.len){if(if(fold){folded(a[i])!=folded(b[i])}else{a[i]!=b[i]}){return false;}i+=1;}return true;}
// Struct field matching follows encoding/json's case-insensitive last match;
// tool argument map keys use exact matching instead.
pub fn usize field(ref json::Tree t,usize object,ref u8[] name,bool fold){let absent=json::count(t);if(object>=absent || json::token(t,object).kind!=1){return absent;}var usize found=absent;var usize i=object+1;while(json::token(t,i).kind!=2){if(equal(json::value(t,i),name,fold)){found=i+2;}i=json::next(t,i+2);if(json::token(t,i).kind==6){i+=1;}}return found;}
pub fn u32 kind(ref json::Tree t,usize at){return if(at<json::count(t)){json::token(t,at).kind}else{0};}
pub fn ref u8[] string(ref json::Tree t,usize at){return if(kind(t,at)==7){json::value(t,at)}else{json::source(t)[..0]};}
pub fn usize fields(ref json::Tree t,usize object){if(kind(t,object)!=1){return 0;}var usize count=0;var usize i=object+1;while(json::token(t,i).kind!=2){count+=1;i=json::next(t,i+2);if(json::token(t,i).kind==6){i+=1;}}return count;}
pub fn ref u8[] raw(ref json::Tree t,usize at){if(at>=json::count(t)){return json::source(t)[..0];}let first=json::token(t,at);let last=json::token(t,json::next(t,at)-1);return json::source(t)[first.start..last.end];}
pub fn bool empty_params(ref json::Tree t,usize at){if(at==json::count(t)){return true;}if(kind(t,at)!=1){return false;}let n=fields(t,at);return n==0 || n==1 && kind(t,field(t,at,b"_meta",false))==1;}
pub fn bool valid_id(ref json::Tree t,usize at){if(at==json::count(t)){return true;}let data=raw(t,at);if(data.len>512){return false;}if(kind(t,at)==7){return true;}if(kind(t,at)!=8 || data.len>20){return false;}var usize i=if(data.len>0 && data[0]==45){1}else{0};if(i==data.len){return false;}while(i<data.len){if(data[i]<48 || data[i]>57){return false;}i+=1;}return true;}
// Per-frame parsing owns all bytes. Duplicate decoded keys are rejected by the
// shared tree reader before any session transition or query selection.
pub fn Result<json::Tree,u32> parse(bytes::Buffer input){if(bytes::len(ref input)>65536 || !text::validate_utf8(bytes::view(ref input)).is_ok){return .err(1);}var usize meter=0;let t=switch(json::parse(input,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(1);}};var usize depth=0;var usize i=0;while(i<json::count(ref t)){let k=json::token(ref t,i).kind;if(k==2 || k==4){depth-=1;}else if(k==1 || k==3 || k>=7){if(depth>16){return .err(1);}if(k==1 || k==3){depth+=1;}}i+=1;}return .ok(t);}

pub fn bool valid_output(ref u8[] data){if(data.len>4194304 || !text::validate_utf8(data).is_ok){return false;}var bytes::Buffer input=bytes::empty();if(!bytes::append(ref mut input,data).is_ok){return false;}var usize meter=0;let t=switch(json::parse(input,ref mut meter)){case .ok(p):{p}case .err(_):{return false;}};return json::count(ref t)>0;}
