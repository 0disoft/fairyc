module project_atlas_binders;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Only declaration shape, bound types and source-independent identity rules.
// This does not admit a map, authenticate current source or collect a body.
fn Result<usize,u32> parameters(ref json::Tree t,usize decl,ref mut usize meter){
 let at=wire::field(t,decl,b"type_parameters");if(at==json::count(t)){return .ok(0);}
 let rows=model::rows(t,at,4096,meter)?;var usize i=0;
 while(i<v::len<usize>(ref rows)){
  let name=json::value(t,*v::at<usize>(ref rows,i));if(!model::identifier(name)){return .err(7);}var usize j=0;
  while(j<i){let prior=json::value(t,*v::at<usize>(ref rows,j));if(!work::charge(meter,name.len+prior.len+1)){return .err(3);}if(bytes::equal(name,prior)){return .err(7);}j+=1;}i+=1;
 }return .ok(v::len<usize>(ref rows));
}
fn bool fields(ref json::Tree t,usize at,usize arity,ref mut usize meter){
 let rows=switch(model::rows(t,at,4096,meter)){case .ok(p):{p}case .err(_):{return false;}};var usize i=0;
 while(i<v::len<usize>(ref rows)){if(!model::bound_type(t,wire::field(t,*v::at<usize>(ref rows,i),b"type"),0,arity,true,meter)){return false;}i+=1;}return true;
}
fn bool aggregate_self(ref json::Tree t,usize decl,usize arity,ref mut usize meter){
 let key=wire::field(t,decl,b"key");let type=wire::field(t,decl,b"type");let module_name=wire::string(t,key,b"module");let name=wire::string(t,key,b"name");let written=wire::string(t,type,b"name");
 if(!bytes::equal(wire::string(t,type,b"kind"),b"nominal") || written.len!=module_name.len+name.len+2 || !bytes::equal(written[..module_name.len],module_name) || !bytes::equal(written[module_name.len..module_name.len+2],b"::") || !bytes::equal(written[module_name.len+2..],name) || wire::count(t,wire::field(t,decl,b"parameters"))!=0){return false;}
 let args=switch(model::rows(t,wire::field(t,type,b"arguments"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};if(v::len<usize>(ref args)!=arity){return false;}var usize i=0;
 while(i<arity){let arg=*v::at<usize>(ref args,i);let index=switch(wire::number(wire::string(t,arg,b"name"))){case .ok(p):{p}case .err(_):{return false;}};if(!work::charge(meter,1) || !bytes::equal(wire::string(t,arg,b"kind"),b"parameter") || index!=i){return false;}i+=1;}return true;
}
pub fn Result<usize,u32> validate(ref json::Tree t,usize decl,ref mut usize meter){
 let shape=wire::declaration_shape(t,decl,meter);if(shape!=0){return .err(if(shape==3){3}else{7});}
 let arity=parameters(t,decl,meter)?;let key=wire::field(t,decl,b"key");let kind=wire::string(t,key,b"kind");let owner=wire::field(t,decl,b"owner");
 if(arity!=0){
  if(!bytes::equal(kind,b"function") && !bytes::equal(kind,b"struct") && !bytes::equal(kind,b"resource") || wire::boolean(t,decl,b"foreign") || json::token(t,owner).kind!=11){return .err(7);}
  if(!bytes::equal(kind,b"function") && !aggregate_self(t,decl,arity,meter)){return .err(7);}
 }
 if(!model::bound_type(t,wire::field(t,decl,b"type"),0,arity,true,meter) || json::token(t,owner).kind!=11 && !model::bound_type(t,owner,0,0,true,meter)){return .err(7);}
 if(!fields(t,wire::field(t,decl,b"parameters"),arity,meter) || !fields(t,wire::field(t,decl,b"fields"),arity,meter)){return .err(7);}
 let variants=model::rows(t,wire::field(t,decl,b"variants"),4096,meter)?;var usize i=0;
 while(i<v::len<usize>(ref variants)){if(!fields(t,wire::field(t,*v::at<usize>(ref variants,i),b"fields"),arity,meter)){return .err(7);}i+=1;}return .ok(arity);
}
