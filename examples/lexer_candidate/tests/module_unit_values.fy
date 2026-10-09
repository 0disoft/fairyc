module tests::module_unit_values;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_unit_plan as units;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(ref sources,ref metadata,ref ready);}
@test fn void call_results_reach_generic_cleanup_and_ordinary_locals(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn u8 main(){let inferred=p::go<u8>(1);let copy=(inferred);return (copy);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{u8 n;cleanup{let inferred=helper();discard inferred;return;}}fn u8 helper(){return 2;}pub fn T go<T>(T n){let Item item=Item{.n=1};let inferred=helper();let copy=inferred;discard item;return copy;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::count(ref p)==3 && values::inferred(ref p)==5 && values::checked_returns(ref p)==3 && values::work(ref p)<=1048576);var usize i=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);let f=model::frame(c);let original=v::at<syntax::Parsed>(ref sources,f.module_index);var usize n=1;while(n<=model::expression_count(c)){let e=model::expression_at(c,n);let s=syntax::expression_at(original,e.source_index);assert(e.start==s.start && e.end==s.end && e.kind==s.kind);n+=1;}n=1;while(n<=model::local_count(c)){let s=model::local_at(c,n);if(s.expected==0 && s.type!=0){assert(values::type_at(ref p,s.type).scalar==3 && syntax::entry_at(original,s.source_index-1).expression==s.initializer);}n+=1;}i+=1;}
}
@test fn void literals_take_only_direct_local_and_return_context(){
    let text=b"module main;fn u8 byte(){let u8 small=(255);let copy=small;return (copy);}fn i64 wide(){return (9223372036854775807);}fn ptr i32 pointer(){let ptr i32 p=(null);return p;}fn ref str text(){let word=\"ok\";return word;}fn ref u8[] raw(){let data=b\"ok\";return data;}fn bool yes(){let flag=true;return flag;}fn ref CStr ctext(){let c=c\"ok\";return c;}";
    let p=switch(checked(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==7 && values::inferred(ref p)==5 && values::checked_returns(ref p)==7);var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);assert(model::return_count(c)==1 && model::return_at(c,0).checked);i+=1;}
}
@test fn void mismatches_and_literal_overflow_keep_provider_spans(){
    let wrong=b"module main;fn u8 byte(){return true;}";switch(checked(wrong)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1006 && p.module_index==0 && bytes::equal(wrong[p.start..p.end],b"true") && p.expected!=p.actual);}};
    let local=b"module main;fn bool flag(){return true;}fn i32 main(){let i32 n=flag();return 0;}";switch(checked(local)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1006 && bytes::equal(local[p.start..p.end],b"flag()"));}};
    let call=b"module main;fn bool flag(){return true;}fn i32 main(){return flag();}";switch(checked(call)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1006 && bytes::equal(call[p.start..p.end],b"flag()"));}};
    let overflow=b"module main;fn u8 byte(){return (256);}";switch(checked(overflow)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1013 && bytes::equal(overflow[p.start..p.end],b"256"));}};
    let empty=b"module main;fn i32 main(){return;}";switch(checked(empty)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1006 && bytes::equal(empty[p.start..p.end],b"return;"));}};
    switch(checked(b"module main;fn i32 main(){return null;}")){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1006);}};
}
@test fn void variants_remain_unchecked_after_argument_matching(){
    let p=switch(checked(b"module main;fn u8 leaf(u8 n){return n;}fn u8 main(){let inferred=leaf(1);let pending=1+2;discard pending;return inferred+1;}fn Option<i32> option(){return .some(1);}enum Flag{A;B;}fn Flag flag(){return .A;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::inferred(ref p)==2 && values::checked_returns(ref p)==2 && values::unresolved(ref p)>0 && values::checked_calls(ref p)==1 && values::checked_arguments(ref p)==1);var usize i=0;var usize deferred=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize n=0;while(n<model::return_count(c)){if(!model::return_at(c,n).checked){deferred+=1;}n+=1;}i+=1;}assert(deferred==2);
}
@test fn void stale_sources_are_rejected_before_value_publication(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn i32 main(){return 1;}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};discard v::pop<syntax::Parsed>(ref mut sources);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn i32 main(){return 2;}")).is_ok);let fresh=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};switch(values::build(ref sources,ref fresh,ref ready)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.module_index==0);}};
}
@test fn void callable_names_do_not_become_inferred_values(){
    let text=b"module main;fn i32 leaf(){return 1;}fn i32 main(){let pending=leaf;return 0;}";switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1006 && bytes::equal(text[p.start..p.end],b"leaf"));}};
    let wrapped=b"module main;fn i32 leaf(){return 1;}fn i32 main(){let pending=(leaf);return 0;}";switch(checked(wrapped)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1006 && bytes::equal(wrapped[p.start..p.end],b"(leaf)"));}};
}
@test fn void value_admission_keeps_the_active_local_bound(){
    var bytes::Buffer text=switch(bytes::copy(b"module main;fn i32 main(){")){case .ok(b):{b}case .err(_):{assert(false);return;}};var usize i=0;var u8 high=65;var u8 low=65;while(i<128){assert(bytes::append(ref mut text,b"let a").is_ok);assert(bytes::append_byte(ref mut text,high).is_ok);assert(bytes::append_byte(ref mut text,low).is_ok);assert(bytes::append(ref mut text,b"=1;").is_ok);low+=1;if(low==91){low=65;high+=1;}i+=1;}let prefix=bytes::view(ref text).len;assert(bytes::append(ref mut text,b"return 0;}").is_ok);let p=switch(checked(bytes::view(ref text))){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::locals(ref p)==128 && values::inferred(ref p)==128);
    assert(bytes::resize_zeroed(ref mut text,prefix).is_ok);assert(bytes::append(ref mut text,b"let extra=1;return 0;}").is_ok);var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref text))).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};switch(units::build(ref sources,ref metadata)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==10 && p.module_index==0);}};
}
