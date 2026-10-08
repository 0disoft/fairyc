module main;
import scanner as scanner;
import syntax as syntax;
import module_signatures as signatures;
import sdk_ast_compact as compact;
import std::vec as v;
import file_source as files;
import std::bytes as bytes;
import std::io as io;
import std::process as process;

@effects(filesystem,console)
fn i32 main(){
    var bytes::Buffer arg=bytes::empty();assert(bytes::resize_zeroed(ref mut arg,4096).is_ok);
    let usize n=switch(process::arg(1,bytes::view_mut(ref mut arg))){case .ok(n):{n}case .err(_):{return 2;}};
    let manifest=switch(files::manifest(bytes::view(ref arg)[..n])){case .ok(b):{b}case .err(_):{return 2;}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    let list=bytes::view(ref manifest);var bytes::Buffer joined=bytes::empty();var usize start=0;var usize count=0;var usize cooked=0;
    while(start<list.len){var usize end=start;while(end<list.len && list[end]!=10){end+=1;}
        let usize stop=if(end>start && list[end-1]==13){end-1}else{end};
        let input=switch(files::load(list[start..stop])){case .ok(b):{b}case .err(_):{return 2;}};
        if(count!=0){assert(bytes::append(ref mut joined,b"\n").is_ok);}assert(bytes::append(ref mut joined,bytes::view(ref input)).is_ok);
        let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{return 2;}};
        cooked+=scanner::storage(ref s).cooked_bytes;
        let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{return 2;}};assert(syntax::problem_count(ref p)==0);assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);count+=1;start=end+1;
    }
    discard io::write("files=");discard io::write_usize(count);discard io::write(" cooked=");discard io::write_usize(cooked);discard io::write(" bytes=");discard io::write_usize(bytes::len(ref joined));
    let duplicate=switch(bytes::copy(bytes::view(ref joined))){case .ok(b):{b}case .err(_):{return 2;}};
    switch(scanner::collect(duplicate,false,1048576,1048576,256,16777216)){case .ok(_):{discard io::write(" status=0");}case .err(code):{discard io::write(" status=");discard io::write_i32(code);}};
    switch(scanner::collect_joined(joined)){case .ok(s):{discard io::write(" sharedStatus=0 owned=");discard io::write_usize(scanner::storage(ref s).cooked_bytes);var usize logical=0;var usize i=0;while(i<scanner::token_count(ref s)){let t=scanner::token_info(ref s,i);if(t.kind>=5 && t.kind<=9){logical+=scanner::token_cooked(ref s,i).len;}i+=1;}discard io::write(" diagnostics=");discard io::write_usize(scanner::diagnostic_count(ref s));discard io::write(" logical=");discard io::write_usize(logical);discard io::write("\n");}case .err(code):{discard io::write(" sharedStatus=");discard io::write_i32(code);discard io::write("\n");}};
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{return 2;}};
    switch(compact::compact_in_place(ref mut sources,ref metadata)){case .ok(_):{}case .err(_):{return 2;}};
    var usize declarations=0;var usize expressions=0;var usize statements=0;var usize entries=0;var usize types=0;var usize fields=0;var usize attributes=0;var usize arguments=0;var usize at=0;
    while(at<v::len<syntax::Parsed>(ref sources)){let ref syntax::Parsed p=v::at<syntax::Parsed>(ref sources,at);
        declarations+=syntax::declaration_count(p);expressions+=syntax::expression_count(p);statements+=syntax::statement_count(p);entries+=syntax::entry_count(p);types+=syntax::type_count(p);fields+=syntax::field_count(p);attributes+=syntax::attribute_count(p);arguments+=syntax::attribute_argument_count(p);
        if(declarations>512 || expressions>16384 || statements>8192 || entries>1024 || types>4096 || fields>1024 || attributes>4096 || arguments>4096){
            discard io::write("arenaModule=");discard io::write_usize(at);discard io::write(" declarations=");discard io::write_usize(declarations);discard io::write(" expressions=");discard io::write_usize(expressions);discard io::write(" statements=");discard io::write_usize(statements);discard io::write(" entries=");discard io::write_usize(entries);discard io::write(" types=");discard io::write_usize(types);discard io::write(" fields=");discard io::write_usize(fields);discard io::write(" attributes=");discard io::write_usize(attributes);discard io::write(" arguments=");discard io::write_usize(arguments);discard io::write("\n");break;
        }at+=1;
    }
    return 0;
}
