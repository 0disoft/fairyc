module tests::generic_expand;
import tests::hir_nominal as support;
import hir_model as h;
@test fn void same_instantiation_has_one_function(){let p=support::retained(b"module main;fn T id<T>(T n){return n;}fn i32 main(){assert(id<bool>(true));return id<i32>(3)+id<i32>(4);}");assert(h::function_count(ref p)==3);var usize calls=0;var usize i=1;while(i<=h::expression_count(ref p)){if(h::expression_code(h::expression_at(ref p,i).kind)==6){calls+=1;}i+=1;}assert(calls==3);}
@test fn void ordered_arguments_keep_distinct_signatures(){let p=support::retained(b"module main;fn A first<A,B>(A a,B b){return a;}fn i32 main(){assert(first<bool,i32>(true,7));return first<i32,bool>(7,true);}");assert(h::function_count(ref p)==3);}
