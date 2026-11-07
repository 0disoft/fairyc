module main;
import std::vec as v;
import std::bytes as bytes;
fn i32 main(){
 var v::Vec<i32> items=v::empty<i32>();var i32 offset=40;
 let add=fn [i32 offset] i32(i32 value){return offset+value;};offset=100;
 for(let ref i32 item in ref items){discard add(*item);assert(false);}
 assert(v::push<i32>(ref mut items,1).is_ok && v::push<i32>(ref mut items,2).is_ok);
 var i32 visits=0;var i32 total=0;
 let adjacent=fn [] i32(i32 value){return value;};
 for(let ref i32 item in ref items){
  if(visits==0){assert(*item==1 && add(*item)==41);}
  if(visits==1){assert(*item==2 && add(*item)==42);}
  total+=add(adjacent(*item));visits+=1;
 }
 assert(visits==2 && total==83);
 let ref v::Vec<i32> view=ref items;
 for(let ref i32 item in view){assert(add(*item)==40+*item);}
 for(let ref i32 add in view){assert(adjacent(*add)==*add);}
 assert(add(2)==42);
 return add(2);
}
