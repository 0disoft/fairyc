module tests::native_coff;
import native_coff as coff;
import native_coff_layout as l;
import native_pe as read;
import std::vec as v;
import std::bytes as bytes;
import std::fs as fs;
import std::process as process;
import std::text as text;
import std::io as io;

@effects(filesystem) fn usize number(ref mut fs::File file){var u8[4] raw=[0;4];assert(fs::read_exact(file,ref mut raw).is_ok);return raw[0] as usize+(raw[1] as usize)*256+(raw[2] as usize)*65536+(raw[3] as usize)*16777216;}
@effects(filesystem) fn bytes::Buffer part(ref mut fs::File file){let usize n=number(file);assert(n<=67108864);var bytes::Buffer out=bytes::empty();assert(bytes::resize_zeroed(ref mut out,n).is_ok);assert(fs::read_exact(file,bytes::view_mut(ref mut out)).is_ok);return out;}
@effects(filesystem) fn fs::File fixture(){var u8[4096] raw=[0;4096];let usize n=switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] all=ref raw;let path=switch(text::validate_utf8(all[..n])){case .ok(p):{p}case .err(_):{assert(false);return fixture();}};return switch(fs::open_read_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return fixture();}};}
fn void append(ref mut bytes::Buffer out,ref u8[] value){assert(bytes::append(out,value).is_ok);}
fn void value(ref mut bytes::Buffer out,usize n,bool last){assert(bytes::append_usize(out,n).is_ok);append(out,if(last){b"\n"}else{b"/"});}
fn void name(ref u8[] input,l::Name n,ref mut bytes::Buffer out){append(out,input[n.start..n.end]);append(out,b"/");}
fn void render(ref u8[] input,ref coff::Report report,ref mut bytes::Buffer out){
    let layout=coff::layout(report);let h=l::header(layout);value(out,34404,false);value(out,h.flags,false);append(out,if(h.big){b"true/"}else{b"false/"});value(out,if(h.big){2}else{0},false);value(out,h.count,false);value(out,v::len<l::Symbol>(l::symbols(layout)),false);value(out,coff::relocations(report),true);var usize i=0;
    while(i<h.count){let s=*v::at<l::Section>(l::sections(layout),i);name(input,s.name,out);value(out,s.offset,false);value(out,s.size,false);value(out,s.count,false);value(out,s.flags,false);value(out,s.selection,false);value(out,s.parent,true);i+=1;}
    i=0;while(i<v::len<l::Symbol>(l::symbols(layout))){let s=*v::at<l::Symbol>(l::symbols(layout),i);value(out,s.index,false);name(input,s.name,out);value(out,s.value,false);
        let usize sign=if(h.big){2147483648}else{32768};let usize modulus=if(h.big){4294967296}else{65536};if(s.section>=sign){append(out,b"-");value(out,modulus-s.section,false);}else{value(out,s.section,false);}
        value(out,s.type,false);value(out,s.storage,false);value(out,s.aux,false);value(out,s.fallback,false);if(s.storage==105 && !h.big){let usize index=*v::at<usize>(l::indices(layout),s.fallback);name(input,v::at<l::Symbol>(l::symbols(layout),index-1).name,out);}else{append(out,b"/");}value(out,s.search,true);i+=1;
    }
    i=0;while(i<h.count){let s=*v::at<l::Section>(l::sections(layout),i);var usize j=s.stored-s.count;while(j<s.stored){let usize at=s.rel+j*10;let usize target=read::word(input,at+4);let usize index=*v::at<usize>(l::indices(layout),target);value(out,i+1,false);name(input,s.name,out);value(out,read::word(input,at),false);value(out,target,false);name(input,v::at<l::Symbol>(l::symbols(layout),index-1).name,out);value(out,read::half(input,at+8),true);j+=1;}i+=1;}
}
@effects(console) fn void mismatch(ref u8[] name,ref u8[] phase,u32 code){discard io::write_stderr_bytes(phase);discard io::write_stderr_bytes(name);discard io::write_stderr_bytes(b" code=");discard io::write_stderr_u32(code);discard io::write_stderr_bytes(b"\n");assert(false);}
@test @effects(filesystem,console) fn void matches_go_coff_reports_and_project_admission(){
    var fs::File file=fixture();var u8[15] magic=[0;15];assert(fs::read_exact(ref mut file,ref mut magic).is_ok && bytes::equal(ref magic,b"FYCCOFFVERIFY1\n"));let usize count=number(ref mut file);assert(count>1000 && count<16384);var usize i=0;
    while(i<count){let name=part(ref mut file);let data=part(ref mut file);let expected=part(ref mut file);let bool project=number(ref mut file)==1;let input=bytes::view(ref data);var bytes::Buffer actual=bytes::empty();var u32 problem=0;
        switch(coff::inspect(input)){case .ok(report):{render(input,ref report,ref mut actual);}case .err(code):{problem=code;}};
        if(!bytes::equal(bytes::view(ref actual),bytes::view(ref expected))){mismatch(bytes::view(ref name),b"COFF: ",problem);}let u32 code=coff::project_problem(input);if((code==0)!=project){mismatch(bytes::view(ref name),b"project: ",code);}i+=1;
    }
    var u8[1] last=[0;1];let usize n=switch(fs::read(ref mut file,ref mut last)){case .ok(n):{n}case .err(_):{assert(false);1}};assert(n==0);assert(fs::close(file).is_ok);
}
