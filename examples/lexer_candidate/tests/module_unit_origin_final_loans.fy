module tests::module_unit_origin_final_loans;
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

@test fn void scalar_callers_are_checked_after_transitive_return_summaries_converge(){
    let text=b"module main;fn ref i32 first(ref i32 n){return second(n);}fn ref i32 second(ref i32 n){return third(n);}fn ref i32 third(ref i32 n){return n;}fn i32 main(){var i32 n=0;let ref i32 view=first(ref n);n=1;return *view;}";
    let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let source=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref source)==0);
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,source).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let ready=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let primary=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let plan=switch(origins::build(ref sources,ref metadata,ref primary)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(origins::rounds(ref plan)>=4 && origins::rounds(ref plan)<=64 && origins::work(ref plan)<1048576);
    var bool conflict=false;var usize i=0;while(i<origins::conflict_count(ref plan)){let c=origins::conflict_at(ref plan,i);if(c.code==2101){let f=inputs::frame(origins::owned_inputs(ref plan),c.frame);if(f.value.declaration==5 && bytes::equal(text[c.start..c.end],b"n")){conflict=true;}}i+=1;}assert(conflict);
}
