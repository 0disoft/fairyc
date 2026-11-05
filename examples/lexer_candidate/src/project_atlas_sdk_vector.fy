module project_atlas_sdk_vector;
import project_compiler_input as input;
import sdk_authority as sdk;
import project_path as work;
import syntax as syntax;
import std::bytes as bytes;

// Neither a nominal kind nor a module spelling grants SDK template authority.
// Charge the exact retained bytes whose existing SDK pin is reauthenticated.
pub fn Result<bool,u32> source(ref input::Input p,usize module_index,ref mut usize meter){
 if(module_index>=input::count(p)){return .err(7);}let parsed=input::source(p,module_index);
 if(syntax::declaration_count(parsed)==0){return .ok(false);}let d=syntax::declaration_at(parsed,0);
 if(d.kind!=1){return .ok(false);}let raw=syntax::source_view(parsed);let name=raw[d.name_start..d.name_end];
 if(!bytes::equal(name,b"std::vec")){return .ok(false);}if(!work::charge(meter,raw.len+1)){return .err(3);}
 return .ok(sdk::authentic(name,raw));
}
pub fn Result<bool,u32> declaration(ref input::Input p,usize module_index,usize declaration,ref mut usize meter){
 if(!source(p,module_index,meter)?){return .ok(false);}let parsed=input::source(p,module_index);
 if(declaration==0 || declaration>syntax::declaration_count(parsed)){return .err(7);}let d=syntax::declaration_at(parsed,declaration-1);
 return .ok(d.kind==6 && syntax::generic_parameter_count(parsed,declaration-1)==1 && d.field_count==6 && d.owner_start==d.owner_end && bytes::equal(syntax::declaration_name(parsed,declaration),b"Vec"));
}
// The exact authenticated source owns this private, single-intrinsic helper.
// A source name alone must never authorize an optimized-away implementation.
pub fn Result<usize,u32> take_leaf(ref input::Input p,usize module_index,ref mut usize meter){
 if(!source(p,module_index,meter)?){return .err(7);}let parsed=input::source(p,module_index);var usize found=0;var usize i=0;
 while(i<syntax::declaration_count(parsed)){if(!work::charge(meter,1)){return .err(3);}let d=syntax::declaration_at(parsed,i);
  if(bytes::equal(syntax::declaration_name(parsed,i+1),b"take_slot")){
   if(found!=0 || d.kind!=3 || d.public || d.unsafe_end!=0 || d.attribute_count!=0 || d.body==0 || syntax::generic_parameter_count(parsed,i)!=1 || !bytes::equal(syntax::type_text(parsed,d.result_type),b"T")){return .err(7);}found=i+1;
  }i+=1;
 }return if(found!=0){.ok(found)}else{.err(7)};
}
