module tests::module_source_set;
import candidate_limits as limits;
import module_units as units;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

@effects(console) fn void failed(ref str phase,u32 code,usize module_index){
    discard io::write(phase);discard io::write(" code ");discard io::write_u32(code);
    discard io::write(" module ");discard io::write_usize(module_index);discard io::write("\n");assert(false);
}
@effects(filesystem,console) fn syntax::Parsed input(usize index){
    var u8[4096] path=[0;4096];
    let length=switch(process::arg(index,ref mut path)){case .ok(n):{n}case .err(_):{failed("argument",1,index-1);return input(index);}};
    let source=switch(files::load(ref path[..length])){case .ok(b):{b}case .err(c):{failed("load",c,index-1);return input(index);}};
    let stream=switch(scanner::collect(source,false,limits::source_bytes(),1048576,256,16777216)){case .ok(s):{s}case .err(c):{failed("scanner",if(c==3){3}else{4},index-1);return input(index);}};
    assert(scanner::diagnostic_count(ref stream)==0);
    let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(c):{failed("parse",if(c==3){3}else{4},index-1);return input(index);}};
    assert(syntax::problem_count(ref parsed)==0);return parsed;
}

// Arguments are exact main.fy followed by its complete private/SDK import closure.
@test @effects(filesystem,console) fn void original_main_sources_fit_catalog_and_units(){
    let count=process::arg_count()-1;assert(count>200 && count<=limits::source_files());
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;var usize bytes_total=0;
    while(i<count){let p=input(i+1);bytes_total+=syntax::source_view(ref p).len;assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);i+=1;}
    discard io::write("sources ");discard io::write_usize(count);discard io::write(" bytes ");discard io::write_usize(bytes_total);discard io::write("\n");
    let metadata=switch(catalog::build(ref sources)){case .ok(c):{c}case .err(e):{failed("catalog",e.code,e.module_index);return;}};
    assert(catalog::module_count(ref metadata)==count);i=0;
    while(i<count){assert(catalog::source_matches(ref metadata,i,v::at<syntax::Parsed>(ref sources,i)));i+=1;}
    let plan=switch(units::build(ref sources)){case .ok(p):{p}case .err(e):{failed("units",e.code,e.module_index);return;}};
    assert(units::file_count(ref plan)==count && units::source_bytes(ref plan)==bytes_total+count-1);
    var usize next=0;var usize grouped_bytes=0;i=0;
    while(i<units::count(ref plan)){
        let unit=units::at(ref plan,i);assert(unit.first==next && unit.stop>unit.first && unit.stop<=count);
        assert(unit.usage.bytes<=limits::program_bytes() && unit.usage.declarations<=512 && unit.usage.expressions<=limits::program_expressions());
        assert(unit.usage.statements<=8192 && unit.usage.entries<=1024 && unit.usage.types<=4096 && unit.usage.fields<=1024);
        assert(unit.usage.attributes<=4096 && unit.usage.arguments<=4096);grouped_bytes+=unit.usage.bytes;next=unit.stop;i+=1;
    }
    assert(next==count && grouped_bytes+units::count(ref plan)-1==units::source_bytes(ref plan));
    discard io::write("units ");discard io::write_usize(units::count(ref plan));discard io::write(" joined bytes ");discard io::write_usize(units::source_bytes(ref plan));discard io::write("\n");
}
