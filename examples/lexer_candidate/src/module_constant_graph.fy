module module_constant_graph;
import module_constant_model as m;
import module_unit_value_model as problem;
import syntax as syntax;
import type_state as scalars;
import std::bytes as bytes;
import std::vec as v;

struct Range{usize start;usize count;}
struct Cursor{usize entry;usize next;}
fn problem::Problem dependency(ref mut v::Vec<usize> edges,usize target,usize module_index,syntax::Expression e){return if(v::push<usize>(edges,target).is_ok){m::ok()}else{m::fail(module_index,11,e.start,e.end)};}
fn problem::Problem walk(ref v::Vec<syntax::Parsed> sources,ref v::Vec<m::Entry> entries,ref v::Vec<m::Parameter> parameters,usize module_index,usize id,usize context,ref mut v::Vec<usize> edges,ref mut usize work,usize depth){
    let source=v::at<syntax::Parsed>(sources,module_index);if(id==0 || id>syntax::expression_count(source)){return m::fail(module_index,4302,0,0);}let e=syntax::expression_at(source,id);if(depth>=64 || !m::charge(work)){return m::fail(module_index,4304,e.start,e.end);}let raw=syntax::expression_text(source,id);
    if(e.kind==1){return if(syntax::literal_kind(source,id)==5 || bytes::equal(raw,b"true") || bytes::equal(raw,b"false")){m::ok()}else{m::fail(module_index,4302,e.start,e.end)};}
    if(e.kind==2){let parameter=switch(m::parameter(source,entries,parameters,context,raw,e.start,e.end,work)){case .ok(n):{n}case .err(p):{return p;}};if(parameter!=0){return m::ok();}let target=switch(m::find(sources,entries,module_index,raw,e.start,e.end,work)){case .ok(n):{n}case .err(p):{return p;}};return if(v::at<m::Entry>(entries,target-1).callable){m::fail(module_index,4302,e.start,e.end)}else{dependency(edges,target,module_index,e)};}
    if(e.kind==3 || e.kind==4 && (bytes::equal(raw,b"+") || bytes::equal(raw,b"-") || bytes::equal(raw,b"!") || bytes::equal(raw,b"~"))){return walk(sources,entries,parameters,module_index,e.left,context,edges,work,depth+1);}
    if(e.kind==5 && (scalars::operation(raw) || scalars::comparison(raw) || bytes::equal(raw,b"&&") || bytes::equal(raw,b"||"))){let p=walk(sources,entries,parameters,module_index,e.left,context,edges,work,depth+1);return if(p.code!=0){p}else{walk(sources,entries,parameters,module_index,e.right,context,edges,work,depth+1)};}
    if(e.kind==6){let target=switch(m::callee(sources,entries,parameters,module_index,id,context,work)){case .ok(n):{n}case .err(p):{return p;}};let p=dependency(edges,target,module_index,e);if(p.code!=0){return p;}var usize i=0;while(i<syntax::call_argument_count(source,id)){let p=walk(sources,entries,parameters,module_index,syntax::call_argument(source,id,i),context,edges,work,depth+1);if(p.code!=0){return p;}i+=1;}return m::ok();}
    if(e.kind==23 && syntax::type_at(source,e.right).kind==1 && m::admitted(scalars::primitive(source,e.right))){return walk(sources,entries,parameters,module_index,e.left,context,edges,work,depth+1);}
    if(e.kind==12){let tail=switch(m::tail(source,module_index,e.left)){case .ok(n):{n}case .err(p):{return p;}};return walk(sources,entries,parameters,module_index,tail,context,edges,work,depth+1);}
    if(e.kind==13){let branch=syntax::expression_at(source,e.right);let tail=switch(m::tail(source,module_index,branch.left)){case .ok(n):{n}case .err(p):{return p;}};if(branch.right==0){return m::fail(module_index,4302,e.start,e.end);}let p=walk(sources,entries,parameters,module_index,e.left,context,edges,work,depth+1);if(p.code!=0){return p;}let selected=walk(sources,entries,parameters,module_index,tail,context,edges,work,depth+1);return if(selected.code!=0){selected}else{walk(sources,entries,parameters,module_index,branch.right,context,edges,work,depth+1)};}
    return m::fail(module_index,4302,e.start,e.end);
}
// Explicit DFS frames keep long constant chains off the native call stack.
// Edges include all unselected branches; parameter names never become edges.
pub fn problem::Problem check(ref v::Vec<syntax::Parsed> sources,ref v::Vec<m::Entry> entries,ref v::Vec<m::Parameter> parameters,ref mut usize work){
    var v::Vec<usize> edges=v::empty<usize>();var v::Vec<Range> ranges=v::empty<Range>();var v::Vec<u32> states=v::empty<u32>();var usize i=0;
    while(i<v::len<m::Entry>(entries)){let row=*v::at<m::Entry>(entries,i);let start=v::len<usize>(ref edges);let p=walk(sources,entries,parameters,row.module_index,row.expression,if(row.callable){i+1}else{0},ref mut edges,work,0);if(p.code!=0){return p;}if(!v::push<Range>(ref mut ranges,Range{.start,.count=v::len<usize>(ref edges)-start}).is_ok || !v::push<u32>(ref mut states,0).is_ok){return m::fail(row.module_index,11,row.start,row.end);}i+=1;}
    var v::Vec<Cursor> stack=v::empty<Cursor>();i=0;
    while(i<v::len<m::Entry>(entries)){if(*v::at<u32>(ref states,i)==0){if(!v::push<Cursor>(ref mut stack,Cursor{.entry=i+1,.next=0}).is_ok){return m::fail(0,11,0,0);}while(v::len<Cursor>(ref stack)!=0){let index=v::len<Cursor>(ref stack)-1;let cursor=*v::at<Cursor>(ref stack,index);let row=*v::at<m::Entry>(entries,cursor.entry-1);if(!m::charge(work)){return m::fail(row.module_index,4304,row.start,row.end);}{let cell=v::at_mut<u32>(ref mut states,cursor.entry-1);*cell=1;}let range=*v::at<Range>(ref ranges,cursor.entry-1);
        if(cursor.next==range.count){{let cell=v::at_mut<u32>(ref mut states,cursor.entry-1);*cell=2;}discard v::truncate<Cursor>(ref mut stack,index);continue;}
        {let cell=v::at_mut<Cursor>(ref mut stack,index);cell.next+=1;}let target=*v::at<usize>(ref edges,range.start+cursor.next);let state=*v::at<u32>(ref states,target-1);if(state==1){let dependency=*v::at<m::Entry>(entries,target-1);let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,dependency.module_index),dependency.declaration-1);return m::fail(dependency.module_index,4304,d.name_start,d.name_end);}if(state==0 && !v::push<Cursor>(ref mut stack,Cursor{.entry=target,.next=0}).is_ok){return m::fail(row.module_index,11,row.start,row.end);}
    }}i+=1;}
    var usize module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);i=1;while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(d.kind==8){discard v::truncate<usize>(ref mut edges,0);let p=walk(sources,entries,parameters,module_index,d.expression,0,ref mut edges,work,0);if(p.code!=0){return p;}}i+=1;}module_index+=1;}return m::ok();
}
