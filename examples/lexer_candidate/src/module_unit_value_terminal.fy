module module_unit_value_terminal;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_types as types;
import syntax as syntax;
import std::vec as v;

fn m::Problem ok(m::Frame f){return m::fail(f,0,0,0,0,0);}
fn Result<bool,m::Problem> contains_resource(m::Frame f,m::Expression e,usize id,ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,ref mut usize work,usize depth){
    if(depth>=64 || !m::charge(work,1)){return .err(m::fail(f,10,e.start,e.end,0,0));}let t=types::at(graph,id);if(t.kind==3 || t.kind==13){return .ok(true);}
    if(t.kind==5 || t.kind==6 || t.kind==16){let first=switch(contains_resource(f,e,t.first,imported,graph,work,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(first || t.kind!=6){return .ok(first);}return contains_resource(f,e,t.second,imported,graph,work,depth+1);}
    if(t.kind!=2){return .ok(false);}var usize i=0;while(i<interfaces::shape_count(imported)){if(!m::charge(work,1)){return .err(m::fail(f,10,e.start,e.end,0,0));}let s=interfaces::shape_at(imported,i);if(s.type==id){var usize j=0;while(j<s.field_count){let field=interfaces::field_at(imported,s.field_start+j);let child=switch(contains_resource(f,e,field.type,imported,graph,work,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(child){return .ok(true);}j+=1;}return .ok(false);}i+=1;}return .err(m::fail(f,12,e.start,e.end,0,id));
}
fn m::Problem burn_value(m::Frame f,m::Expression e,ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,ref mut usize work){
    let t=types::at(graph,e.type);if(e.kind!=2 || e.place.kind!=1 || e.place.root==0 || t.kind!=3 || t.module_index!=f.module_index){return m::fail(f,1109,e.start,e.end,0,e.type);}
    var usize i=0;while(i<interfaces::shape_count(imported)){if(!m::charge(work,1)){return m::fail(f,10,e.start,e.end,0,0);}let s=interfaces::shape_at(imported,i);if(s.type==e.type){if(s.module_index!=t.module_index || s.declaration!=t.declaration){return m::fail(f,12,e.start,e.end,0,e.type);}if((s.abilities & 1)!=0){return m::fail(f,1109,e.start,e.end,0,e.type);}var usize j=0;while(j<s.field_count){let field=interfaces::field_at(imported,s.field_start+j);let child=switch(contains_resource(f,e,field.type,imported,graph,work,0)){case .ok(b):{b}case .err(p):{return p;}};if(child){return m::fail(f,1109,e.start,e.end,0,e.type);}j+=1;}return ok(f);}i+=1;}return m::fail(f,12,e.start,e.end,0,e.type);
}
fn m::Problem deferred(m::Frame f,m::Consumer c,m::Expression e,ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,ref v::Vec<m::Invocation> calls,ref v::Vec<m::Argument> arguments,ref mut usize work,ref mut bool ready){
    if(e.kind!=6 || e.builtin.kind!=0 || types::at(graph,e.type).kind!=1 || types::at(graph,e.type).scalar!=7){return m::fail(f,2,c.start,c.end,0,e.type);}
    var usize low=0;var usize high=v::len<m::Invocation>(calls);while(low<high){if(!m::charge(work,1)){return m::fail(f,10,e.start,e.end,0,0);}let usize mid=low+(high-low)/2;if(v::at<m::Invocation>(calls,mid).expression<c.expression){low=mid+1;}else{high=mid;}}if(low<v::len<m::Invocation>(calls)){let call=*v::at<m::Invocation>(calls,low);if(call.expression==c.expression){if(!call.checked){return ok(f);}var bool found=false;
        if(call.kind==2){let plan=interfaces::specializations(imported);if(call.instance==0 || call.instance>generics::count(plan)){return m::fail(f,12,e.start,e.end,0,0);}let target=generics::at(plan,call.instance);if(target.module_index!=call.target_module || target.declaration!=call.target_declaration || target.result!=e.type || target.parameter_count!=call.argument_count){return m::fail(f,12,e.start,e.end,0,0);}found=true;}
        else if(call.kind==1){var usize j=0;while(j<interfaces::callable_count(imported)){if(!m::charge(work,1)){return m::fail(f,10,e.start,e.end,0,0);}let target=interfaces::callable_at(imported,j);if(target.module_index==call.target_module && target.declaration==call.target_declaration){if(target.foreign){return m::fail(f,2,c.start,c.end,0,e.type);}if(target.template || target.result!=e.type || target.parameter_count!=call.argument_count){return m::fail(f,12,e.start,e.end,0,0);}found=true;break;}j+=1;}}
        if(!found || call.argument_start>v::len<m::Argument>(arguments) || call.argument_count>v::len<m::Argument>(arguments)-call.argument_start){return m::fail(f,12,e.start,e.end,0,0);}var usize j=0;while(j<call.argument_count){if(!m::charge(work,1)){return m::fail(f,10,e.start,e.end,0,0);}let a=*v::at<m::Argument>(arguments,call.argument_start+j);if(!a.checked){return ok(f);}let t=types::at(graph,a.actual);if(t.kind!=1 || t.scalar<1 || t.scalar>6){return m::fail(f,2,c.start,c.end,0,a.actual);}j+=1;}*ready=true;return ok(f);
    }}return m::fail(f,2,c.start,c.end,0,e.type);
}
pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame f,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref v::Vec<m::Invocation> calls,ref v::Vec<m::Argument> arguments,ref mut m::Consumer c,ref mut usize work){
    if(c.kind!=13 && c.kind!=16 || c.expression==0 || c.expression>v::len<m::Expression>(expressions)){return m::fail(f,12,c.start,c.end,0,0);}let p=scalar::evaluate(source,f,c.expression,0,imported,graph,atoms,expressions,work,0);if(p.code!=0){return p;}let e=*v::at<m::Expression>(expressions,c.expression-1);if(e.state==2){return m::fail(f,if(c.kind==16){1109}else{2},if(c.kind==16){e.start}else{c.start},if(c.kind==16){e.end}else{c.end},0,0);}if(e.type==0){return ok(f);}var bool ready=c.kind==16;let checked=if(c.kind==16){burn_value(f,e,imported,graph,work)}else{deferred(f,*c,e,imported,graph,calls,arguments,work,ref mut ready)};if(checked.code!=0){return checked;}if(ready){c.expected=e.type;c.actual=e.type;c.checked=true;}return ok(f);
}
