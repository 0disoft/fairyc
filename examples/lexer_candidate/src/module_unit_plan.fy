module module_unit_plan;
import module_units as units;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_unit_bodies as bodies;
import module_unit_drops as drops;
import module_unit_names as names;
import module_signatures as signatures;
import module_ast_join as join;
import syntax as syntax;
import std::vec as v;

pub resource Plan {v::Vec<units::Unit> units;usize files;usize bytes;usize attempts;usize splits;usize work;usize binding_work;usize interface_work;usize body_work;usize drop_work;usize name_work;cleanup{}}
struct Budget {usize attempts;usize splits;usize work;usize binding_work;usize interface_work;usize body_work;usize drop_work;usize name_work;}
fn join::Problem fail(u32 code,usize module_index){return join::Problem{.code,.module_index,.start=0,.end=0};}
fn bool charge(ref mut Budget budget,usize amount){if(budget.work>1048576 || amount>1048576-budget.work){return false;}budget.work+=amount;return true;}
fn units::Usage measured(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){
    var units::Usage result=units::Usage{.bytes=stop-first-1,.declarations=0,.expressions=0,.statements=0,.entries=0,.types=0,.fields=0,.attributes=0,.arguments=0};var usize i=first;
    while(i<stop){let u=units::measured(v::at<syntax::Parsed>(sources,i));result.bytes+=u.bytes;result.declarations+=u.declarations;result.expressions+=u.expressions;result.statements+=u.statements;result.entries+=u.entries;result.types+=u.types;result.fields+=u.fields;result.attributes+=u.attributes;result.arguments+=u.arguments;i+=1;}return result;
}
fn join::Problem admit(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop,ref mut Budget budget){
    let bound=switch(binding::bind(sources,metadata,first,stop)){case .ok(b):{b}case .err(p):{return p;}};
    if(binding::work(ref bound)>budget.binding_work){budget.binding_work=binding::work(ref bound);}
    let imported=switch(interfaces::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{return p;}};
    if(interfaces::work(ref imported)>budget.interface_work){budget.interface_work=interfaces::work(ref imported);}
    let drop_plan=switch(drops::build(ref imported)){case .ok(p):{p}case .err(p):{return p;}};if(drops::work(ref drop_plan)>budget.drop_work){budget.drop_work=drops::work(ref drop_plan);}
    let owned=switch(bodies::build(sources,metadata,ref imported)){case .ok(p):{p}case .err(p):{return p;}};
    let resolved=switch(names::build(sources,metadata,ref imported,ref owned)){case .ok(p):{p}case .err(p):{return p;}};if(names::work(ref resolved)>budget.name_work){budget.name_work=names::work(ref resolved);}
    if(bodies::work(ref owned)>budget.body_work){budget.body_work=bodies::work(ref owned);}if(!bodies::fits(ref owned,units::measured(binding::parsed(ref bound)))){return fail(10,first);}return fail(0,0);
}
fn join::Problem refine(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop,ref mut v::Vec<units::Unit> ranges,ref mut Budget budget,usize depth){
    if(depth>10 || !charge(budget,stop-first+1)){return fail(10,first);}budget.attempts+=1;let problem=admit(sources,metadata,first,stop,budget);
    if(problem.code==0){if(!v::push<units::Unit>(ranges,units::Unit{.first,.stop,.usage=measured(sources,first,stop)}).is_ok){return fail(11,first);}return fail(0,0);}
    if((problem.code!=10 && problem.code!=4503) || stop-first==1){return problem;}
    budget.splits+=1;let usize middle=first+(stop-first)/2;let left=refine(sources,metadata,first,middle,ranges,budget,depth+1);if(left.code!=0){return left;}return refine(sources,metadata,middle,stop,ranges,budget,depth+1);
}
pub fn Result<Plan,join::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){
    if(v::len<syntax::Parsed>(sources)!=signatures::module_count(metadata)){return .err(fail(12,0));}var usize i=0;
    while(i<v::len<syntax::Parsed>(sources)){if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return .err(fail(12,i));}i+=1;}
    let initial=switch(units::build(sources)){case .ok(p):{p}case .err(p):{return .err(p);}};var v::Vec<units::Unit> ranges=v::empty<units::Unit>();var Budget budget=Budget{.attempts=0,.splits=0,.work=0,.binding_work=0,.interface_work=0,.body_work=0,.drop_work=0,.name_work=0};i=0;
    while(i<units::count(ref initial)){let u=units::at(ref initial,i);let p=refine(sources,metadata,u.first,u.stop,ref mut ranges,ref mut budget,0);if(p.code!=0){return .err(p);}i+=1;}
    return .ok(Plan{.units=ranges,.files=units::file_count(ref initial),.bytes=units::source_bytes(ref initial),.attempts=budget.attempts,.splits=budget.splits,.work=budget.work,.binding_work=budget.binding_work,.interface_work=budget.interface_work,.body_work=budget.body_work,.drop_work=budget.drop_work,.name_work=budget.name_work});
}
pub fn usize count(ref Plan p){return v::len<units::Unit>(ref p.units);}
pub fn units::Unit at(ref Plan p,usize index){return *v::at<units::Unit>(ref p.units,index);}
pub fn usize file_count(ref Plan p){return p.files;}
pub fn usize source_bytes(ref Plan p){return p.bytes;}
pub fn usize attempts(ref Plan p){return p.attempts;}
pub fn usize splits(ref Plan p){return p.splits;}
pub fn usize work(ref Plan p){return p.work;}
pub fn usize binding_work(ref Plan p){return p.binding_work;}
pub fn usize interface_work(ref Plan p){return p.interface_work;}
pub fn usize body_work(ref Plan p){return p.body_work;}
pub fn usize drop_work(ref Plan p){return p.drop_work;}
pub fn usize name_work(ref Plan p){return p.name_work;}
