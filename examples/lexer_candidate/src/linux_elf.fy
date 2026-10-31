module linux_elf;
import std::bytes as bytes;

// Bounded ELF64/LSB/x86-64 inspection for the private GNU companion. This is
// artifact admission, not a linker, symbol resolver or target conformance claim.
fn usize half(ref u8[] data,usize at){return (data[at] as usize)+(data[at+1] as usize)*256;}
fn usize word(ref u8[] data,usize at){return half(data,at)+half(data,at+2)*65536;}
fn bool range(ref u8[] data,usize at,usize size){return at<=data.len && size<=data.len-at;}
fn Result<usize,u32> quad(ref u8[] data,usize at){
    if(!range(data,at,8) || word(data,at+4)>2147483647){return .err(2);}return .ok(word(data,at)+word(data,at+4)*4294967296);
}
fn bool power(usize n){return n>0 && (n&(n-1))==0;}
// 1 identity/size, 2 header/table/unsupported extended numbering, 3 section
// bounds, 4 segment bounds, 5 missing executable entry. All reads are bounded.
pub fn Result<usize,u32> inspect(ref u8[] data,bool image){
    if(data.len<64 || data.len>67108864 || !bytes::equal(data[..4],b"\x7fELF") || data[4]!=2 || data[5]!=1 || data[6]!=1 || half(data,18)!=62 || word(data,20)!=1){return .err(1);}
    let kind=half(data,16);if((image && kind!=2 && kind!=3) || (!image && kind!=1) || half(data,52)!=64){return .err(2);}
    let sections=quad(data,40)?;let count=half(data,60);let names=half(data,62);
    if(count==0 && sections!=0 || count>0 && (sections<64 || half(data,58)!=64 || !range(data,sections,count*64)) || names!=0 && names>=count){return .err(2);}
    var usize work=0;var usize i=0;while(i<count){let at=sections+i*64;let ty=word(data,at+4);let offset=quad(data,at+24)?;let size=quad(data,at+32)?;let alignment=quad(data,at+48)?;
        if(alignment>1 && !power(alignment) || ty!=0 && ty!=8 && !range(data,offset,size)){return .err(3);}
        if((ty==2 || ty==11) && (quad(data,at+56)?!=24 || size%24!=0 || word(data,at+40)>=count)){return .err(3);}
        if(names>0){let table=sections+names*64;let base=quad(data,table+24)?;let length=quad(data,table+32)?;
            if(word(data,table+4)!=3 || !range(data,base,length) || word(data,at)>=length){return .err(3);}var usize cursor=word(data,at);while(cursor<length && data[base+cursor]!=0){if(work==4194304){return .err(3);}work+=1;cursor+=1;}if(cursor==length){return .err(3);}
        }i+=1;
    }
    let programs=quad(data,32)?;let loads=half(data,56);
    if(loads==65535 || loads>0 && (programs<64 || half(data,54)!=56 || !range(data,programs,loads*56))){return .err(2);}
    let entry=quad(data,24)?;var bool executable=false;i=0;
    while(i<loads){let at=programs+i*56;let offset=quad(data,at+8)?;let address=quad(data,at+16)?;let file_size=quad(data,at+32)?;let memory_size=quad(data,at+40)?;let alignment=quad(data,at+48)?;
        if(!range(data,offset,file_size)){return .err(4);}if(word(data,at)==1){
            if(file_size>memory_size || alignment>1 && (!power(alignment) || offset%alignment!=address%alignment)){return .err(4);}
            if((word(data,at+4)&1)!=0 && entry>=address && entry-address<file_size){executable=true;}
        }i+=1;
    }
    if(image && (entry==0 || !executable)){return .err(5);}return .ok(count);
}
pub fn bool object(ref u8[] data){return inspect(data,false).is_ok;}
pub fn bool executable(ref u8[] data){return inspect(data,true).is_ok;}
