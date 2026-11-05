module module_unit_aggregate_plan;
import module_signatures as signatures;
import generic_aggregate_types as aggregates;
import generic_source_plan as plan;
import generic_source_plan_input as input;
import generic_source_catalog as templates;
import generic_concrete_types as concrete;
import generic_transport_scalars as transport;
import generic_scalar_arguments as descriptors;
import module_types as types;
import syntax as syntax;
import std::vec as v;

fn bool charge(ref mut usize work,usize amount){if(*work>1048576 || amount>1048576-*work){return false;};*work=*work+amount;return true;}
fn u32 scalar(ref v::Vec<types::Type> graph,usize id){if(descriptors::descriptor(id)){return transport::named(descriptors::name(id));}if(id==0 || id>types::count(graph)){return 0;}let t=types::at(graph,id);return if(t.kind==1 && transport::valid(t.scalar)){t.scalar}else{0};}
// Match the original declaration and ordered scalar tuple, never generated names.
// Zero is the ordinary body context; generic contexts must join the sealed plan.
pub fn Result<usize,u32> context(ref signatures::Program metadata,ref v::Vec<types::Type> graph,usize module_index,usize declaration,ref v::Vec<usize> arguments,ref mut usize work){
 if(!signatures::aggregate_types(metadata)){return .err(9);}let count=v::len<usize>(arguments);if(count==0){return .ok(0);}let p=aggregates::source_plan(signatures::aggregate_source(metadata));let catalog=input::templates(plan::input(p));let known=input::types(plan::input(p));var usize i=0;
 while(i<plan::count(p)){if(!charge(work,count+1)){return .err(4503);}let n=plan::at(p,i);let d=templates::template_at(catalog,n.template_index);if(d.module_index==module_index && d.declaration+1==declaration && n.count==count){var usize j=0;var bool same=true;while(j<count){let actual=scalar(graph,*v::at<usize>(arguments,j));if(actual==0 || n.arguments[j]==0 || n.arguments[j]>concrete::count(known)){return .err(4502);}let expected=concrete::at(known,n.arguments[j]-1);if(expected.kind!=1 || actual!=expected.scalar){same=false;break;}j+=1;}if(same){return .ok(i+1);}}i+=1;}
 return .err(12);
}
// application=true names the generic application child; false names its record.
pub fn Result<usize,u32> construction(ref signatures::Program metadata,ref syntax::Parsed source,usize module_index,usize declaration,ref v::Vec<usize> arguments,usize expression,bool application,ref v::Vec<types::Type> graph,ref mut usize work){
 if(module_index>=signatures::module_count(metadata) || !signatures::source_matches(metadata,module_index,source)){return .err(12);}let node=switch(context(metadata,graph,module_index,declaration,arguments,work)){case .ok(n):{n}case .err(code):{return .err(code);}};let p=signatures::aggregate_source(metadata);var usize selected=0;var usize i=0;
 while(i<aggregates::use_count(p)){if(!charge(work,1)){return .err(4503);}let u=aggregates::use_at(p,i);if(u.kind==4 && u.node==node && u.module_index==module_index && u.declaration==declaration){if(u.source_index==0 || u.source_index>syntax::expression_count(source)){return .err(12);}let e=syntax::expression_at(source,u.source_index);if(e.kind!=10){return .err(12);}if(if(application){e.left==expression}else{u.source_index==expression}){if(selected!=0 || u.type==0 || u.type>types::count(graph)){return .err(12);}let t=types::at(graph,u.type);if(t.kind!=20 && t.kind!=21){return .err(12);}selected=u.type;}}i+=1;}
 return if(selected==0){.err(12)}else{.ok(selected)};
}
// Cleanup receives the concrete tuple retained by the resource type itself.
pub fn Result<v::Vec<usize>,u32> arguments(ref signatures::Program metadata,ref v::Vec<types::Type> graph,usize id,ref mut usize work){
 if(!signatures::aggregate_types(metadata) || id==0 || id>types::count(graph)){return .err(12);}let t=types::at(graph,id);if(t.kind!=21){return .err(9);}let prepared=signatures::aggregate_source(metadata);var usize matches=0;var usize i=0;while(i<aggregates::count(prepared)){if(!charge(work,1)){return .err(4503);}let a=aggregates::at(prepared,i);if(a.type==id){if(a.module_index!=t.module_index || a.declaration!=t.declaration){return .err(12);}matches+=1;}i+=1;}if(matches!=1){return .err(12);}let original=signatures::type_at(metadata,id);if(original.kind!=t.kind || original.first!=t.first || original.scalar!=t.scalar || original.second!=t.second || original.module_index!=t.module_index || original.declaration!=t.declaration){return .err(12);}var out=v::empty<usize>();var usize tail=t.first;
 while(tail!=0){if(!charge(work,1)){return .err(4503);}if(tail>types::count(graph) || v::len<usize>(ref out)>=8){return .err(12);}let row=types::at(graph,tail);if(row.kind!=22 || scalar(graph,row.first)==0 || row.second>=tail){return .err(12);}if(!v::push<usize>(ref mut out,row.first).is_ok){return .err(11);}tail=row.second;}
 discard switch(context(metadata,graph,t.module_index,t.declaration,ref out,work)){case .ok(n):{n}case .err(code):{return .err(code);}};return .ok(out);
}
