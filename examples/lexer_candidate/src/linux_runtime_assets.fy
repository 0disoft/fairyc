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
    var bytes::Buffer output=bytes::empty();var u32 bit=1;var usize i=15;
    // Feature macros from rooted/process adapters precede every libc include.
    while(i<=25){if((plan.hosts&bit)!=0){let usize n=part(bundle,i,ref mut output)?;}bit*=2;i+=1;}
    var u32 id=1;while(id<=10){if(selection::contains(ref plan,id)){
        let usize asset=if(id==1){2}else if(id==2){3}else if(id==3){5}else if(id==4){6}else if(id==5){7}else if(id==6){8}else if(id==7){10}else if(id==8){11}else if(id==9){12}else{13};
        let usize n=part(bundle,asset,ref mut output)?;if(id==2 || id==6){let usize extra=part(bundle,asset+1,ref mut output)?;}
    }id+=1;}if(plan.vectors){let usize n=part(bundle,14,ref mut output)?;}let usize n=scalar(bundle,trap,ref mut output)?;return .ok(output);
}
