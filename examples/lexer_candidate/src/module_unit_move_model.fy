module module_unit_move_model;
import module_unit_value_model as m;
import module_unit_values as values;
import std::vec as v;

// Zero is out of scope, 1 Live, 2 Dead and 3 MaybeLive. Identity is the
// lexical local ID in one immutable Values frame, never the identifier text.
pub struct Cell {u32 state;usize move_start;usize move_end;}
pub struct Diagnostic {u32 code;usize frame;usize module_index;usize start;usize end;usize declaration_start;usize declaration_end;usize move_start;usize move_end;usize header_start;usize header_end;usize field_module;usize field_start;usize field_end;}
pub struct Fact {usize frame;usize binding;u32 kind;u32 before;u32 after;usize start;usize end;usize move_start;usize move_end;usize type;u32 exit;}
pub struct Context {m::Frame value;usize frame;usize work;usize loop;usize cleanup_end;usize pending_count;usize[1024] pending_types;usize[1024] pending_loops;}
pub resource Edge {usize loop;bool back;usize start;usize end;v::Vec<Cell> cells;cleanup{}}
pub fn Edge edge(usize loop,bool back,usize start,usize end,v::Vec<Cell> cells){return Edge{.loop,.back,.start,.end,.cells};}
pub fn usize edge_loop(ref Edge p){return p.loop;}
pub fn bool edge_back(ref Edge p){return p.back;}
pub fn usize edge_start(ref Edge p){return p.start;}
pub fn usize edge_end(ref Edge p){return p.end;}
pub fn ref v::Vec<Cell> edge_cells(ref Edge p){return ref p.cells;}
pub fn Cell empty(){return Cell{.state=0,.move_start=0,.move_end=0};}
pub fn m::Problem fail(ref Context c,u32 code,usize start,usize end){return m::fail(c.value,code,start,end,0,0);}
pub fn Result<bool,m::Problem> owned(ref values::Plan primary,usize type,ref mut Context c,usize depth){
    if(type==0){return .ok(false);}if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}let t=values::type_at(primary,type);if(t.kind==3 || t.kind==13){return .ok(true);}if(t.kind!=5 && t.kind!=6){return .ok(false);}let first=switch(owned(primary,t.first,c,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};return if(first || t.kind==5){.ok(first)}else{owned(primary,t.second,c,depth+1)};
}
pub fn Result<m::Expression,m::Problem> expression(ref m::Values body,usize original,ref mut Context c){var usize low=1;var usize high=m::expression_count(body)+1;while(low<high){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}let mid=low+(high-low)/2;if(m::expression_at(body,mid).source_index<original){low=mid+1;}else{high=mid;}}if(low>m::expression_count(body) || m::expression_at(body,low).source_index!=original){return .err(fail(c,12,0,0));}return .ok(m::expression_at(body,low));}
pub fn Result<usize,m::Problem> local(ref m::Values body,usize original,u32 kind,ref mut Context c){var usize i=1;while(i<=m::local_count(body)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}let n=m::local_at(body,i);if(n.source_index==original && (kind==2 && (n.kind==2 || n.kind==3) || kind!=2 && n.kind==kind)){return .ok(i);}i+=1;}return .ok(0);}
pub fn Result<v::Vec<Cell>,m::Problem> copy(ref v::Vec<Cell> source,ref mut Context c){var v::Vec<Cell> result=v::empty<Cell>();var usize i=0;while(i<v::len<Cell>(source)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}if(!v::push<Cell>(ref mut result,*v::at<Cell>(source,i)).is_ok){return .err(fail(c,11,0,0));}i+=1;}return .ok(result);}
pub fn m::Problem replace(ref v::Vec<Cell> source,ref mut v::Vec<Cell> target,ref mut Context c){if(v::len<Cell>(source)!=v::len<Cell>(target)){return fail(c,12,0,0);}var usize i=0;while(i<v::len<Cell>(source)){if(!m::charge(ref mut c.work,1)){return fail(c,10,0,0);}let slot=v::at_mut<Cell>(target,i);*slot=*v::at<Cell>(source,i);i+=1;}return fail(c,0,0,0);}
pub fn m::Problem report(ref m::Values body,usize binding,u32 code,usize start,usize end,Cell cell,usize header_start,usize header_end,ref mut v::Vec<Diagnostic> diagnostics,ref mut Context c){
    if(binding==0 || binding>m::local_count(body)){return fail(c,12,start,end);}let n=m::local_at(body,binding);var usize i=0;while(i<v::len<Diagnostic>(diagnostics)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let d=*v::at<Diagnostic>(diagnostics,i);if(d.frame==c.frame && d.code==code && d.start==start && d.end==end && d.declaration_start==n.start && d.declaration_end==n.end){return fail(c,0,0,0);}i+=1;}
    if(v::len<Diagnostic>(diagnostics)>=8192 || !m::charge(ref mut c.work,1)){return fail(c,10,start,end);}return if(v::push<Diagnostic>(diagnostics,Diagnostic{.code,.frame=c.frame,.module_index=c.value.module_index,.start,.end,.declaration_start=n.start,.declaration_end=if(n.kind==10){c.cleanup_end}else{n.end},.move_start=cell.move_start,.move_end=cell.move_end,.header_start,.header_end,.field_module=0,.field_start=0,.field_end=0}).is_ok){fail(c,0,0,0)}else{fail(c,11,start,end)};
}
pub fn m::Problem fact(usize binding,u32 kind,Cell before,Cell after,usize start,usize end,ref mut v::Vec<Fact> facts,ref mut Context c){if(v::len<Fact>(facts)>=16384 || !m::charge(ref mut c.work,1)){return fail(c,10,start,end);}return if(v::push<Fact>(facts,Fact{.frame=c.frame,.binding,.kind,.before=before.state,.after=after.state,.start,.end,.move_start=after.move_start,.move_end=after.move_end,.type=0,.exit=0}).is_ok){fail(c,0,0,0)}else{fail(c,11,start,end)};}
pub fn m::Problem produce(ref m::Values body,usize binding,usize start,usize end,ref mut v::Vec<Cell> env,ref mut v::Vec<Fact> facts,ref mut Context c){if(binding==0){return fail(c,0,0,0);}let before=*v::at<Cell>(env,binding-1);let after=Cell{.state=1,.move_start=0,.move_end=0};let recorded=fact(binding,3,before,after,start,end,facts,c);if(recorded.code!=0){return recorded;}let slot=v::at_mut<Cell>(env,binding-1);*slot=after;return fail(c,0,0,0);}
pub fn Cell joined(Cell a,Cell b){if(a.state==b.state){if(a.move_end==0 || b.move_end!=0 && b.move_start<a.move_start){return b;}return a;}let first=if(a.move_end==0 || b.move_end!=0 && b.move_start<a.move_start){b}else{a};return Cell{.state=3,.move_start=first.move_start,.move_end=first.move_end};}
pub fn m::Problem merge(ref m::Values body,ref v::Vec<Cell> base,ref v::Vec<Cell> incoming,ref mut v::Vec<Cell> target,bool first,usize start,usize end,ref mut v::Vec<Diagnostic> diagnostics,ref mut Context c){var usize i=0;while(i<v::len<Cell>(base)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let old=*v::at<Cell>(base,i);if(old.state!=0){let next=*v::at<Cell>(incoming,i);let prior=*v::at<Cell>(target,i);let result=if(first){next}else{joined(prior,next)};if(result.state==3 && old.state!=3 && prior.state!=3){let p=report(body,i+1,1104,start,end,result,0,0,diagnostics,c);if(p.code!=0){return p;}}let slot=v::at_mut<Cell>(target,i);*slot=result;}i+=1;}return fail(c,0,0,0);}

// Partial transfers retain the defining field location across provider modules.
pub fn m::Problem partial(ref m::Values body,m::Expression e,usize binding,usize field_module,usize field_start,usize field_end,ref mut v::Vec<Diagnostic> diagnostics,ref mut Context c){
    if(binding==0 || binding>m::local_count(body) || field_start>=field_end){return fail(c,12,e.start,e.end);}let n=m::local_at(body,binding);
    if(v::len<Diagnostic>(diagnostics)>=8192 || !m::charge(ref mut c.work,1)){return fail(c,10,e.start,e.end);}
    return if(v::push<Diagnostic>(diagnostics,Diagnostic{.code=1103,.frame=c.frame,.module_index=c.value.module_index,.start=e.start,.end=e.end,.declaration_start=n.start,.declaration_end=if(n.kind==10){c.cleanup_end}else{n.end},.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module,.field_start,.field_end}).is_ok){fail(c,0,0,0)}else{fail(c,11,e.start,e.end)};
}

// Cleanup requests record semantic exit points, not backend drop order.
// Reasons: scope=1, return=2, break=3, continue=4, replace=5,
// propagation-error=6, consumed is_ok=7, discarded temporary=8, match-arm=9.
pub fn m::Problem schedule(usize binding,usize type,u32 exit,usize start,usize end,ref mut v::Vec<Fact> facts,ref mut Context c){
    if(start>=end || type==0 || exit==0 || exit>9){return fail(c,12,start,end);}if(v::len<Fact>(facts)>=16384 || !m::charge(ref mut c.work,1)){return fail(c,10,start,end);}
    return if(v::push<Fact>(facts,Fact{.frame=c.frame,.binding,.kind=5,.before=1,.after=1,.start,.end,.move_start=0,.move_end=0,.type,.exit}).is_ok){fail(c,0,0,0)}else{fail(c,11,start,end)};
}
pub fn m::Problem retain_pending(usize type,ref mut Context c){if(c.pending_count>=1024 || !m::charge(ref mut c.work,1)){return fail(c,10,0,0);}let index=c.pending_count;let loop=c.loop;c.pending_types[index]=type;c.pending_loops[index]=loop;c.pending_count=index+1;return fail(c,0,0,0);}
// Partially evaluated fields/arguments are still owned by their surrounding
// expression. Only function exit, propagation failure or an exiting loop drops them.
pub fn m::Problem cleanup_pending(u32 exit,usize start,usize end,ref mut v::Vec<Fact> facts,ref mut Context c){var usize i=0;while(i<c.pending_count){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}if(exit==2 || exit==6 || (exit==3 || exit==4) && c.pending_loops[i]>=c.loop){let p=schedule(0,c.pending_types[i],exit,start,end,facts,c);if(p.code!=0){return p;}}i+=1;}return fail(c,0,0,0);}
pub fn m::Problem cleanup_live(ref m::Values body,ref v::Vec<Cell> env,u32 exit,usize start,usize end,ref mut v::Vec<Fact> facts,ref mut Context c){var usize i=0;while(i<v::len<Cell>(env)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let n=m::local_at(body,i+1);if(v::at<Cell>(env,i).state==1 && n.kind!=10){let p=schedule(i+1,n.type,exit,start,end,facts,c);if(p.code!=0){return p;}}i+=1;}return cleanup_pending(exit,start,end,facts,c);}
pub fn m::Problem cleanup_new(ref m::Values body,ref v::Vec<Cell> before,ref v::Vec<Cell> env,u32 exit,usize start,usize end,ref mut v::Vec<Fact> facts,ref mut Context c){var usize i=0;while(i<v::len<Cell>(env)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let n=m::local_at(body,i+1);if(v::at<Cell>(before,i).state==0 && v::at<Cell>(env,i).state==1 && n.kind!=10){let p=schedule(i+1,n.type,exit,start,end,facts,c);if(p.code!=0){return p;}}i+=1;}return fail(c,0,0,0);}
