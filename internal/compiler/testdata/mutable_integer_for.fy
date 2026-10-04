module main;
fn void check_u8(){
 var u8[5] data=[0,1,200,3,4];
 let ref mut u8[] view=ref mut data;
 let ref mut u8[] part=view[1..4];
 var usize visits=0;
 for(let ref mut u8 item in part){
  visits+=1;
  let u8 copy=*item;
  *item=*item+1;
  assert(*item==copy+1);
  if(copy==1){continue;}
  if(copy==3){break;}
 }
 assert(visits==3 && view[0]==0 && view[1]==2 && view[2]==200+1 && view[3]==4 && view[4]==4);
 let ref mut u8[] empty=view[5..];
 for(let ref mut u8 item in empty){assert(false);}
 view[0]=9;
 assert(data[0]==9);
}
fn void check_i32(){
 var i32[5] data=[0,1,2000000000,3,4];
 let ref mut i32[] view=ref mut data;
 let ref mut i32[] part=view[1..4];
 var usize visits=0;
 for(let ref mut i32 item in part){
  visits+=1;
  let i32 copy=*item;
  *item=*item+1;
  assert(*item==copy+1);
  if(copy==1){continue;}
  if(copy==3){break;}
 }
 assert(visits==3 && view[0]==0 && view[1]==2 && view[2]==2000000000+1 && view[3]==4 && view[4]==4);
 let ref mut i32[] empty=view[5..];
 for(let ref mut i32 item in empty){assert(false);}
 view[0]=9;
 assert(data[0]==9);
}
fn void check_i64(){
 var i64[5] data=[0,1,4000000000,3,4];
 let ref mut i64[] view=ref mut data;
 let ref mut i64[] part=view[1..4];
 var usize visits=0;
 for(let ref mut i64 item in part){
  visits+=1;
  let i64 copy=*item;
  *item=*item+1;
  assert(*item==copy+1);
  if(copy==1){continue;}
  if(copy==3){break;}
 }
 assert(visits==3 && view[0]==0 && view[1]==2 && view[2]==4000000000+1 && view[3]==4 && view[4]==4);
 let ref mut i64[] empty=view[5..];
 for(let ref mut i64 item in empty){assert(false);}
 view[0]=9;
 assert(data[0]==9);
}
fn void check_u32(){
 var u32[5] data=[0,1,4000000000,3,4];
 let ref mut u32[] view=ref mut data;
 let ref mut u32[] part=view[1..4];
 var usize visits=0;
 for(let ref mut u32 item in part){
  visits+=1;
  let u32 copy=*item;
  *item=*item+1;
  assert(*item==copy+1);
  if(copy==1){continue;}
  if(copy==3){break;}
 }
 assert(visits==3 && view[0]==0 && view[1]==2 && view[2]==4000000000+1 && view[3]==4 && view[4]==4);
 let ref mut u32[] empty=view[5..];
 for(let ref mut u32 item in empty){assert(false);}
 view[0]=9;
 assert(data[0]==9);
}
fn void check_usize(){
 var usize[5] data=[0,1,4294967296,3,4];
 let ref mut usize[] view=ref mut data;
 let ref mut usize[] part=view[1..4];
 var usize visits=0;
 for(let ref mut usize item in part){
  visits+=1;
  let usize copy=*item;
  *item=*item+1;
  assert(*item==copy+1);
  if(copy==1){continue;}
  if(copy==3){break;}
 }
 assert(visits==3 && view[0]==0 && view[1]==2 && view[2]==4294967296+1 && view[3]==4 && view[4]==4);
 let ref mut usize[] empty=view[5..];
 for(let ref mut usize item in empty){assert(false);}
 view[0]=9;
 assert(data[0]==9);
}
fn void bump(ref mut i64 value){*value=*value+4000000000;}
fn Option<ref mut i64> first(ref mut i64[] values){for(let ref mut i64 item in values){return .some(item);}return .none;}
fn Result<i32,u32> fail(){return .err(7);}
fn Result<i32,u32> visit(ref mut i64[] values){for(let ref mut i64 item in values){bump(item);let i32 value=fail()?;}return .ok(0);}
fn void nested(ref mut i64[] a,ref mut i64[] b){for(let ref mut i64 x in a){for(let ref mut i64 y in b){*y=*y+*x;}*x=*x+1;}}
fn i32 main(){
 check_u8();check_i32();check_i64();check_u32();check_usize();
 var i64[2] data=[1,2];let ref mut i64[] values=ref mut data;
 switch(first(values)){case .some(item):{*item=42;}case .none:{assert(false);}};
 switch(visit(values)){case .ok(_):{assert(false);}case .err(code):{assert(code==7);}};
 assert(values[0]==4000000042 && values[1]==2);
 var i64[1] other=[7];let ref mut i64[] right=ref mut other;
 nested(values,right);
 assert(values[0]==4000000043 && values[1]==3 && right[0]==4000000051);
 return 0;
}
