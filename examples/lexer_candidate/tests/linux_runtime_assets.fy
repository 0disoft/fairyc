module tests::linux_runtime_assets;
import linux_runtime_assets as assets;
import std::bytes as bytes;
import std::fs as fs;

@effects(filesystem) fn usize number(ref mut fs::File file){var u8[4] data=[0;4];assert(fs::read_exact(file,ref mut data).is_ok);return data[0] as usize+(data[1] as usize)*256+(data[2] as usize)*65536+(data[3] as usize)*16777216;}
@effects(filesystem) fn bytes::Buffer part(ref mut fs::File file){let n=number(file);assert(n<=1048576);var bytes::Buffer data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,n).is_ok && fs::read_exact(file,bytes::view_mut(ref mut data)).is_ok);return data;}
@effects(filesystem) fn bytes::Buffer read(ref str path){var file=switch(fs::open_read_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};let n=switch(fs::size(ref file)){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};assert(n<=1048576);var bytes::Buffer data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,n).is_ok);assert(fs::read_exact(ref mut file,bytes::view_mut(ref mut data)).is_ok);assert(fs::close(file).is_ok);return data;}
@effects(filesystem) fn void compare(ref str pack,ref str cases){
    let bundle=switch(assets::load(read(pack))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var file=switch(fs::open_read_utf8(cases)){case .ok(p):{p}case .err(_):{assert(false);return;}};let count=number(ref mut file);assert(count>=40 && count<=256);var usize i=0;
    while(i<count){let trap=number(ref mut file)==1;let header=part(ref mut file);let expected=part(ref mut file);let result=switch(assets::source(ref bundle,bytes::view(ref header),trap)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref result),bytes::view(ref expected)));i+=1;}
    var u8[1] eof=[0;1];let n=switch(fs::read(ref mut file,ref mut eof)){case .ok(n):{n}case .err(_):{assert(false);1}};assert(n==0);assert(fs::close(file).is_ok);
    var usize variant=0;while(variant<3){var data=read(pack);if(variant==0){bytes::view_mut(ref mut data)[0]=0;}else if(variant==1){let n=bytes::len(ref data);assert(bytes::truncate(ref mut data,n-1).is_ok);}else{assert(bytes::append_byte(ref mut data,0).is_ok);}assert(!assets::load(data).is_ok);variant+=1;}
}
