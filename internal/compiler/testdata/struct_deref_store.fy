module main;
import std::vec as vec;
struct Span {usize start;usize end;}
struct Token {i64 value;Span position;}
fn void replace(ref mut Token item,Token value){*item=value;}
fn ref mut Token selected(ref mut Token[] items,usize index){return ref mut items[index];}
fn ref Token read(ref Token[] items,usize index){return ref items[index];}
fn Option<ref mut Token> tagged(ref mut Token[] items){return .some(selected(items,1));}
fn i32 main(){
 var vec::Vec<Token> owner=vec::empty<Token>();
 discard vec::push<Token>(ref mut owner,Token{.value=7,.position=Span{.start=1,.end=4}});
 discard vec::push<Token>(ref mut owner,Token{.value=8,.position=Span{.start=5,.end=9}});
 let ref mut Token[] items=vec::view_mut<Token>(ref mut owner);
 var usize calls=0;
 let ref mut Token item=selected(items,{calls+=1;0});
 var Token replacement=Token{.value=4000000000,.position=Span{.start=11,.end=14}};
 var Token old=*item;
 replace(item,replacement);replacement.position.start=99;
 assert(item.value==4000000000 && item.position.start==11 && old.position.start==1 && calls==1);
 *(item)=*item;assert(item.position.end==14);
 let Option<ref mut Token> result=tagged(items);
 switch(result){case .some(value):{*value=Token{.value=9223372036854775807,.position=Span{.start=15,.end=19}};}case .none:{assert(false);}};
 for(let ref mut Token value in items){*value=Token{.value=value.value-1,.position=value.position};}
 let ref Token first=read(items,0);let ref Token second=read(items,1);
 assert(first.value==3999999999 && second.value==9223372036854775806 && second.position.end==19);
 discard vec::clear<Token>(ref mut owner);assert(old.value==7 && replacement.position.start==99);
 return 0;
}
