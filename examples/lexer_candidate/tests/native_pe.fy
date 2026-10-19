module tests::native_pe;
import native_pe as pe;
import native_pe_unwind as unwind;
import native_msvc_artifacts as artifact;
import std::bytes as bytes;
import std::fs as fs;
import std::process as process;
import std::text as text;
import std::io as io;

@effects(filesystem) fn usize number(ref mut fs::File file){var u8[4] raw=[0;4];assert(fs::read_exact(file,ref mut raw).is_ok);return raw[0] as usize+(raw[1] as usize)*256+(raw[2] as usize)*65536+(raw[3] as usize)*16777216;}
@effects(filesystem) fn bytes::Buffer part(ref mut fs::File file){let usize n=number(file);assert(n<=67108864);var bytes::Buffer out=bytes::empty();assert(bytes::resize_zeroed(ref mut out,n).is_ok);assert(fs::read_exact(file,bytes::view_mut(ref mut out)).is_ok);return out;}
@effects(filesystem) fn fs::File fixture(){var u8[4096] raw=[0;4096];let usize n=switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] all=ref raw;let path=switch(text::validate_utf8(all[..n])){case .ok(p):{p}case .err(_):{assert(false);return fixture();}};return switch(fs::open_read_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return fixture();}};}
fn void append(ref mut bytes::Buffer out,ref u8[] value){assert(bytes::append(out,value).is_ok);}
fn void value(ref mut bytes::Buffer out,usize n, bool last){assert(bytes::append_usize(out,n).is_ok);append(out,if(last){b"\n"}else{b"/"});}
fn void render_pe(ref u8[] input,pe::Header h,ref mut bytes::Buffer out){
    let usize o=h.optional;value(out,34404,false);value(out,pe::half(input,o-2),false);value(out,pe::word(input,o+24)+pe::word(input,o+28)*4294967296,false);value(out,pe::word(input,o+16),false);value(out,pe::half(input,o+68),false);value(out,pe::word(input,o+32),false);value(out,pe::word(input,o+36),false);value(out,pe::word(input,o+56),false);value(out,pe::word(input,o+60),false);value(out,h.count,false);
    append(out,if(pe::directory_rva(input,h,1)>0 && pe::directory_size(input,h,1)>0){b"true/"}else{b"false/"});append(out,if(pe::directory_rva(input,h,3)>0 && pe::directory_size(input,h,3)>0){b"true\n"}else{b"false\n"});var usize i=0;
    while(i<h.count){let s=switch(pe::section(input,h,i)){case .ok(s):{s}case .err(_):{assert(false);return;}};append(out,input[s.name..s.name_end]);append(out,b"/");value(out,s.address,false);value(out,s.virtual_size,false);value(out,s.offset,false);value(out,s.size,false);value(out,s.flags,true);i+=1;}
}
fn void render_unwind(ref u8[] input,unwind::Report r,ref mut bytes::Buffer out){
    value(out,r.functions,false);value(out,r.unique_infos,false);value(out,r.handlers,false);value(out,r.chains,false);value(out,r.version2,false);value(out,r.maximum_codes,true);
    let h=switch(pe::inspect(input)){case .ok(h):{h}case .err(_):{assert(false);return;}};let mapped=switch(pe::map(input,h,pe::directory_rva(input,h,3),pe::directory_size(input,h,3))){case .ok(m):{m}case .err(_):{assert(false);return;}};var usize i=0;
    while(i<r.functions){value(out,pe::word(input,mapped.offset+i*12),false);value(out,pe::word(input,mapped.offset+i*12+4),false);value(out,pe::word(input,mapped.offset+i*12+8),true);i+=1;}
}
@effects(console) fn void compare(ref u8[] name,ref u8[] actual,ref u8[] expected,ref u8[] phase){
    if(!bytes::equal(actual,expected)){discard io::write_stderr_bytes(phase);discard io::write_stderr_bytes(name);discard io::write_stderr_bytes(b"\nactual: ");discard io::write_stderr_bytes(actual);discard io::write_stderr_bytes(b"\nexpected: ");discard io::write_stderr_bytes(expected);assert(false);}
}
@test @effects(filesystem,console) fn void matches_go_pe_and_unwind_reports(){
    var fs::File file=fixture();var u8[13] magic=[0;13];assert(fs::read_exact(ref mut file,ref mut magic).is_ok && bytes::equal(ref magic,b"FYCPEVERIFY1\n"));let usize count=number(ref mut file);assert(count>1000 && count<16384);var usize i=0;
    while(i<count){let name=part(ref mut file);let image=part(ref mut file);let expected_pe=part(ref mut file);let expected_unwind=part(ref mut file);let input=bytes::view(ref image);var bytes::Buffer actual_pe=bytes::empty();var bytes::Buffer actual_unwind=bytes::empty();
        switch(pe::inspect(input)){case .ok(h):{render_pe(input,h,ref mut actual_pe);}case .err(_):{}};
        switch(unwind::inspect(input)){case .ok(r):{render_unwind(input,r,ref mut actual_unwind);}case .err(_):{}};
        compare(bytes::view(ref name),bytes::view(ref actual_pe),bytes::view(ref expected_pe),b"PE: ");compare(bytes::view(ref name),bytes::view(ref actual_unwind),bytes::view(ref expected_unwind),b"unwind: ");
        let bool has_pe=bytes::len(ref actual_pe)>0;let bool has_unwind=bytes::len(ref actual_unwind)>0;var bool no_directory=false;if(has_pe){let h=switch(pe::inspect(input)){case .ok(h):{h}case .err(_):{assert(false);return;}};no_directory=pe::directory_rva(input,h,3)==0 && pe::directory_size(input,h,3)==0;}
        assert(artifact::executable(input)==(has_pe && (has_unwind || no_directory)));i+=1;
    }
    var u8[1] last=[0;1];let usize n=switch(fs::read(ref mut file,ref mut last)){case .ok(n):{n}case .err(_):{assert(false);1}};assert(n==0);assert(fs::close(file).is_ok);
}
