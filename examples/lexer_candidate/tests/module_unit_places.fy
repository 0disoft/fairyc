module tests::module_unit_places;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import module_types as types;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void named_borrows_and_dereferences_reach_arguments_casts_and_returns(){
    let p=switch(checked(b"module main;fn i64 leaf(ref i64 n){return *n;}fn i64 main(){var i64 n=4000000000;let shared=ref n;let alias=shared;let got=leaf(alias);let mutable=ref mut n;let u8 byte=1;let byte_ref=ref byte;let widened=(*byte_ref) as i64;discard widened;return got+*mutable;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::checked_returns(ref p)==2 && values::checked_arguments(ref p)==1 && values::inferred(ref p)==6 && values::unresolved(ref p)==0);let c=values::at(ref p,1);var usize borrow_count=0;var usize dereferences=0;var usize i=1;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==21 || e.kind==22){let t=values::type_at(ref p,e.type);assert(e.state==1 && t.kind==if(e.kind==21){7}else{8} && (values::type_at(ref p,t.first).scalar==2 || values::type_at(ref p,t.first).scalar==3) && e.place.root>0 && e.place.kind==0);borrow_count+=1;}else if(e.kind==4 && e.place.kind==3){assert(e.place.root>0 && e.place.indirect && !e.place.borrowable && e.state==1);dereferences+=1;}i+=1;}assert(borrow_count==3 && dereferences==2);assert(model::local_at(c,2).origin.root==1 && model::local_at(c,3).origin.root==1 && model::local_at(c,5).origin.root==1);
}
@test fn void nested_fields_keep_whole_owner_and_reference_mutability(){
    let p=switch(checked(b"module main;struct Inner{i64 n;}struct Outer{Inner inner;}fn i64 main(ref mut Outer owner){let alias=owner;let shared=ref alias.inner.n;let got=*shared;let mutable=ref mut owner.inner.n;return got+*mutable;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,0);var usize fields=0;var usize i=1;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==8){assert(e.place.kind==2 && e.place.root==1 && e.place.scope==1 && e.place.indirect && e.place.mutable && e.place.borrowable && e.field.id>0 && !e.place.resource_field);fields+=1;}i+=1;}assert(fields==4);
}
@test fn void immutable_and_nonplace_borrows_keep_reference_diagnostics(){
    failure(b"module main;fn i32 main(){let i32 n=1;let r=ref mut n;return *r;}",1008,b"n");
    failure(b"module main;fn i32 main(){let r=ref (1+2);return *r;}",2,b"ref (1+2)");
    failure(b"module main;fn i32 main(){let r=ref mut (1+2);return *r;}",1008,b"(1+2)");
    failure(b"module main;fn i32 main(ref i32 r){let p=ref *r;return *p;}",2,b"ref *r");
    failure(b"module main;fn i32 main(ref mut i32 r){let p=ref mut *r;return *p;}",1008,b"*r");
    failure(b"module main;fn i32 main(){var i32 x=1;let r=ref (x);return *r;}",2,b"ref (x)");
    failure(b"module main;struct S{i32 n;}fn i32 main(ref S s){let r=ref mut s.n;return *r;}",1008,b"s.n");
    failure(b"module main;fn i32 main(i32 n){return *n;}",2,b"*n");
    failure(b"module main;fn u8 main(ref i32 n){return *n;}",1009,b"*n");
}
@test fn void generic_and_cleanup_borrows_preserve_provider_places(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn u8 main(ref u8 n){return p::go<u8>(n);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{u8 n;cleanup{let r=ref this.n;let loaded=*r;discard loaded;return;}}pub fn T go<T>(ref T n){let Item item=Item{.n=1};let alias=n;discard item;return *alias;}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==3);var usize i=0;var bool cleanup_seen=false;var bool generic=false;while(i<values::count(ref p)){let c=values::at(ref p,i);let f=model::frame(c);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.place.kind==3){assert(e.place.root>0 && e.place.scope>0 && e.state==1 && values::type_at(ref p,e.type).scalar==3);if(f.kind==2){assert(f.module_index==1 && e.place.resource_field);cleanup_seen=true;}else if(f.kind==1){assert(f.module_index==1);generic=true;}}j+=1;}i+=1;}assert(cleanup_seen && generic);
    failure(b"module main;resource R{i32 x;cleanup{let r=ref mut this.x;discard r;return;}}fn void main(){let R item=R{.x=1};discard item;return;}",1008,b"this.x");
}
@test fn void arrays_views_and_unsafe_pointers_remain_explicitly_pending(){
    let p=switch(checked(b"module main;fn void array(i32[4] a){let r=ref a;discard r;return;}unsafe fn i32 raw(ptr i32 p){return *p;}fn void view(ref u8[] data){let r=ref data;discard r;return;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==2 && values::unresolved(ref p)>=3);var usize i=0;var usize pending=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==21 || e.kind==4){assert(e.state==0 && e.type==0);pending+=1;}j+=1;}i+=1;}assert(pending==3);
}
