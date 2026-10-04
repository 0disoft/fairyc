module main;
fn void check_u8(){
 var u8[5] data=[0,1,200,3,4];
 var usize visits=0;
 for(let u8 item in data){visits+=1;assert(item==data[visits-1]);}
 assert(visits==5 && data[2]==200);
 visits=0;
 for(let ref mut u8 item in data){
  visits+=1;let u8 copy=*item;*item=*item+1;assert(*item==copy+1);
  if(copy==0){continue;}if(copy==3){break;}
 }
 assert(visits==4 && data[0]==1 && data[1]==2 && data[2]==200+1 && data[3]==4 && data[4]==4);
 for(let ref mut u8 item in data){*item=*item+1;}
 assert(data[0]==2 && data[4]==5);
 let u8[1] single=[7];var u8 total=0;
 for(let u8 item in single){total+=item;}
 assert(total==7);
}
fn void check_i32(){
 var i32[5] data=[0,1,2000000000,3,4];
 var usize visits=0;
 for(let i32 item in data){visits+=1;assert(item==data[visits-1]);}
 assert(visits==5 && data[2]==2000000000);
 visits=0;
 for(let ref mut i32 item in data){
  visits+=1;let i32 copy=*item;*item=*item+1;assert(*item==copy+1);
  if(copy==0){continue;}if(copy==3){break;}
 }
 assert(visits==4 && data[0]==1 && data[1]==2 && data[2]==2000000000+1 && data[3]==4 && data[4]==4);
 for(let ref mut i32 item in data){*item=*item+1;}
 assert(data[0]==2 && data[4]==5);
 let i32[1] single=[7];var i32 total=0;
 for(let i32 item in single){total+=item;}
 assert(total==7);
}
fn void check_i64(){
 var i64[5] data=[0,1,4000000000,3,4];
 var usize visits=0;
 for(let i64 item in data){visits+=1;assert(item==data[visits-1]);}
 assert(visits==5 && data[2]==4000000000);
 visits=0;
 for(let ref mut i64 item in data){
  visits+=1;let i64 copy=*item;*item=*item+1;assert(*item==copy+1);
  if(copy==0){continue;}if(copy==3){break;}
 }
 assert(visits==4 && data[0]==1 && data[1]==2 && data[2]==4000000000+1 && data[3]==4 && data[4]==4);
 for(let ref mut i64 item in data){*item=*item+1;}
 assert(data[0]==2 && data[4]==5);
 let i64[1] single=[7];var i64 total=0;
 for(let i64 item in single){total+=item;}
 assert(total==7);
}
fn void check_u32(){
 var u32[5] data=[0,1,4000000000,3,4];
 var usize visits=0;
 for(let u32 item in data){visits+=1;assert(item==data[visits-1]);}
 assert(visits==5 && data[2]==4000000000);
 visits=0;
 for(let ref mut u32 item in data){
  visits+=1;let u32 copy=*item;*item=*item+1;assert(*item==copy+1);
  if(copy==0){continue;}if(copy==3){break;}
 }
 assert(visits==4 && data[0]==1 && data[1]==2 && data[2]==4000000000+1 && data[3]==4 && data[4]==4);
 for(let ref mut u32 item in data){*item=*item+1;}
 assert(data[0]==2 && data[4]==5);
 let u32[1] single=[7];var u32 total=0;
 for(let u32 item in single){total+=item;}
 assert(total==7);
}
fn void check_usize(){
 var usize[5] data=[0,1,4294967296,3,4];
 var usize visits=0;
 for(let usize item in data){visits+=1;assert(item==data[visits-1]);}
 assert(visits==5 && data[2]==4294967296);
 visits=0;
 for(let ref mut usize item in data){
  visits+=1;let usize copy=*item;*item=*item+1;assert(*item==copy+1);
  if(copy==0){continue;}if(copy==3){break;}
 }
 assert(visits==4 && data[0]==1 && data[1]==2 && data[2]==4294967296+1 && data[3]==4 && data[4]==4);
 for(let ref mut usize item in data){*item=*item+1;}
 assert(data[0]==2 && data[4]==5);
 let usize[1] single=[7];var usize total=0;
 for(let usize item in single){total+=item;}
 assert(total==7);
}
fn i64 sum(i64[3] data){var i64 total=0;for(let i64 item in data){total+=item;}return total;}
fn void early(){var i64[2] data=[1,2];for(let ref mut i64 item in data){*item=9;return;}assert(false);}
fn Result<i32,u32> fail(){return .err(7);}
fn Result<i32,u32> propagation(){var i64[2] data=[1,2];for(let ref mut i64 item in data){*item=9;let i32 unused=fail()?;}return .ok(0);}
fn void nested(){var i64[2] left=[1,2];var i64[1] right=[7];for(let ref mut i64 x in left){for(let ref mut i64 y in right){*y=*y+*x;}*x=*x+1;}assert(left[0]==2 && left[1]==3 && right[0]==10);}
fn void shadow(){var i64[2] data=[1,2];for(let ref mut i64 data in data){*data=*data+1;}assert(data[0]==2 && data[1]==3);}
fn i32 main(){check_u8();check_i32();check_i64();check_u32();check_usize();early();nested();shadow();assert(sum([1,4000000000,3])==4000000004);switch(propagation()){case .ok(_):{assert(false);}case .err(code):{assert(code==7);}};return 0;}
