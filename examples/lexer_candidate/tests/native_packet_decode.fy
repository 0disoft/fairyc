module tests::native_packet_decode;
import native_packet_decode as packet;
import std::bytes as bytes;
import std::fs as fs;
import std::process as process;
import std::text as text;

fn bytes::Buffer copy(ref u8[] value){return switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};}
@effects(filesystem) fn bytes::Buffer fixture_path(){
    var u8[4096] raw=[0;4096];let usize n=switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref raw;return copy(view[..n]);
}
@effects(filesystem) fn fs::File open(ref u8[] path){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return open(b"");}};
    return switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return open(b"");}};
}
@effects(filesystem) fn usize number(ref mut fs::File file){
    var u8[4] raw=[0;4];assert(fs::read_exact(file,ref mut raw).is_ok);return raw[0] as usize+(raw[1] as usize)*256+(raw[2] as usize)*65536+(raw[3] as usize)*16777216;
}
@test @effects(filesystem) fn void matches_go_admission_and_part_lengths(){
    let path=fixture_path();var fs::File file=open(bytes::view(ref path));var u8[15] magic=[0;15];assert(fs::read_exact(ref mut file,ref mut magic).is_ok && bytes::equal(ref magic,b"FYCPACKETTEST1\n"));
    let usize count=number(ref mut file);assert(count>300 && count<2048);var usize i=0;
    while(i<count){var u8[1] admitted=[0;1];assert(fs::read_exact(ref mut file,ref mut admitted).is_ok);let usize length=number(ref mut file);let usize header=number(ref mut file);let usize literals=number(ref mut file);let usize units=number(ref mut file);assert(length<=100663296);
        var bytes::Buffer data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,length).is_ok);assert(fs::read_exact(ref mut file,bytes::view_mut(ref mut data)).is_ok);
        switch(packet::decode(data)){
            case .ok(p):{assert(admitted[0]==1 && packet::header(ref p).len==header && packet::literals(ref p).len==literals && packet::unit_count(ref p)==units);var usize u=0;while(u<units){assert(packet::unit(ref p,u).len>0);u+=1;}}
            case .err(_):{assert(admitted[0]==0);}
        };i+=1;
    }
    var u8[1] end=[0;1];let usize remaining=switch(fs::read(ref mut file,ref mut end)){case .ok(n):{n}case .err(_):{assert(false);1}};assert(remaining==0);assert(fs::close(file).is_ok);
}
fn void part(ref mut bytes::Buffer data,ref u8[] source){assert(bytes::append_usize(data,source.len).is_ok && bytes::append_byte(data,10).is_ok && bytes::append(data,source).is_ok);}
@test fn void owns_the_decoded_bytes_and_exposes_exact_parts(){
    let ref u8[] header=b"#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#endif\n";
    let ref u8[] literals=b"/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\n";
    let ref u8[] unit=b"/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\nextern int32_t fy_fn_1(void){return 0;}\nint32_t fy_candidate_entry(void){return fy_fn_1();}\n";
    var bytes::Buffer data=copy(b"FYCPARTS1\n");part(ref mut data,header);part(ref mut data,literals);assert(bytes::append(ref mut data,b"1\n").is_ok);part(ref mut data,unit);
    switch(packet::decode(data)){case .ok(p):{assert(bytes::equal(packet::header(ref p),header) && bytes::equal(packet::literals(ref p),literals) && packet::unit_count(ref p)==1 && bytes::equal(packet::unit(ref p,0),unit));}case .err(_):{assert(false);}};
}
@test fn void rejects_packet_over_the_total_bound_before_scanning(){
    var bytes::Buffer data=bytes::empty();assert(bytes::append(ref mut data,b"FYCPARTS1\n").is_ok && bytes::resize_zeroed(ref mut data,100663297).is_ok);
    switch(packet::decode(data)){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==1 && e.offset==0);}};
}
@test @effects(filesystem) fn void reads_fresh_packet_and_rejects_nonpacket_and_missing_files(){
    let path=fixture_path();var bytes::Buffer fresh=copy(bytes::view(ref path));assert(bytes::append(ref mut fresh,b".packet").is_ok);
    switch(packet::read(bytes::view(ref fresh))){case .ok(p):{assert(packet::unit_count(ref p)>0 && packet::header(ref p).len>0);}case .err(_):{assert(false);}};
    var bytes::Buffer invalid=copy(bytes::view(ref path));assert(bytes::append(ref mut invalid,b".invalid").is_ok);
    switch(packet::read(bytes::view(ref invalid))){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==7 && e.status==1);}};
    var bytes::Buffer missing=copy(bytes::view(ref path));assert(bytes::append(ref mut missing,b".missing").is_ok);
    switch(packet::read(bytes::view(ref missing))){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==3);}};
    switch(packet::read(b"")){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==3 || e.stage==2);}};
}
