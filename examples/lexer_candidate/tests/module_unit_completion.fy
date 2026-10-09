module tests::module_unit_completion;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import module_unit_value_completion as completion;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let bound=switch(binding::bind(sources,ref metadata,0,1)){case .ok(b):{b}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources);}
@effects(console) fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{discard io::write("unexpected success\n");assert(false);}case .err(p):{if(p.code!=code || p.module_index!=0 || !bytes::equal(text[p.start..p.end],at)){discard io::write_u32(p.code);discard io::write(" at ");discard io::write_bytes(text[p.start..p.end]);discard io::write("\n");assert(false);}}};}

fn void completed(ref values::Plan p){var usize i=0;while(i<values::count(p)){assert(model::checked_completion(values::at(p,i)));i+=1;}assert(values::unresolved(p)==0 && values::work(p)<=1048576);}
@test fn void ordinary_returns_branch_returns_unsafe_and_literal_traps_complete(){
    let p=switch(checked(b"module main;fn i64 choose(bool flag){if(flag){return 4000000000;}else if(false){assert(false);}else{return 4000000001;};}fn i64 main(){unsafe{return choose(true);};}")){case .ok(p):{p}case .err(_):{assert(false);return;}};completed(ref p);assert(values::checked_returns(ref p)==3);
    let trap=switch(checked(b"module main;fn i32 main(){assert(false);}")){case .ok(p):{p}case .err(_):{assert(false);return;}};completed(ref trap);assert(values::checked_returns(ref trap)==0);
    let tails=switch(checked(b"module main;fn i32 main(){if(true){1}else{2};}")){case .ok(p):{p}case .err(_):{assert(false);return;}};completed(ref tails);
    let source=parsed(b"module main;fn i32 main(){assert(false);}");var usize work=1048576;switch(completion::check(ref source,model::frame(values::at(ref trap,0)),false,ref mut work)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==10 && p.module_index==0 && p.start==25 && p.end==41 && work==1048576);}};
}
@test fn void inferred_void_values_and_aliases_keep_canonical_identity(){
    let p=switch(checked(b"module main;fn void empty(){}fn void main(){let n=empty();let alias=n;discard alias;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};completed(ref p);assert(values::count(ref p)==2 && values::inferred(ref p)==2 && values::checked_calls(ref p)==1);let c=values::at(ref p,1);assert(model::local_count(c)==2);let local=model::local_at(c,1);let alias=model::local_at(c,2);assert(local.expected==0 && alias.expected==0 && local.type==alias.type && values::type_at(ref p,local.type).kind==1 && values::type_at(ref p,local.type).scalar==7 && model::checked_consumers(c)==1);
}
@test @effects(console) fn void missing_returns_keep_reference_body_spans(){
    failure(b"module main;fn i32 main(){}",2,b"{}");
    failure(b"module main;fn i32 main(){let n=1;}",2,b"{let n=1;}");
    failure(b"module main;fn i32 main(){while(true){return 1;}}",2,b"{while(true){return 1;}}");
    failure(b"module main;fn i32 main(){if(true){return 1;};}",2,b"{if(true){return 1;};}");
    failure(b"module main;fn i32 main(){assert((false));}",2,b"{assert((false));}");
    failure(b"module main;fn i32 main(){{return 1;};}",2,b"{{return 1;};}");
    failure(b"module main;enum Flag{A;B;}fn i32 main(){let Flag f=.A;switch(f){case .A:{return 1;}case .B:{return 2;}};}",2,b"{let Flag f=.A;switch(f){case .A:{return 1;}case .B:{return 2;}};}");
    failure(b"module main;fn void main(){return 1;}",1009,b"1");
}
@test fn void function_bodies_keep_the_statement_block_grammar(){
    let input=switch(bytes::copy(b"module main;fn i32 main(){1}")){case .ok(b):{b}case .err(_):{assert(false);return;}};let tokens=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return;}};let p=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref p)!=0);
}
