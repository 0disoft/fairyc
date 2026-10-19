module native_msvc_artifacts;
import std::bytes as bytes;
import std::fs as fs;
import std::text as text;

pub struct Problem{u32 stage;u32 status;u32 os_code;}
fn Problem fail(u32 stage,u32 status,u32 os_code){return Problem{.stage,.status,.os_code};}
@effects(filesystem) pub fn Result<bytes::Buffer,Problem> read(ref u8[] path){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(fail(1,1,0));}};var fs::File file=switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(e):{return .err(fail(1,e.status,e.os_code));}};
    let usize observed=switch(fs::size(ref file)){case .ok(n):{n}case .err(e):{return .err(fail(2,e.status,e.os_code));}};let kind=switch(fs::kind(ref file)){case .ok(k):{k}case .err(e):{return .err(fail(2,e.status,e.os_code));}};
    if(!fs::kind_is_disk(kind) || observed==0 || observed>67108864){return .err(fail(2,1,0));}var bytes::Buffer data=bytes::empty();
    while(true){let usize used=bytes::len(ref data);let usize maximum=if(67108865-used<65536){67108865-used}else{65536};let usize n=switch(fs::read_append(ref mut file,ref mut data,maximum)){case .ok(n):{n}case .err(e):{return .err(fail(3,if(e.file_status!=0){e.file_status}else{e.allocation_status},e.os_code));}};if(bytes::len(ref data)>67108864){return .err(fail(2,1,0));}if(n==0){break;}}
    if(bytes::len(ref data)!=observed){return .err(fail(2,1,0));}switch(fs::close(file)){case .ok(_):{}case .err(e):{return .err(fail(4,e.status,e.os_code));}};return .ok(data);
}
fn usize half(ref u8[] data,usize at){return (data[at] as usize)+(data[at+1] as usize)*256;}
fn usize word(ref u8[] data,usize at){return half(data,at)+half(data,at+2)*65536;}
fn bool range(ref u8[] data,usize offset,usize length){return offset<=data.len && length<=data.len-offset;}
// Bounded transport checks, not yet the complete pecoff COMDAT/unwind proof.
// All artifacts in the integration harness also pass the existing Go inspector.
pub fn bool object(ref u8[] data){return object_problem(data)==0;}
// Header, sections, symbols, strings, raw bytes, overflow count, relocations.
pub fn u32 object_problem(ref u8[] data){
    if(data.len<20 || data.len>67108864 || half(data,0)!=34404 || half(data,16)!=0){return 1;}let usize count=half(data,2);if(count==0 || count>16384 || !range(data,20,count*40)){return 2;}
    let usize symbols=word(data,8);let usize entries=word(data,12);if(entries>262144 || entries>0 && (symbols<20+count*40 || !range(data,symbols,entries*18+4))){return 3;}
    if(entries>0){let usize strings=symbols+entries*18;let usize length=word(data,strings);if(length<4 || !range(data,strings,length)){return 4;}}
    var usize i=0;var usize relocations=0;var bool code=false;while(i<count){let usize section=20+i*40;let usize size=word(data,section+16);let usize offset=word(data,section+20);let usize rel=word(data,section+24);var usize n=half(data,section+32);let usize flags=word(data,section+36);if(size>0 && (flags&32!=0 || flags&536870912!=0)){code=true;}
        if(size>0 && (offset==0 && flags&128==0 || offset!=0 && (offset<20+count*40 || !range(data,offset,size)))){return 5;}
        if(flags&16777216!=0){if(n!=65535 || !range(data,rel,10)){return 6;}let usize extended=word(data,rel);if(extended<65536){return 6;}n=extended;}
        if(n>1048576-relocations || n>0 && (rel<20+count*40 || !range(data,rel,n*10))){return 7;}relocations+=n;i+=1;
    }return if(!code && count>96){2}else{0};
}
fn bool power(usize value){return value>0 && value&(value-1)==0;}
pub fn bool executable(ref u8[] data){
    if(data.len<64 || data.len>67108864 || data[0]!=77 || data[1]!=90){return false;}let usize pe=word(data,60);if(pe<64 || !range(data,pe,24) || !bytes::equal(data[pe..pe+4],b"PE\0\0") || half(data,pe+4)!=34404){return false;}
    let usize sections=half(data,pe+6);let usize optional=half(data,pe+20);let usize flags=half(data,pe+22);let usize first=pe+24;
    if(sections==0 || sections>96 || optional<240 || flags&2==0 || flags&32==0 || flags&8192!=0 || !range(data,first,optional+sections*40) || half(data,first)!=523 || half(data,first+68)!=3){return false;}
    let usize entry=word(data,first+16);let usize align=word(data,first+32);let usize file_align=word(data,first+36);let usize headers=word(data,first+60);let usize image=word(data,first+56);
    if(entry==0 || (word(data,first+24)==0 && word(data,first+28)==0) || word(data,first+24)%65536!=0 || !power(align) || !power(file_align) || file_align<512 || file_align>65536 || align<file_align || headers==0 || headers>data.len || headers%file_align!=0 || image==0 || image%align!=0){return false;}
    var usize i=0;var usize end=0;var bool entry_found=false;var bool text_found=false;
    while(i<sections){let usize section=first+optional+i*40;let usize virtual_size=word(data,section+8);let usize address=word(data,section+12);let usize size=word(data,section+16);let usize offset=word(data,section+20);let usize attributes=word(data,section+36);let usize span=if(virtual_size>size){virtual_size}else{size};
        if(address==0 || address%align!=0 || address<end || span>image || address>image-span || size>0 && (offset<headers || offset%file_align!=0 || size%file_align!=0 || !range(data,offset,size))){return false;}end=address+span;
        if(attributes&32!=0 && attributes&536870912!=0 && size>0){text_found=true;if(entry>=address && entry-address<size){entry_found=true;}}i+=1;
    }return entry_found && text_found;
}
