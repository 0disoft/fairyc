module module_unit_origin_match;
import module_unit_origin_inputs as inputs;
import module_unit_origin_flow_model as g;
import module_unit_values as values;
import module_unit_value_model as m;
import std::vec as v;

// Bind one builtin payload by lexical symbol ID, never by a forwarded place.
// Enum payloads are inert under the owning shape contract and need no origins.
pub fn Result<usize,m::Problem> seed(ref inputs::Inputs input,ref values::Plan primary,ref m::Values body,ref mut v::Vec<g::Unknown> unknowns,m::Expression subject,m::Expression pattern,g::Value value,ref mut v::Vec<g::Value> env,ref mut g::Context c){
    if(subject.type==0){return .err(g::fail(c,2,pattern.start,pattern.end));}if(!m::charge(ref mut c.work,1)){return .err(g::fail(c,10,pattern.start,pattern.end));}let t=inputs::type_at(input,primary,c.frame,subject.type);if(t.kind!=5 && t.kind!=6){return .ok(0);}if(pattern.kind!=16 || pattern.expected!=subject.type){return .err(g::fail(c,12,pattern.start,pattern.end));}if(pattern.left==0){return .ok(0);}
    let binding=switch(g::expression(body,pattern.left,c)){case .ok(p):{p}case .err(p):{return .err(p);}};if(binding.kind!=17 || binding.target==0){return .err(g::fail(c,12,binding.start,binding.end));}let usize symbol=binding.place.root;if(symbol==0){return .ok(0);}if(symbol>m::local_count(body) || symbol>v::len<g::Value>(env)){return .err(g::fail(c,12,binding.start,binding.end));}let local=m::local_at(body,symbol);if(local.kind!=4 || local.source_index!=pattern.left || local.type!=binding.target){return .err(g::fail(c,12,binding.start,binding.end));}
    let leaves=switch(inputs::borrow_leaves(input,primary,c.frame,binding.target,ref mut c.work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(leaves>1){return .err(g::fail(c,2,pattern.start,pattern.end));}if(leaves==1){let actual=if(value.origin==0){g::Value{.origin=1,.allocation=0}}else{value};let slot=v::at_mut<g::Value>(env,symbol-1);*slot=actual;if(actual.origin==1){let error=g::unknown(unknowns,c,3,pattern.start,pattern.end);if(error.code!=0){return .err(error);}}}return .ok(symbol);
}
