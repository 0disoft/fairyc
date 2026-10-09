module module_unit_value_result;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import std::vec as v;

pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);let usize child=m::index(expressions,e.left);
    if(child==0 || e.end<=e.start || frame.result==0 || frame.result>types::count(graph)){return m::fail(frame,12,e.start,e.end,0,0);}
    // The success context must not manufacture a Result type for the operand.
    let p=scalar::evaluate(source,frame,child,0,imported,graph,atoms,expressions,work,depth+1);
    // An unsupported operand has no Result type. The reference emits E4102
    // before the CLI's unsupported-construct fallback in this consumer.
    if(p.code==2){return m::fail(frame,4102,e.end-1,e.end,0,0);}if(p.code!=0){return p;}
    let actual=*v::at<m::Expression>(expressions,child-1);if(actual.state==0 && actual.type==0){return m::fail(frame,0,0,0,0,0);}
    if(actual.state==2 || types::at(graph,actual.type).kind!=6){return m::fail(frame,4102,e.end-1,e.end,0,actual.type);}
    let operand=types::at(graph,actual.type);let enclosing=types::at(graph,frame.result);
    if(enclosing.kind!=6 || enclosing.second!=operand.second){return m::fail(frame,4103,e.end-1,e.end,if(enclosing.kind==6){enclosing.second}else{0},operand.second);}
    let slot=v::at_mut<m::Expression>(expressions,id-1);slot.state=1;slot.type=operand.first;slot.expected=expected;slot.target=actual.type;slot.place=m::no_place();return m::fail(frame,0,0,0,0,0);
}
