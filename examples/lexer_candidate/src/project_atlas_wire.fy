module project_atlas_wire;
import project_json_tree as json;
import project_json_decode as decode;
import project_json_string as quoted;
import project_path as work;
import sha256_stream as sha;
import std::bytes as bytes;
import std::text as text;

// Private transport-only view. No semantic/current-source trust is implied.
// V3 transport includes original generic declarations and separate concrete
// function/aggregate/cleanup records. Semantic/current-source admission remains
// separate; V1 rejects generic fields rather than discarding them.
pub resource Document{json::Tree tree;usize map;usize header;u32 version;cleanup{}}
pub fn ref json::Tree tree(ref Document d){return ref d.tree;}
pub fn usize header(ref Document d){return d.header;}
pub fn usize map(ref Document d){return d.map;}
pub fn u32 version(ref Document d){return d.version;}
pub fn decode::Problem fail(u32 code){return decode::Problem{.code,.start=0,.end=0};}
fn ref u8[] layout(u32 id){
 if(id==1){return b"schema|encoding|checksum_algorithm|checksum|map";}
 if(id==2){return b"header|files|modules|symbols|declarations|functions|completeness";}
 if(id==3){return b"logical_schema|snapshot_model|compiler_build|compiler_build_algorithm|language|package_uid|package_instance|package_coordinate|package_source|package_source_kind|runtime_abi|target_scope|target|package_instance_algorithm|package_coordinate_algorithm|package_source_algorithm|source_file_algorithm|line_map_algorithm|symbol_algorithm|interface_algorithm|interface_hash|features|dependencies|package_references|generation_status|omissions|counts";}
 if(id==4){return b"files|modules|symbols|functions|relations|completeness";}
 if(id==5){return b"uid|instance|coordinate";}
 if(id==6){return b"package_instance|path|digest|line_map|kind";}
 if(id==7){return b"package_instance|path|source";}
 if(id==8){return b"package_instance|module|kind|name";}
 if(id==9){return b"package_instance|source|start_byte|end_byte";}
 if(id==10){return b"kind|name|package_coordinate|mutable|length|arguments";}
 if(id==11){return b"name|type|mutable|span";}
 if(id==12){return b"name|type|span";}
 if(id==13){return b"name|span|fields";}
 if(id==14){return b"key|span|provenance|public|unsafe|foreign|abi|link_name|type|owner|receiver|parameters|effects|fields|variants|repr_c|drop_kind|drop_effects";}
 if(id==15){return b"declaration|visible|full";}
 if(id==16){return b"span|target_kind|target|certainty|abi|link_name";}
 if(id==17){return b"span|kind|escapes";}
 if(id==18){return b"function|provenance|calls|trap_categories|unsafe_sites|move_sites|drop_sites|allocation_sites|return_error_sites|borrows";}
 if(id==19){return b"relation|status|resolved|external|unknown|reasons";}
 if(id==20){return b"logical_schema|snapshot_model|compiler_build|compiler_build_algorithm|language|package_uid|package_instance|package_coordinate|package_source|package_source_kind|runtime_abi|target_scope|target|package_instance_algorithm|package_coordinate_algorithm|package_source_algorithm|source_file_algorithm|line_map_algorithm|symbol_algorithm|interface_algorithm|interface_hash|generic_interface_algorithm|generic_interface_hash|features|dependencies|package_references|generation_status|omissions|counts";}
 if(id==21){return b"type_parameters|key|span|provenance|public|unsafe|foreign|abi|link_name|type|owner|receiver|parameters|effects|fields|variants|repr_c|drop_kind|drop_effects";}
 if(id==22){return b"specializations|header|files|modules|symbols|declarations|functions|completeness";}
 if(id==23){return b"id|declaration|type_arguments|function";}
 if(id==24){return b"id|declaration|type_arguments|aggregate";}
 if(id==25){return b"id|declaration|type_arguments|aggregate|cleanup";}
 if(id==26){return b"specializations|files|modules|symbols|functions|relations|completeness";}
 if(id==27){return b"specialization|span|target_kind|target|certainty|abi|link_name";}
 return b"";
}
fn u32 child(u32 id,usize index){
 if(id==1){if(index==4){return 2;}return 100;}
 if(id==2){if(index==0){return 3;}if(index==1){return 206;}if(index==2){return 207;}if(index==3){return 215;}if(index==4){return 214;}if(index==5){return 218;}if(index==6){return 219;}return 100;}
 if(id==3){if(index==21){return 300;}if(index==22){return 300;}if(index==23){return 205;}if(index==25){return 300;}if(index==26){return 4;}return 100;}
 if(id==4){if(index==0){return 101;}if(index==1){return 101;}if(index==2){return 101;}if(index==3){return 101;}if(index==4){return 101;}if(index==5){return 101;}return 100;}
 if(id==5){return 100;}
 if(id==6){return 100;}
 if(id==7){return 100;}
 if(id==8){return 100;}
 if(id==9){if(index==2){return 101;}if(index==3){return 101;}return 100;}
 if(id==10){if(index==3){return 102;}if(index==4){return 101;}if(index==5){return 210;}return 100;}
 if(id==11){if(index==1){return 10;}if(index==2){return 102;}if(index==3){return 9;}return 100;}
 if(id==12){if(index==1){return 10;}if(index==2){return 9;}return 100;}
 if(id==13){if(index==1){return 9;}if(index==2){return 212;}return 100;}
 if(id==14){if(index==0){return 8;}if(index==1){return 9;}if(index==3){return 102;}if(index==4){return 102;}if(index==5){return 102;}if(index==8){return 10;}if(index==9){return 103;}if(index==11){return 211;}if(index==12){return 300;}if(index==13){return 212;}if(index==14){return 213;}if(index==15){return 102;}if(index==17){return 300;}return 100;}
 if(id==15){if(index==0){return 8;}return 100;}
 if(id==16){if(index==0){return 9;}if(index==2){return 8;}return 100;}
 if(id==17){if(index==0){return 9;}if(index==2){return 102;}return 100;}
 if(id==18){if(index==0){return 8;}if(index==2){return 216;}if(index==3){return 300;}if(index==4){return 209;}if(index==5){return 209;}if(index==6){return 209;}if(index==7){return 209;}if(index==8){return 209;}if(index==9){return 217;}return 100;}
 if(id==19){if(index==2){return 101;}if(index==3){return 101;}if(index==4){return 101;}if(index==5){return 300;}return 100;}
 if(id==20){if(index==23 || index==24 || index==27){return 300;}if(index==25){return 205;}if(index==28){return 4;}return 100;}
 if(id==21){return if(index==0){300}else{child(14,index-1)};}
 if(id==22){return if(index==0){223}else{child(2,index-1)};}
 if(id==23 || id==24 || id==25){if(index==1){return 8;}if(index==2){return 210;}if(index==3){return if(id==23){18}else{14};}if(index==4){return 18;}return 100;}
 if(id==26){return 101;}
 if(id==27){return if(index==0){100}else{child(16,index-1)};}
 return 0;
}
pub fn usize field(ref json::Tree t,usize at,ref u8[] name){if(at>=json::count(t) || json::token(t,at).kind!=1){return json::count(t);}var usize i=at+1;while(json::token(t,i).kind!=2){if(bytes::equal(json::value(t,i),name)){return i+2;}i=json::next(t,i+2);if(json::token(t,i).kind==6){i+=1;}}return json::count(t);}
pub fn ref u8[] string(ref json::Tree t,usize at,ref u8[] name){let i=field(t,at,name);return if(i<json::count(t)){json::value(t,i)}else{json::source(t)[..0]};}
pub fn usize count(ref json::Tree t,usize at){if(at>=json::count(t) || json::token(t,at).kind!=3){return 0;}var usize n=0;var usize i=at+1;while(json::token(t,i).kind!=4){n+=1;i=json::next(t,i);if(json::token(t,i).kind==6){i+=1;}}return n;}
pub fn usize item(ref json::Tree t,usize at,usize index){var usize left=index;var usize i=at+1;while(left>0 && json::token(t,i).kind!=4){i=json::next(t,i);if(json::token(t,i).kind==6){i+=1;}left-=1;}return i;}
pub fn Result<usize,u32> number(ref u8[] raw){if(raw.len==0 || raw.len>20 || raw.len>1 && raw[0]==48){return .err(1);}var usize n=0;var usize i=0;while(i<raw.len){let c=raw[i];if(c<48 || c>57 || n>(18446744073709551615-((c-48) as usize))/10){return .err(1);}n=n*10+((c-48) as usize);i+=1;}return .ok(n);}
pub fn usize integer(ref json::Tree t,usize at,ref u8[] name){let n=field(t,at,name);if(n>=json::count(t)){return 0;}return switch(number(json::value(t,n))){case .ok(n):{n}case .err(_):{0}};}
pub fn bool boolean(ref json::Tree t,usize at,ref u8[] name){let n=field(t,at,name);return n<json::count(t) && json::token(t,n).kind==9;}
pub fn bool hex(ref u8[] value,ref u8[] prefix,usize digits,bool nonzero){if(value.len!=prefix.len+digits || !bytes::equal(value[..prefix.len],prefix)){return false;}var usize i=prefix.len;var bool any=false;while(i<value.len){let c=value[i];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}if(c!=48){any=true;}i+=1;}return !nonzero || any;}
fn u32 shape(ref json::Tree t,usize at,u32 id,usize depth,ref mut usize meter){
 if(depth>160 || !work::charge(meter,1)){return 3;}let kind=json::token(t,at).kind;
 if(id==100){return if(kind==7){0}else{5};}if(id==101){return if(kind==8 && number(json::value(t,at)).is_ok){0}else{5};}if(id==102){return if(kind==9 || kind==10){0}else{5};}if(id==103){return if(kind==11){0}else{shape(t,at,10,depth+1,meter)};}
 if(id>=200){if(kind!=3){return 5;}var usize i=at+1;var usize n=0;while(json::token(t,i).kind!=4){if(n>=65536){return 3;}let code=shape(t,i,if(id==300){100}else{id-200},depth+1,meter);if(code!=0){return code;}n+=1;i=json::next(t,i);if(json::token(t,i).kind==6){i+=1;}}return 0;}
 if(kind!=1){return 5;}let u32 owner=if(id==2 && field(t,at,b"specializations")<json::count(t)){22}else if(id==3 && field(t,at,b"generic_interface_algorithm")<json::count(t)){20}else if(id==4 && field(t,at,b"specializations")<json::count(t)){26}else if(id==14 && field(t,at,b"type_parameters")<json::count(t)){21}else if(id==16 && field(t,at,b"specialization")<json::count(t)){27}else if(id==23 && field(t,at,b"aggregate")<json::count(t)){if(field(t,at,b"cleanup")<json::count(t)){25}else{24}}else{id};let names=layout(owner);var usize start=0;var usize index=0;var usize i=at+1;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}if(json::token(t,i).kind!=7 || !bytes::equal(json::value(t,i),names[start..end])){return 5;}let code=shape(t,i+2,child(owner,index),depth+1,meter);if(code!=0){return code;}i=json::next(t,i+2);if(json::token(t,i).kind==6){i+=1;}index+=1;start=end+1;}
 return if(json::token(t,i).kind==2){0}else{5};
}
pub fn u32 declaration_shape(ref json::Tree t,usize at,ref mut usize meter){if(at>=json::count(t)){return 5;}return shape(t,at,14,0,meter);}
fn u32 generic_fields(ref json::Tree t,u32 version,ref mut usize meter){
 var usize i=0;while(i+2<json::count(t)){
  if(!work::charge(meter,1)){return 3;}
  if(json::token(t,i).kind==7 && json::token(t,i+1).kind==5){let name=json::value(t,i);
   let array=bytes::equal(name,b"type_parameters") || bytes::equal(name,b"specializations") && json::token(t,i+2).kind==3;
   let count=bytes::equal(name,b"specializations") && json::token(t,i+2).kind==8;
   let text=bytes::equal(name,b"specialization") || bytes::equal(name,b"generic_interface_algorithm") || bytes::equal(name,b"generic_interface_hash");
   if(array || count || text){if(version!=3 || array && json::token(t,i+3).kind==4 || count && bytes::equal(json::value(t,i+2),b"0") || text && json::value(t,i+2).len==0){return 5;}}
  }i+=1;
 }return 0;
}
fn bool framed(ref mut sha::State state,ref u8[] data){var u8[8] size=[0;8];var usize n=data.len;var usize i=8;while(i>0){i-=1;let hex=b"0123456789abcdef";let u8 hi=hex[(n%256)/16];let u8 lo=hex[n%16];size[i]=((if(hi<=57){hi-48}else{hi-87})<<4)|(if(lo<=57){lo-48}else{lo-87});n/=256;}return sha::update(state,ref size) && sha::update(state,data);}
// Codes: JSON codes 1..4; 5 schema/canonical, 6 checksum. Caller meter is shared.
pub fn Result<Document,decode::Problem> read(bytes::Buffer source,usize maximum,ref mut usize meter){
 let size=bytes::len(ref source);if(maximum==0 || maximum>4194304 || size==0 || size>maximum){return .err(fail(3));}if(!text::validate_utf8(bytes::view(ref source)).is_ok){return .err(fail(5));}
 let t=switch(json::parse(source,meter)){case .ok(t):{t}case .err(p):{return .err(p);}};
 var usize i=0;var usize end=0;while(i<json::count(ref t)){let tok=json::token(ref t,i);if(tok.start!=end){return .err(fail(5));}end=tok.end;if(tok.kind==7){let value=switch(quoted::quote(json::value(ref t,i),meter)){case .ok(p):{p}case .err(_):{return .err(fail(3));}};if(!bytes::equal(bytes::view(ref value),json::source(ref t)[tok.start..tok.end])){return .err(fail(5));}}i+=1;}if(end!=size){return .err(fail(5));}
 let code=shape(ref t,0,1,0,meter);if(code!=0){return .err(fail(code));}
 let schema=string(ref t,0,b"schema");let encoding=string(ref t,0,b"encoding");let u32 version=if(bytes::equal(schema,b"fairyc.fymap/v1") && bytes::equal(encoding,b"fairyc.fymap-json/v1")){1}else if(bytes::equal(schema,b"fairyc.fymap/v3") && bytes::equal(encoding,b"fairyc.fymap-json/v3")){3}else{0};
 if(version==0 || !bytes::equal(string(ref t,0,b"checksum_algorithm"),b"fairyc.fymap-checksum/v1-sha256")){return .err(fail(5));}
 let map=field(ref t,0,b"map");let header=field(ref t,map,b"header");if(!bytes::equal(string(ref t,header,b"logical_schema"),if(version==1){b"fairyc.fymap-logical/v1"}else{b"fairyc.fymap-logical/v3"})){return .err(fail(5));}
 let generic=generic_fields(ref t,version,meter);if(generic!=0){return .err(fail(generic));}
 let digest=string(ref t,0,b"checksum");if(!hex(digest,b"fymap1-sha256-",64,false)){return .err(fail(6));}
 let begin=json::token(ref t,map).start;let stop=json::token(ref t,json::next(ref t,map)-1).end;var sha::State state=sha::empty();if(!sha::update(ref mut state,b"fairyc.fymap-checksum\0\x01") || !framed(ref mut state,schema) || !framed(ref mut state,encoding) || !framed(ref mut state,json::source(ref t)[begin..stop]) || !sha::matches(ref state,digest[14..])){return .err(fail(6));}
 return .ok(Document{.tree=t,.map,.header,.version});
}
