module module_unit_move_walk;
import module_unit_move_model as g;
import module_unit_move_control as control;
import module_unit_value_model as m;
import module_unit_values as values;
import syntax as syntax;
import std::vec as v;
import std::bytes as bytes;

// Keep earlier consumed operands alive until the whole call/record is complete.
// Restoring in the caller covers every normal, unreachable and error return.
fn Result<bool,m::Problem> operands(ref v::Vec<syntax::Parsed> sources,ref values::Plan primary,ref m::Values body,ref v::Vec<u8> owners,ref mut v::Vec<g::Cell> env,ref mut v::Vec<g::Edge> edges,ref mut v::Vec<g::Diagnostic> diagnostics,ref mut v::Vec<g::Fact> facts,m::Expression e,usize depth,ref mut g::Context c){
    let source=v::at<syntax::Parsed>(sources,c.value.module_index);let count=if(e.kind==6){syntax::call_argument_count(source,e.source_index)}else if(e.kind==10){syntax::record_field_count(source,e.source_index)}else{syntax::array_element_count(source,e.source_index)};var usize i=0;
    while(i<count){let original=if(e.kind==6){syntax::call_argument(source,e.source_index,i)}else if(e.kind==10){syntax::expression_at(source,syntax::record_field(source,e.source_index,i)).left}else{syntax::array_element(source,e.source_index,i)};let actual=switch(g::expression(body,original,c)){case .ok(e):{e}case .err(p):{return .err(p);}};let consumes=switch(g::owned(primary,actual.type,c,0)){case .ok(b):{b}case .err(p):{return .err(p);}};let reachable=switch(expression(sources,primary,body,owners,env,edges,diagnostics,facts,original,consumes,depth+1,c)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!reachable){return .ok(false);}if(consumes){let p=g::retain_pending(actual.type,actual.source_index,c);if(p.code!=0){return .err(p);}}i+=1;}return .ok(true);
}

pub fn Result<bool,m::Problem> expression(ref v::Vec<syntax::Parsed> sources,ref values::Plan primary,ref m::Values body,ref v::Vec<u8> owners,ref mut v::Vec<g::Cell> env,ref mut v::Vec<g::Edge> edges,ref mut v::Vec<g::Diagnostic> diagnostics,ref mut v::Vec<g::Fact> facts,usize id,bool consuming,usize depth,ref mut g::Context c){
    if(id==0){return .ok(true);}let e=switch(g::expression(body,id,c)){case .ok(e):{e}case .err(p):{return .err(p);}};if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,e.start,e.end));}let source=v::at<syntax::Parsed>(sources,c.value.module_index);
    if(e.kind==2){if(e.binding==0 || e.binding>v::len<g::Cell>(env) || *v::at<u8>(owners,e.binding-1)==0){return .ok(true);}let before=*v::at<g::Cell>(env,e.binding-1);if(before.state==0){return .err(g::fail(c,12,e.start,e.end));}var after=before;
        if(before.state==2){let p=g::report(body,e.binding,1101,e.start,e.end,before,0,0,diagnostics,c);if(p.code!=0){return .err(p);}}
        else if(before.state==1 && consuming){after=g::Cell{.state=2,.move_start=e.start,.move_end=e.end};let slot=v::at_mut<g::Cell>(env,e.binding-1);*slot=after;}
        let p=g::fact(e.binding,if(consuming){2}else{1},before,after,e.start,e.end,facts,c);return if(p.code==0){.ok(true)}else{.err(p)};
    }
    if(e.kind==12 || e.kind==31){return control::block(sources,primary,body,owners,env,edges,diagnostics,facts,e.left,consuming,depth+1,c);}
    if(e.kind==23){return expression(sources,primary,body,owners,env,edges,diagnostics,facts,e.left,false,depth+1,c);}
    if(e.kind==3){return expression(sources,primary,body,owners,env,edges,diagnostics,facts,e.left,consuming,depth+1,c);}
    if(e.kind==13 || e.kind==15){return control::branch(sources,primary,body,owners,env,edges,diagnostics,facts,e,consuming,depth+1,c);}
    let op=syntax::expression_text(source,id);
    if(e.kind==5 && (bytes::equal(op,b"&&") || bytes::equal(op,b"||"))){let left=switch(expression(sources,primary,body,owners,env,edges,diagnostics,facts,e.left,false,depth+1,c)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!left){return .ok(false);}let base=switch(g::copy(env,c)){case .ok(v):{v}case .err(p):{return .err(p);}};var right=switch(g::copy(env,c)){case .ok(v):{v}case .err(p):{return .err(p);}};let reachable=switch(expression(sources,primary,body,owners,ref mut right,edges,diagnostics,facts,e.right,false,depth+1,c)){case .ok(b):{b}case .err(p):{return .err(p);}};if(reachable){let p=g::merge(body,ref base,ref right,env,false,e.start,e.end,diagnostics,c);if(p.code!=0){return .err(p);}}return .ok(true);}
    if(e.kind==6 || e.kind==10 || e.kind==24){let base=c.pending_count;let result=operands(sources,primary,body,owners,env,edges,diagnostics,facts,e,depth,c);c.pending_count=base;return result;}
    if(e.kind==1 || e.kind==28 || e.kind==17 || e.kind==16 || e.kind==32 || e.kind==33 || e.kind==20){return .ok(true);}
    // Reading a field/borrow/dereference selects its owner without consuming it.
    // ? consumes the tagged operand on both its success and return alternatives.
    var bool child_mode=e.kind==30;
    if(e.kind==8 && bytes::equal(op,b"is_ok")){let parent=switch(g::expression(body,e.left,c)){case .ok(e):{e}case .err(p):{return .err(p);}};child_mode=switch(g::owned(primary,parent.type,c,0)){case .ok(b):{b}case .err(p):{return .err(p);}};}
    if(e.kind==8 && consuming && e.field.id!=0){let receiver=switch(g::expression(body,e.left,c)){case .ok(e):{e}case .err(p):{return .err(p);}};let moves=switch(g::owned(primary,e.type,c,0)){case .ok(b):{b}case .err(p):{return .err(p);}};
        if(moves && receiver.kind==2 && receiver.binding!=0){let defining=v::at<syntax::Parsed>(sources,e.field.module_index);let field=syntax::field_at(defining,e.field.source_index);let p=g::partial(body,e,receiver.binding,e.field.module_index,field.name_start,field.name_end,diagnostics,c);if(p.code!=0){return .err(p);}}
    }
    let left=switch(expression(sources,primary,body,owners,env,edges,diagnostics,facts,e.left,child_mode,depth+1,c)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!left){return .ok(false);}
    if(e.kind==30){let dropped=g::cleanup_live(body,env,6,e.start,e.end,facts,c);if(dropped.code!=0){return .err(dropped);}}
    if(e.kind==8 && child_mode){let operand=switch(g::expression(body,e.left,c)){case .ok(e):{e}case .err(p):{return .err(p);}};let dropped=g::schedule_value(operand.source_index,operand.type,7,e.start,e.end,facts,c);if(dropped.code!=0){return .err(dropped);}}
    if(e.kind==26){let bounds=syntax::expression_at(source,e.right);let first=switch(expression(sources,primary,body,owners,env,edges,diagnostics,facts,bounds.left,false,depth+1,c)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!first){return .ok(false);}return expression(sources,primary,body,owners,env,edges,diagnostics,facts,bounds.right,false,depth+1,c);}
    return expression(sources,primary,body,owners,env,edges,diagnostics,facts,e.right,false,depth+1,c);
}
