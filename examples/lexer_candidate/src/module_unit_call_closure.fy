module module_unit_call_closure;
import module_unit_interfaces as interfaces;
import module_unit_bodies as bodies;
import module_unit_names as names;
import module_unit_ordinary_names as ordinary;
import module_unit_calls as calls;
import module_unit_bind as binding;
import module_unit_body_ast as ast;
import module_unit_drops as drops;
import module_units as units;
import module_signatures as signatures;
import module_ast_join as join;
import syntax as syntax;
import std::vec as v;

pub resource Ready {interfaces::Interfaces imported;bodies::Plan owned;names::Plan concrete;ordinary::Plan original;calls::Plan calls;drops::Plan drops;usize passes;usize work;cleanup{}}
fn join::Problem fail(u32 code,usize module_index){return join::Problem{.code,.module_index,.start=0,.end=0};}
fn Result<Ready,join::Problem> collect(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref v::Vec<binding::Coordinate> seeds,usize passes,bool unit_only){
    let imported=switch(if(unit_only){interfaces::build_unit_seeded(sources,metadata,bound,seeds)}else{interfaces::build_seeded(sources,metadata,bound,seeds)}){case .ok(p):{p}case .err(p):{return .err(p);}};
    let owned=switch(bodies::build(sources,metadata,ref imported)){case .ok(p):{p}case .err(p):{return .err(p);}};if(!bodies::fits(ref owned,units::measured(binding::parsed(bound)))){return .err(fail(10,binding::first(bound)));}
    let concrete=switch(names::build(sources,metadata,ref imported,ref owned)){case .ok(p):{p}case .err(p):{return .err(p);}};
    let original=switch(ordinary::build(sources,metadata,bound,ref imported)){case .ok(p):{p}case .err(p):{return .err(p);}};
    if(names::work(ref concrete)+ordinary::work(ref original)>1048576 || names::symbol_count(ref concrete)>16384-ordinary::symbols(ref original) || names::reference_count(ref concrete)>16384-ordinary::references(ref original)){return .err(fail(10,binding::first(bound)));}
    let typed=switch(calls::build(sources,metadata,ref imported,ref owned,ref concrete,ref original)){case .ok(p):{p}case .err(p):{return .err(p);}};
    let dropped=switch(drops::build(ref imported)){case .ok(p):{p}case .err(p):{return .err(p);}};
    return .ok(Ready{.imported,.owned,.concrete,.original,.calls=typed,.drops=dropped,.passes,.work=0});
}
fn Result<Ready,join::Problem> build_mode(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,bool unit_only){
    var v::Vec<binding::Coordinate> seeds=v::empty<binding::Coordinate>();var usize passes=0;var usize work=0;
    while(passes<64){passes+=1;var next=switch(collect(sources,metadata,bound,ref seeds,passes,unit_only)){case .ok(p):{p}case .err(p):{return .err(p);}};
        if(!ast::charge(ref mut work,calls::work(ref next.calls)+1)){return .err(fail(10,binding::first(bound)));}
        if(calls::target_count(ref next.calls)==0){next.work=work;return .ok(next);}
        var usize i=0;while(i<calls::target_count(ref next.calls)){if(!ast::charge(ref mut work,1) || v::len<binding::Coordinate>(ref seeds)>=512){return .err(fail(10,binding::first(bound)));}if(!v::push<binding::Coordinate>(ref mut seeds,calls::target_at(ref next.calls,i)).is_ok){return .err(fail(11,binding::first(bound)));}i+=1;}
    }return .err(fail(10,binding::first(bound)));
}
pub fn Result<Ready,join::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound){return build_mode(sources,metadata,bound,false);}
pub fn Result<Ready,join::Problem> build_unit(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound){return build_mode(sources,metadata,bound,true);}
pub fn ref interfaces::Interfaces imported(ref Ready p){return ref p.imported;}
pub fn ref bodies::Plan owned(ref Ready p){return ref p.owned;}
pub fn ref names::Plan resolved_names(ref Ready p){return ref p.concrete;}
pub fn ref ordinary::Plan original_names(ref Ready p){return ref p.original;}
pub fn ref calls::Plan call_plan(ref Ready p){return ref p.calls;}
pub fn ref drops::Plan drop_plan(ref Ready p){return ref p.drops;}
pub fn usize passes(ref Ready p){return p.passes;}
pub fn usize work(ref Ready p){return p.work;}
