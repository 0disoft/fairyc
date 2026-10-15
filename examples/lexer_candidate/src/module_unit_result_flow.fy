module module_unit_result_flow;
import module_unit_value_model as m;
import module_unit_values as values;
import result_flow_graph as graph;
import result_flow as flow;
import type_state as state;
import tagged_types as tagged;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn bool result(ref values::Plan typed,usize type){return type!=0 && values::type_at(typed,type).kind==6;}
fn m::Problem body(ref syntax::Parsed source,ref values::Plan typed,ref m::Values body,ref mut usize work){
    let frame=m::frame(body);var bool needed=false;var usize i=1;
    while(i<=m::expression_count(body)){if(!m::charge(work,1)){return m::fail(frame,10,0,0,0,0);}needed=needed || result(typed,m::expression_at(body,i).type);i+=1;}
    i=1;while(i<=m::local_count(body)){if(!m::charge(work,1)){return m::fail(frame,10,0,0,0,0);}needed=needed || result(typed,m::local_at(body,i).type);i+=1;}
    if(!needed){return m::fail(frame,0,0,0,0,0);}
    let size=if(syntax::expression_count(source)>syntax::entry_count(source)){syntax::expression_count(source)}else{syntax::entry_count(source)};
    if(!m::charge(work,size)){return m::fail(frame,10,0,0,0,0);}var tables=state::empty();
    if(!state::initialize(ref mut tables,0,syntax::expression_count(source),syntax::entry_count(source))){return m::fail(frame,11,0,0,0,0);}
    // One membership marker suffices: payload types remain owned by Values.
    let marker=tagged::make_result(1,1);i=1;
    while(i<=m::expression_count(body)){let e=m::expression_at(body,i);
        if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}
        if(result(typed,e.type)){state::set_expressions(ref mut tables,e.source_index-1,marker);}
        state::set_uses(ref mut tables,e.source_index-1,e.binding);i+=1;
    }
    i=1;while(i<=m::expression_count(body)){let e=m::expression_at(body,i);
        if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}
        if(e.kind==8 && e.left!=0 && tagged::result(state::expressions(ref tables,e.left-1)) && bytes::equal(syntax::expression_text(source,e.source_index),b"is_ok")){state::set_flags(ref mut tables,e.source_index-1,2048);}i+=1;
    }
    var v::Vec<usize> columns=v::empty<usize>();var v::Vec<graph::Node> parameters=v::empty<graph::Node>();var usize width=0;i=0;
    while(i<=m::local_count(body)){var usize col=0;
        if(i!=0){let n=m::local_at(body,i);if(!m::charge(work,1)){return m::fail(frame,10,n.start,n.end,0,0);}
            if(result(typed,n.type)){width+=1;col=width;
                if(n.kind==1 || n.kind==10){if(!v::push<graph::Node>(ref mut parameters,graph::Node{.kind=2,.column=col,.start=n.start,.end=n.end,.next=0,.alternative=0}).is_ok){return m::fail(frame,11,n.start,n.end,0,0);}}
            }
            if(col!=0 && n.source_index!=0){if(n.kind==4){state::set_uses(ref mut tables,n.source_index-1,i);}else if(n.kind==1 || n.kind==2 || n.kind==3 || n.kind==10){state::set_entries(ref mut tables,n.source_index-1,i);}}
        }
        if(!v::push<usize>(ref mut columns,col).is_ok){return m::fail(frame,11,0,0,0,0);}i+=1;
    }
    let d=syntax::declaration_at(source,frame.declaration-1);
    let cfg=switch(graph::projected(source,ref tables,columns,width,parameters,if(frame.kind==2){d.cleanup_body}else{d.body})){case .ok(p):{p}case .err(p):{return m::fail(frame,p.code,p.start,p.end,0,0);}};
    let p=flow::analyze(ref cfg);return m::fail(frame,if(p.code==42){4101}else{p.code},p.start,p.end,0,0);
}
// This phase consumes current checked Values before any IR or cache publication.
pub fn m::Problem check(ref v::Vec<syntax::Parsed> sources,ref values::Plan typed){
    var usize work=0;var usize i=0;var first=m::Problem{.code=0,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};
    while(i<values::count(typed)){let current=values::at(typed,i);let p=body(v::at<syntax::Parsed>(sources,m::frame(current).module_index),typed,current,ref mut work);
        if(p.code!=0){if(p.code!=4101){return p;}if(first.code==0 || p.module_index<first.module_index || p.module_index==first.module_index && (p.start<first.start || p.start==first.start && p.end<first.end)){first=p;}}i+=1;
    }return first;
}
