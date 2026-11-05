module project_atlas_declarations;
import project_atlas_write as w;
import project_atlas_symbols as symbols;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_atlas_facts as facts;
import project_compiler_input as input;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;
import syntax as syntax;
import project_atlas_types as types;
import project_atlas_sdk_vector as vector;

pub struct Row{usize module_index;usize declaration;}
pub fn Result<bytes::Buffer,u32> name(ref syntax::Parsed p,usize declaration,ref mut usize meter){
 let d=syntax::declaration_at(p,declaration-1);var out=bytes::empty();if(d.owner_end!=0){discard w::raw(ref mut out,syntax::source_view(p)[d.owner_start..d.owner_end],meter)?;discard w::raw(ref mut out,b"::",meter)?;}discard w::raw(ref mut out,syntax::declaration_name(p,declaration),meter)?;return .ok(out);
}
pub fn ref u8[] kind(u32 kind){if(kind==4){return b"struct";}if(kind==6){return b"resource";}if(kind==11){return b"enum";}if(kind==9){return b"foreign-function";}return b"function";}
pub fn Result<v::Vec<Row>,u32> collect(ref input::Input p,usize owner,ref mut usize meter){return collect_mode(p,owner,false,false,false,false,meter);}
pub fn Result<v::Vec<Row>,u32> collect_all(ref input::Input p,usize owner,ref mut usize meter){return collect_mode(p,owner,true,false,false,false,meter);}
pub fn Result<v::Vec<Row>,u32> collect_generics(ref input::Input p,usize owner,ref mut usize meter){return collect_mode(p,owner,true,true,false,false,meter);}
pub fn Result<v::Vec<Row>,u32> collect_aggregates(ref input::Input p,usize owner,ref mut usize meter){return collect_mode(p,owner,true,true,true,false,meter);}
pub fn Result<v::Vec<Row>,u32> collect_sdk_vectors(ref input::Input p,usize owner,ref mut usize meter){return collect_mode(p,owner,true,true,false,true,meter);}
fn Result<bool,u32> before(ref input::Input p,Row a,Row b,ref mut usize meter){
 let left=input::source(p,a.module_index);let right=input::source(p,b.module_index);
 let lm=types::module_name(input::file(p,a.module_index),meter)?;let rm=types::module_name(input::file(p,b.module_index),meter)?;
 let l=bytes::view(ref lm);let r=bytes::view(ref rm);if(!bytes::equal(l,r)){return .ok(model::before(l,r));}
 let ld=syntax::declaration_at(left,a.declaration);let rd=syntax::declaration_at(right,b.declaration);
 let lk=kind(ld.kind);let rk=kind(rd.kind);if(!bytes::equal(lk,rk)){return .ok(model::before(lk,rk));}
 let left_name=if(ld.owner_end!=0){name(left,a.declaration+1,meter)?}else{bytes::empty()};let right_name=if(rd.owner_end!=0){name(right,b.declaration+1,meter)?}else{bytes::empty()};
 let ln=if(ld.owner_end!=0){bytes::view(ref left_name)}else{syntax::declaration_name(left,a.declaration+1)};let rn=if(rd.owner_end!=0){bytes::view(ref right_name)}else{syntax::declaration_name(right,b.declaration+1)};
 if(!work::charge(meter,ln.len+rn.len+1)){return .err(3);}return .ok(model::before(ln,rn));
}
fn Result<v::Vec<Row>,u32> collect_mode(ref input::Input p,usize owner,bool all,bool generics,bool aggregates,bool vectors,ref mut usize meter){
 var rows=v::empty<Row>();var usize m=0;
 while(m<input::count(p)){if(input::origin(p,m).owner==owner){
  let parsed=input::source(p,m);if(!all && syntax::closure_prepared(parsed)){return .err(101);}let sdk=if(vectors){vector::source(p,m,meter)?}else{false};var usize i=0;
  while(i<syntax::declaration_count(parsed)){
   if(!work::charge(meter,1)){return .err(3);}let d=syntax::declaration_at(parsed,i);let row=Row{.module_index=m,.declaration=i};i+=1;
   if(d.kind==1 || d.kind==2 || all && (d.kind==7 || d.kind==8)){continue;}
   let sdk_resource=sdk && d.kind==6 && bytes::equal(syntax::declaration_name(parsed,i),b"Vec");
   if(d.generic_parameters!=0 && (!generics || d.kind!=3 && !sdk_resource && (!aggregates || d.kind!=4 && d.kind!=6) || d.owner_end!=0) || !all && d.owner_end!=0){return .err(101);}
   if(all){if(d.kind!=3 && d.kind!=4 && d.kind!=6 && d.kind!=9 && d.kind!=11){return .err(101);}}
   else if(d.kind!=3 || d.unsafe_end!=d.unsafe_start || d.attribute_count!=0 || d.field_count!=0){return .err(101);}
   if(v::len<Row>(ref rows)>=4096){return .err(3);}if(!v::push<Row>(ref mut rows,row).is_ok){return .err(2);}
  }}m+=1;}
 var usize i=1;while(i<v::len<Row>(ref rows)){let row=*v::at<Row>(ref rows,i);var usize at=i;
  while(at>0){let prior=*v::at<Row>(ref rows,at-1);if(!before(p,row,prior,meter)?){break;}
   {let slot=v::at_mut<Row>(ref mut rows,at);*slot=prior;}{let slot=v::at_mut<Row>(ref mut rows,at-1);*slot=row;}at-=1;}i+=1;}
 return .ok(rows);
}
pub fn Result<bool,u32> key(ref mut bytes::Buffer out,ref input::Input p,Row row,ref mut usize meter){let parsed=input::source(p,row.module_index);let path=input::file(p,row.module_index);var bytes::Buffer module_name=bytes::empty();var usize i=0;while(i+3<path.len){if(path[i]==47){discard w::raw(ref mut module_name,b"::",meter)?;}else{discard w::raw(ref mut module_name,path[i..i+1],meter)?;}i+=1;}discard w::raw(out,b"{",meter)?;discard w::pair(out,b"package_instance",input::instance_id(p,row.module_index),meter)?;discard w::raw(out,b",",meter)?;discard w::pair(out,b"module",bytes::view(ref module_name),meter)?;discard w::raw(out,b",",meter)?;discard w::pair(out,b"kind",kind(syntax::declaration_at(parsed,row.declaration).kind),meter)?;discard w::raw(out,b",",meter)?;let written=if(syntax::declaration_at(parsed,row.declaration).owner_end!=0){name(parsed,row.declaration+1,meter)?}else{bytes::empty()};discard w::pair(out,b"name",if(syntax::declaration_at(parsed,row.declaration).owner_end!=0){bytes::view(ref written)}else{syntax::declaration_name(parsed,row.declaration+1)},meter)?;return w::raw(out,b"}",meter);}
pub fn Result<bool,u32> span(ref mut bytes::Buffer out,ref input::Input p,usize module_index,usize start,usize end,ref mut usize meter){if(start>end || end>syntax::source_view(input::source(p,module_index)).len){return .err(9);}discard w::raw(out,b"{",meter)?;discard w::pair(out,b"package_instance",input::instance_id(p,module_index),meter)?;discard w::raw(out,b",",meter)?;discard w::pair(out,b"source",input::file(p,module_index),meter)?;discard w::raw(out,b",\"start_byte\":",meter)?;discard w::number(out,start,meter)?;discard w::raw(out,b",\"end_byte\":",meter)?;discard w::number(out,end,meter)?;return w::raw(out,b"}",meter);}
fn Result<bool,u32> type(ref mut bytes::Buffer out,ref syntax::Parsed p,usize id,ref mut usize meter){if(id==0 || syntax::type_at(p,id).kind!=1){return .err(102);}let name=syntax::type_text(p,id);if(!bytes::equal(name,b"i32") && !bytes::equal(name,b"i64") && !bytes::equal(name,b"u8") && !bytes::equal(name,b"usize") && !bytes::equal(name,b"bool") && !bytes::equal(name,b"void")){return .err(102);}discard w::raw(out,b"{\"kind\":\"builtin\",",meter)?;discard w::pair(out,b"name",name,meter)?;return w::raw(out,b",\"package_coordinate\":\"\",\"mutable\":false,\"length\":0,\"arguments\":[]}",meter);}
pub fn Result<bytes::Buffer,u32> declaration(ref input::Input p,Row row,ref mut usize meter){let parsed=input::source(p,row.module_index);let d=syntax::declaration_at(parsed,row.declaration);let source=syntax::source_view(parsed);var bytes::Buffer out=bytes::empty();discard w::raw(ref mut out,b"{\"key\":",meter)?;discard key(ref mut out,p,row,meter)?;discard w::raw(ref mut out,b",\"span\":",meter)?;discard span(ref mut out,p,row.module_index,d.start,d.end,meter)?;discard w::raw(ref mut out,b",\"provenance\":\"compiler-declared\",\"public\":",meter)?;discard w::raw(ref mut out,if(d.public){b"true"}else{b"false"},meter)?;discard w::raw(ref mut out,b",\"unsafe\":false,\"foreign\":false,\"abi\":\"\",\"link_name\":\"\",\"type\":",meter)?;discard type(ref mut out,parsed,d.result_type,meter)?;discard w::raw(ref mut out,b",\"owner\":null,\"receiver\":\"\",\"parameters\":[",meter)?;var usize i=0;var usize parameters=0;while(i<d.entry_count){let e=syntax::entry_at(parsed,d.entry_start+i);if(e.kind!=1){i+=1;continue;}if(parameters>0){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"{",meter)?;discard w::pair(ref mut out,b"name",source[e.name_start..e.name_end],meter)?;discard w::raw(ref mut out,b",\"type\":",meter)?;discard type(ref mut out,parsed,e.type_root,meter)?;discard w::raw(ref mut out,b",\"mutable\":false,\"span\":",meter)?;discard span(ref mut out,p,row.module_index,e.start,e.end,meter)?;discard w::raw(ref mut out,b"}",meter)?;parameters+=1;i+=1;}discard w::raw(ref mut out,b"],\"effects\":[],\"fields\":[],\"variants\":[],\"repr_c\":false,\"drop_kind\":\"\",\"drop_effects\":[]}",meter)?;return .ok(out);}
pub fn Result<bytes::Buffer,u32> digest(ref facts::Facts f,ref u8[] declaration,ref mut usize meter){
 // Symbol identity consumes only the current UID and nominal-coordinate table.
 // Parsing the full file/lock/hash expectation for every declaration charges
 // unrelated source metadata D times; Ready still checks that full expectation.
 var joined=bytes::empty();discard w::raw(ref mut joined,b"{\"header\":{",meter)?;
 discard w::pair(ref mut joined,b"package_uid",facts::value(f,b"package_uid"),meter)?;
 discard w::raw(ref mut joined,b",\"package_references\":",meter)?;discard w::raw(ref mut joined,facts::array(f,b"package_references"),meter)?;
 discard w::raw(ref mut joined,b"},\"declaration\":",meter)?;discard w::raw(ref mut joined,declaration,meter)?;discard w::raw(ref mut joined,b"}",meter)?;
 let tree=switch(json::parse(joined,meter)){case .ok(p):{p}case .err(_):{return .err(9);}};
 return symbols::derive(ref tree,wire::field(ref tree,0,b"header"),wire::field(ref tree,0,b"declaration"),meter);
}
