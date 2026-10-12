module module_unit_constants;
import module_constant_engine as engine;
import module_unit_value_model as m;
import module_signatures as signatures;
import syntax as syntax;
import std::vec as v;

// The value/lookup boundary stays separate from evaluator-owned graph state.
pub struct Value{u32 scalar;usize bits;}
pub struct Entry{usize module_index;usize declaration;usize start;usize end;Value value;}
pub resource Plan{engine::Plan inner;cleanup{}}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources){return switch(engine::build(sources)){case .ok(p):{.ok(Plan{.inner=p})}case .err(p):{.err(p)}};}
pub fn Result<Plan,m::Problem> build_owned(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners){return switch(engine::build_owned(sources,owners)){case .ok(p):{.ok(Plan{.inner=p})}case .err(p):{.err(p)}};}
pub fn Result<Plan,m::Problem> build_project(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){return switch(engine::build_project(sources,metadata)){case .ok(p):{.ok(Plan{.inner=p})}case .err(p):{.err(p)}};}
pub fn Result<Value,m::Problem> get(ref Plan p,usize module_index,usize declaration,ref mut usize work){return switch(engine::get(ref p.inner,module_index,declaration,work)){case .ok(v):{.ok(Value{.scalar=v.scalar,.bits=v.bits})}case .err(p):{.err(p)}};}
pub fn usize work(ref Plan p){return engine::work(ref p.inner);}
