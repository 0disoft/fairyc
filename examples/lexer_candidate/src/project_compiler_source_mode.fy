module project_compiler_source_mode;
import project_compiler_input as input;
import project_path as work;
import syntax as syntax;
import sdk_authority as sdk;
import std::bytes as bytes;

// Only exact retained source selects the explicit aggregate factory. A map,
// cache header or selected symbol cannot upgrade the compilation mode.
// The authenticated SDK Vec template keeps its existing ordinary compiler lane.
fn Result<bool,u32> vector(ref syntax::Parsed source,ref mut usize meter){
 if(syntax::declaration_count(source)==0){return .ok(false);}let d=syntax::declaration_at(source,0);
 if(d.kind!=1){return .ok(false);}let raw=syntax::source_view(source);let name=raw[d.name_start..d.name_end];
 if(!bytes::equal(name,b"std::vec")){return .ok(false);}if(!work::charge(meter,raw.len+1)){return .err(3);}
 return .ok(sdk::authentic(name,raw));
}
pub fn Result<bool,u32> aggregates(ref input::Input source,ref mut usize meter){
 var bool found=false;var usize module_index=0;
 while(module_index<input::count(source)){
  let p=input::source(source,module_index);let vector=switch(vector(p,meter)){case .ok(p):{p}case .err(code):{return .err(code);}};var usize declaration=0;
  while(declaration<syntax::declaration_count(p)){
   if(!work::charge(meter,1)){return .err(3);}
   let d=syntax::declaration_at(p,declaration);
   if(!vector && (d.kind==4 || d.kind==6) && d.generic_parameters!=0){found=true;}
   declaration+=1;
  }module_index+=1;
 }return .ok(found);
}
