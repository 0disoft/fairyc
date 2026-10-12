module tests::module_const_eval_calls;
import module_unit_constants as constants;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(bytes::Buffer source){let s=switch(scanner::collect(source,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}
fn constants::Plan plan(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return plan(b"module empty;");}};assert(v::push<syntax::Parsed>(ref mut sources,parsed(b)).is_ok);return switch(constants::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);plan(b"module empty;")}};}
fn void rejected(ref u8[] text,u32 code,ref u8[] span){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(v::push<syntax::Parsed>(ref mut sources,parsed(b)).is_ok);switch(constants::build(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && bytes::equal(text[p.start..p.end],span));}};}
fn constants::Value value(ref constants::Plan p,usize declaration){var usize work=0;return switch(constants::get(p,0,declaration,ref mut work)){case .ok(v):{v}case .err(_):{assert(false);constants::Value{.scalar=0,.bits=0}}};}
@test fn void calls_bind_parameters_shadow_constants_and_keep_distinct_arguments(){
    let p=plan(b"module main;const i32 n=99;@const_eval fn i32 square(i32 n){return n*n;}@const_eval fn i32 sum(i32 a,i32 b){return a+b;}@const_eval fn i32 twice(i32 n){return sum(n,n);}const i32 NESTED=twice(sum(1,2));const i32 DISTINCT=square(2)+square(3);const i32 LABELED=sum(.a=40,.b=2);static_assert(NESTED==6 && DISTINCT==13 && LABELED==42);");
    assert(value(ref p,6).bits==6 && value(ref p,7).bits==13 && value(ref p,8).bits==42);
}
@test fn void conditionals_casts_and_unused_traps_fold_only_selected_values(){
    let p=plan(b"module main;@const_eval fn i32 square(i32 n){return n*n;}@const_eval fn u32 widen(u8 n){return n as u32;}@const_eval fn i32 trap(){return 1/0;}const i32 ANSWER=square(6)+6;const u8 BYTE=40+2;const u32 WIDE=widen(BYTE);const bool READY=false || (ANSWER==42 && WIDE==42);const i32 SELECTED=if(READY){ANSWER}else{trap()};const i64 NEG=(-2147483648) as i64;const i64 HIGH=(WIDE+4294967253) as i64;const bool SAFE=true || (trap()==0);static_assert(READY && SELECTED==42 && SAFE);static_assert(NEG<0 && HIGH==4294967295);");
    assert(value(ref p,9).bits==42 && value(ref p,10).scalar==2 && value(ref p,10).bits==18446744071562067968 && value(ref p,11).bits==4294967295 && value(ref p,12).bits==1);
}
@test fn void combined_cycles_excluded_calls_and_argument_types_keep_source_spans(){
    rejected(b"module main;const i32 X=read();@const_eval fn i32 read(){return X;}",4304,b"X");
    rejected(b"module main;@const_eval fn i32 first(){return second();}@const_eval fn i32 second(){return first();}",4304,b"first");
    rejected(b"module main;@const_eval fn i32 loop(){return if(false){loop()}else{0};}",4304,b"loop");
    rejected(b"module main;@const_eval fn i32 bad(){return ordinary();}fn i32 ordinary(){return 0;}",4302,b"ordinary()");
    rejected(b"module main;@const_eval fn i32 take(i32 n){return n;}const i32 X=take();",1010,b"take()");
    rejected(b"module main;@const_eval fn i32 take(i32 n){return n;}const u32 WRONG=0;const i32 X=take(WRONG);",1006,b"WRONG");
    rejected(b"module main;@const_eval fn i32 take(i32 n){return n;}const i32 X=take(.wrong=0);",1014,b".wrong=0");
    rejected(b"module main;@const_eval fn i32 bad(){return source_location().len as i32;}",4302,b"source_location().len");
    rejected(b"module main;const bool X=if(true){true}else{1};",1006,b"if(true){true}else{1}");
    rejected(b"module main;const i32 X=if(1){0}else{1};",1007,b"1");
}
fn bool name(ref mut bytes::Buffer source,usize index){var usize remaining=index;var u8 head=97;var u8 tail=97;while(remaining>=26){head+=1;remaining-=26;}while(remaining!=0){tail+=1;remaining-=1;}return bytes::append(source,b"f").is_ok && bytes::append_byte(source,head).is_ok && bytes::append_byte(source,tail).is_ok;}
fn bytes::Buffer chain(usize count){var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module main;").is_ok);var usize i=0;while(i<count){assert(bytes::append(ref mut source,b"@const_eval fn i32 ").is_ok && name(ref mut source,i) && bytes::append(ref mut source,b"(){return ").is_ok);if(i+1<count){assert(name(ref mut source,i+1) && bytes::append(ref mut source,b"();}").is_ok);}else{assert(bytes::append(ref mut source,b"42;}").is_ok);}i+=1;}assert(bytes::append(ref mut source,b"const i32 ANSWER=faa();static_assert(ANSWER==42);").is_ok);return source;}
@test fn void exact_call_depth_boundary_counts_calls_not_constant_references(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(chain(32))).is_ok);let p=switch(constants::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(value(ref p,34).bits==42);
    var v::Vec<syntax::Parsed> too_deep=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut too_deep,parsed(chain(33))).is_ok);switch(constants::build(ref too_deep)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4304 && p.start<p.end);}};
}
