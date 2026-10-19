module native_packet_decode;
import std::bytes as bytes;
import std::vec as v;
import std::fs as fs;
import std::text as text;
import sha256 as sha;

// Private FYCPARTS1 reader. Bounds and recognized C records mirror
// internal/candidatebuild.Decode; this is not a C parser or a sandbox.
// Decode kinds: 1 packet bound/magic, 2 number, 3 part, 4 unit count,
// 5 trailing bytes, 6 C markers, 7 shared storage, 8 literal order/count,
// 9 literal byte, 10 literal digest, 11 frame order/bound, 12 entry, 13 allocation.
pub struct DecodeProblem {u32 kind;usize offset;}
struct Part {usize start;usize end;}
pub resource Packet {bytes::Buffer data;Part header;Part literals;v::Vec<Part> units;cleanup{}}
fn DecodeProblem invalid(u32 kind,usize offset){return DecodeProblem{.kind,.offset};}
fn bool at(ref u8[] data,usize start,ref u8[] prefix){return start<=data.len && prefix.len<=data.len-start && bytes::equal(data[start..start+prefix.len],prefix);}
fn usize find(ref u8[] data,usize start,ref u8[] pattern){
    var usize i=start;while(i<data.len){if(data[i]==pattern[0] && at(data,i,pattern)){return i;}i+=1;}return data.len;
}
fn Result<usize,DecodeProblem> read_number(ref u8[] data,ref mut usize cursor){
    let usize first=*cursor;var usize value=0;
    while(*cursor<data.len && data[*cursor]!=10){
        let u8 digit=data[*cursor];if(*cursor-first>=8 || digit<48 || digit>57 || *cursor>first && data[first]==48){return .err(invalid(2,first));}
        value=value*10+(digit-48) as usize;*cursor+=1;
    }
    if(*cursor==first || *cursor==data.len){return .err(invalid(2,first));};*cursor+=1;return .ok(value);
}
fn Result<Part,DecodeProblem> read_part(ref u8[] data,ref mut usize cursor){
    let usize count=read_number(data,cursor)?;let usize first=*cursor;
    if(count==0 || count>2097152 || count>data.len-first){return .err(invalid(3,first));};*cursor+=count;return .ok(Part{.start=first,.end=*cursor});
}
fn bool hex_identity(ref u8[] data,usize first){
    if(first>data.len || data.len-first<64){return false;}var usize i=0;
    while(i<64){let u8 c=data[first+i];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}i+=1;}return true;
}
fn bool identity_after(ref u8[] data,usize previous,usize current){
    var usize i=0;while(i<64){if(data[previous+i]!=data[current+i]){return data[previous+i]<data[current+i];}i+=1;}return false;
}
fn usize declaration(ref u8[] header,usize first){
    let ref u8[] prefix=b"extern const uint8_t fy_unit_literal_";var usize cursor=first;
    while(cursor<header.len){let usize found=find(header,cursor,prefix);if(found==header.len){return found;}let usize digest=found+prefix.len;
        if(hex_identity(header,digest) && at(header,digest+64,b"[];")){return digest;}cursor=found+1;
    }return header.len;
}
fn Result<usize,DecodeProblem> literal_values(ref u8[] line,usize first,ref mut bytes::Buffer payload){
    if(!bytes::clear(payload).is_ok){return .err(invalid(13,first));}var usize cursor=first;let usize stop=line.len-2;
    while(cursor<stop){var u8 value=0;let usize begin=cursor;
        while(cursor<stop && line[cursor]!=44){let u8 digit=line[cursor]-48;if(value>25 || value==25 && digit>5){return .err(invalid(9,cursor));}value=value*10+digit;cursor+=1;}
        if(cursor==begin || !bytes::append_byte(payload,value).is_ok){return .err(invalid(if(cursor==begin){9}else{13},cursor));}
        if(cursor<stop){cursor+=1;if(cursor==stop){return .err(invalid(9,cursor));}}
    }return .ok(bytes::len(payload));
}
fn Result<usize,DecodeProblem> shared_literals(ref u8[] header,ref u8[] literals){
    if(find(header,0,b"static const uint8_t fy_unit_literal_")<header.len){return .err(invalid(7,0));}
    let ref u8[] prefix=b"const uint8_t fy_unit_literal_";var usize declaration_cursor=0;var usize previous=0;var usize count=0;var usize cursor=0;var bytes::Buffer payload=bytes::empty();
    while(cursor<literals.len){let usize first=cursor;while(cursor<literals.len && literals[cursor]!=10){cursor+=1;}let line=literals[first..cursor];if(cursor<literals.len){cursor+=1;}
        let usize digest=prefix.len;let usize values=digest+68;
        if(!at(line,0,prefix) || !hex_identity(line,digest) || !at(line,digest+64,b"[]={") || line.len<=values+2 || !at(line,line.len-2,b"};")){continue;}
        // Match the same anchored [0-9,]+ records as the Go decoder. Other C
        // text is left to the native compiler, just as in that decoder.
        var bool recognized=true;var usize i=values;while(i<line.len-2){let u8 c=line[i];if(!(c>=48 && c<=57 || c==44)){recognized=false;break;}i+=1;}if(!recognized){continue;}
        let usize declared=declaration(header,declaration_cursor);if(declared==header.len || !bytes::equal(header[declared..declared+64],line[digest..digest+64]) || count>0 && !identity_after(literals,previous,first+digest)){return .err(invalid(8,first));}
        declaration_cursor=declared+67;previous=first+digest;count+=1;let usize length=literal_values(line,values,ref mut payload)?;
        if(!sha::matches(bytes::view(ref payload),line[digest..digest+64]) && !(length==1 && bytes::view(ref payload)[0]==0 && sha::matches(b"",line[digest..digest+64]))){return .err(invalid(10,first));}
    }
    if(declaration(header,declaration_cursor)<header.len){return .err(invalid(8,header.len));}return .ok(count);
}
fn Result<usize,DecodeProblem> frame_line(ref u8[] line,usize last){
    if(!at(line,0,b"extern ")){return .ok(last);}let ref u8[] prefix=b" fy_fn_";var usize cursor=8;var usize selected=line.len;var usize stop=line.len;var usize close=line.len;var usize i=0;
    while(i+1<line.len){if(line[i]==41 && line[i+1]==123){close=i;}i+=1;}if(close==line.len){return .ok(last);}
    // Go's prefix is greedy: the last complete definition on the line wins.
    while(cursor<line.len){let usize found=find(line,cursor,prefix);if(found==line.len){break;}let usize first=found+prefix.len;i=first;while(i<line.len && line[i]>=48 && line[i]<=57){i+=1;}
        if(i>first && i<line.len && line[i]==40 && i<close){selected=first;stop=i;}cursor=found+1;
    }
    if(selected==line.len){return .ok(last);}var usize value=0;i=selected;while(i<stop){value=value*10+(line[i]-48) as usize;if(value>16384){return .err(invalid(11,i));}i+=1;}
    return if(value<=last){.err(invalid(11,selected))}else{.ok(value)};
}
fn Result<usize,DecodeProblem> body_frames(ref u8[] unit,ref mut usize last){
    if(!at(unit,0,b"/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\n") || find(unit,0,b"const uint8_t fy_unit_literal_")<unit.len){return .err(invalid(6,0));}
    var usize cursor=0;var usize entries=0;while(cursor<unit.len){let usize first=cursor;while(cursor<unit.len && unit[cursor]!=10){cursor+=1;}*last=frame_line(unit[first..cursor],*last)?;if(cursor<unit.len){cursor+=1;}}
    cursor=0;let ref u8[] entry=b"int32_t fy_candidate_entry(void){";while(cursor<unit.len){let usize found=find(unit,cursor,entry);if(found==unit.len){break;}entries+=1;cursor=found+entry.len;}return .ok(entries);
}
pub fn Result<Packet,DecodeProblem> decode(bytes::Buffer data){
    let input=bytes::view(ref data);if(input.len>100663296 || !at(input,0,b"FYCPARTS1\n")){return .err(invalid(1,0));}var usize cursor=10;
    let header=read_part(input,ref mut cursor)?;let literals=read_part(input,ref mut cursor)?;let usize count=read_number(input,ref mut cursor)?;
    if(count==0 || count>4096){return .err(invalid(4,cursor));}var v::Vec<Part> units=v::empty<Part>();var usize i=0;
    while(i<count){let part=read_part(input,ref mut cursor)?;if(!v::push<Part>(ref mut units,part).is_ok){return .err(invalid(13,cursor));}i+=1;}
    if(cursor!=input.len){return .err(invalid(5,cursor));}let h=input[header.start..header.end];let l=input[literals.start..literals.end];
    if(!at(h,0,b"#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n") || h.len<7 || !at(h,h.len-7,b"#endif\n") || !at(l,0,b"/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\n")){return .err(invalid(6,header.start));}
    let usize literal_count=shared_literals(h,l)?;var usize last=0;var usize entries=0;i=0;
    while(i<count){let part=*v::at<Part>(ref units,i);entries+=body_frames(input[part.start..part.end],ref mut last)?;i+=1;}
    if(last==0 || entries!=1){return .err(invalid(12,0));}return .ok(Packet{.data,.header,.literals,.units});
}
pub fn ref u8[] header(ref Packet packet){return bytes::view(ref packet.data)[packet.header.start..packet.header.end];}
pub fn ref u8[] literals(ref Packet packet){return bytes::view(ref packet.data)[packet.literals.start..packet.literals.end];}
pub fn usize unit_count(ref Packet packet){return v::len<Part>(ref packet.units);}
pub fn ref u8[] unit(ref Packet packet,usize index){let part=*v::at<Part>(ref packet.units,index);return bytes::view(ref packet.data)[part.start..part.end];}
pub struct ReadProblem {u32 stage;u32 status;u32 os_code;usize completed;u32 close_status;u32 close_os_code;}
fn ReadProblem read_fail(u32 stage,u32 status,u32 os_code,usize completed){return ReadProblem{.stage,.status,.os_code,.completed,.close_status=0,.close_os_code=0};}
// Fresh input only: no cache digest or frontend fallback. The same open handle
// is checked and bounded reads include one overflow byte even if it grows.
// Read stages: 2 path, 3 open, 4 kind/size/read/allocation, 6 close, 7 decode.
@effects(filesystem) pub fn Result<Packet,ReadProblem> read(ref u8[] path){
    let value=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(read_fail(2,1,0,0));}};
    var fs::File file=switch(fs::open_read_utf8(value)){case .ok(p):{p}case .err(e):{return .err(read_fail(3,e.status,e.os_code,e.completed));}};
    let kind=switch(fs::kind(ref file)){case .ok(p):{p}case .err(e):{return .err(read_fail(4,e.status,e.os_code,e.completed));}};
    let usize size=switch(fs::size(ref file)){case .ok(p):{p}case .err(e):{return .err(read_fail(4,e.status,e.os_code,e.completed));}};
    if(!fs::kind_is_disk(kind) || size>100663296){return .err(read_fail(4,1,0,0));}
    var bytes::Buffer data=bytes::empty();var ReadProblem error=read_fail(0,0,0,0);
    while(error.stage==0){let usize used=bytes::len(ref data);let usize maximum=if(100663297-used<65536){100663297-used}else{65536};
        switch(fs::read_append(ref mut file,ref mut data,maximum)){
            case .ok(n):{if(bytes::len(ref data)>100663296){error=read_fail(4,1,0,bytes::len(ref data));}else if(n==0){break;}}
            case .err(e):{error=read_fail(4,if(e.file_status!=0){e.file_status}else{e.allocation_status},e.os_code,bytes::len(ref data));}
        };
    }
    switch(fs::close(file)){case .ok(_):{}case .err(e):{if(error.stage==0){error=read_fail(6,e.status,e.os_code,bytes::len(ref data));}else{error.close_status=e.status;error.close_os_code=e.os_code;}}};
    if(error.stage!=0){return .err(error);}return switch(decode(data)){case .ok(p):{.ok(p)}case .err(e):{.err(read_fail(7,e.kind,0,e.offset))}};
}
