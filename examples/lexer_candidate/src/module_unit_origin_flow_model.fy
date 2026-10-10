module module_unit_origin_flow_model;
import module_unit_origin_inputs as inputs;
import module_unit_origin_sets as sets;
import module_unit_value_model as m;
import std::vec as v;
import module_unit_origin_provenance as p;
import module_unit_loan_model as loan;
import module_unit_loan_facts as facts;

pub struct Context {usize place_root;bool return_mutable;bool returned_subview;usize statement;usize frame;m::Frame value;bool borrower;bool tagged_return;usize sites;usize returned;usize allocation;bool seen;usize work;usize loop_depth;usize probes;}
// kind 4 is a Region lifetime violation; allocation is an input site ID.
pub struct Unknown {u32 kind;usize after;usize frame;usize start;usize end;usize origin;usize allocation;}
pub struct Value {bool subview;usize origin;usize allocation;usize loan;}
pub fn Value value(Flow f){return Value{.subview=false,.loan=f.loan,.origin=f.origin,.allocation=f.allocation};}
pub struct Flow {usize origin;usize allocation;usize loan;usize outcomes;}
pub struct Returned {usize frame;usize origin;usize allocation;usize start;usize end;}
pub resource Edge {usize depth;u32 kind;v::Vec<Value> values;cleanup{}}
pub fn usize edge_depth(ref Edge e){return e.depth;}
pub fn u32 edge_kind(ref Edge e){return e.kind;}
pub fn ref v::Vec<Value> edge_values(ref Edge e){return ref e.values;}
pub fn m::Problem fail(ref Context c,u32 code,usize start,usize end){return m::fail(c.value,code,start,end,0,0);}
pub fn m::Problem unknown(ref mut v::Vec<Unknown> unknowns,ref mut Context c,u32 kind,usize start,usize end){if(c.probes!=0){return fail(c,0,0,0);}if(!m::charge(ref mut c.work,1) || v::len<Unknown>(unknowns)>=8192){return fail(c,10,start,end);}return if(v::push<Unknown>(unknowns,Unknown{.origin=0,.allocation=0,.kind,.after=c.sites,.frame=c.frame,.start,.end}).is_ok){fail(c,0,0,0)}else{fail(c,11,start,end)};}

pub fn Result<usize,m::Problem> join(ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut Context c,usize a,usize b,usize start,usize end){return switch(sets::join(entries,members,a,b,ref mut c.work)){case .ok(n):{.ok(n)}case .err(code):{.err(fail(c,code,start,end))}};}
pub fn Result<m::Expression,m::Problem> expression(ref m::Values body,usize original,ref mut Context c){
    var usize low=0;var usize high=m::expression_count(body);while(low<high){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}let usize mid=low+(high-low)/2;if(m::expression_at(body,mid+1).source_index<original){low=mid+1;}else{high=mid;}}
    if(low>=m::expression_count(body) || m::expression_at(body,low+1).source_index!=original){return .err(fail(c,12,0,0));}return .ok(m::expression_at(body,low+1));
}
pub fn Result<v::Vec<Value>,m::Problem> copy(ref v::Vec<Value> env,ref mut Context c,usize start,usize end){
    if(!m::charge(ref mut c.work,v::len<Value>(env)+1)){return .err(fail(c,10,start,end));}var v::Vec<Value> result=v::empty<Value>();var usize i=0;while(i<v::len<Value>(env)){if(!v::push<Value>(ref mut result,*v::at<Value>(env,i)).is_ok){return .err(fail(c,11,start,end));}i+=1;}return .ok(result);
}
pub fn m::Problem replace(ref mut v::Vec<Value> env,ref v::Vec<Value> source,ref mut Context c,usize start,usize end){
    if(v::len<Value>(env)!=v::len<Value>(source)){return fail(c,12,start,end);}if(!m::charge(ref mut c.work,v::len<Value>(env)+1)){return fail(c,10,start,end);}var usize i=0;while(i<v::len<Value>(env)){let slot=v::at_mut<Value>(env,i);*slot=*v::at<Value>(source,i);i+=1;}return fail(c,0,0,0);
}
// Only bindings live on entry survive the branch/block. A missing live binding
// is Unknown; zero is reserved for a binding outside the lexical environment.
pub fn m::Problem merge(ref v::Vec<Value> base,ref v::Vec<Value> source,ref mut v::Vec<Value> env,bool first,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<p::Row> allocations,ref mut Context c,usize start,usize end){
    if(v::len<Value>(base)!=v::len<Value>(env) || v::len<Value>(source)!=v::len<Value>(env)){return fail(c,12,start,end);}var usize i=0;while(i<v::len<Value>(env)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}var Value result=Value{.subview=false,.loan=0,.origin=0,.allocation=0};if(v::at<Value>(base,i).origin!=0){let actual=*v::at<Value>(source,i);let next=if(actual.origin==0){Value{.subview=false,.loan=0,.origin=1,.allocation=0}}else{actual};result=if(first){next}else{switch(join_value(entries,members,allocations,c,*v::at<Value>(env,i),next,start,end)){case .ok(n):{n}case .err(p):{return p;}}};}let slot=v::at_mut<Value>(env,i);*slot=result;i+=1;}return fail(c,0,0,0);
}
pub fn Result<bool,m::Problem> same(ref v::Vec<Value> a,ref v::Vec<Value> b,ref mut Context c,usize start,usize end){
    if(v::len<Value>(a)!=v::len<Value>(b)){return .err(fail(c,12,start,end));}var usize i=0;while(i<v::len<Value>(a)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,start,end));}if(v::at<Value>(a,i).origin!=v::at<Value>(b,i).origin || v::at<Value>(a,i).allocation!=v::at<Value>(b,i).allocation || v::at<Value>(a,i).loan!=v::at<Value>(b,i).loan){return .ok(false);}i+=1;}return .ok(true);
}
pub fn m::Problem capture(ref v::Vec<Value> env,u32 kind,ref mut v::Vec<Edge> edges,ref mut Context c,usize start,usize end){
    if(c.loop_depth==0){return fail(c,12,start,end);}if(v::len<Edge>(edges)>=16384){return fail(c,10,start,end);}let values=switch(copy(env,c,start,end)){case .ok(e):{e}case .err(p):{return p;}};return if(v::push<Edge>(edges,Edge{.depth=c.loop_depth,.kind,.values}).is_ok){fail(c,0,0,0)}else{fail(c,11,start,end)};
}
pub fn void drop_edges(ref mut v::Vec<Edge> edges,usize base){while(v::len<Edge>(edges)>base){discard switch(v::pop<Edge>(edges)){case .some(_):{}case .none:{assert(false);}};}}
pub fn Result<usize,m::Problem> leaf(ref inputs::Inputs input,m::Expression e,ref mut Context c){let id=switch(inputs::storage_owner(input,c.frame,e)){case .ok(n):{n}case .err(p):{return .err(p);}};return if(id==0){.ok(1)}else{switch(sets::owner(id)){case .ok(n):{.ok(n)}case .err(code):{.err(fail(c,code,e.start,e.end))}}};}
pub fn Result<Value,m::Problem> join_value(ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<p::Row> allocations,ref mut Context c,Value a,Value b,usize start,usize end){
    let origin=switch(join(entries,members,c,a.origin,b.origin,start,end)){case .ok(n):{n}case .err(p):{return .err(p);}};let usize allocation=if(origin<=1 || origin==sets::empty()){0}else{switch(p::join(allocations,a.allocation,b.allocation,ref mut c.work)){case .ok(n):{n}case .err(code):{return .err(fail(c,code,start,end));}}};let loan_set=switch(join(entries,members,c,a.loan,b.loan,start,end)){case .ok(n):{n}case .err(p):{return .err(p);}};return .ok(Value{.subview=a.subview || b.subview,.loan=loan_set,.origin,.allocation});
}
pub fn Result<Value,m::Problem> allocated(ref mut v::Vec<p::Row> allocations,ref mut Context c,usize origin,usize site,usize start,usize end){
    if(origin==0 || origin==1){return .ok(Value{.subview=false,.loan=0,.origin=1,.allocation=0});}let allocation=switch(p::single(allocations,origin,site,ref mut c.work)){case .ok(n):{n}case .err(code):{return .err(fail(c,code,start,end));}};return .ok(Value{.subview=false,.loan=0,.origin,.allocation});
}
pub fn m::Problem returned(Value value,ref inputs::Inputs input,ref mut v::Vec<loan::Loan> loan_records,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<p::Row> allocations,ref mut v::Vec<Returned> returns,ref mut Context c,usize start,usize end){
    if(!c.borrower || c.probes!=0){return fail(c,0,0,0);}var usize ordinal=0;while(ordinal<sets::count(entries,members,value.loan)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let id=sets::member(entries,members,value.loan,ordinal);if(id<3 || id-2>v::len<loan::Loan>(loan_records)){return fail(c,12,start,end);}c.returned_subview=c.returned_subview || v::at<loan::Loan>(loan_records,id-3).subview;ordinal+=1;}let escaped=facts::escaping(value,input,entries,members,loan_records,c,start,end);if(escaped.code!=0){return escaped;}if(!m::charge(ref mut c.work,1) || v::len<Returned>(returns)>=8192){return fail(c,10,start,end);}let actual=if(value.origin==0 || value.origin==sets::empty() && !c.tagged_return){Value{.subview=false,.loan=0,.origin=1,.allocation=0}}else{value};if(!v::push<Returned>(returns,Returned{.frame=c.frame,.origin=actual.origin,.allocation=actual.allocation,.start,.end}).is_ok){return fail(c,11,start,end);}c.sites+=1;c.seen=true;let result=switch(join_value(entries,members,allocations,c,Value{.subview=false,.loan=0,.origin=c.returned,.allocation=c.allocation},actual,start,end)){case .ok(n):{n}case .err(p):{return p;}};c.returned=result.origin;c.allocation=result.allocation;return fail(c,0,0,0);
}

pub fn m::Problem expire_values(ref mut v::Vec<Value> env,ref mut v::Vec<p::Row> allocations,usize leaf,ref mut Context c,usize start,usize end){
    var usize i=0;while(i<v::len<Value>(env)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let old=*v::at<Value>(env,i);let map=switch(p::expire(allocations,old.allocation,leaf,ref mut c.work)){case .ok(n):{n}case .err(code):{return fail(c,code,start,end);}};let slot=v::at_mut<Value>(env,i);slot.allocation=map;i+=1;}return fail(c,0,0,0);
}
fn m::Problem expired_edge(ref Edge source,ref mut v::Vec<Edge> pending,ref mut v::Vec<p::Row> allocations,usize leaf,ref mut Context c,usize start,usize end){
    var v::Vec<Value> values=v::empty<Value>();var usize i=0;while(i<v::len<Value>(ref source.values)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}if(!v::push<Value>(ref mut values,*v::at<Value>(ref source.values,i)).is_ok){return fail(c,11,start,end);}i+=1;}let problem=expire_values(ref mut values,allocations,leaf,c,start,end);if(problem.code!=0){return problem;}return if(v::push<Edge>(pending,Edge{.depth=source.depth,.kind=source.kind,.values}).is_ok){fail(c,0,0,0)}else{fail(c,11,start,end)};
}
pub fn m::Problem expire_edges(ref mut v::Vec<Edge> edges,usize first,ref mut v::Vec<p::Row> allocations,usize leaf,ref mut Context c,usize start,usize end){
    if(first>v::len<Edge>(edges)){return fail(c,12,start,end);}var v::Vec<Edge> pending=v::empty<Edge>();while(v::len<Edge>(edges)>first){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let problem=switch(v::pop<Edge>(edges)){case .some(edge):{expired_edge(ref edge,ref mut pending,allocations,leaf,c,start,end)}case .none:{fail(c,12,start,end)}};if(problem.code!=0){return problem;}}
    while(v::len<Edge>(ref pending)>0){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let problem=switch(v::pop<Edge>(ref mut pending)){case .some(edge):{if(v::push<Edge>(edges,edge).is_ok){fail(c,0,0,0)}else{fail(c,11,start,end)}}case .none:{fail(c,12,start,end)}};if(problem.code!=0){return problem;}}return fail(c,0,0,0);
}
pub fn m::Problem check_lifetime(ref inputs::Inputs input,Value value,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref v::Vec<p::Row> allocations,ref mut v::Vec<Unknown> unknowns,ref mut Context c,usize start,usize end){
    if(c.probes!=0 || value.origin<=2 || value.origin==sets::empty()){return fail(c,0,0,0);}var usize i=0;while(i<sets::count(entries,members,value.origin)){if(!m::charge(ref mut c.work,1)){return fail(c,10,start,end);}let leaf=sets::member(entries,members,value.origin,i);if(leaf==2){i+=1;continue;}let owner=sets::owner_id(leaf);if(owner==0 || owner>inputs::owner_count(input)){return fail(c,12,start,end);}let source=inputs::owner(input,owner);if(source.kind==14){let expired=switch(p::expired(allocations,value.allocation,leaf,ref mut c.work)){case .ok(b):{b}case .err(code):{return fail(c,code,start,end);}};if(expired || source.frame!=c.frame || start<source.start || end>source.end){let allocation=switch(p::lookup(allocations,value.allocation,leaf,ref mut c.work)){case .ok(n):{n}case .err(code):{return fail(c,code,start,end);}};if(allocation==0 || allocation>inputs::allocation_count(input)){return fail(c,12,start,end);}if(v::len<Unknown>(unknowns)>=8192 || !m::charge(ref mut c.work,1)){return fail(c,10,start,end);}if(!v::push<Unknown>(unknowns,Unknown{.kind=4,.after=c.sites,.frame=c.frame,.start,.end,.origin=leaf,.allocation}).is_ok){return fail(c,11,start,end);}}}i+=1;}return fail(c,0,0,0);
}

pub fn Context context(usize frame,m::Frame value,bool borrower,bool tagged_return,usize work){return Context{.place_root=0,.return_mutable=false,.returned_subview=false,.statement=0,.frame,.value,.borrower,.tagged_return,.sites=0,.returned=0,.allocation=0,.seen=false,.work,.loop_depth=0,.probes=0};}
