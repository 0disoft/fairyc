module native_packet;
import module_unit_c_emit as emit;
import std::bytes as bytes;
import std::fs as fs;

// 1 packet bound, 2 number-buffer allocation, 3 file write. The caller owns
// the fresh file and must sync/close it before invoking the native driver.
pub struct Problem{u32 kind;u32 status;u32 os_code;usize completed;}
fn Problem fail(u32 kind,u32 status,u32 os_code,usize completed){return Problem{.kind,.status,.os_code,.completed};}
fn usize digits(usize value){var usize n=value;var usize count=1;while(n>=10){n/=10;count+=1;}return count;}
fn Result<usize,Problem> part_size(usize retained,usize length){if(length>100663296 || retained>100663296 || length+digits(length)+1>100663296-retained){return .err(fail(1,0,0,0));}return .ok(retained+digits(length)+1+length);}
pub fn Result<usize,Problem> size(ref emit::Bundle bundle){
    var usize total=switch(part_size(10,emit::project_header(bundle).len)){case .ok(n):{n}case .err(e):{return .err(e);}};
    total=switch(part_size(total,emit::project_literals(bundle).len)){case .ok(n):{n}case .err(e):{return .err(e);}};let count=emit::project_unit_count(bundle);
    if(count==0 || count>8192 || digits(count)+1>100663296-total){return .err(fail(1,0,0,0));}total+=digits(count)+1;var usize i=0;
    while(i<count){total=switch(part_size(total,emit::project_source(bundle,i).len)){case .ok(n):{n}case .err(e):{return .err(e);}};i+=1;}return .ok(total);
}
@effects(filesystem) fn Result<usize,Problem> put(ref mut fs::File file,ref u8[] data,usize completed){return switch(fs::write_all(file,data)){
    case .ok(n):{if(n!=data.len){.err(fail(3,4,0,completed+n))}else{.ok(completed+n)}}
    case .err(e):{.err(fail(3,e.status,e.os_code,completed+e.completed))}
};}
@effects(filesystem) fn Result<usize,Problem> number(ref mut fs::File file,usize value,usize completed){var bytes::Buffer text=bytes::empty();if(!bytes::append_usize(ref mut text,value).is_ok || !bytes::append_byte(ref mut text,10).is_ok){return .err(fail(2,0,0,completed));}return put(file,bytes::view(ref text),completed);}
@effects(filesystem) fn Result<usize,Problem> part(ref mut fs::File file,ref u8[] data,usize completed){let count=switch(number(file,data.len,completed)){case .ok(n):{n}case .err(e):{return .err(e);}};return put(file,data,count);}
// Admission completes before the first write. Only a complete successful result
// authorizes the caller to hand this fresh packet to the native driver.
@effects(filesystem) pub fn Result<usize,Problem> write(ref emit::Bundle bundle,ref mut fs::File file){
    let expected=switch(size(bundle)){case .ok(n):{n}case .err(e):{return .err(e);}};
    var usize total=switch(put(file,b"FYCPARTS1\n",0)){case .ok(n):{n}case .err(e):{return .err(e);}};
    total=switch(part(file,emit::project_header(bundle),total)){case .ok(n):{n}case .err(e):{return .err(e);}};
    total=switch(part(file,emit::project_literals(bundle),total)){case .ok(n):{n}case .err(e):{return .err(e);}};
    total=switch(number(file,emit::project_unit_count(bundle),total)){case .ok(n):{n}case .err(e):{return .err(e);}};var usize i=0;
    while(i<emit::project_unit_count(bundle)){total=switch(part(file,emit::project_source(bundle,i),total)){case .ok(n):{n}case .err(e):{return .err(e);}};i+=1;}
    return if(total==expected){.ok(total)}else{.err(fail(1,0,0,total))};
}
