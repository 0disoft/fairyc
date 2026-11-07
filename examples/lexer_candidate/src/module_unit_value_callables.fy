module module_unit_value_callables;
import module_unit_value_model as m;
import module_types as types;
import syntax as syntax;
import std::vec as v;

pub fn Result<bool,m::Problem> ready(ref syntax::Parsed source,m::Frame frame,m::Invocation c,ref v::Vec<types::Type> graph,ref v::Vec<m::Expression> expressions){
    if(c.kind!=4 || c.expression==0 || c.expression>v::len<m::Expression>(expressions) || c.target_module!=0 || c.target_declaration!=0 || c.instance!=0 || c.argument_count!=1){return .err(m::fail(frame,12,c.start,c.end,0,0));}
    let e=*v::at<m::Expression>(expressions,c.expression-1);if(e.kind!=6 || e.source_index!=c.source_index){return .err(m::fail(frame,12,c.start,c.end,0,0));}
    let local=m::index(expressions,e.left);if(local==0){return .err(m::fail(frame,12,c.start,c.end,0,0));}let callee=*v::at<m::Expression>(expressions,local-1);
    if(callee.kind!=2 || callee.binding==0 || callee.place.kind!=1 || callee.place.mutable || syntax::expression_at(source,e.left).kind!=2){return .err(m::fail(frame,1006,callee.start,callee.end,0,callee.type));}
    if(callee.state==2){return .err(m::fail(frame,1006,callee.start,callee.end,0,0));}if(callee.type==0){return .ok(false);}if(callee.type>types::count(graph)){return .err(m::fail(frame,12,callee.start,callee.end,0,callee.type));}
    let t=types::at(graph,callee.type);if(t.kind!=23 || t.scalar!=0 || t.module_index!=0 || t.declaration!=0 || t.first!=0 || t.second!=0){return .err(m::fail(frame,1006,callee.start,callee.end,0,callee.type));}
    return .ok(true);
}
