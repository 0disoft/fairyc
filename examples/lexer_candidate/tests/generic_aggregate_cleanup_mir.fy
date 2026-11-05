module tests::generic_aggregate_cleanup_mir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cleanup_mir_verify as proof;
import module_unit_checked_bodies as checking;
import module_unit_moves as moves;
import module_unit_values as values;
import module_unit_interfaces as interfaces;
import module_types as types;
import std::vec as v;

pub fn void verify(ref clean::Program p,ref checking::Plan owner){
 let checked=checking::checked_values(owner);let imported=checking::checked_interfaces(owner);let moved=checking::checked_moves(owner);
 assert(clean::aggregates(p) && proof::check(p).code==0 && clean::type_count(p)==values::type_count(checked) && clean::frame_count(p)==values::count(checked));
 assert(clean::drop_count(p)==moves::cleanup_count(moved));var usize i=0;
 while(i<clean::drop_count(p)){let drop=clean::drop_at(p,i);let operand=clean::operand_at(p,drop.operand);let expected=moves::cleanup_at(moved,i);assert(drop.exit==expected.exit && drop.start==expected.start && drop.end==expected.end && operand.frame==expected.frame && operand.type==expected.type && operand.id==if(expected.binding!=0){expected.binding}else{expected.expression});i+=1;}
 i=0;while(i<interfaces::shape_count(imported)){let shape=interfaces::shape_at(imported,i);if(clean::type_at(p,shape.type).kind==21){let glue=clean::glue_at(p,shape.type);assert(glue.kind==1 && glue.nonempty==shape.cleanup_nonempty && glue.effects==shape.cleanup_effects && glue.abilities==shape.abilities && glue.field_count==shape.field_count);var usize j=0;while(j<glue.field_count){let field=clean::field_at(p,glue.field_start+j);assert(field.owner==shape.type && field.ordinal==j && field.type==interfaces::field_at(imported,shape.field_start+j).type);j+=1;}}i+=1;}
 var u32 mode=0;while(mode<=13){if((mode!=8 || clean::operand_count(p)!=0) && (mode!=7 || clean::field_count(p)!=0)){assert(rebuild(p,mode)==if(mode==0){0}else{1001});}mode+=1;}
}
fn u32 rebuild(ref clean::Program p,u32 mode){
 var graph=v::empty<types::Type>();var frames=v::empty<clean::Frame>();var glues=v::empty<clean::Glue>();var fields=v::empty<clean::Field>();var operands=v::empty<clean::Operand>();var drops=v::empty<clean::Drop>();var usize tuple=0;var usize resource_type=0;var usize i=1;
 while(i<=clean::type_count(p)){let t=clean::type_at(p,i);if(t.kind==22 && tuple==0){tuple=i;}if(t.kind==21 && resource_type==0){resource_type=i;}assert(v::push<types::Type>(ref mut graph,t).is_ok && v::push<clean::Glue>(ref mut glues,clean::glue_at(p,i)).is_ok);i+=1;}assert(tuple!=0 && resource_type!=0);
 i=1;while(i<=clean::frame_count(p)){var f=clean::frame_at(p,i);if(mode==6 && i==1){f.result=tuple;}if(mode==10 && f.kind==2){f.receiver_type=tuple;}assert(v::push<clean::Frame>(ref mut frames,f).is_ok);i+=1;}
 i=0;while(i<clean::field_count(p)){var f=clean::field_at(p,i);if(mode==7 && i==0){f.type=tuple;}assert(v::push<clean::Field>(ref mut fields,f).is_ok);i+=1;}
 i=1;while(i<=clean::operand_count(p)){var o=clean::operand_at(p,i);if(mode==8 && i==1){o.type=tuple;}assert(v::push<clean::Operand>(ref mut operands,o).is_ok);i+=1;}
 i=0;while(i<clean::drop_count(p)){assert(v::push<clean::Drop>(ref mut drops,clean::drop_at(p,i)).is_ok);i+=1;}
 if(mode==3){let slot=v::at_mut<types::Type>(ref mut graph,tuple-1);slot.first=tuple;}if(mode==4){let scalar=clean::type_at(p,tuple).first;let slot=v::at_mut<types::Type>(ref mut graph,resource_type-1);slot.first=scalar;}if(mode==5){let slot=v::at_mut<clean::Glue>(ref mut glues,resource_type-1);slot.kind=0;}
 if(mode==9){assert(v::push<types::Type>(ref mut graph,types::Type{.kind=5,.scalar=0,.module_index=0,.declaration=0,.first=tuple,.second=0}).is_ok && v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=2,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);}
 if(mode==11){assert(v::push<types::Type>(ref mut graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=tuple,.second=1}).is_ok && v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);}
 if(mode==12){var usize tail=tuple;var usize n=0;while(n<9){assert(v::push<types::Type>(ref mut graph,types::Type{.kind=22,.scalar=0,.module_index=0,.declaration=0,.first=clean::type_at(p,tuple).first,.second=tail}).is_ok && v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);tail=v::len<types::Type>(ref graph);n+=1;}}
 if(mode==13){return switch(clean::assemble_project(graph,frames,glues,fields,operands,drops,clean::work(p))){case .ok(_):{0}case .err(e):{e.code}};}
 if(mode==1){return switch(clean::assemble(graph,frames,glues,fields,operands,drops,clean::work(p))){case .ok(_):{0}case .err(e):{e.code}};}if(mode==2){var usize work=0;return switch(clean::assemble_artifact(graph,frames,glues,fields,operands,drops,false,ref mut work)){case .ok(_):{0}case .err(e):{e.code}};}
 return switch(clean::assemble_aggregates(graph,frames,glues,fields,operands,drops,clean::work(p))){case .ok(_):{0}case .err(e):{e.code}};
}
