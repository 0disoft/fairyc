module tests::module_unit_cleanup_operands;
import module_unit_moves as moves;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
// Every returned request is observed after the parser, declarations and typed
// values are destroyed. Operand IDs belong to their explicit immutable frame.
fn moves::Plan retained(ref u8[] text){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained(text);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return retained(text);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return retained(text);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return retained(text);}};
    return switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(_):{assert(false);retained(text)}};
}
@test fn void repeated_partial_operands_keep_distinct_expression_identity(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}resource Pair{R a;R b;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Pair,i32> f(){return .ok(Pair{.a=make(),.b=make(),.n=number()?});}");
    assert(moves::count(ref p)==0 && moves::cleanup_count(ref p)==2);let a=moves::cleanup_at(ref p,0);let b=moves::cleanup_at(ref p,1);
    assert(a.binding==0 && b.binding==0 && a.expression!=0 && b.expression!=0 && a.expression!=b.expression);
    assert(a.frame==b.frame && a.type==b.type && a.exit==6 && b.exit==6 && a.start==b.start && a.end==b.end);
}
@test fn void roots_and_consumed_temporary_operands_are_unambiguous(){
    let root=retained(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void f(){let R value=make();}");assert(moves::cleanup_count(ref root)==1);let a=moves::cleanup_at(ref root,0);assert(a.binding!=0 && a.expression==0 && a.exit==1);
    let temporary=retained(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void f(){discard make();}");assert(moves::cleanup_count(ref temporary)==1);let b=moves::cleanup_at(ref temporary,0);assert(b.binding==0 && b.expression!=0 && b.exit==8);
    let tagged=retained(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<R,i32> boxed(){return .ok(make());}fn void f(){discard boxed().is_ok;}");
    assert(moves::cleanup_count(ref tagged)==1);let c=moves::cleanup_at(ref tagged,0);assert(c.binding==0 && c.expression!=0 && c.exit==7);
}
@test fn void generic_clones_qualify_same_expression_ids_by_frame(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}resource Pair{R a;R b;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Pair,i32> f<T>(){return .ok(Pair{.a=make(),.b=make(),.n=number()?});}fn Result<Pair,i32> one(){return f<i32>();}fn Result<Pair,i32> two(){return f<i64>();}");
    assert(moves::count(ref p)==0 && moves::cleanup_count(ref p)==4);let a=moves::cleanup_at(ref p,0);let b=moves::cleanup_at(ref p,1);let c=moves::cleanup_at(ref p,2);let d=moves::cleanup_at(ref p,3);
    assert(a.frame==b.frame && c.frame==d.frame && a.frame!=c.frame && a.expression==c.expression && b.expression==d.expression);
}
