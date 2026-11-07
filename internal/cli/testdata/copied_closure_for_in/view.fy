module main;
fn i32 main(){
 var i32 offset=40;
 let add=fn [i32 offset] i32(i32 value){return offset+value;};
 offset=100;
 let i32[2] data=[1,2];let ref i32[] view=ref data[..];
 var i32 visits=0;var i32 total=0;
 let adjacent=fn [] i32(i32 value){return value;};
 for(let i32 item in view){
  if(visits==0){assert(item==1 && add(item)==41);}
  if(visits==1){assert(item==2 && add(item)==42);}
  total+=add(adjacent(item));visits+=1;
 }
 assert(visits==2 && total==83);
 for(let i32 item in view[..0]){discard add(item);assert(false);}
 for(let i32 add in view){assert(adjacent(add)==add);}
 assert(add(2)==42);
 for(let i32 offset in view){assert(add(offset)==40+offset);}
 for(let ref i32 item in view){assert(add(*item)==40+*item);}
 for(let i32 item in view){let i32 add=item;assert(add==item);}
 var i32 nested=0;
 for(let i32 item in view){for(let i32 other in view){nested+=adjacent(item+other);}}
 assert(nested==12 && add(2)==42);
 return add(2);
}
