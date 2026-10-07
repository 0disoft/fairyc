module tests::resource_cleanup;
import tests::result_types as fixture;
import type_check as typing;
import name_resolution as names;
import std::bytes as bytes;

fn void accepted(ref u8[] source){let r=fixture::resolved(source);assert(names::problem(ref r).code==0);let t=typing::check(r);assert(typing::problem(ref t).code==0);}
fn void rejected(ref u8[] source,u32 code,ref u8[] spelling){let r=fixture::resolved(source);assert(names::problem(ref r).code==0);let t=typing::check(r);let p=typing::problem(ref t);assert(p.code==code);assert(bytes::equal(source[p.start..p.end],spelling));}

@test
fn void receiver_is_scoped_mutable_and_cannot_escape_as_a_second_owner(){
    accepted(b"module main;resource R{i32 n;cleanup{let i32 x=this.n;this.n=x+1;}}fn i32 main(){return 0;}");
    rejected(b"module main;resource R{cleanup{take(this);}}fn void take(R r){}",48,b"this");
    rejected(b"module main;resource R{cleanup{let R other=this;}}",48,b"this");
    rejected(b"module main;resource R{cleanup{this=R{};}}",48,b"this");
}
@test
fn void cleanup_checks_types_and_is_infallible(){
    rejected(b"module main;resource R{i32 n;cleanup{this.n=true;}}",1,b"true");
    rejected(b"module main;resource R{cleanup{return 1;}}",1,b"1");
    rejected(b"module main;resource R{cleanup{let i32 x=f()?;}}fn Result<i32,bool> f(){return .ok(1);}",41,b"?");
}
@test
fn void ordinary_cleanup_locals_keep_move_and_result_obligations(){
    accepted(b"module main;resource R{i32 n;cleanup{var i32 i=0;while(i<2){this.n+=1;i+=1;}return;}}fn i32 main(){return 0;}");
    rejected(b"module main;resource R{cleanup{let S s=S{};take(s);take(s);}}resource S{cleanup{}}fn void take(S s){}",45,b"s");
    rejected(b"module main;resource R{cleanup{let Result<i32,bool> x=f();}}fn Result<i32,bool> f(){return .ok(1);}",42,b"f()");
}
@test
fn void receiver_shadow_and_nested_fields_do_not_change_other_owners(){
    accepted(b"module main;struct Pair{i32 n;}resource R{Pair p;cleanup{{let i32 this=5;let i32 x=this;};let i32 y=this.p.n;}}");
    rejected(b"module main;struct Pair{i32 n;}resource R{Pair p;cleanup{this.p.n=3;}}",4,b"this.p.n");
    rejected(b"module main;resource S{cleanup{}}resource R{S s;cleanup{take(this.s);}}fn void take(S s){}",44,b"this.s");
    let r=fixture::resolved(b"module main;resource R{cleanup{let i32 x=1;}}fn void f(){let i32 x=this.n;}");assert(names::problem(ref r).code==1);
}
