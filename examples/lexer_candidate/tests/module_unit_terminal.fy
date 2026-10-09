module tests::module_unit_terminal;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let bound=switch(binding::bind(sources,ref metadata,0,1)){case .ok(b):{b}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources);}
@effects(console) fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{discard io::write("unexpected success\n");assert(false);}case .err(p):{if(p.code!=code || p.module_index!=0 || !bytes::equal(text[p.start..p.end],at)){discard io::write_u32(p.code);discard io::write(" at ");discard io::write_bytes(text[p.start..p.end]);discard io::write("\n");assert(false);}}};}


fn void consumers(ref values::Plan p,usize deferred,usize burned){var usize d=0;var usize b=0;var usize i=0;while(i<values::count(p)){let c=values::at(p,i);var usize j=0;while(j<model::consumer_count(c)){let s=model::consumer_at(c,j);if(s.kind==13 || s.kind==16){let e=model::expression_at(c,s.expression);assert(s.checked && s.expected==s.actual && s.actual==e.type);let t=values::type_at(p,e.type);if(s.kind==13){assert(t.kind==1 && t.scalar==7);d+=1;}else{assert(t.kind==3 && e.kind==2 && e.place.kind==1);b+=1;}}j+=1;}i+=1;}assert(d==deferred && b==burned && values::unresolved(p)==0 && values::work(p)<=1048576);}
@test fn void internal_void_scalar_and_generic_defer_calls_are_checked(){
    let p=switch(checked(b"module main;fn void record(i32 a,i64 b,u8 c,u32 d,usize e,bool f){}fn void empty(){}fn void typed<T>(T n){}fn void main(){defer record(1,4000000000,1,1,1,true);if(true){defer empty();}defer typed<i64>(4000000000);}")){case .ok(p):{p}case .err(_):{assert(false);return;}};consumers(ref p,3,0);
}
@test fn void declaration_owned_no_drop_local_and_parameter_burns_are_checked(){
    let p=switch(checked(b"module main;@abilities(store)resource Ticket{i32 n;cleanup{}}fn void consume(Ticket t){burn t;}fn void main(){let Ticket first=Ticket{.n=1};burn first;consume(Ticket{.n=2});}")){case .ok(p):{p}case .err(_):{assert(false);return;}};consumers(ref p,0,2);
}
@test @effects(console) fn void defer_and_burn_rejections_keep_reference_spans(){
    failure(b"module main;fn i32 number(){return 1;}fn void main(){defer number();}",2,b"defer number();");
    failure(b"module main;fn void observe(ref i32 n){}fn void main(){let i32 n=1;defer observe(ref n);}",2,b"defer observe(ref n);");
    failure(b"module main;fn void main(){defer source_location();}",2,b"defer source_location();");
    failure(b"module main;fn void main(){defer 1;}",2,b"defer 1;");
    failure(b"module main;unsafe extern C fn void foreign();fn void main(){unsafe{defer foreign();};}",2,b"defer foreign();");
    failure(b"module main;fn void take(i32 n){}fn void main(){defer take(true);}",1006,b"true");
    failure(b"module main;resource Ticket{i32 n;cleanup{}}fn void main(){let Ticket t=Ticket{.n=1};burn t;}",1109,b"t");
    failure(b"module main;fn void main(){let i32 n=1;burn n;}",1109,b"n");
    failure(b"module main;@abilities(store)resource Ticket{i32 n;cleanup{}}fn void main(){let Ticket t=Ticket{.n=1};burn (t);}",1109,b"(t)");
    failure(b"module main;resource Child{i32 n;cleanup{}}@abilities(store)resource Parent{Child child;cleanup{}}fn void main(){let Parent p=Parent{.child=Child{.n=1}};burn p;}",1109,b"p");
    failure(b"module main;@abilities(unknown)resource Ticket{i32 n;cleanup{}}fn void main(){let Ticket t=Ticket{.n=1};burn t;}",1107,b"unknown");
    failure(b"module main;@abilities(store,store)resource Ticket{i32 n;cleanup{}}fn void main(){let Ticket t=Ticket{.n=1};burn t;}",1107,b"store");
    failure(b"module main;@abilities(key)resource Ticket{i32 n;cleanup{}}fn void main(){let Ticket t=Ticket{.n=1};burn t;}",1107,b"key");
    failure(b"module main;@abilities()resource Ticket{i32 n;cleanup{}}fn void main(){let Ticket t=Ticket{.n=1};burn t;}",1107,b"@abilities()");
}
@test fn void imported_generic_and_cleanup_defers_keep_original_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(){defer p::done(1);return p::go<i64>(4000000000);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{defer record(this.n);}}fn void record(i32 n){}pub fn void done(i32 n){}pub fn T go<T>(T n){let Item item=Item{.n=1};defer record(1);discard item;return n;}" )).is_ok);let p=switch(projected(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};consumers(ref p,3,0);var usize i=0;var usize provider=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=0;while(j<model::consumer_count(c)){let s=model::consumer_at(c,j);if(s.kind==13){let e=model::expression_at(c,s.expression);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,model::frame(c).module_index),e.source_index);assert(e.start==original.start && e.end==original.end);if(model::frame(c).module_index==1){provider+=1;}}j+=1;}i+=1;}assert(provider==2);
    var v::Vec<syntax::Parsed> denied=v::empty<syntax::Parsed>();let main=b"module main;import provider as p;fn void main(){let p::Ticket t=p::make();burn t;}";assert(v::push<syntax::Parsed>(ref mut denied,parsed(main)).is_ok);assert(v::push<syntax::Parsed>(ref mut denied,parsed(b"module provider;@abilities(store)pub resource Ticket{pub i32 n;cleanup{}}pub fn Ticket make(){return Ticket{.n=1};}" )).is_ok);switch(projected(ref denied)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1109 && p.module_index==0 && bytes::equal(main[p.start..p.end],b"t"));}};
}
