module tests::sdk_vector_plan_capacity;
import sdk_vector_plan as plan;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

fn syntax::Parsed parsed(bytes::Buffer input){
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(bytes::empty());}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;
}
fn bytes::Buffer copied(ref u8[] input){return switch(bytes::copy(input)){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem) fn bytes::Buffer sdk_source(){
    var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};
    return switch(files::load(ref path[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};
}
fn bytes::Buffer many_vectors(ref u8[] module_name,usize count){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module ").is_ok && bytes::append(ref mut source,module_name).is_ok);
    assert(bytes::append(ref mut source,b";import std::vec as v;").is_ok);var usize i=0;
    while(i<count){
        assert(bytes::append(ref mut source,b"struct S").is_ok && bytes::append_usize(ref mut source,i).is_ok && bytes::append(ref mut source,b"{i32 n;}fn v::Vec<S").is_ok);
        assert(bytes::append_usize(ref mut source,i).is_ok && bytes::append(ref mut source,b"> f").is_ok && bytes::append_usize(ref mut source,i).is_ok && bytes::append(ref mut source,b"(){}").is_ok);i+=1;
    }return source;
}
fn void providers(ref mut v::Vec<syntax::Parsed> sources,usize total){
    var usize remaining=total;var usize index=0;
    while(remaining>0){
        var bytes::Buffer name=bytes::empty();assert(bytes::append(ref mut name,b"part").is_ok && bytes::append_usize(ref mut name,index).is_ok);
        let usize count=if(remaining>60){60}else{remaining};
        assert(v::push<syntax::Parsed>(sources,parsed(many_vectors(bytes::view(ref name),count))).is_ok);
        remaining-=count;index+=1;
    }
}
@test @effects(filesystem,console) fn void source_wide_vector_descriptors_have_an_independent_boundary(){
    let source=sdk_source();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(copied(bytes::view(ref source)))).is_ok);
    let sdk=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write("SDK code ");discard io::write_u32(e.code);discard io::write("\n");assert(false);return;}};
    let builtins=signatures::vector_count(ref sdk);assert(builtins>0 && builtins<128);
    discard v::clear<syntax::Parsed>(ref mut sources);
    providers(ref mut sources,256-builtins);let sdk_index=v::len<syntax::Parsed>(ref sources);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(copied(bytes::view(ref source)))).is_ok);
    let admitted=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write("admission code ");discard io::write_u32(e.code);discard io::write(" module ");discard io::write_usize(e.module_index);discard io::write("\n");assert(false);return;}};
    assert(plan::limit()==256 && signatures::vector_count(ref admitted)==256);
    var usize functions=0;var usize id=1;while(id<=signatures::count(ref admitted)){let item=signatures::at(ref admitted,id);if(item.module_index<sdk_index){assert(signatures::type_at(ref admitted,item.result).kind==13);functions+=1;}id+=1;}assert(functions==256-builtins);
    var usize scalar=0;var usize nominal=0;var usize i=0;
    while(i<256){let item=signatures::vector_at(ref admitted,i);assert(item.module_index==sdk_index && signatures::type_at(ref admitted,item.type).kind==13);let element=signatures::type_at(ref admitted,item.element);if(element.kind==1){scalar+=1;}else{assert(element.kind==2 && element.module_index<sdk_index);nominal+=1;}i+=1;}assert(scalar==builtins && nominal==256-builtins);
    discard v::clear<syntax::Parsed>(ref mut sources);
    providers(ref mut sources,257-builtins);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(copied(bytes::view(ref source)))).is_ok);
    switch(signatures::build(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4503);}};
    discard io::write("SDK descriptors ");discard io::write_usize(builtins);discard io::write(" accepted 256 rejected 257\n");
}
