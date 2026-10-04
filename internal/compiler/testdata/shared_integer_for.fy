module main;
fn Option<ref u8> find_u8(ref u8[] values, u8 needle){
 for(let ref u8 value in values){if(*value==needle){return .some(value);}}
 return .none;
}
fn void check_u8(){
 var u8[3] data=[1,200,3];var usize visits=0;
 for(let ref u8 item in data){visits+=1;assert(*item==data[visits-1]);}
 assert(visits==3 && data[1]==200);
 let ref mut u8[] exclusive=ref mut data;
 for(let ref u8 item in exclusive){let u8 copy=*item;assert(copy==*item);}
 exclusive[0]=2;
 let ref u8[] values=ref data;
 visits=0;for(let ref u8 item in values){visits+=1;if(*item==2){continue;}if(*item==200){break;}}
 assert(visits==2);
 let ref u8[] part=values[1..];switch(find_u8(part,200)){case .some(item):{assert(*item==200);}case .none:{assert(false);}};
 let ref u8[] empty=values[3..];switch(find_u8(empty,1)){case .some(_):{assert(false);}case .none:{}};
 switch(find_u8(values,9)){case .some(_):{assert(false);}case .none:{}};
 var Option<ref u8> last=.none;for(let ref u8 item in data){last=.some(item);}
 switch(last){case .some(item):{assert(*item==3);}case .none:{assert(false);}};
 data[0]=7;assert(data[0]==7);
}
fn Option<ref i32> find_i32(ref i32[] values, i32 needle){
 for(let ref i32 value in values){if(*value==needle){return .some(value);}}
 return .none;
}
fn void check_i32(){
 var i32[3] data=[1,2000000000,3];var usize visits=0;
 for(let ref i32 item in data){visits+=1;assert(*item==data[visits-1]);}
 assert(visits==3 && data[1]==2000000000);
 let ref mut i32[] exclusive=ref mut data;
 for(let ref i32 item in exclusive){let i32 copy=*item;assert(copy==*item);}
 exclusive[0]=2;
 let ref i32[] values=ref data;
 visits=0;for(let ref i32 item in values){visits+=1;if(*item==2){continue;}if(*item==2000000000){break;}}
 assert(visits==2);
 let ref i32[] part=values[1..];switch(find_i32(part,2000000000)){case .some(item):{assert(*item==2000000000);}case .none:{assert(false);}};
 let ref i32[] empty=values[3..];switch(find_i32(empty,1)){case .some(_):{assert(false);}case .none:{}};
 switch(find_i32(values,9)){case .some(_):{assert(false);}case .none:{}};
 var Option<ref i32> last=.none;for(let ref i32 item in data){last=.some(item);}
 switch(last){case .some(item):{assert(*item==3);}case .none:{assert(false);}};
 data[0]=7;assert(data[0]==7);
}
fn Option<ref i64> find_i64(ref i64[] values, i64 needle){
 for(let ref i64 value in values){if(*value==needle){return .some(value);}}
 return .none;
}
fn void check_i64(){
 var i64[3] data=[1,4000000000,3];var usize visits=0;
 for(let ref i64 item in data){visits+=1;assert(*item==data[visits-1]);}
 assert(visits==3 && data[1]==4000000000);
 let ref mut i64[] exclusive=ref mut data;
 for(let ref i64 item in exclusive){let i64 copy=*item;assert(copy==*item);}
 exclusive[0]=2;
 let ref i64[] values=ref data;
 visits=0;for(let ref i64 item in values){visits+=1;if(*item==2){continue;}if(*item==4000000000){break;}}
 assert(visits==2);
 let ref i64[] part=values[1..];switch(find_i64(part,4000000000)){case .some(item):{assert(*item==4000000000);}case .none:{assert(false);}};
 let ref i64[] empty=values[3..];switch(find_i64(empty,1)){case .some(_):{assert(false);}case .none:{}};
 switch(find_i64(values,9)){case .some(_):{assert(false);}case .none:{}};
 var Option<ref i64> last=.none;for(let ref i64 item in data){last=.some(item);}
 switch(last){case .some(item):{assert(*item==3);}case .none:{assert(false);}};
 data[0]=7;assert(data[0]==7);
}
fn Option<ref u32> find_u32(ref u32[] values, u32 needle){
 for(let ref u32 value in values){if(*value==needle){return .some(value);}}
 return .none;
}
fn void check_u32(){
 var u32[3] data=[1,4000000000,3];var usize visits=0;
 for(let ref u32 item in data){visits+=1;assert(*item==data[visits-1]);}
 assert(visits==3 && data[1]==4000000000);
 let ref mut u32[] exclusive=ref mut data;
 for(let ref u32 item in exclusive){let u32 copy=*item;assert(copy==*item);}
 exclusive[0]=2;
 let ref u32[] values=ref data;
 visits=0;for(let ref u32 item in values){visits+=1;if(*item==2){continue;}if(*item==4000000000){break;}}
 assert(visits==2);
 let ref u32[] part=values[1..];switch(find_u32(part,4000000000)){case .some(item):{assert(*item==4000000000);}case .none:{assert(false);}};
 let ref u32[] empty=values[3..];switch(find_u32(empty,1)){case .some(_):{assert(false);}case .none:{}};
 switch(find_u32(values,9)){case .some(_):{assert(false);}case .none:{}};
 var Option<ref u32> last=.none;for(let ref u32 item in data){last=.some(item);}
 switch(last){case .some(item):{assert(*item==3);}case .none:{assert(false);}};
 data[0]=7;assert(data[0]==7);
}
fn Option<ref usize> find_usize(ref usize[] values, usize needle){
 for(let ref usize value in values){if(*value==needle){return .some(value);}}
 return .none;
}
fn void check_usize(){
 var usize[3] data=[1,4294967296,3];var usize visits=0;
 for(let ref usize item in data){visits+=1;assert(*item==data[visits-1]);}
 assert(visits==3 && data[1]==4294967296);
 let ref mut usize[] exclusive=ref mut data;
 for(let ref usize item in exclusive){let usize copy=*item;assert(copy==*item);}
 exclusive[0]=2;
 let ref usize[] values=ref data;
 visits=0;for(let ref usize item in values){visits+=1;if(*item==2){continue;}if(*item==4294967296){break;}}
 assert(visits==2);
 let ref usize[] part=values[1..];switch(find_usize(part,4294967296)){case .some(item):{assert(*item==4294967296);}case .none:{assert(false);}};
 let ref usize[] empty=values[3..];switch(find_usize(empty,1)){case .some(_):{assert(false);}case .none:{}};
 switch(find_usize(values,9)){case .some(_):{assert(false);}case .none:{}};
 var Option<ref usize> last=.none;for(let ref usize item in data){last=.some(item);}
 switch(last){case .some(item):{assert(*item==3);}case .none:{assert(false);}};
 data[0]=7;assert(data[0]==7);
}
fn i64 total(i64[3] values){var i64 sum=0;for(let ref i64 item in values){sum+=*item;}return sum;}
fn Result<i32,u32> fail(){return .err(7);}
fn Result<i32,u32> propagation(ref i64[] values){for(let ref i64 item in values){assert(*item>0);let i32 unused=fail()?;}return .ok(0);}
fn void nested(){let i64[2] data=[1,2];var usize visits=0;for(let ref i64 a in data){for(let ref i64 b in data){assert(*a>0 && *b>0);visits+=1;}}assert(visits==4);}
fn void shadow(){let i64[1] data=[7];for(let ref i64 data in data){assert(*data==7);}}
fn void static_bytes(){let ref u8[] values=b"ab";var u32 sum=0;for(let ref u8 item in values){sum+=*item as u32;}assert(sum==195);}
fn i32 main(){check_u8();check_i32();check_i64();check_u32();check_usize();nested();shadow();static_bytes();assert(total([1,4000000000,3])==4000000004);var i64[1] data=[1];let ref i64[] values=ref data;switch(propagation(values)){case .ok(_):{assert(false);}case .err(code):{assert(code==7);}};data[0]=9;return 0;}
