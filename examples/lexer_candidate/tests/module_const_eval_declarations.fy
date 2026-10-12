module tests::module_const_eval_declarations;
import module_const_eval_declarations as declarations;
import module_unit_constants as constants;
import module_signatures as signatures;
import effect_declarations as effects;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn void rejected(ref u8[] source,ref u8[] span){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(source)).is_ok);
    let first=declarations::read(v::at<syntax::Parsed>(ref sources,0),1);assert(first.code==4301 && bytes::equal(source[first.start..first.end],span));
    switch(signatures::build(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4301 && bytes::equal(source[p.start..p.end],span));}};
    switch(constants::build(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4301 && bytes::equal(source[p.start..p.end],span));}};
}
@test fn void eligible_scalar_functions_retain_return_and_parameter_ids(){
    let source=parsed(b"module main;@must_use @const_eval fn i64 calculate(i64 n,u8 a,u32 b,bool ready,i32 c){return n;}fn i32 ordinary(){return 0;}");
    let info=declarations::read(ref source,1);assert(info.annotated && info.code==0 && info.result==2 && info.parameters==5 && info.expression!=0);
    assert(bytes::equal(syntax::expression_text(ref source,info.expression),b"n"));let contract=effects::read(ref source,1);assert(contract.code==0 && contract.bits==0 && contract.must_use);
    let plain=declarations::read(ref source,2);assert(!plain.annotated && plain.code==0);
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,source).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(signatures::count(ref metadata)==2);
    // A valid unused function is validated but not executed with invented inputs.
    let checked=switch(constants::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(constants::work(ref checked)>0);
}
@test fn void invalid_arity_duplicates_and_misplacement_report_attribute_spans(){
    rejected(b"module main;@const_eval(1) fn i32 bad(){return 0;}",b"@const_eval(1)");
    rejected(b"module main;@const_eval @const_eval fn i32 bad(){return 0;}",b"@const_eval");
    rejected(b"module main;@const_eval struct Bad{i32 value;}",b"@const_eval");
    rejected(b"module main;@const_eval enum Bad{Value;}",b"@const_eval");
    rejected(b"module main;@const_eval resource Bad{cleanup{}}",b"@const_eval");
    rejected(b"module main;@const_eval unsafe extern C fn i32 bad();",b"@const_eval");
    rejected(b"module main;resource Bad{@const_eval cleanup{}}",b"@const_eval");
    rejected(b"module main;@const_eval const i32 BAD=0;",b"@const_eval");
}
@test fn void invalid_visibility_signature_effects_and_body_report_function_names(){
    rejected(b"module main;@const_eval pub fn i32 bad(){return 0;}",b"bad");
    rejected(b"module main;@const_eval unsafe fn i32 bad(){return 0;}",b"bad");
    rejected(b"module main;@const_eval fn usize bad(){return 0;}",b"bad");
    rejected(b"module main;@const_eval fn i32 bad(ref i32 n){return 0;}",b"bad");
    rejected(b"module main;@const_eval @effects(console) fn i32 bad(){return 0;}",b"bad");
    rejected(b"module main;@const_eval fn i32 bad(){let i32 n=0;return n;}",b"bad");
    rejected(b"module main;@const_eval fn i32 bad(){return 0;return 1;}",b"bad");
    rejected(b"module main;@const_eval fn void bad(){return;}",b"bad");
    rejected(b"module main;@const_eval fn i32 bad(){while(false){}return 0;}",b"bad");
}
