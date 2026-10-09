module module_unit_origin_flow_model;
import module_unit_origin_inputs as inputs;
import module_unit_origin_sets as sets;
import module_unit_value_model as m;
import std::vec as v;

pub struct Context {usize frame;m::Frame value;bool borrower;usize returned;bool seen;usize work;usize loop_depth;usize probes;}
pub struct Flow {usize origin;usize outcomes;}
pub struct Returned {usize frame;usize origin;usize start;usize end;}
pub resource Edge {usize depth;u32 kind;v::Vec<usize> values;cleanup{}}
pub fn usize edge_depth(ref Edge e){return e.depth;}
pub fn u32 edge_kind(ref Edge e){return e.kind;}
pub fn ref v::Vec<usize> edge_values(ref Edge e){return ref e.values;}
pub fn m::Problem fail(ref Context c,u32 code,usize start,usize end){return m::fail(c.value,code,start,end,0,0);}
pub fn Result<usize,m::Problem> join(ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut Context c,usize a,usize b,usize start,usize end){return switch(sets::join(entries,members,a,b,ref mut c.work)){case .ok(n):{.ok(n)}case .err(code):{.err(fail(c,code,start,end))}};}
pub fn Result<m::Expression,m::Problem> expression(ref m::Values body,usize original,ref mut Context c){
    var usize low=0;var usize high=m::expression_count(body);while(low<high){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}let usize mid=low+(high-low)/2;if(m::expression_at(body,mid+1).source_index<original){low=mid+1;}else{high=mid;}}
    if(low>=m::expression_count(body) || m::expression_at(body,low+1).source_index!=original){return .err(fail(c,12,0,0));}return .ok(m::expression_at(body,low+1));
}
pub fn Result<v::Vec<usize>,m::Problem> copy(ref v::Vec<usize> env,ref mut Context c,usize start,usize end){
    if(!m::charge(ref mut c.work,v::len<usize>(env)+1)){return .err(fail(c,10,start,end));}var v::Vec<usize> result=v::empty<usize>();var usize i=0;while(i<v::len<usize>(env)){if(!v::push<usize>(ref mut result,*v::at<usize>(env,i)).is_ok){return .err(fail(c,11,start,end));}i+=1;}return .ok(result);
}
pub fn m::Problem replace(ref mut v::Vec<usize> env,ref v::Vec<usize> source,ref mut Context c,usize start,usize end){
    if(v::len<usize>(env)!=v::len<usize>(source)){return fail(c,12,start,end);}if(!m::charge(ref mut c.work,v::len<usize>(env)+1)){return fail(c,10,start,end);}var usize i=0;while(i<v::len<usize>(env)){let slot=v::at_mut<usize>(env,i);*slot=*v::at<usize>(source,i);i+=1;}return fail(c,0,0,0);
}
// Only bindings live on entry survive the branch/block. A missing live binding
// is Unknown; zero is reserved for a binding outside the lexical environment.
pub fn m::Problem merge(ref v::Vec<usize> base,ref v::Vec<usize> source,ref mut v::Vec<usize> env,bool first,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut Context c,usize start,usize end){
    if(v::len<usize>(base)!=v::len<usize>(env) || v::len<usize>(source)!=v::len<usize>(env)){return fail(c,12,start,end);}var usize i=0;while(i<v::len<usize>(env)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}var usize result=0;if(*v::at<usize>(base,i)!=0){let usize actual=*v::at<usize>(source,i);let usize next=if(actual==0){1}else{actual};result=if(first){next}else{switch(join(entries,members,c,*v::at<usize>(env,i),next,start,end)){case .ok(n):{n}case .err(p):{return p;}}};}let slot=v::at_mut<usize>(env,i);*slot=result;i+=1;}return fail(c,0,0,0);
}
pub fn Result<bool,m::Problem> same(ref v::Vec<usize> a,ref v::Vec<usize> b,ref mut Context c,usize start,usize end){
    if(v::len<usize>(a)!=v::len<usize>(b)){return .err(fail(c,12,start,end));}var usize i=0;while(i<v::len<usize>(a)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,start,end));}if(*v::at<usize>(a,i)!=*v::at<usize>(b,i)){return .ok(false);}i+=1;}return .ok(true);
}
pub fn m::Problem capture(ref v::Vec<usize> env,u32 kind,ref mut v::Vec<Edge> edges,ref mut Context c,usize start,usize end){
    if(c.loop_depth==0){return fail(c,12,start,end);}if(v::len<Edge>(edges)>=16384){return fail(c,10,start,end);}let values=switch(copy(env,c,start,end)){case .ok(e):{e}case .err(p):{return p;}};return if(v::push<Edge>(edges,Edge{.depth=c.loop_depth,.kind,.values}).is_ok){fail(c,0,0,0)}else{fail(c,11,start,end)};
}
pub fn void drop_edges(ref mut v::Vec<Edge> edges,usize base){while(v::len<Edge>(edges)>base){discard switch(v::pop<Edge>(edges)){case .some(_):{}case .none:{assert(false);}};}}
pub fn Result<usize,m::Problem> leaf(ref inputs::Inputs input,m::Expression e,ref mut Context c){let id=switch(inputs::binding_owner(input,c.frame,e)){case .ok(n):{n}case .err(p):{return .err(p);}};return if(id==0){.ok(1)}else{switch(sets::owner(id)){case .ok(n):{.ok(n)}case .err(code):{.err(fail(c,code,e.start,e.end))}}};}
pub fn m::Problem returned(usize origin,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<Returned> returns,ref mut Context c,usize start,usize end){
    if(!c.borrower || c.probes!=0){return fail(c,0,0,0);}if(!m::charge(ref mut c.work,1) || v::len<Returned>(returns)>=8192){return fail(c,10,start,end);}let usize actual=if(origin==0){1}else{origin};if(!v::push<Returned>(returns,Returned{.frame=c.frame,.origin=actual,.start,.end}).is_ok){return fail(c,11,start,end);}c.seen=true;c.returned=switch(join(entries,members,c,c.returned,actual,start,end)){case .ok(n):{n}case .err(p):{return p;}};return fail(c,0,0,0);
}
