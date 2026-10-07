module effect_declarations;
import syntax as syntax;
import std::bytes as bytes;
// Closed language/11 authority set. Allocation and unsafe are not effects.
pub struct Info { bool must_use; usize must_start; usize must_end; u32 bits; u32 code; usize start; usize end; }
pub fn u32 bit(ref u8[] name) {
    return if(bytes::equal(name,b"console")){1}else if(bytes::equal(name,b"filesystem")){2}
        else if(bytes::equal(name,b"environment")){4}else if(bytes::equal(name,b"process")){8}
        else if(bytes::equal(name,b"network")){16}else if(bytes::equal(name,b"clock")){32}
        else if(bytes::equal(name,b"random")){64}else if(bytes::equal(name,b"block_context")){128}
        else if(bytes::equal(name,b"emit")){256}else if(bytes::equal(name,b"external_call")){512}
        else if(bytes::equal(name,b"storage_read")){1024}else if(bytes::equal(name,b"storage_write")){2048}else{0};
}
pub fn bool includes(u32 authority,u32 required){return (authority&required)==required;}
pub fn Info read(ref syntax::Parsed p,usize declaration){
    let d=syntax::declaration_at(p,declaration);var Info info=Info{.must_use=false,.must_start=0,.must_end=0,.bits=0,.code=0,.start=d.start,.end=d.end};
    var bool seen=false;var usize i=0;
    while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);let name=syntax::source_view(p)[a.name_start..a.name_end];
        if(bytes::equal(name,b"must_use")){
            if((d.kind!=3 && d.kind!=9 && d.kind!=12) || info.must_use || a.argument_count!=0){info.code=4201;info.start=a.start;info.end=a.end;return info;}
            info.must_use=true;info.must_start=a.start;info.must_end=a.end;i+=1;continue;
        }
        if(bytes::equal(name,b"link_name") && d.kind==9){i+=1;continue;}
        if(!bytes::equal(name,b"effects") && d.kind==6){i+=1;continue;}
        if(!bytes::equal(name,b"effects")){info.code=3;info.start=a.start;info.end=a.end;return info;}
        if(seen || a.argument_count==0){info.code=7101;info.start=a.start;info.end=a.end;return info;}seen=true;
        var usize j=0;while(j<a.argument_count){let arg=syntax::attribute_argument_at(p,a.argument_start+j);let u32 b=bit(syntax::attribute_argument_text(p,a.argument_start+j));
            if(arg.kind!=1 || b==0 || (info.bits&b)!=0){info.code=7101;info.start=arg.start;info.end=arg.end;return info;}info.bits|=b;j+=1;}i+=1;
    }
    if(seen && d.kind==6 && syntax::block_statement_count(p,d.cleanup_body)==0 && syntax::statement_at(p,d.cleanup_body).expression==0){info.code=7101;}
    return info;
}
// Generic clones retain source spans and the original declaration authority.
// Only source diagnostics use spans; MIR checks exact function IDs.
pub fn u32 caller(ref syntax::Parsed p,usize start,usize end){var usize i=0;
    while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if((d.kind==3 || d.kind==6) && (d.body!=0 || d.cleanup_body!=0)){let b=syntax::statement_at(p,if(d.kind==6){d.cleanup_body}else{d.body});if(b.start<=start && end<=b.end){return read(p,i).bits;}}i+=1;}return 0;
}
