module module_unit_origin_flow;
import module_unit_origin_inputs as inputs;
import module_unit_origin_sets as sets;
import module_unit_origin_flow_model as g;
import module_unit_origin_flow_control as control;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_value_builtin as borrows;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn bool borrowed(ref inputs::Inputs input,ref values::Plan primary,usize frame,usize type){return type!=0 && borrows::borrowed(inputs::type_at(input,primary,frame,type));}
fn Result<usize,m::Problem> place(ref inputs::Inputs input,ref values::Plan primary,ref m::Values body,usize original,ref v::Vec<usize> env,ref mut g::Context c,usize depth){
    let e=switch(g::expression(body,original,c)){case .ok(e):{e}case .err(p):{return .err(p);}};if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,e.start,e.end));}
    if(e.kind==2 && e.binding!=0){if(e.binding>v::len<usize>(env)){return .err(g::fail(c,12,e.start,e.end));}if(borrowed(input,primary,c.frame,e.type)){let usize n=*v::at<usize>(env,e.binding-1);return .ok(if(n==0){1}else{n});}return g::leaf(input,e,c);}
    if(e.kind==3 || e.kind==8 || e.kind==9 || e.kind==26 || e.kind==4 && e.place.kind==3){return place(input,primary,body,e.left,env,c,depth+1);}return .ok(1);
}
// Arguments are evaluated once in source order. Later argument side effects
// cannot retroactively change an earlier argument's recorded origin.
fn Result<usize,m::Problem> substitute(ref inputs::Inputs input,ref m::Values body,usize target,m::Invocation call,ref v::Vec<usize> arguments,ref v::Vec<usize> summaries,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut g::Context c){
    if(target==0){return .ok(1);}let summary=*v::at<usize>(summaries,target-1);if(summary==0 || summary==1){return .ok(1);}let f=inputs::frame(input,target);if(f.parameter_count!=call.argument_count || v::len<usize>(arguments)!=call.argument_count){return .err(g::fail(c,12,call.start,call.end));}var usize result=0;var usize i=0;
    while(i<sets::count(entries,members,summary)){if(!m::charge(ref mut c.work,1)){return .err(g::fail(c,10,call.start,call.end));}let leaf=sets::member(entries,members,summary,i);var usize actual=2;if(leaf!=2){let usize owner=sets::owner_id(leaf);if(owner==0 || owner>inputs::owner_count(input)){return .err(g::fail(c,12,call.start,call.end));}let local=inputs::owner(input,owner);if(local.kind==11){actual=leaf;}else{if(local.frame!=target || local.kind!=1 || !local.borrowed){return .ok(1);}var usize ordinal=0;var bool found=false;while(ordinal<f.parameter_count){if(!m::charge(ref mut c.work,1)){return .err(g::fail(c,10,call.start,call.end));}if(inputs::parameter(input,target,ordinal)==owner){found=true;break;}ordinal+=1;}if(!found){return .err(g::fail(c,12,call.start,call.end));}actual=*v::at<usize>(arguments,ordinal);if(actual==0){actual=1;}}}
        result=switch(g::join(entries,members,c,result,actual,call.start,call.end)){case .ok(n):{n}case .err(p):{return .err(p);}};i+=1;
    }return .ok(result);
}
pub fn Result<g::Flow,m::Problem> expression(ref v::Vec<syntax::Parsed> sources,ref inputs::Inputs input,ref values::Plan primary,ref v::Vec<usize> summaries,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<g::Returned> returns,ref mut v::Vec<usize> env,ref mut v::Vec<g::Edge> edges,ref mut g::Context c,usize original,usize depth){
    if(original==0){return .ok(g::Flow{.origin=0,.outcomes=1});}let source=v::at<syntax::Parsed>(sources,c.value.module_index);let body=inputs::body(input,primary,c.frame);let e=switch(g::expression(body,original,c)){case .ok(e):{e}case .err(p):{return .err(p);}};if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,e.start,e.end));}
    if(e.kind==12 || e.kind==31){return control::block(sources,input,primary,summaries,entries,members,returns,env,edges,c,e.left,depth+1);}if(e.kind==13 || e.kind==15){return control::branch(sources,input,primary,summaries,entries,members,returns,env,edges,c,e,depth+1);}
    let bool descriptor=borrowed(input,primary,c.frame,e.type);
    if(e.kind==1 || e.kind==28 || e.kind==17 || e.kind==16 || e.kind==32 || e.kind==33){var usize origin=0;if(e.kind==1 && descriptor){let owner=switch(inputs::static_owner(input,primary,c.frame,original,ref mut c.work)){case .ok(n):{n}case .err(p):{return .err(p);}};origin=owner+2;}return .ok(g::Flow{.origin,.outcomes=1});}
    if(e.kind==2){var usize origin=0;if(descriptor){if(e.binding==0 || e.binding>v::len<usize>(env)){origin=1;}else{origin=*v::at<usize>(env,e.binding-1);if(origin==0){origin=1;}}}return .ok(g::Flow{.origin,.outcomes=1});}
    let left=switch(expression(sources,input,primary,summaries,entries,members,returns,env,edges,c,e.left,depth+1)){case .ok(f):{f}case .err(p):{return .err(p);}};if((left.outcomes & 1)==0){return .ok(left);}
    if(e.kind==3){return .ok(left);}if(e.kind==21 || e.kind==22){let origin=switch(place(input,primary,body,e.left,env,c,depth+1)){case .ok(n):{n}case .err(p):{return .err(p);}};return .ok(g::Flow{.origin,.outcomes=left.outcomes});}
    if(e.kind==20 || e.kind==23 || e.kind==8 || e.kind==4){return .ok(g::Flow{.origin=if(descriptor){left.origin}else{0},.outcomes=left.outcomes});}
    if(e.kind==30){return .ok(g::Flow{.origin=if(descriptor){left.origin}else{0},.outcomes=left.outcomes | 2});}
    if(e.kind==6 || e.kind==10 || e.kind==24){var v::Vec<usize> order=v::empty<usize>();var v::Vec<usize> arguments=v::empty<usize>();var usize link=e.right;let u32 kind=if(e.kind==6){7}else if(e.kind==10){11}else{25};
        while(link!=0){let row=syntax::expression_at(source,link);if(!m::charge(ref mut c.work,1) || v::len<usize>(ref order)>=1024){return .err(g::fail(c,10,row.start,row.end));}if(row.kind!=kind){return .err(g::fail(c,12,row.start,row.end));}if(!v::push<usize>(ref mut order,row.left).is_ok){return .err(g::fail(c,11,row.start,row.end));}link=row.right;}
        var usize outcomes=left.outcomes;var usize i=v::len<usize>(ref order);while(i>0 && (outcomes & 1)!=0){i-=1;let next=switch(expression(sources,input,primary,summaries,entries,members,returns,env,edges,c,*v::at<usize>(ref order,i),depth+1)){case .ok(f):{f}case .err(p):{return .err(p);}};outcomes=(outcomes & 14) | next.outcomes;if(!v::push<usize>(ref mut arguments,next.origin).is_ok){return .err(g::fail(c,11,e.start,e.end));}}
        var usize origin=0;if(descriptor && (outcomes & 1)!=0){origin=1;if(e.kind==6 && e.builtin.kind==200){origin=switch(inputs::static_owner(input,primary,c.frame,original,ref mut c.work)){case .ok(n):{n+2}case .err(p):{return .err(p);}};}else if(e.kind==6 && (e.builtin.kind>=9 && e.builtin.kind<=12 || e.builtin.kind==106 || e.builtin.kind==107)){if(v::len<usize>(ref arguments)==0){return .err(g::fail(c,12,e.start,e.end));}origin=*v::at<usize>(ref arguments,0);}else if(e.kind==6){i=0;while(i<m::call_count(body)){if(!m::charge(ref mut c.work,1)){return .err(g::fail(c,10,e.start,e.end));}let call=m::call_at(body,i);if(call.source_index==original){if(!call.checked){return .err(g::fail(c,12,e.start,e.end));}origin=switch(substitute(input,body,inputs::target(input,c.frame,i),call,ref arguments,summaries,entries,members,c)){case .ok(n):{n}case .err(p):{return .err(p);}};break;}i+=1;}}}return .ok(g::Flow{.origin,.outcomes});
    }
    // && and || have both the skipped and the evaluated right path.
    let op=syntax::expression_text(source,original);if(e.kind==5 && (bytes::equal(op,b"&&") || bytes::equal(op,b"||"))){let base=switch(g::copy(env,c,e.start,e.end)){case .ok(v):{v}case .err(p):{return .err(p);}};var branch=switch(g::copy(env,c,e.start,e.end)){case .ok(v):{v}case .err(p):{return .err(p);}};let right=switch(expression(sources,input,primary,summaries,entries,members,returns,ref mut branch,edges,c,e.right,depth+1)){case .ok(f):{f}case .err(p):{return .err(p);}};if((right.outcomes & 1)!=0){let p=g::merge(ref base,ref branch,env,false,entries,members,c,e.start,e.end);if(p.code!=0){return .err(p);}}return .ok(g::Flow{.origin=0,.outcomes=left.outcomes | right.outcomes | 1});}
    let right=switch(expression(sources,input,primary,summaries,entries,members,returns,env,edges,c,e.right,depth+1)){case .ok(f):{f}case .err(p):{return .err(p);}};return .ok(g::Flow{.origin=if(descriptor){left.origin}else{0},.outcomes=(left.outcomes & 14) | right.outcomes});
}
