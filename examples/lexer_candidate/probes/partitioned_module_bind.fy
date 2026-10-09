module main;
import module_units as units;
import module_unit_bind as binding;
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
    let bound_metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{return 2;}};
    let plan=switch(units::build(ref sources)){case .ok(p):{p}case .err(p):{number("planError=",p.code as usize);number(" module=",p.module_index);discard io::write("\n");return 2;}};
    number("files=",units::file_count(ref plan));number(" joinedBytes=",units::source_bytes(ref plan));number(" units=",units::count(ref plan));discard io::write("\n");
    var usize modules=0;var usize expressions=0;var usize entries=0;var usize joined_bytes=0;var usize targets=0;var usize max_work=0;i=0;
    while(i<units::count(ref plan)){let unit=units::at(ref plan,i);let bound=switch(binding::bind(ref sources,ref bound_metadata,unit.first,unit.stop)){case .ok(p):{p}case .err(p):{number("bindError=",p.code as usize);number(" module=",p.module_index);number(" start=",p.start);number(" end=",p.end);discard io::write("\n");return 2;}};let ref syntax::Parsed p=binding::parsed(ref bound);
        assert(syntax::module_count(p)==unit.stop-unit.first && syntax::source_view(p).len==unit.usage.bytes && syntax::declaration_count(p)==unit.usage.declarations && syntax::expression_count(p)==unit.usage.expressions && syntax::statement_count(p)==unit.usage.statements && syntax::entry_count(p)==unit.usage.entries && syntax::type_count(p)==unit.usage.types && syntax::field_count(p)==unit.usage.fields && syntax::attribute_count(p)==unit.usage.attributes && syntax::attribute_argument_count(p)==unit.usage.arguments);
        var usize d=1;while(d<=syntax::declaration_count(p)){let c=binding::definition_at(ref bound,d);let original=syntax::declaration_at(v::at<syntax::Parsed>(ref sources,c.module_index),c.declaration-1);assert(c.kind==original.kind && c.start==original.start && c.end==original.end);d+=1;}
        d=1;while(d<=binding::target_count(ref bound)){let c=binding::target_at(ref bound,d);let original=syntax::declaration_at(v::at<syntax::Parsed>(ref sources,c.module_index),c.declaration-1);assert(c.kind==original.kind && c.start==original.start && c.end==original.end);d+=1;}
        targets+=binding::target_count(ref bound);if(binding::work(ref bound)>max_work){max_work=binding::work(ref bound);}assert(binding::work(ref bound)<=1048576);
        modules+=syntax::module_count(p);expressions+=syntax::expression_count(p);entries+=syntax::entry_count(p);joined_bytes+=syntax::source_view(p).len;
        number("unit=",i);number(" first=",unit.first);number(" stop=",unit.stop);number(" bytes=",unit.usage.bytes);number(" declarations=",unit.usage.declarations);number(" expressions=",unit.usage.expressions);number(" statements=",unit.usage.statements);number(" entries=",unit.usage.entries);number(" types=",unit.usage.types);number(" fields=",unit.usage.fields);number(" attributes=",unit.usage.attributes);number(" arguments=",unit.usage.arguments);number(" targets=",binding::target_count(ref bound));number(" work=",binding::work(ref bound));discard io::write("\n");i+=1;
    }
    assert(modules==units::file_count(ref plan) && joined_bytes+units::count(ref plan)-1==units::source_bytes(ref plan));number("joinedModules=",modules);number(" totalExpressions=",expressions);number(" totalEntries=",entries);number(" totalTargets=",targets);number(" maxWork=",max_work);discard io::write("\n");return 0;
}
