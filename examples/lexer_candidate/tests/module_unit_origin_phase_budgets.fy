module tests::module_unit_origin_phase_budgets;
import module_unit_origin_returns as origins;
import module_unit_origin_inputs as inputs;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

import module_unit_origin_diagnostics as diagnostics;
@test fn void completed_summary_history_does_not_exhaust_final_checks_or_publication(){
    let text=b"module main;fn ref i32 identity(ref i32 n){return n;}";
    let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let source=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref source)==0);
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,source).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let ready=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let primary=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let baseline=switch(origins::build(ref sources,ref metadata,ref primary)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let cost=origins::summary_work(ref baseline);assert(cost>0 && cost<1048576 && origins::work(ref baseline)>cost+1);
    let plan=switch(origins::build_budgeted(ref sources,ref metadata,ref primary,1048576-cost-1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(origins::summary_work(ref plan)==1048575 && origins::work(ref plan)>1048576 && origins::work(ref plan)<2097152);
    let report=switch(diagnostics::collect(ref plan)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(diagnostics::exit_code(ref report)==0 && diagnostics::work(ref report)>origins::work(ref plan) && diagnostics::work(ref report)<3145728);
    switch(diagnostics::collect_budgeted(ref plan,1048576)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10);}};
}
