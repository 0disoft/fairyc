module main;
import std::vec as vec;
struct Span{usize start;usize end;}
struct Token{i64 value;Span position;bool active;}
fn Option<Token> first(ref mut Token[] items){for(let Token item in items){return .some(item);}return .none;}
fn Result<i32,u32> fail(){return .err(7);}
fn Result<i32,u32> propagate(ref mut Token[] items){for(let Token item in items){items[0].value=item.value+1;let i32 ignored=fail()?;}return .ok(0);}
fn void edit(ref mut Token[] items){
 var usize visits=0;
 for(let Token item in items){
  visits+=1;
  var Token copy=item;copy.value=99;copy.position.start=100;
  assert(item.value!=99 && item.position.start!=100);
  if(visits==1){
   items[0]=Token{.value=9,.position=Span{.start=40,.end=50},.active=false};
   items[1]=Token{.value=5000000000,.position=Span{.start=77,.end=88},.active=true};
   assert(item.value==1 && item.position.start==1 && item.active);
   continue;
  }
  if(visits==2){assert(item.value==5000000000 && item.position.start==77);}
  if(visits==3){assert(item.value==3);break;}
 }
 assert(visits==3);
}
fn i32 main(){
 var vec::Vec<Token> owner=vec::empty<Token>();
 for(var u32 i=1;i<=4;i+=1){let usize offset=i as usize;assert(vec::push<Token>(ref mut owner,Token{.value=i as i64,.position=Span{.start=offset,.end=offset+10},.active=true}).is_ok);}
 let ref mut Token[] items=vec::view_mut<Token>(ref mut owner);
 edit(items);
 assert(items[0].value==9 && !items[0].active && items[1].value==5000000000 && items[2].value==3 && items[3].value==4);
 let ref mut Token[] part=items[2..];var usize index=0;
 for(let Token item in part){part[index].value=item.value+10;assert(item.value<10);index+=1;}
 assert(index==2);
 var usize visits=0;for(let Token outer in items){for(let Token inner in items){assert(outer.value>0 && inner.value>0);visits+=1;}}assert(visits==16);
 var Option<Token> last=.none;for(let Token item in items){last=.some(item);}
 switch(last){case .some(copy):{assert(copy.value==14);}case .none:{assert(false);}};
 switch(propagate(items)){case .ok(_):{assert(false);}case .err(code):{assert(code==7);}};
 let Option<Token> saved=first(items);
 let ref mut Token[] empty=items[4..];for(let Token item in empty){assert(false);}
 for(let Token items in items){assert(items.value>0);}
 assert(vec::clear<Token>(ref mut owner)==4);
 switch(saved){case .some(copy):{assert(copy.value==10 && copy.position.start==40);}case .none:{assert(false);}};
 assert(vec::len<Token>(ref owner)==0);
 let ref mut Token[] cleared=vec::view_mut<Token>(ref mut owner);for(let Token item in cleared){assert(false);}
 return 0;
}
