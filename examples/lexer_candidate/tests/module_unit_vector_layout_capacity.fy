module tests::module_unit_vector_layout_capacity;
import module_unit_interfaces as interfaces;
import module_unit_bind as binding;
import module_unit_generics as generics;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn syntax::Parsed parsed(bytes::Buffer b){let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};if(syntax::problem_count(ref p)!=0){discard io::write("parse problems ");discard io::write_usize(syntax::problem_count(ref p));discard io::write("\n");}assert(syntax::problem_count(ref p)==0);return p;}
@test @effects(filesystem,console) fn void sixty_five_vector_layouts_do_not_consume_executable_specializations(){
    var bytes::Buffer root=bytes::empty();assert(bytes::append(ref mut root,b"module main;import std::vec as v;import provider as p;").is_ok);var bytes::Buffer provider=bytes::empty();assert(bytes::append(ref mut provider,b"module provider;").is_ok);var usize i=0;
    while(i<65){assert(bytes::append(ref mut provider,b"pub struct S").is_ok);assert(bytes::append_usize(ref mut provider,i).is_ok);assert(bytes::append(ref mut provider,b"{i32 n;}").is_ok);assert(bytes::append(ref mut root,b"fn v::Vec<p::S").is_ok);assert(bytes::append_usize(ref mut root,i).is_ok);assert(bytes::append(ref mut root,b"> f").is_ok);assert(bytes::append_usize(ref mut root,i).is_ok);assert(bytes::append(ref mut root,b"(){}").is_ok);i+=1;}
    var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{discard io::write("argument failed\n");assert(false);return;}};let sdk=switch(files::load(ref path[..n])){case .ok(b):{b}case .err(_):{discard io::write("load failed\n");assert(false);return;}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(sdk)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write("signatures ");discard io::write_u32(e.code);discard io::write("\n");assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{discard io::write("binding ");discard io::write_u32(e.code);discard io::write("\n");assert(false);return;}};let seeds=v::empty<binding::Coordinate>();let p=switch(interfaces::build_unit_seeded(ref sources,ref metadata,ref bound,ref seeds)){case .ok(p):{p}case .err(e):{discard io::write("code ");discard io::write_u32(e.code);discard io::write("\n");assert(false);return;}};
    var usize vectors=0;i=0;while(i<interfaces::shape_count(ref p)){let s=interfaces::shape_at(ref p,i);if(interfaces::type_at(ref p,s.type).kind==13){vectors+=1;}i+=1;}var usize generic_functions=0;i=1;while(i<=generics::count(interfaces::specializations(ref p))){if(generics::at(interfaces::specializations(ref p),i).argument_count!=0){generic_functions+=1;}i+=1;}
    discard io::write("vector layouts ");discard io::write_usize(vectors);discard io::write(" generic functions ");discard io::write_usize(generic_functions);discard io::write("\n");assert(vectors==65 && generic_functions==0 && generics::cleanup_count(interfaces::specializations(ref p))==65);
}
