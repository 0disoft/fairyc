module tests::module_budget_probe;
import file_source as files;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
import std::vec as v;
import module_ast_join as join;

fn bytes::Buffer argument(usize index){var bytes::Buffer b=bytes::empty();assert(bytes::resize_zeroed(ref mut b,4096).is_ok);let usize n=switch(process::arg(index,bytes::view_mut(ref mut b))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};assert(bytes::resize_zeroed(ref mut b,n).is_ok);return b;}
@effects(filesystem,console) @test fn void token_cli_input_exceeds_the_existing_join_entry_limit(){
    let name=argument(1);let root=argument(2);
    let manifest=switch(files::load(bytes::view(ref name))){case .ok(b):{b}case .err(_):{assert(false);return;}};
    let lines=bytes::view(ref manifest);var usize start=0;var usize entries=0;var usize types=0;var usize fields=0;var usize declarations=0;var usize expressions=0;
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    while(start<lines.len){var usize end=start;while(end<lines.len && lines[end]!=10){end+=1;}
        let usize stop=if(end>start && lines[end-1]==13){end-1}else{end};
        var bytes::Buffer path=bytes::empty();assert(bytes::append(ref mut path,bytes::view(ref root)).is_ok && bytes::append_byte(ref mut path,47).is_ok && bytes::append(ref mut path,lines[start..stop]).is_ok);
        let input=switch(files::load(bytes::view(ref path))){case .ok(b):{b}case .err(_):{assert(false);return;}};
        let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return;}};
        let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref p)==0);
        entries+=syntax::entry_count(ref p);types+=syntax::type_count(ref p);fields+=syntax::field_count(ref p);declarations+=syntax::declaration_count(ref p);expressions+=syntax::expression_count(ref p);
        assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);start=end+1;
    }
    assert(entries>1024);switch(join::join(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==10);}};
    discard io::write("entries=");discard io::write_usize(entries);discard io::write(" types=");discard io::write_usize(types);discard io::write(" fields=");discard io::write_usize(fields);discard io::write(" declarations=");discard io::write_usize(declarations);discard io::write(" expressions=");discard io::write_usize(expressions);discard io::write("\n");
}
