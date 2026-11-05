module tests::generic_aggregate_values;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_interfaces as interfaces;
import module_unit_drops as drops;
import module_types as types;
import syntax as syntax;
import std::vec as v;

pub fn void verify(ref closure::Ready ready,ref values::Plan checked,ref v::Vec<syntax::Parsed> sources){
 assert(values::checked_returns(checked)==values::returns(checked));
 assert(values::checked_calls(checked)==values::invocation_count(checked) && values::checked_arguments(checked)==values::arguments(checked));
 let imported=closure::imported(ready);var usize i=1;
 while(i<=interfaces::type_count(imported)){let t=interfaces::type_at(imported,i);if(t.kind==20 || t.kind==21){let drop=switch(drops::required(closure::drop_plan(ready),i)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(drop.complete && drop.needed==(t.kind==21));if(t.kind==21){assert(drop.context!=0);}}i+=1;}
 i=0;while(i<values::count(checked)){let context=values::at(checked,i);let frame=model::frame(context);let source=v::at<syntax::Parsed>(sources,frame.module_index);var usize j=1;
  while(j<=model::expression_count(context)){let e=model::expression_at(context,j);if(e.kind==10 && e.target!=0){let t=values::type_at(checked,e.target);if(t.kind==20 || t.kind==21){assert(e.state==1 && e.type==e.target);let original=syntax::expression_at(source,e.source_index);assert(original.kind==10 && original.start==e.start && original.end==e.end);}}j+=1;}
  j=1;while(j<=model::local_count(context)){assert(model::local_at(context,j).type!=0);j+=1;}i+=1;
 }
}
