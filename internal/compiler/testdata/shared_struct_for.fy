module main;
import std::vec as vec;
struct Span {usize start;usize end;}
struct Token {u32 kind;i64 value;bool active;Span position;}
fn T identity<T>(T value){return value;}
fn Option<ref Token> find(ref Token[] items,u32 kind){
 for(let ref Token item in items){if(item.kind==kind){return .some(item);}}
 return .none;
}
fn Option<ref Span> position(ref mut Token[] items,u32 kind){
 for(let ref Token item in items){if(item.kind==kind){return .some(ref item.position);}}
 return .none;
}
fn Result<i32,u32> fail(){return .err(7);}
fn Result<i32,u32> propagate(ref Token[] items){for(let ref Token item in items){assert(item.active);let i32 ignored=fail()?;}return .ok(0);}
fn i32 main(){
 var vec::Vec<Token> owner=vec::empty<Token>();
 for(var u32 kind=1;kind<=4;kind+=1){
  let usize start=kind as usize;
  assert(vec::push<Token>(ref mut owner,Token{.kind=kind,.value=4000000000+kind as i64,.active=true,.position=Span{.start=start,.end=start+10}}).is_ok);
 }
 let ref mut Token[] exclusive=vec::view_mut<Token>(ref mut owner);
 switch(position(exclusive,2)){case .some(picked):{assert(picked.start==2 && picked.end==12);}case .none:{assert(false);}};
 var usize visits=0;
 for(let ref Token item in exclusive){visits+=1;assert(item.value>4000000000);let ref Span child=ref item.position;assert(child.end-child.start==10);if(item.kind==1){continue;}if(item.kind==3){break;}}
 assert(visits==3);
 exclusive[0].value=9;
 let ref Token[] items=vec::view<Token>(ref owner);
 let ref Token[] part=items[1..];
 switch(find(part,2)){case .some(picked):{assert(picked.kind==2 && picked.value==4000000002);let ref Span child=ref picked.position;assert(child.start==2);}case .none:{assert(false);}};
 switch(find(items,1)){case .some(picked):{assert(picked.value==9);}case .none:{assert(false);}};
 switch(find(items,4)){case .some(picked):{assert(picked.position.end==14);}case .none:{assert(false);}};
 switch(find(items,99)){case .some(_):{assert(false);}case .none:{}};
 let ref Token[] empty=items[4..];switch(find(empty,1)){case .some(_):{assert(false);}case .none:{}};
 var Option<ref Token> last=.none;
 for(let ref Token item in items){switch(last){case .some(old):{assert(old.kind<item.kind);}case .none:{}};last=.some(item);}
 switch(last){case .some(item):{assert(item.kind==4);}case .none:{assert(false);}};
 visits=0;for(let ref Token a in items){for(let ref Token b in items){assert(a.active && b.active);visits+=1;}}assert(visits==16);
 for(let ref Token items in items){assert(items.active);let i64 value=identity<i64>(items.value);var vec::Vec<i64> scratch=vec::empty<i64>();assert(vec::push<i64>(ref mut scratch,value).is_ok);assert(vec::len<i64>(ref scratch)==1);}
 switch(propagate(items)){case .ok(_):{assert(false);}case .err(code):{assert(code==7);}};
 assert(vec::clear<Token>(ref mut owner)==4);
 assert(vec::len<Token>(ref owner)==0);
 let ref Token[] cleared=vec::view<Token>(ref owner);for(let ref Token item in cleared){assert(false);}
 return 0;
}
