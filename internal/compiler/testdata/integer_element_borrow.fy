module main;
fn ref u8 first_u8(ref u8[] view){return ref view[0];}
fn ref mut u8 last_u8(ref mut u8[] view){return ref mut view[1];}
fn void check_u8(){
 var u8[3] data=[1,255,3];
 let ref mut u8[] view=ref mut data;
 let ref mut u8[] child=view[1..];
 let ref mut u8 item=ref mut child[0];
 let u8 copy=*item;*item=7;assert(copy==255 && *item==7);
 let ref mut u8 returned=last_u8(view);*returned=*returned+2;
 let ref u8 shared=first_u8(view);let ref u8 sibling=ref view[1];
 assert(*shared==1 && *sibling==9 && data[2]==3);
}
fn ref i32 first_i32(ref i32[] view){return ref view[0];}
fn ref mut i32 last_i32(ref mut i32[] view){return ref mut view[1];}
fn void check_i32(){
 var i32[3] data=[1,2147483647,3];
 let ref mut i32[] view=ref mut data;
 let ref mut i32[] child=view[1..];
 let ref mut i32 item=ref mut child[0];
 let i32 copy=*item;*item=7;assert(copy==2147483647 && *item==7);
 let ref mut i32 returned=last_i32(view);*returned=*returned+2;
 let ref i32 shared=first_i32(view);let ref i32 sibling=ref view[1];
 assert(*shared==1 && *sibling==9 && data[2]==3);
}
fn ref i64 first_i64(ref i64[] view){return ref view[0];}
fn ref mut i64 last_i64(ref mut i64[] view){return ref mut view[1];}
fn void check_i64(){
 var i64[3] data=[1,4000000000,3];
 let ref mut i64[] view=ref mut data;
 let ref mut i64[] child=view[1..];
 let ref mut i64 item=ref mut child[0];
 let i64 copy=*item;*item=7;assert(copy==4000000000 && *item==7);
 let ref mut i64 returned=last_i64(view);*returned=*returned+2;
 let ref i64 shared=first_i64(view);let ref i64 sibling=ref view[1];
 assert(*shared==1 && *sibling==9 && data[2]==3);
}
fn ref u32 first_u32(ref u32[] view){return ref view[0];}
fn ref mut u32 last_u32(ref mut u32[] view){return ref mut view[1];}
fn void check_u32(){
 var u32[3] data=[1,4294967295,3];
 let ref mut u32[] view=ref mut data;
 let ref mut u32[] child=view[1..];
 let ref mut u32 item=ref mut child[0];
 let u32 copy=*item;*item=7;assert(copy==4294967295 && *item==7);
 let ref mut u32 returned=last_u32(view);*returned=*returned+2;
 let ref u32 shared=first_u32(view);let ref u32 sibling=ref view[1];
 assert(*shared==1 && *sibling==9 && data[2]==3);
}
fn ref usize first_usize(ref usize[] view){return ref view[0];}
fn ref mut usize last_usize(ref mut usize[] view){return ref mut view[1];}
fn void check_usize(){
 var usize[3] data=[1,4294967296,3];
 let ref mut usize[] view=ref mut data;
 let ref mut usize[] child=view[1..];
 let ref mut usize item=ref mut child[0];
 let usize copy=*item;*item=7;assert(copy==4294967296 && *item==7);
 let ref mut usize returned=last_usize(view);*returned=*returned+2;
 let ref usize shared=first_usize(view);let ref usize sibling=ref view[1];
 assert(*shared==1 && *sibling==9 && data[2]==3);
}
fn Option<ref mut i64> tagged(ref mut i64[] view){return .some(ref mut view[0]);}
fn i32 main(){
 check_u8();check_i32();check_i64();check_u32();check_usize();
 var i64[2] data=[4000000000,7];let ref mut i64[] view=ref mut data;
 var usize calls=0;let ref mut i64 item=ref mut ((view)[{calls+=1;0}]);*item=*item+10;
 let Option<ref mut i64> result=tagged(view);
 switch(result){case .some(value):{*value=*value+1;}case .none:{assert(false);}};
 assert(data[0]==4000000011 && calls==1);
 let ref u8[] text=b"Az";let ref u8 letter=ref text[1];assert(*letter==122);
 return 0;
}
