module tests::module_project_diagnostics;
import module_project_cli as cli;
import module_project_compile as compile;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@test
fn void project_failure_retains_checked_type_details(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn i32 main(){return true;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let expected=switch(compile::build(ref sources,ref metadata)){case .ok(_):{assert(false);return;}case .err(e):{e}};
    assert(expected.expected!=0 && expected.actual!=0 && expected.expected!=expected.actual);
    switch(cli::build(ref sources)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==2 && e.stage==expected.stage && e.code==expected.code && e.module_index==expected.module_index && e.start==expected.start && e.end==expected.end && e.expected==expected.expected && e.actual==expected.actual);}};
}
@test
fn void nonchecking_failures_keep_zero_type_and_limit_details(){
    let empty=v::empty<syntax::Parsed>();switch(cli::build(ref empty)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==1 && e.code==1001 && e.expected==0 && e.actual==0);}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn void main(){}")).is_ok);
    switch(cli::build(ref sources)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==5 && e.code==1001 && e.expected==0 && e.actual==0);}};
}
