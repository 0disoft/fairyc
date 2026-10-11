module tests::project_toml_compatibility;
import project_toml_document as document;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@test
@effects(filesystem,console)
fn void matches_pinned_go_table_context_acceptance(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;
    let input=switch(files::manifest(path[..length])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};let ref u8[] data=bytes::view(ref input);assert(data.len>=9 && bytes::equal(data[..9],b"FYTBL001\n"));var usize offset=9;var usize count=0;
    while(offset<data.len){assert(data.len-offset>=3);let usize size=(data[offset] as usize)+(data[offset+1] as usize)*256;assert(data[offset+2]<=1);let bool expected=data[offset+2]==1;offset+=3;assert(size<=data.len-offset);let usize start=offset;offset+=size;
        let bool accepted=switch(document::parse(data[start..offset])){case .ok(_):{true}case .err(e):{assert(e.start<=e.end && e.end<=size);false}};
        if(accepted!=expected){discard io::write_usize(count);discard io::write(" table context acceptance mismatch\n");}assert(accepted==expected);count+=1;
    }assert(count>0);
}
