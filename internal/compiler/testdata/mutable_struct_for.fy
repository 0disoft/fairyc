module main;
import std::vec as vec;
struct Span {usize start;usize end;}
struct Token {i64 value;Span position;}
fn void bump(ref mut Token token){token.value+=4000000000;}
fn void edit(ref mut Token[] items){
 var usize visits=0;
 for(let ref mut Token item in items){
  visits+=1;
  if(item.value==1){item.value=11;continue;}
  bump(item);
  let ref mut Span position=ref mut item.position;
  position.start+=3;assert(position.end>position.start);
  if(item.value==4000000003){break;}
 }
 assert(visits==3);
}
fn Option<ref mut Token> first(ref mut Token[] items){for(let ref mut Token item in items){return .some(item);}return .none;}
fn Result<i32,u32> failure(){return .err(7);}
fn Result<i32,u32> visit(ref mut Token[] items){for(let ref mut Token item in items){item.value+=1;let i32 value=failure()?;}return .ok(0);}
fn i32 main(){
 var vec::Vec<Token> owner=vec::empty<Token>();
 var usize offset=1;
 for(var i32 i=1;i<=4;i+=1){discard vec::push<Token>(ref mut owner,Token{.value=i as i64,.position=Span{.start=offset,.end=offset+10}});offset+=1;}
 let ref mut Token[] items=vec::view_mut<Token>(ref mut owner);
 edit(items);
 assert(items[0].value==11 && items[1].value==4000000002 && items[2].value==4000000003 && items[3].value==4);
 assert(items[1].position.start==5 && items[2].position.start==6);
 switch(first(items)){case .some(picked):{picked.value=42;assert(picked.value==42);}case .none:{assert(false);}};
 switch(visit(items)){case .ok(_):{assert(false);}case .err(code):{assert(code==7);}};
 assert(items[0].value==43 && items[1].value==4000000002);
 let ref mut Token[] empty=items[4..];for(let ref mut Token item in empty){assert(false);}
 discard vec::clear<Token>(ref mut owner);assert(vec::len<Token>(ref owner)==0);
 return 0;
}
