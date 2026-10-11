module project_module_path;
import std::bytes as bytes;
import project_path as work;

fn bool initial(u8 byte){return byte==95 || byte>=65 && byte<=90 || byte>=97 && byte<=122;}
fn bool continuation(u8 byte){return initial(byte) || byte>=48 && byte<=57;}
// Input is an exact canonical relative path from the source catalog. No path
// cleaning or filesystem case normalization is allowed at this binding step.
// 1 invalid Stage 0 source path, 2 allocation, 3 path/work bound.
pub fn Result<bytes::Buffer,u32> expected(ref u8[] path,ref mut usize meter){
    if(path.len>4095 || !work::charge(meter,path.len+1)){return .err(3);}if(path.len<4 || !bytes::equal(path[path.len-3..],b".fy")){return .err(1);}
    let usize end=path.len-3;var usize at=0;var bytes::Buffer output=bytes::empty();
    while(at<end){let usize start=at;if(!initial(path[at])){return .err(1);}at+=1;while(at<end && continuation(path[at])){at+=1;}
        if(!bytes::append(ref mut output,path[start..at]).is_ok){return .err(2);}if(at==end){return .ok(output);}if(path[at]!=47 || at+1==end){return .err(1);}if(!bytes::append(ref mut output,b"::").is_ok){return .err(2);}at+=1;
    }return .err(1);
}
