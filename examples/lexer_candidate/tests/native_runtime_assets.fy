module tests::native_runtime_assets;
import native_runtime_assets as assets;
import native_runtime as selection;
import std::bytes as bytes;
import std::fs as fs;
import std::text as text;
import std::process as process;

@effects(filesystem) fn usize number(ref mut fs::File file){var u8[4] raw=[0;4];assert(fs::read_exact(file,ref mut raw).is_ok);return raw[0] as usize+(raw[1] as usize)*256+(raw[2] as usize)*65536+(raw[3] as usize)*16777216;}
@effects(filesystem) fn bytes::Buffer part(ref mut fs::File file){let usize n=number(file);assert(n<=2097152);var bytes::Buffer out=bytes::empty();assert(bytes::resize_zeroed(ref mut out,n).is_ok);assert(fs::read_exact(file,bytes::view_mut(ref mut out)).is_ok);return out;}
@effects(filesystem) fn bytes::Buffer path(){var u8[4096] data=[0;4096];let usize n=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] raw=ref data;return switch(bytes::copy(raw[..n])){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};}
@effects(filesystem) fn fs::File open(ref u8[] path){let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return open(b"");}};return switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return open(b"");}};}
@effects(filesystem) fn bytes::Buffer asset_bytes(){var bytes::Buffer name=path();assert(bytes::append(ref mut name,b".assets").is_ok);var fs::File file=open(bytes::view(ref name));var bytes::Buffer out=bytes::empty();let usize n=switch(fs::size(ref file)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(n<=1048576 && bytes::resize_zeroed(ref mut out,n).is_ok);assert(fs::read_exact(ref mut file,bytes::view_mut(ref mut out)).is_ok);assert(fs::close(file).is_ok);return out;}
@test @effects(filesystem) fn void matches_complete_go_runtime_bytes_and_admission(){
    let bundle=switch(assets::load(asset_bytes())){case .ok(p):{p}case .err(_):{assert(false);return;}};let name=path();var fs::File file=open(bytes::view(ref name));var u8[15] magic=[0;15];assert(fs::read_exact(ref mut file,ref mut magic).is_ok && bytes::equal(ref magic,b"FYCRTPACKTEST1\n"));
    let usize count=number(ref mut file);assert(count>200 && count<1024);var usize i=0;
    while(i<count){let bool trap=number(ref mut file)==1;let bool admitted=number(ref mut file)==1;let header=part(ref mut file);let expected=part(ref mut file);
        switch(assets::source(ref bundle,bytes::view(ref header),trap)){
            case .ok(output):{assert(admitted && bytes::equal(bytes::view(ref output),bytes::view(ref expected)));}
            case .err(e):{assert(!admitted && e.kind==2);var bytes::Buffer diagnostic=bytes::empty();assert(selection::diagnostic(bytes::view(ref header),selection::Problem{.kind=e.symbol_kind,.start=e.start,.end=e.end},ref mut diagnostic));assert(bytes::equal(bytes::view(ref diagnostic),bytes::view(ref expected)));}
        };i+=1;
    }
    var u8[1] end=[0;1];let usize n=switch(fs::read(ref mut file,ref mut end)){case .ok(n):{n}case .err(_):{assert(false);1}};assert(n==0);assert(fs::close(file).is_ok);
}
@test @effects(filesystem) fn void rejects_tampered_reordered_truncated_and_extended_assets(){
    var usize variant=0;while(variant<5){var bytes::Buffer input=asset_bytes();let usize n=bytes::len(ref input);
        if(variant==0){let ref mut u8[] raw=bytes::view_mut(ref mut input);raw[0]=0;}
        else if(variant==1){let ref mut u8[] raw=bytes::view_mut(ref mut input);let u8 saved=raw[1000];raw[1000]=raw[1001];raw[1001]=saved;if(raw[1000]==raw[1001]){raw[1000]=0;}}
        else if(variant==2){assert(bytes::truncate(ref mut input,n-1).is_ok);}
        else if(variant==3){assert(bytes::append_byte(ref mut input,0).is_ok);}
        else{let ref mut u8[] raw=bytes::view_mut(ref mut input);raw[n/2]=0;}
        switch(assets::load(input)){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==1);}};variant+=1;
    }
    switch(assets::load(bytes::empty())){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==1);}};
}
