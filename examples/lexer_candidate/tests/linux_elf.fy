module tests::linux_elf;
import linux_elf as elf;
import std::bytes as bytes;

fn u8 byte(usize n){var usize i=0;var u8 result=0;while(i<n){result+=1;i+=1;}return result;}
fn void put(ref mut bytes::Buffer data,usize at,usize value){var usize n=value;var usize i=0;while(i<8){bytes::view_mut(data)[at+i]=byte(n%256);n/=256;i+=1;}}
fn void put_half(ref mut bytes::Buffer data,usize at,usize value){bytes::view_mut(data)[at]=byte(value%256);bytes::view_mut(data)[at+1]=byte(value/256);}
fn bytes::Buffer header(bool image){
    var bytes::Buffer data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,if(image){128}else{208}).is_ok);
    let ref u8[] magic=b"\x7fELF";var usize i=0;while(i<4){bytes::view_mut(ref mut data)[i]=magic[i];i+=1;}
    bytes::view_mut(ref mut data)[4]=2;bytes::view_mut(ref mut data)[5]=1;bytes::view_mut(ref mut data)[6]=1;
    put(ref mut data,16,if(image){2}else{1});put(ref mut data,18,62);put(ref mut data,20,1);put(ref mut data,52,64);
    if(image){put(ref mut data,24,4194424);put(ref mut data,32,64);put(ref mut data,54,56);put(ref mut data,56,1);
        put(ref mut data,64,1);put(ref mut data,68,5);put(ref mut data,72,120);put(ref mut data,80,4194424);put(ref mut data,96,8);put(ref mut data,104,8);put(ref mut data,112,1);
    }else{put(ref mut data,40,64);put(ref mut data,58,64);put(ref mut data,60,2);put(ref mut data,132,1);put(ref mut data,152,192);put(ref mut data,160,16);put(ref mut data,176,1);}
    return data;
}
@test fn void validates_relocatable_and_executable_ranges(){
    var object=header(false);assert(elf::object(bytes::view(ref object)) && !elf::executable(bytes::view(ref object)));
    var image=header(true);assert(elf::executable(bytes::view(ref image)) && !elf::object(bytes::view(ref image)));
    put_half(ref mut image,16,3);assert(elf::executable(bytes::view(ref image)));
    put(ref mut image,24,4194432);assert(!elf::executable(bytes::view(ref image)));put(ref mut image,24,4194424);
    put(ref mut image,68,4);assert(!elf::executable(bytes::view(ref image)));
    put(ref mut object,132,8);put(ref mut object,152,99999);assert(elf::object(bytes::view(ref object)));
    put(ref mut object,132,1);assert(!elf::object(bytes::view(ref object)));
}
@test fn void rejects_truncated_foreign_and_overflowed_headers(){
    assert(!elf::object(b"") && !elf::object(b"MZ") && !elf::executable(b"\x7fELF"));
    var object=header(false);var usize i=0;while(i<64){assert(!elf::object(bytes::view(ref object)[..i]));i+=1;}
    bytes::view_mut(ref mut object)[5]=2;assert(!elf::object(bytes::view(ref object)));bytes::view_mut(ref mut object)[5]=1;
    put(ref mut object,18,183);assert(!elf::object(bytes::view(ref object)));put(ref mut object,18,62);put(ref mut object,20,1);
    put(ref mut object,40,2147483647);assert(!elf::object(bytes::view(ref object)));put(ref mut object,40,64);
    put(ref mut object,160,2147483647);assert(!elf::object(bytes::view(ref object)));
    var image=header(true);put(ref mut image,96,9);assert(!elf::executable(bytes::view(ref image)));
    put(ref mut image,96,8);put(ref mut image,112,3);assert(!elf::executable(bytes::view(ref image)));
    put(ref mut image,112,1);bytes::view_mut(ref mut image)[39]=255;assert(!elf::executable(bytes::view(ref image)));
}
