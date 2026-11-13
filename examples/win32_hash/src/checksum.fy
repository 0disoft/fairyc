module checksum;
import comparison as comparison;
import hash_file as hash;
import std::bytes as bytes;
import std::fs as fs;

fn bool space(u8 c){return c==32 || (c>=9 && c<=13);}
pub fn bool extension(ref u8[] path){
    let length=path.len;if(length<7){return false;}
    let ref u8[] suffix=b".sha256";
    for(var usize i=0;i<7;i+=1){var u8 c=path[length-7+i];if(c>=65 && c<=90){c+=32;}if(c!=suffix[i]){return false;}}
    return true;
}
// One raw/GNU/BSD record only. Filename is informational, never opened.
pub fn bool parse(ref u8[] data,ref mut u8[] expected){
    var usize start=0;var usize end=data.len;
    if(end>=3 && data[0]==239 && data[1]==187 && data[2]==191){start=3;}
    while(start<end && space(data[start])){start+=1;}
    while(end>start && space(data[end-1])){end-=1;}
    let ref u8[] line=data[start..end];let length=line.len;
    for(var usize i=0;i<length;i+=1){if(line[i]==0 || line[i]==10 || line[i]==13){return false;}}
    if(length==64){return comparison::normalize(line,expected);}
    if(length>66 && line[64]==32 && (line[65]==32 || line[65]==42)){
        return comparison::normalize(line[..64],expected);
    }
    if(length>76 && bytes::equal(line[..8],b"SHA256 (") && bytes::equal(line[length-68..length-64],b") = ")){
        return comparison::normalize(line[length-64..],expected);
    }
    return false;
}
@effects(filesystem)
pub fn Result<i32,u32> read(ref u8[] path,ref mut u8[] expected){
    var file=switch(hash::open(path)){case .ok(v):{v}case .err(e):{return .err(e);}};
    let size=switch(fs::size(ref file)){case .ok(v):{v}case .err(_):{return .err(2);}};
    if(size>65536){return .err(13);}
    var bytes::Buffer content=bytes::empty();
    while(true){
        let length=bytes::view(ref content).len;
        let count=switch(fs::read_append(ref mut file,ref mut content,65537-length)){
            case .ok(v):{v}case .err(_):{return .err(2);}
        };
        if(bytes::view(ref content).len>65536){return .err(13);}
        if(count==0){break;}
    }
    return if(parse(bytes::view(ref content),expected)){.ok(0)}else{.err(13)};
}
pub fn bool format(ref u8[] path,ref hash::Digest value,ref mut bytes::Buffer output){
    var usize start=0;let length=path.len;
    for(var usize i=0;i<length;i+=1){if(path[i]==47 || path[i]==92){start=i+1;}if(path[i]==0 || path[i]==10 || path[i]==13){return false;}}
    return start<length && bytes::clear(output).is_ok && hash::append_hex(value,output) &&
        bytes::append(output,b"  ").is_ok && bytes::append(output,path[start..]).is_ok && bytes::append_byte(output,10).is_ok;
}
