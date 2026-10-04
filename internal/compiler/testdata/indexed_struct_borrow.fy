module main;
import std::vec as vec;
struct Span {usize start;usize end;}
struct Token {i64 value;Span position;}
fn void edit(ref mut Token item){item.value+=1000000000;item.position.start+=1;}
fn ref Token selected(ref Token[] items,usize index){return ref items[index];}
fn ref mut Token selected_mut(ref mut Token[] items,usize index){return ref mut items[index];}
fn Option<ref mut Token> tagged(ref mut Token[] items){return .some(selected_mut(items,1));}
fn i32 main(){
 var vec::Vec<Token> owner=vec::empty<Token>();
 discard vec::push<Token>(ref mut owner,Token{.value=4000000000,.position=Span{.start=1,.end=4}});
 discard vec::push<Token>(ref mut owner,Token{.value=7,.position=Span{.start=5,.end=9}});
 let ref mut Token[] items=vec::view_mut<Token>(ref mut owner);
 var usize calls=0;
 let ref mut Token item=ref mut ((items)[{calls+=1;0}]);
 var Token copy=*item;
 edit(item);assert(item.value==5000000000 && item.position.start==2);
 copy.value=9;assert(copy.position.start==1 && item.value==5000000000);
 let ref Token shared=ref items[0];let ref Token also=ref items[1];assert(shared.value==5000000000 && also.value==7);
 edit(ref mut items[1]);assert(items[1].value==1000000007);
 let Option<ref mut Token> result=tagged(items);
 switch(result){case .some(value):{let ref mut Span position=ref mut value.position;position.end+=2;value.value+=1;}case .none:{assert(false);}};
 assert(items[1].value==1000000008 && items[1].position.end==11 && calls==1);
 let ref Token[] shared_view=vec::view<Token>(ref owner);
 let ref Token returned=selected(shared_view,1);assert(returned.value==1000000008);
 discard vec::clear<Token>(ref mut owner);assert(copy.value==9);
 return 0;
}
