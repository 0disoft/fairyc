module tests::module_source_closure;
import candidate_limits as limits;
import module_signatures as signatures;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_bodies as bodies;
import module_unit_values as values;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

@effects(console) fn void failed(ref str phase,u32 code,usize module_index,usize start,usize end){
    discard io::write(phase);discard io::write(" code ");discard io::write_u32(code);
    discard io::write(" module ");discard io::write_usize(module_index);
    discard io::write(" at ");discard io::write_usize(start);discard io::write("..");discard io::write_usize(end);discard io::write("\n");assert(false);
}
@effects(filesystem,console) fn syntax::Parsed input(usize index){
    var u8[4096] path=[0;4096];let n=switch(process::arg(index,ref mut path)){case .ok(n):{n}case .err(_):{failed("argument",1,index-1,0,0);return input(index);}};
    let source=switch(files::load(ref path[..n])){case .ok(b):{b}case .err(c):{failed("load",c,index-1,0,0);return input(index);}};
    let stream=switch(scanner::collect(source,false,limits::source_bytes(),1048576,256,16777216)){case .ok(s):{s}case .err(c):{failed("scanner",if(c==3){3}else{4},index-1,0,0);return input(index);}};
    assert(scanner::diagnostic_count(ref stream)==0);
    let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(c):{failed("parse",if(c==3){3}else{4},index-1,0,0);return input(index);}};
    assert(syntax::problem_count(ref parsed)==0);return parsed;
}
@test @effects(filesystem,console) fn void original_main_call_closure_and_values(){
    let count=process::arg_count()-1;assert(count>200 && count<=limits::source_files());
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<count){assert(v::push<syntax::Parsed>(ref mut sources,input(i+1)).is_ok);i+=1;}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{failed("signatures",e.code,e.module_index,e.start,e.end);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{failed("binding",e.code,e.module_index,e.start,e.end);return;}};
    discard io::write("root bound\n");
    let ready=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{failed("closure",e.code,e.module_index,e.start,e.end);return;}};
    discard io::write("retained bodies ");discard io::write_usize(bodies::count(closure::owned(ref ready)));
    discard io::write(" passes ");discard io::write_usize(closure::passes(ref ready));discard io::write("\n");
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{failed("values",e.code,e.module_index,e.start,e.end);return;}};
    discard io::write("typed bodies ");discard io::write_usize(values::count(ref typed));discard io::write(" work ");discard io::write_usize(values::work(ref typed));discard io::write("\n");
}
