module tests::module_unit_imported_metadata_capacity;
import candidate_limits as limits;
import module_signatures as signatures;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@test @effects(filesystem,console) fn void original_origin_check_retains_all_imported_fields_above_the_source_arena_cap(){
    let count=process::arg_count()-1;assert(count>200 && count<=limits::source_files());var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<count){var u8[4096] path=[0;4096];let n=switch(process::arg(i+1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let b=switch(files::load(ref path[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};let s=switch(scanner::collect(b,false,limits::source_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref p)==0);assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);i+=1;}
    let source=v::at<syntax::Parsed>(ref sources,136);let d=syntax::declaration_at(source,0);assert(bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],b"module_unit_origin_check"));
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,136,137)){case .ok(p):{p}case .err(_):{assert(false);return;}};let seeds=v::empty<binding::Coordinate>();let p=switch(interfaces::build_unit_seeded(ref sources,ref metadata,ref bound,ref seeds)){case .ok(p):{p}case .err(e):{discard io::write("interfaces ");discard io::write_u32(e.code);discard io::write(" module ");discard io::write_usize(e.module_index);discard io::write(" at ");discard io::write_usize(e.start);discard io::write("..");discard io::write_usize(e.end);discard io::write("\n");assert(false);return;}};
    assert(interfaces::field_count(ref p)>1024 && interfaces::field_count(ref p)<=limits::metadata_fields() && interfaces::work(ref p)<=1048576);var usize vectors=0;i=0;while(i<interfaces::shape_count(ref p)){let s=interfaces::shape_at(ref p,i);if(interfaces::type_at(ref p,s.type).kind==13){vectors+=1;}i+=1;}assert(vectors>64 && vectors<=256);
    discard io::write("fields ");discard io::write_usize(interfaces::field_count(ref p));discard io::write(" shapes ");discard io::write_usize(interfaces::shape_count(ref p));discard io::write(" vectors ");discard io::write_usize(vectors);discard io::write(" work ");discard io::write_usize(interfaces::work(ref p));discard io::write("\n");
}
