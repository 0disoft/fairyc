module project_atlas_model;
import project_atlas_wire as wire;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub fn bool text(ref u8[] s){if(s.len==0 || s.len>4096){return false;}var usize i=0;while(i<s.len){if(s[i]<32 || s[i]==127 || s[i]==194 && i+1<s.len && s[i+1]>=128 && s[i+1]<=159){return false;}i+=1;}return true;}
pub fn bool identifier(ref u8[] s){if(s.len==0){return false;}var usize i=0;while(i<s.len){let c=s[i];if(!(c==95 || c>=65 && c<=90 || c>=97 && c<=122 || i>0 && c>=48 && c<=57)){return false;}i+=1;}return true;}
pub fn bool qualified(ref u8[] s){var usize start=0;var usize i=0;while(i<s.len){if(s[i]==58){if(i+1==s.len || s[i+1]!=58 || !identifier(s[start..i])){return false;}i+=2;start=i;}else{i+=1;}}return identifier(s[start..]);}
pub fn bool before(ref u8[] a,ref u8[] b){var usize i=0;while(i<a.len && i<b.len){if(a[i]!=b[i]){return a[i]<b[i];}i+=1;}return a.len<b.len;}
pub fn bool eq(ref json::Tree t,usize a,usize b,ref u8[] name){return bytes::equal(wire::string(t,a,name),wire::string(t,b,name));}
pub fn i32 keys(ref json::Tree t,usize a,usize b,ref mut usize meter){
 if(!work::charge(meter,wire::string(t,a,b"package_instance").len+wire::string(t,b,b"package_instance").len+1)){return 2;}if(!eq(t,a,b,b"package_instance")){return if(before(wire::string(t,a,b"package_instance"),wire::string(t,b,b"package_instance"))){-1}else{1};}
 if(!work::charge(meter,wire::string(t,a,b"module").len+wire::string(t,b,b"module").len+1)){return 2;}if(!eq(t,a,b,b"module")){return if(before(wire::string(t,a,b"module"),wire::string(t,b,b"module"))){-1}else{1};}
 if(!work::charge(meter,wire::string(t,a,b"kind").len+wire::string(t,b,b"kind").len+1)){return 2;}if(!eq(t,a,b,b"kind")){return if(before(wire::string(t,a,b"kind"),wire::string(t,b,b"kind"))){-1}else{1};}
 if(!work::charge(meter,wire::string(t,a,b"name").len+wire::string(t,b,b"name").len+1)){return 2;}if(!eq(t,a,b,b"name")){return if(before(wire::string(t,a,b"name"),wire::string(t,b,b"name"))){-1}else{1};}
 return 0;
}
pub fn i32 spans(ref json::Tree t,usize a,usize b,ref mut usize meter){if(!work::charge(meter,wire::string(t,a,b"package_instance").len+wire::string(t,b,b"package_instance").len+wire::string(t,a,b"source").len+wire::string(t,b,b"source").len+1)){return 2;}if(!eq(t,a,b,b"package_instance")){return if(before(wire::string(t,a,b"package_instance"),wire::string(t,b,b"package_instance"))){-1}else{1};}if(!eq(t,a,b,b"source")){return if(before(wire::string(t,a,b"source"),wire::string(t,b,b"source"))){-1}else{1};}let x=wire::integer(t,a,b"start_byte");let y=wire::integer(t,b,b"start_byte");if(x!=y){return if(x<y){-1}else{1};}let u=wire::integer(t,a,b"end_byte");let w=wire::integer(t,b,b"end_byte");return if(u==w){0}else if(u<w){-1}else{1};}
pub fn bool kind(ref u8[] s){return bytes::equal(s,b"struct") || bytes::equal(s,b"enum") || bytes::equal(s,b"resource") || bytes::equal(s,b"function") || bytes::equal(s,b"foreign-function");}
pub fn Result<v::Vec<usize>,u32> rows(ref json::Tree t,usize at,usize maximum,ref mut usize meter){if(at>=json::count(t) || json::token(t,at).kind!=3){return .err(7);}var v::Vec<usize> out=v::empty<usize>();var usize i=at+1;while(json::token(t,i).kind!=4){if(v::len<usize>(ref out)>=maximum || !work::charge(meter,1)){return .err(3);}if(!v::push<usize>(ref mut out,i).is_ok){return .err(2);}i=json::next(t,i);if(json::token(t,i).kind==6){i+=1;}}return .ok(out);}
pub fn bool sorted(ref json::Tree t,usize at,ref mut usize meter){let values=switch(rows(t,at,4096,meter)){case .ok(p):{p}case .err(_):{return false;}};var usize i=0;while(i<v::len<usize>(ref values)){let a=*v::at<usize>(ref values,i);if(!text(json::value(t,a)) || i>0 && !before(json::value(t,*v::at<usize>(ref values,i-1)),json::value(t,a))){return false;}i+=1;}return true;}
pub fn usize find(ref json::Tree t,ref v::Vec<usize> rows,ref u8[] field,ref u8[] value,ref mut usize meter){var usize i=0;while(i<v::len<usize>(rows)){let at=*v::at<usize>(rows,i);let s=wire::string(t,at,field);if(!work::charge(meter,s.len+value.len+1)){return json::count(t);}if(bytes::equal(s,value)){return at;}i+=1;}return json::count(t);}
pub fn usize find_key(ref json::Tree t,ref v::Vec<usize> rows,ref u8[] field,usize key,ref mut usize meter){var usize i=0;while(i<v::len<usize>(rows)){if(!work::charge(meter,256)){return json::count(t);}let at=*v::at<usize>(rows,i);if(keys(t,wire::field(t,at,field),key,meter)==0){return at;}i+=1;}return json::count(t);}
pub fn bool module_path(ref u8[] path,ref u8[] module_name){if(path.len<4 || !bytes::equal(path[path.len-3..],b".fy")){return false;}var usize a=0;var usize b=0;while(a<path.len-3){if(path[a]==47){if(b+1>=module_name.len || module_name[b]!=58 || module_name[b+1]!=58){return false;}b+=2;}else{if(b>=module_name.len || path[a]!=module_name[b]){return false;}b+=1;}a+=1;}return b==module_name.len && qualified(module_name);}
pub fn bool span(ref json::Tree t,usize at,ref u8[] instance,ref v::Vec<usize> files,ref mut usize meter){return bytes::equal(wire::string(t,at,b"package_instance"),instance) && wire::integer(t,at,b"start_byte")<=wire::integer(t,at,b"end_byte") && wire::integer(t,at,b"end_byte")<=9223372036854775807 && find(t,files,b"path",wire::string(t,at,b"source"),meter)<json::count(t);}
pub fn bool type(ref json::Tree t,usize at,usize depth,ref mut usize meter){return bound_type(t,at,depth,0,false,meter);}
// Original declaration binders are canonical decimal indices, not display names.
// The ordinary reader keeps arity zero and disallows nominal arguments.
pub fn bool bound_type(ref json::Tree t,usize at,usize depth,usize arity,bool aggregates,ref mut usize meter){if(depth>=64 || !work::charge(meter,1)){return false;}let k=wire::string(t,at,b"kind");let name=wire::string(t,at,b"name");let coordinate=wire::string(t,at,b"package_coordinate");let args=switch(rows(t,wire::field(t,at,b"arguments"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};let n=v::len<usize>(ref args);
 if(wire::boolean(t,at,b"mutable") && !bytes::equal(k,b"borrow") && !bytes::equal(k,b"pointer") || wire::integer(t,at,b"length")!=0 && !bytes::equal(k,b"fixed-array")){return false;}
 if(bytes::equal(k,b"parameter")){let index=switch(wire::number(name)){case .ok(p):{p}case .err(_):{return false;}};if(index>=arity || coordinate.len!=0 || n!=0){return false;}}
 else if(bytes::equal(k,b"builtin")){if(!text(name) || coordinate.len!=0 || n!=0){return false;}}
 else if(bytes::equal(k,b"nominal")){if(!qualified(name) || !wire::hex(coordinate,b"fypc1-sha256-",64,false) || !aggregates && n!=0){return false;}}
 else{let usize expected=if(bytes::equal(k,b"result")){2}else if(bytes::equal(k,b"slice") || bytes::equal(k,b"borrow") || bytes::equal(k,b"pointer") || bytes::equal(k,b"fixed-array") || bytes::equal(k,b"option")){1}else{0};if(expected==0 || n!=expected || name.len!=0 || coordinate.len!=0){return false;}if(bytes::equal(k,b"slice")){let element=wire::string(t,*v::at<usize>(ref args,0),b"kind");if(!bytes::equal(element,b"nominal") && !(arity>0 && bytes::equal(element,b"parameter"))){return false;}}}
 var usize i=0;while(i<n){if(!bound_type(t,*v::at<usize>(ref args,i),depth+1,arity,aggregates,meter)){return false;}i+=1;}return true;
}
// Exact raw JSON of a structurally checked node; scalar values use token.end.
pub fn ref u8[] raw(ref json::Tree t,usize at){let a=json::token(t,at);let b=json::token(t,json::next(t,at)-1);return json::source(t)[a.start..b.end];}
