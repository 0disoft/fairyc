module native_run_control;
import std::bytes as bytes;
import std::text as text;
import project_json_string as quoted;
pub struct Outcome{u32 status;u32 trap;u32 pid;}
pub fn ref u8[] category(u32 trap){let ref u8[] names=b"boundsinteger-overflowdivision-by-zerosigned-division-overflowinvalid-shiftassertion-failedunreachable";return if(trap==1){names[..6]}else if(trap==2){names[6..22]}else if(trap==3){names[22..38]}else if(trap==4){names[38..62]}else if(trap==5){names[62..75]}else if(trap==6){names[75..91]}else if(trap==7){names[91..102]}else{names[..0]};}
pub fn Result<Outcome,u32> classify(u32 exit_code,ref u8[] receipt,u32 pid){
    if(receipt.len==0){return .ok(Outcome{.status=exit_code,.trap=0,.pid});}
    if(exit_code<129 || exit_code>135 || (receipt[0] as u32)!=exit_code-128){return .err(10);}
    if(receipt.len!=1){
        if(receipt.len<21 || receipt.len>max_receipt_bytes() || receipt[1]!=1){return .err(10);}
        let length=(receipt[2] as usize)+(receipt[3] as usize)*256;
        if(length==0 || length>256 || receipt.len!=20+length || location_start(receipt)>location_end(receipt)){return .err(10);}
        let source=location_source(receipt);if(!text::validate_utf8(source).is_ok){return .err(10);}
        var usize i=0;while(i<source.len){if(source[i]==0){return .err(10);}i+=1;}
    }
    return .ok(Outcome{.status=5,.trap=canonical(exit_code-128),.pid});
}
// The runtime's v1 location record is a 20-byte LE header and up to 256 UTF-8 bytes.
// Accessors are used only after classify accepts the record; legacy records have no location.
pub fn usize max_receipt_bytes(){return 276;}
fn usize offset(ref u8[] data,usize at){var usize value=0;var usize i=8;while(i>0){i-=1;value=value*256+(data[at+i] as usize);}return value;}
pub fn ref u8[] location_source(ref u8[] data){return if(data.len>20){data[20..]}else{data[..0]};}
pub fn usize location_start(ref u8[] data){return if(data.len>20){offset(data,4)}else{0};}
pub fn usize location_end(ref u8[] data){return if(data.len>20){offset(data,12)}else{0};}
// Use the compilation snapshot, with Unicode columns and an end-exclusive byte range.
// Unknown or invalid locations keep the category-only diagnostic.
pub fn bool append_location(ref u8[] path,ref u8[] source,usize start,usize end,ref mut bytes::Buffer out){
    if(path.len==0 || start>end || end>source.len || !text::validate_utf8(source).is_ok){return true;}
    if(start<source.len && source[start]>=128 && source[start]<192 || end<source.len && source[end]>=128 && source[end]<192){return true;}
    var usize line=1;var usize column=1;var usize i=0;while(i<start){if(source[i]==10){line+=1;column=1;}else if(source[i]<128 || source[i]>=192){column+=1;}i+=1;}
    var bool escaped=false;i=0;while(i<path.len){if(path[i]==9 || path[i]==10 || path[i]==13){escaped=true;}i+=1;}
    if(!bytes::append(out,b"  at ").is_ok){return false;}
    if(escaped){var usize work=0;let display=switch(quoted::quote_diagnostic(path,ref mut work)){case .ok(p):{p}case .err(_):{return false;}};if(!bytes::append(out,bytes::view(ref display)).is_ok){return false;}}
    else if(!bytes::append(out,path).is_ok){return false;}
    return bytes::append_byte(out,58).is_ok && bytes::append_usize(out,line).is_ok && bytes::append_byte(out,58).is_ok && bytes::append_usize(out,column).is_ok && bytes::append_byte(out,10).is_ok;
}
// The private candidate C emitter's established tags are overflow=1,
// division=2, signed division=3, shift=4, assertion=5, bounds=6, unreachable=7.
// Only authenticated receipts are normalized to public category numbering.
pub fn u32 canonical(u32 raw){return if(raw==6){1}else if(raw>=1 && raw<=5){raw+1}else if(raw==7){7}else{0};}
