module main;
import module_units as units;
import module_signatures as signatures;
import sdk_ast_compact as compact;
import module_cli as cli;
import file_source as files;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import std::process as process;

@effects(console)
fn void number(ref str key,usize value){discard io::write(key);discard io::write_usize(value);}
@effects(filesystem,console)
fn i32 main(){
    var bytes::Buffer arg=bytes::empty();assert(bytes::resize_zeroed(ref mut arg,4096).is_ok);
    let usize n=switch(process::arg(1,bytes::view_mut(ref mut arg))){case .ok(n):{n}case .err(_):{return 2;}};
    let manifest=switch(files::manifest(bytes::view(ref arg)[..n])){case .ok(b):{b}case .err(_):{return 2;}};
    let source=bytes::view(ref manifest);var v::Vec<cli::Path> paths=v::empty<cli::Path>();assert(cli::paths(source,ref mut paths)==0);
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<v::len<cli::Path>(ref paths)){let path=*v::at<cli::Path>(ref paths,i);
        let input=switch(files::load(source[path.start..path.end])){case .ok(b):{b}case .err(_):{return 2;}};
        let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{return 2;}};assert(scanner::diagnostic_count(ref stream)==0);
        let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{return 2;}};assert(syntax::problem_count(ref p)==0);assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);i+=1;
    }
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{return 2;}};
    switch(compact::compact_in_place(ref mut sources,ref metadata)){case .ok(_):{}case .err(_):{return 2;}};
    let plan=switch(units::build(ref sources)){case .ok(p):{p}case .err(p):{number("planError=",p.code as usize);number(" module=",p.module_index);discard io::write("\n");return 2;}};
    number("files=",units::file_count(ref plan));number(" joinedBytes=",units::source_bytes(ref plan));number(" units=",units::count(ref plan));discard io::write("\n");
    var usize modules=0;var usize expressions=0;var usize entries=0;var usize joined_bytes=0;i=0;
    while(i<units::count(ref plan)){let unit=units::at(ref plan,i);let p=switch(units::joined(ref plan,ref sources,i)){case .ok(p):{p}case .err(p):{number("joinError=",p.code as usize);number(" module=",p.module_index);discard io::write("\n");return 2;}};
        assert(syntax::module_count(ref p)==unit.stop-unit.first && syntax::source_view(ref p).len==unit.usage.bytes && syntax::declaration_count(ref p)==unit.usage.declarations && syntax::expression_count(ref p)==unit.usage.expressions && syntax::statement_count(ref p)==unit.usage.statements && syntax::entry_count(ref p)==unit.usage.entries && syntax::type_count(ref p)==unit.usage.types && syntax::field_count(ref p)==unit.usage.fields && syntax::attribute_count(ref p)==unit.usage.attributes && syntax::attribute_argument_count(ref p)==unit.usage.arguments);
        modules+=syntax::module_count(ref p);expressions+=syntax::expression_count(ref p);entries+=syntax::entry_count(ref p);joined_bytes+=syntax::source_view(ref p).len;
        number("unit=",i);number(" first=",unit.first);number(" stop=",unit.stop);number(" bytes=",unit.usage.bytes);number(" declarations=",unit.usage.declarations);number(" expressions=",unit.usage.expressions);number(" statements=",unit.usage.statements);number(" entries=",unit.usage.entries);number(" types=",unit.usage.types);number(" fields=",unit.usage.fields);number(" attributes=",unit.usage.attributes);number(" arguments=",unit.usage.arguments);discard io::write("\n");i+=1;
    }
    assert(modules==units::file_count(ref plan) && joined_bytes+units::count(ref plan)-1==units::source_bytes(ref plan));number("joinedModules=",modules);number(" totalExpressions=",expressions);number(" totalEntries=",entries);discard io::write("\n");return 0;
}
