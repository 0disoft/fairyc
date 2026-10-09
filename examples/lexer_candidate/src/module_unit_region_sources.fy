module module_unit_region_sources;
import module_unit_value_model as m;
import module_unit_value_builtin as builtins;
import module_unit_interfaces as interfaces;
import module_types as types;
import std::vec as v;

// Lexical capabilities are manufactured only for the resolved region binding.
// Arena source typing is distinct from later enter/reset and loan verification.
pub fn m::Problem prepare(m::Frame frame,ref mut v::Vec<m::Local> locals,ref v::Vec<m::Expression> expressions,ref mut v::Vec<types::Type> graph,ref mut usize work){
    var usize i=0;while(i<v::len<m::Local>(locals)){let local=*v::at<m::Local>(locals,i);if(!m::charge(work,1)){return m::fail(frame,10,local.start,local.end,0,0);}if(local.kind==6){let usize id=m::index(expressions,local.source_index);if(id==0 || v::at<m::Expression>(expressions,id-1).kind!=17){return m::fail(frame,12,local.start,local.end,0,0);}let e=*v::at<m::Expression>(expressions,id-1);let capability=switch(builtins::canonical(frame,e,types::Type{.kind=19,.scalar=0,.module_index=0,.declaration=0,.first=0,.second=0},graph,work)){case .ok(t):{t}case .err(p):{return p;}};let slot=v::at_mut<m::Local>(locals,i);slot.expected=capability;slot.type=capability;}i+=1;}return m::fail(frame,0,0,0,0,0);
}
pub fn m::Problem validate(m::Frame frame,m::Expression source,ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph){
    let arena=interfaces::region_arena(imported);let t=types::at(graph,source.type);return if(arena!=0 && t.kind==8 && t.first==arena){m::fail(frame,0,0,0,0,0)}else{m::fail(frame,2,source.start,source.end,0,source.type)};
}
pub fn m::Problem complete(m::Frame frame,ref v::Vec<m::Consumer> consumers,ref v::Vec<m::Expression> expressions,ref mut usize work){
    var usize i=0;while(i<v::len<m::Consumer>(consumers)){let c=*v::at<m::Consumer>(consumers,i);if(!m::charge(work,1)){return m::fail(frame,10,c.start,c.end,0,0);}if(c.kind==17 && !c.checked){if(c.expression==0 || c.expression>v::len<m::Expression>(expressions)){return m::fail(frame,12,c.start,c.end,0,0);}let e=*v::at<m::Expression>(expressions,c.expression-1);return m::fail(frame,2,e.start,e.end,0,0);}i+=1;}return m::fail(frame,0,0,0,0,0);
}
