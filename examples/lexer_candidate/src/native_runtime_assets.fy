module native_runtime_assets;
import native_runtime_asset_pin as pin;
import sha256 as sha;
import native_runtime as selection;
import std::bytes as bytes;

// An explicitly supplied distribution asset. No cwd/path search, fallback,
// generated C input, or Go child process. Offsets are trusted only after hash.
pub resource Bundle{bytes::Buffer data;cleanup{}}
pub struct Problem{u32 kind;u32 symbol_kind;usize start;usize end;}
// 1 invalid asset, 2 symbol admission, 3 allocation/bound, 4 scalar template.
fn Problem fail(u32 kind){return Problem{.kind,.symbol_kind=0,.start=0,.end=0};}
pub fn Result<Bundle,Problem> load(bytes::Buffer data){
    let input=bytes::view(ref data);if(input.len!=pin::size() || input.len>1048576 || !sha::matches(input,pin::digest())){return .err(fail(1));}return .ok(Bundle{.data});
}
fn ref u8[] raw(ref Bundle bundle,usize id){assert(id>=1 && id<=20);return bytes::view(ref bundle.data)[pin::start(id)..pin::end(id)];}
fn bool at(ref u8[] data,usize i,ref u8[] prefix){return i<=data.len && prefix.len<=data.len-i && bytes::equal(data[i..i+prefix.len],prefix);}
fn Result<usize,Problem> append(ref mut bytes::Buffer output,ref u8[] value){
    if(value.len>8388608 || bytes::len(output)>8388608-value.len || !bytes::append(output,value).is_ok){return .err(fail(3));}return .ok(bytes::len(output));
}
fn Result<usize,Problem> scalar(ref Bundle bundle,bool trap,ref mut bytes::Buffer output){
    let source=raw(bundle,1);var usize i=0;var usize start=0;var u32 seen=0;
    while(i<source.len){var u32 bit=0;var usize size=0;
        if(at(source,i,b"@TRAP_EXIT_BASE@")){bit=1;size=16;}
        else if(at(source,i,b"@TRAP_RECEIPT_INCLUDE@")){bit=2;size=22;}
        else if(at(source,i,b"@TRAP_RECEIPT_WRITE@")){bit=4;size=20;}
        if(bit!=0){if((seen&bit)!=0){return .err(fail(4));}seen=seen|bit;let usize copied=append(output,source[start..i])?;
            if(bit==1){let usize n=append(output,if(trap){b"128"}else{b"100"})?;}
            else if(bit==2 && trap){let usize n=append(output,b"#include <windows.h>\n")?;}
            else if(bit==4 && trap){let usize n=append(output,raw(bundle,2))?;}
            i+=size;start=i;
        }else{i+=1;}
    }
    if(seen!=7){return .err(fail(4));}return append(output,source[start..]);
}
fn Result<usize,Problem> part(ref Bundle bundle,usize id,ref mut bytes::Buffer output,ref mut bool written){
    if(*written){let usize separator=append(output,b"\n")?;};*written=true;return append(output,raw(bundle,id));
}
pub fn Result<bytes::Buffer,Problem> source(ref Bundle bundle,ref u8[] header,bool trap){
    let plan=switch(selection::select(header,trap)){case .ok(p):{p}case .err(e):{return .err(Problem{.kind=2,.symbol_kind=e.kind,.start=e.start,.end=e.end});}};
    var bytes::Buffer output=bytes::empty();let usize base=scalar(bundle,trap,ref mut output)?;let usize separator=append(ref mut output,b"\n")?;
    if(selection::contains(ref plan,7)){let usize alignment=append(ref mut output,b"#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)\n")?;}
    var bool written=false;var u32 id=1;
    while(id<=14){if(selection::contains(ref plan,id)){
        let usize asset=if(id==3){3}else if(id==5){9}else if(id==6){10}else if(id==7){11}else if(id==8){12}else if(id==9){13}else if(id==10){14}else if(id==11){15}else if(id==12){16}else if(id==14){17}else{0};
        if(id==4){var usize file=4;while(file<=8){let usize n=part(bundle,file,ref mut output,ref mut written)?;file+=1;}}
        else if(asset!=0){let usize n=part(bundle,asset,ref mut output,ref mut written)?;}
        else{return .err(fail(1));}
    }id+=1;}
    if(plan.vectors){let usize n=part(bundle,18,ref mut output,ref mut written)?;}
    if(plan.host>0){let usize n=append(ref mut output,b"\n")?;let usize host=append(ref mut output,raw(bundle,19))?;if(plan.host==2){let usize gap=append(ref mut output,b"\n")?;let usize rooted=append(ref mut output,raw(bundle,20))?;}}
    return .ok(output);
}
