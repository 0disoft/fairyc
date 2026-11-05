module project_atlas_source_mode;
import project_compiler_input as input;
import project_path as work;
import syntax as syntax;

// Only exact retained source selects the explicit aggregate factory. A map,
// cache header or selected symbol cannot upgrade the compilation mode.
pub fn Result<bool,u32> aggregates(ref input::Input source,ref mut usize meter){
 var bool found=false;var usize module_index=0;
 while(module_index<input::count(source)){
  let p=input::source(source,module_index);var usize declaration=0;
  while(declaration<syntax::declaration_count(p)){
   if(!work::charge(meter,1)){return .err(3);}
   let d=syntax::declaration_at(p,declaration);
   if((d.kind==4 || d.kind==6) && d.generic_parameters!=0){found=true;}
   declaration+=1;
  }module_index+=1;
 }return .ok(found);
}
