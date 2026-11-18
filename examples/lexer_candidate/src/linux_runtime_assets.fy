module linux_runtime_assets;
import linux_runtime_asset_pin as pin;
import linux_runtime as selection;
import sha256 as sha;
import std::bytes as bytes;

pub resource Bundle{bytes::Buffer data;cleanup{}}
// 1 asset hash/size, 2 symbol admission, 3 allocation/bound, 4 template.
pub struct Problem{u32 kind;u32 symbol_kind;usize start;usize end;}
fn Problem fail(u32 kind){return Problem{.kind,.symbol_kind=0,.start=0,.end=0};}
pub fn Result<Bundle,Problem> load(bytes::Buffer data){
    let input=bytes::view(ref data);if(input.len!=pin::size() || input.len>1048576 || !sha::matches(input,pin::digest())){return .err(fail(1));}return .ok(Bundle{.data});
}
fn ref u8[] raw(ref Bundle bundle,usize id){assert(id>=1 && id<=pin::count());return bytes::view(ref bundle.data)[pin::start(id)..pin::end(id)];}
fn bool at(ref u8[] data,usize i,ref u8[] prefix){return i<=data.len && prefix.len<=data.len-i && bytes::equal(data[i..i+prefix.len],prefix);}
fn Result<usize,Problem> append(ref mut bytes::Buffer output,ref u8[] data){if(data.len>8388608 || bytes::len(output)>8388608-data.len || !bytes::append(output,data).is_ok){return .err(fail(3));}return .ok(bytes::len(output));}
fn Result<usize,Problem> part(ref Bundle bundle,usize id,ref mut bytes::Buffer output){if(bytes::len(output)!=0){let usize gap=append(output,b"\n")?;}return append(output,raw(bundle,id));}
fn Result<usize,Problem> hosts(ref Bundle bundle,ref selection::Plan plan,usize first,usize last,u32 bit,ref mut bytes::Buffer output){
    var usize i=first;var u32 mask=bit;while(i<=last){if((plan.hosts&mask)!=0){let usize n=part(bundle,i,output)?;}mask*=2;i+=1;}return .ok(bytes::len(output));
}
fn Result<usize,Problem> scalar(ref Bundle bundle,bool trap,ref mut bytes::Buffer output){
    if(bytes::len(output)!=0){let usize gap=append(output,b"\n")?;}let source=raw(bundle,1);var usize i=0;var usize first=0;var u32 seen=0;
    while(i<source.len){var u32 bit=0;var usize size=0;if(at(source,i,b"@TRAP_EXIT_BASE@")){bit=1;size=16;}else if(at(source,i,b"@TRAP_RECEIPT_INCLUDE@")){bit=2;size=22;}else if(at(source,i,b"@TRAP_RECEIPT_WRITE@")){bit=4;size=20;}
        if(bit==0){i+=1;}else{if((seen&bit)!=0){return .err(fail(4));}seen|=bit;let usize copied=append(output,source[first..i])?;
            if(bit==1){let usize n=append(output,b"128")?;}
            else if(bit==2 && trap){let usize n=append(output,b"#include <errno.h>\n#include <unistd.h>\n")?;}
            else if(bit==4 && trap){let usize n=append(output,b"    size_t fy_trap_offset = 0;\n    while (fy_trap_offset < fy_trap_receipt_length) {\n        ssize_t fy_trap_written = write(3, fy_trap_receipt + fy_trap_offset, fy_trap_receipt_length - fy_trap_offset);\n        if (fy_trap_written > 0) { fy_trap_offset += (size_t)fy_trap_written; continue; }\n        if (fy_trap_written == -1 && errno == EINTR) { continue; }\n        _Exit(254);\n    }\n")?;}
            i+=size;first=i;
        }
    }if(seen!=7){return .err(fail(4));}return append(output,source[first..]);
}
pub fn Result<bytes::Buffer,Problem> source(ref Bundle bundle,ref u8[] header,bool trap){
    let plan=switch(selection::select(header)){case .ok(p):{p}case .err(e):{return .err(Problem{.kind=2,.symbol_kind=e.kind,.start=e.start,.end=e.end});}};
    var bytes::Buffer output=bytes::empty();
    // Match LinuxSource exactly: HTTPS feature selection precedes all headers,
    // package adapters follow directory, and clock precedes the HTTPS quartet.
    // part() supplies the prefix's final newline before the first host source.
    if((plan.hosts&(1<<14))!=0){let usize n=append(ref mut output,b"#ifndef _GNU_SOURCE\n#define _GNU_SOURCE 1\n#endif")?;}
    let usize base=hosts(bundle,ref plan,15,20,1,ref mut output)?;
    let usize packages=hosts(bundle,ref plan,26,27,1<<11,ref mut output)?;
    let usize editing=hosts(bundle,ref plan,33,33,1<<18,ref mut output)?;
    let usize tail=hosts(bundle,ref plan,21,25,1<<6,ref mut output)?;
    let usize remote=hosts(bundle,ref plan,28,32,1<<13,ref mut output)?;
    // Component IDs are in link order. Pack IDs remain stable as assets are appended.
    let usize[14] assets=[2,3,34,35,36,37,5,6,7,8,10,11,12,13];
    var u32 id=1;while(id<=14){if(selection::contains(ref plan,id)){
        let usize asset=assets[(id-1) as usize];
        let usize n=part(bundle,asset,ref mut output)?;if(asset==3 || asset==8){let usize extra=part(bundle,asset+1,ref mut output)?;}
    }id+=1;}if(plan.vectors){let usize n=part(bundle,14,ref mut output)?;}let usize n=scalar(bundle,trap,ref mut output)?;return .ok(output);
}
