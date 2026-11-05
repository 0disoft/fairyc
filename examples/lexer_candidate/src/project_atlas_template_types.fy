module project_atlas_template_types;
import project_compiler_input as input;
import module_signatures as signatures;
import module_public_type_text as coordinates;
import project_atlas_types as types;
import project_atlas_write as w;
import project_path as work;
import generic_source_arguments as paths;
import project_json_tree as json;
import project_atlas_wire as wire;
import c_abi_types as c;
import syntax as syntax;
import std::bytes as bytes;

// Source-bound signature projection; it does not infer template-body behavior.
// The owning schema factory has authenticated these exact sources and metadata.
pub fn Result<bool,u32> parameters(ref mut bytes::Buffer out,ref syntax::Parsed p,usize declaration,ref mut usize meter){
 let count=syntax::generic_parameter_count(p,declaration);if(count==0 || count>8){return .err(7);}discard w::raw(out,b"[",meter)?;var usize i=0;
 while(i<count){if(!work::charge(meter,1)){return .err(3);}let node=syntax::generic_parameter_at(p,declaration,i);if(i>0){discard w::raw(out,b",",meter)?;}discard w::string(out,syntax::source_view(p)[node.start..node.end],meter)?;i+=1;}return w::raw(out,b"]",meter);
}
fn Result<bool,u32> parameter(ref mut bytes::Buffer out,usize index,ref mut usize meter){var name=bytes::empty();discard w::number(ref mut name,index,meter)?;discard types::head(out,b"parameter",bytes::view(ref name),b"",false,0,meter)?;return w::raw(out,b"]}",meter);}
fn bool builtin(ref u8[] name){return bytes::equal(name,b"bool") || bytes::equal(name,b"u8") || bytes::equal(name,b"i32") || bytes::equal(name,b"i64") || bytes::equal(name,b"u32") || bytes::equal(name,b"usize") || bytes::equal(name,b"void") || bytes::equal(name,b"str") || bytes::equal(name,b"CStr") || bytes::equal(name,b"Region") || c::named(name)!=0;}
fn Result<bool,u32> nominal_head(ref mut bytes::Buffer out,ref input::Input source,ref coordinates::Coordinates table,usize module_index,usize declaration,ref mut usize meter){
 if(module_index>=input::count(source) || declaration==0){return .err(7);}let p=input::source(source,module_index);if(declaration>syntax::declaration_count(p)){return .err(7);}let d=syntax::declaration_at(p,declaration-1);if(d.kind!=4 && d.kind!=6 && d.kind!=11){return .err(7);}
 var name=types::module_name(input::file(source,module_index),meter)?;discard w::raw(ref mut name,b"::",meter)?;discard w::raw(ref mut name,syntax::declaration_name(p,declaration),meter)?;
 if(coordinates::coordinate_count(table)>512){return .err(7);}let owner=input::origin(source,module_index).owner;var usize found=coordinates::coordinate_count(table);var usize i=0;
 while(i<coordinates::coordinate_count(table)){if(!work::charge(meter,1)){return .err(3);}if(coordinates::coordinate_at(table,i).owner==owner){if(found!=coordinates::coordinate_count(table)){return .err(7);}found=i;}i+=1;}
 if(found==coordinates::coordinate_count(table) || !coordinates::valid_coordinate(coordinates::coordinate_text(table,found))){return .err(7);}return types::head(out,b"nominal",bytes::view(ref name),coordinates::coordinate_text(table,found),false,0,meter);
}
pub fn Result<bool,u32> self(ref mut bytes::Buffer out,ref input::Input source,ref coordinates::Coordinates table,usize module_index,usize declaration,ref mut usize meter){
 if(module_index>=input::count(source)){return .err(7);}let p=input::source(source,module_index);if(declaration>=syntax::declaration_count(p)){return .err(7);}let count=syntax::generic_parameter_count(p,declaration);if(count==0 || count>8){return .err(7);}
 discard nominal_head(out,source,table,module_index,declaration+1,meter)?;var usize i=0;while(i<count){if(i>0){discard w::raw(out,b",",meter)?;}discard parameter(out,i,meter)?;i+=1;}return w::raw(out,b"]}",meter);
}
fn Result<usize,u32> array_length(ref u8[] text,ref mut usize meter){
 var usize base=10;var usize i=0;var usize n=0;var usize digits=0;
 if(text.len>=2 && text[0]==48){if(text[1]==120 || text[1]==88){base=16;i=2;}else if(text[1]==98 || text[1]==66){base=2;i=2;}}
 while(i<text.len){if(!work::charge(meter,1)){return .err(3);}let c=text[i];i+=1;if(c==95){continue;}
  let usize digit=if(c>=48 && c<=57){(c-48) as usize}else if(c>=65 && c<=70){(c-65+10) as usize}else if(c>=97 && c<=102){(c-97+10) as usize}else{base};
  if(digit>=base || n>(18446744073709551615-digit)/base){return .err(7);}n=n*base+digit;digits+=1;
 }return if(n==0 || digits==0){.err(7)}else{.ok(n)};
}
fn Result<bool,u32> walk(ref mut bytes::Buffer out,ref input::Input source,ref coordinates::Coordinates table,ref signatures::Program metadata,usize module_index,usize declaration,usize root,usize depth,ref mut usize meter){
 if(depth>=64 || !work::charge(meter,1)){return .err(3);}let p=input::source(source,module_index);if(root==0 || root>syntax::type_count(p)){return .err(7);}let n=syntax::type_at(p,root);
 if(n.kind==1 || n.kind==6){
  let id=if(n.kind==6){n.left}else{root};if(id==0 || id>syntax::type_count(p) || syntax::type_at(p,id).kind!=1){return .err(7);}let base=syntax::type_at(p,id);
  let name=switch(paths::path(p,base.start,base.end,meter)){case .ok(p):{p}case .err(code):{return .err(if(code==10){3}else if(code==11){2}else{7});}};let text=bytes::view(ref name);let usize count=if(n.kind==6){syntax::type_argument_count(p,root)}else{0};if(count>4096){return .err(3);}
  if(count==0){var usize i=0;while(i<syntax::generic_parameter_count(p,declaration)){let binder=syntax::generic_parameter_at(p,declaration,i);if(!work::charge(meter,text.len+binder.end-binder.start+1)){return .err(3);}if(bytes::equal(text,syntax::source_view(p)[binder.start..binder.end])){return parameter(out,i,meter);}i+=1;}if(builtin(text)){return types::builtin(out,text,meter);}}
  let ref u8[] tag=if(bytes::equal(text,b"Option") && count==1){b"option"}else if(bytes::equal(text,b"Result") && count==2){b"result"}else{b""};
  if(tag.len>0){discard types::head(out,tag,b"",b"",false,0,meter)?;}
  else{var usize probes=0;let target=signatures::declaration_lookup_measured(metadata,module_index,text,ref mut probes);if(!work::charge(meter,probes)){return .err(3);}if(target.code!=0){return .err(7);}let d=signatures::declaration_at(metadata,target.item);let parsed=input::source(source,d.module_index);if(count!=syntax::generic_parameter_count(parsed,d.declaration-1)){return .err(7);}discard nominal_head(out,source,table,d.module_index,d.declaration,meter)?;}
  var usize i=0;while(i<count){if(i>0){discard w::raw(out,b",",meter)?;}discard walk(out,source,table,metadata,module_index,declaration,syntax::type_argument(p,root,i),depth+1,meter)?;i+=1;}return w::raw(out,b"]}",meter);
 }
 if(n.kind==5){return walk(out,source,table,metadata,module_index,declaration,n.left,depth+1,meter);}
 if(n.kind==2 || n.kind==3 || n.kind==11 || n.kind==12 || n.kind==8 || n.kind==4){
  var element=bytes::empty();discard walk(ref mut element,source,table,metadata,module_index,declaration,n.left,depth+1,meter)?;
  if(n.kind==4){
   // A scalar slice normalizes to a builtin view; nominal and bound elements
   // keep explicit slice structure. Other compound elements have no identity.
   let t=switch(json::parse(element,meter)){case .ok(p):{p}case .err(e):{return .err(if(e.code==3){3}else{7});}};let kind=wire::string(ref t,0,b"kind");
   if(bytes::equal(kind,b"builtin")){let name=wire::string(ref t,0,b"name");if(!bytes::equal(name,b"i32") && !bytes::equal(name,b"i64") && !bytes::equal(name,b"u8") && !bytes::equal(name,b"u32") && !bytes::equal(name,b"usize") && !bytes::equal(name,b"bool")){return .err(7);}var view=bytes::empty();discard w::raw(ref mut view,b"[",meter)?;discard w::raw(ref mut view,name,meter)?;discard w::raw(ref mut view,b"]",meter)?;return types::builtin(out,bytes::view(ref view),meter);}
   if(!bytes::equal(kind,b"nominal") && !bytes::equal(kind,b"parameter")){return .err(7);}discard types::head(out,b"slice",b"",b"",false,0,meter)?;discard w::raw(out,json::source(ref t),meter)?;return w::raw(out,b"]}",meter);
  }
  var usize length=0;if(n.kind==8){if(n.right==0 || n.right>syntax::type_count(p) || syntax::type_at(p,n.right).kind!=9){return .err(7);}length=array_length(syntax::type_text(p,n.right),meter)?;}
  discard types::head(out,if(n.kind==4){b"slice"}else if(n.kind==8){b"fixed-array"}else if(n.kind==11 || n.kind==12){b"pointer"}else{b"borrow"},b"",b"",n.kind==3 || n.kind==12,length,meter)?;
  discard w::raw(out,bytes::view(ref element),meter)?;return w::raw(out,b"]}",meter);
 }return .err(7);
}
pub fn Result<bool,u32> render(ref mut bytes::Buffer out,ref input::Input source,ref coordinates::Coordinates table,ref signatures::Program metadata,usize module_index,usize declaration,usize root,ref mut usize meter){
 if(module_index>=input::count(source) || declaration>=syntax::declaration_count(input::source(source,module_index))){return .err(7);}return walk(out,source,table,metadata,module_index,declaration,root,0,meter);
}
