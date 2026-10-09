module module_unit_argument_types;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_types as types;
import module_unit_interfaces as interfaces;
import syntax as syntax;
import std::vec as v;

pub fn m::Scan resolve(ref syntax::Parsed source,m::Frame frame,ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut v::Vec<m::Invocation> calls,ref mut v::Vec<m::Argument> arguments,ref mut usize work){
    var bool changed=false;var usize i=0;while(i<v::len<m::Argument>(arguments)){let a=*v::at<m::Argument>(arguments,i);if(!m::charge(work,1)){return m::Scan{.problem=m::fail(frame,10,a.start,a.end,0,0),.changed};}if(!a.checked){let p=scalar::evaluate(source,frame,a.expression,a.expected,imported,graph,atoms,expressions,work,0);if(p.code!=0){return m::Scan{.problem=p,.changed};}let e=*v::at<m::Expression>(expressions,a.expression-1);if(e.state==2){return m::Scan{.problem=m::fail(frame,1006,a.start,a.end,a.expected,0),.changed};}
            if(e.type!=0){let expected=types::at(graph,a.expected);let actual=types::at(graph,e.type);let bool shared=expected.kind==7 && actual.kind==8 && expected.first==actual.first; if(e.type!=a.expected && !shared){return m::Scan{.problem=m::fail(frame,1006,a.start,a.end,a.expected,e.type),.changed};}let slot=v::at_mut<m::Argument>(arguments,i);slot.actual=e.type;slot.checked=true;slot.shared=shared;changed=true;}
        }i+=1;
    }
    i=0;while(i<v::len<m::Invocation>(calls)){let c=*v::at<m::Invocation>(calls,i);if(!m::charge(work,1)){return m::Scan{.problem=m::fail(frame,10,c.start,c.end,0,0),.changed};}if(!c.checked){var bool complete=true;var usize j=0;while(j<c.argument_count){if(!m::charge(work,1)){return m::Scan{.problem=m::fail(frame,10,c.start,c.end,0,0),.changed};}complete=complete && v::at<m::Argument>(arguments,c.argument_start+j).checked;j+=1;}if(complete){let slot=v::at_mut<m::Invocation>(calls,i);slot.checked=true;changed=true;}}i+=1;}
    return m::Scan{.problem=m::fail(frame,0,0,0,0,0),.changed};
}
