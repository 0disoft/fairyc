module native_packet_file;
import native_packet as packet;
import module_unit_c_emit as emit;
import std::text as text;
import std::fs as fs;

pub struct Problem{u32 stage;u32 status;u32 os_code;usize completed;u32 close_status;u32 close_os_code;}
fn Problem fail(u32 stage,u32 status,u32 os_code,usize completed){return Problem{.stage,.status,.os_code,.completed,.close_status=0,.close_os_code=0};}
@effects(filesystem) pub fn Result<usize,Problem> write(ref emit::Bundle bundle,ref u8[] path){return write_emission(emit::project_emission(bundle),path);}
@effects(filesystem) pub fn Result<usize,Problem> write_emission(ref emit::Emission bundle,ref u8[] path){
    switch(packet::emission_size(bundle)){case .ok(_):{}case .err(e):{return .err(fail(1,e.kind,e.os_code,e.completed));}};
    let value=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(fail(2,1,0,0));}};let name=switch(text::cstring_from_text(value)){case .ok(p):{p}case .err(_):{return .err(fail(2,2,0,0));}};
    var fs::File file=switch(fs::create_new(text::cstring_as_cstr(ref name))){case .ok(p):{p}case .err(e):{return .err(fail(3,e.status,e.os_code,e.completed));}};var Problem error=fail(0,0,0,0);var usize count=0;
    switch(packet::write_emission(bundle,ref mut file)){case .ok(n):{count=n;}case .err(e):{error=fail(4,e.status,e.os_code,e.completed);}};
    if(error.stage==0){switch(fs::sync(ref mut file)){case .ok(_):{}case .err(e):{error=fail(5,e.status,e.os_code,count);}};}
    switch(fs::close(file)){case .ok(_):{}case .err(e):{if(error.stage==0){error=fail(6,e.status,e.os_code,count);}else{error.close_status=e.status;error.close_os_code=e.os_code;}}};
    return if(error.stage==0){.ok(count)}else{.err(error)};
}
