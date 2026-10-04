module main;
fn ref mut u8 identity_u8(ref mut u8 value){return value;}
fn void edit_u8(ref mut u8 value){*value=*value+2;}
fn void check_u8(){
 var u8[3] data=[1,255,3];
 let ref mut u8 item=ref mut ((data)[1]);
 let u8 copy=*item;*item=7;
 assert(copy==255 && *item==7);
 edit_u8(ref mut data[1]);
 let ref mut u8 returned=identity_u8(ref mut data[1]);*returned=*returned+1;
 let ref u8 a=ref data[0];let ref u8 b=ref data[1];
 assert(*a==1 && *b==10 && data[2]==3);
 let u8[1] readonly=[9];let ref u8 shared=ref readonly[0];assert(*shared==9);
}
fn ref mut i32 identity_i32(ref mut i32 value){return value;}
fn void edit_i32(ref mut i32 value){*value=*value+2;}
fn void check_i32(){
 var i32[3] data=[1,2147483647,3];
 let ref mut i32 item=ref mut ((data)[1]);
 let i32 copy=*item;*item=7;
 assert(copy==2147483647 && *item==7);
 edit_i32(ref mut data[1]);
 let ref mut i32 returned=identity_i32(ref mut data[1]);*returned=*returned+1;
 let ref i32 a=ref data[0];let ref i32 b=ref data[1];
 assert(*a==1 && *b==10 && data[2]==3);
 let i32[1] readonly=[9];let ref i32 shared=ref readonly[0];assert(*shared==9);
}
fn ref mut i64 identity_i64(ref mut i64 value){return value;}
fn void edit_i64(ref mut i64 value){*value=*value+2;}
fn void check_i64(){
 var i64[3] data=[1,4000000000,3];
 let ref mut i64 item=ref mut ((data)[1]);
 let i64 copy=*item;*item=7;
 assert(copy==4000000000 && *item==7);
 edit_i64(ref mut data[1]);
 let ref mut i64 returned=identity_i64(ref mut data[1]);*returned=*returned+1;
 let ref i64 a=ref data[0];let ref i64 b=ref data[1];
 assert(*a==1 && *b==10 && data[2]==3);
 let i64[1] readonly=[9];let ref i64 shared=ref readonly[0];assert(*shared==9);
}
fn ref mut u32 identity_u32(ref mut u32 value){return value;}
fn void edit_u32(ref mut u32 value){*value=*value+2;}
fn void check_u32(){
 var u32[3] data=[1,4294967295,3];
 let ref mut u32 item=ref mut ((data)[1]);
 let u32 copy=*item;*item=7;
 assert(copy==4294967295 && *item==7);
 edit_u32(ref mut data[1]);
 let ref mut u32 returned=identity_u32(ref mut data[1]);*returned=*returned+1;
 let ref u32 a=ref data[0];let ref u32 b=ref data[1];
 assert(*a==1 && *b==10 && data[2]==3);
 let u32[1] readonly=[9];let ref u32 shared=ref readonly[0];assert(*shared==9);
}
fn ref mut usize identity_usize(ref mut usize value){return value;}
fn void edit_usize(ref mut usize value){*value=*value+2;}
fn void check_usize(){
 var usize[3] data=[1,4294967296,3];
 let ref mut usize item=ref mut ((data)[1]);
 let usize copy=*item;*item=7;
 assert(copy==4294967296 && *item==7);
 edit_usize(ref mut data[1]);
 let ref mut usize returned=identity_usize(ref mut data[1]);*returned=*returned+1;
 let ref usize a=ref data[0];let ref usize b=ref data[1];
 assert(*a==1 && *b==10 && data[2]==3);
 let usize[1] readonly=[9];let ref usize shared=ref readonly[0];assert(*shared==9);
}
fn ref mut i64 tagged_identity(ref mut i64 value){return value;}
fn i32 main(){
 check_u8();check_i32();check_i64();check_u32();check_usize();
 var i64[2] data=[4000000000,7];var usize calls=0;
 let ref mut i64 item=ref mut data[{calls+=1;1}];*item=42;
 assert(calls==1 && data[0]==4000000000 && data[1]==42);
 let Option<ref mut i64> result=.some(tagged_identity(ref mut data[0]));
 switch(result){case .some(value):{*value=*value+1;}case .none:{assert(false);}};
 assert(data[0]==4000000001);
 var i64[1] before=[7];var i64[1] after=[8];
 var ref mut i64 receiver=ref mut before[0];let ref mut i64 other=ref mut after[0];
 *receiver={receiver=other;99};
 assert(before[0]==99 && after[0]==8);
 return 0;
}
