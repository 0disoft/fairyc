module main;
import std::vec as vec;
struct Span {usize start;usize end;}
struct Token {i64 value;Span position;bool active;}
fn void edit(ref mut Span span){span.start+=2;}
fn ref i64 number(ref Token[] view){return ref view[0].value;}
fn ref mut i64 mutable_number(ref mut Token[] view){return ref mut view[0].value;}
fn Option<ref mut i64> tagged(ref mut Token[] view){return .some(mutable_number(view));}
fn i32 main(){
 var vec::Vec<Token> owner=vec::empty<Token>();
 discard vec::push<Token>(ref mut owner,Token{.value=4000000000,.position=Span{.start=1,.end=4},.active=true});
 let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);
 view[0].value=5000000000;
 view[0].value=view[0].value+1;
 assert(view[0].value==5000000001);
 view[0].active=false;assert(!view[0].active);
 view[0].position.start+=3;
 var Span replacement=Span{.start=4,.end=9};
 view[0].position=replacement;replacement.start=100;
 assert(view[0].position.start==4);
 let ref mut Span span=ref mut view[0].position;
 edit(span);assert(span.start==6);
 let ref mut i64 value=ref mut view[0].value;
 *value=6000000000;assert(*value==6000000000);
 let ref i64 shared=ref view[0].value;assert(*shared==6000000000);
 let ref i64 returned=number(view);assert(*returned==6000000000);
 let Option<ref mut i64> result=tagged(view);
 switch(result){case .some(item):{*item=7000000000;assert(*item==7000000000);}case .none:{assert(false);}};
 assert(view[0].value==7000000000 && view[0].position.start==6);
 var usize calls=0;
 view[{calls+=1;0}].value+=1;
 assert(calls==1 && view[0].value==7000000001);
 discard vec::clear<Token>(ref mut owner);
 return 0;
}
