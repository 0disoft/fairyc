module tests::source_limits;
import candidate_limits as limits;
import module_catalog as catalog;
import module_units as units;
import module_ast_join as join;
import module_program as program;
import scanner as scanner;
import syntax as syntax;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn syntax::Parsed parsed(bytes::Buffer source){
    let stream=switch(scanner::collect(source,false,limits::source_bytes(),1048576,256,16777216)){case .ok(s):{s}case .err(code):{discard io::write_stderr("scanner ");discard io::write_stderr_i32(code);assert(false);return parsed(bytes::empty());}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(code):{discard io::write_stderr("parser ");discard io::write_stderr_i32(code);assert(false);return parsed(bytes::empty());}};
    if(syntax::problem_count(ref p)!=0){discard io::write_stderr("parse problems\n");}assert(syntax::problem_count(ref p)==0);return p;
}
@effects(console) fn syntax::Parsed module_sized(usize index,usize size){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module m").is_ok);
    var u8[2] suffix=[97,97];var usize n=index;while(n>=26){suffix[0]+=1;n-=26;}while(n>0){suffix[1]+=1;n-=1;}
    assert(bytes::append(ref mut source,ref suffix).is_ok && bytes::append(ref mut source,b";").is_ok);
    if(size!=0){assert(size>=bytes::len(ref source)+4 && size<=limits::source_bytes());assert(bytes::append(ref mut source,b"/*").is_ok);while(bytes::len(ref source)<size-2){assert(bytes::append(ref mut source,b" ").is_ok);}assert(bytes::append(ref mut source,b"*/").is_ok);}
    return parsed(source);
}
@effects(console) fn syntax::Parsed module_at(usize index,bool padded){return module_sized(index,if(padded){limits::source_bytes()}else{0});}
@test @effects(console) fn void source_set_boundaries_are_consistent(){
    var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();var usize i=0;
    while(i<limits::source_files()){assert(v::push<syntax::Parsed>(ref mut files,module_at(i,false)).is_ok);i+=1;}
    let owned=switch(catalog::build(ref files)){case .ok(c):{c}case .err(e):{discard io::write_stderr("catalog ");discard io::write_stderr_u32(e.code);assert(false);return;}};assert(catalog::module_count(ref owned)==256);
    let joined=switch(join::join(ref files)){case .ok(p):{p}case .err(e):{discard io::write_stderr("join ");discard io::write_stderr_u32(e.code);discard io::write_stderr_usize(e.module_index);assert(false);return;}};
    assert(v::push<syntax::Parsed>(ref mut files,module_at(256,false)).is_ok);
    assert(!catalog::build(ref files).is_ok && !join::join(ref files).is_ok);
    discard v::clear<syntax::Parsed>(ref mut files);i=0;
    while(i<16){assert(v::push<syntax::Parsed>(ref mut files,module_at(i,true)).is_ok);i+=1;}
    assert(catalog::build(ref files).is_ok); // Exactly 1 MiB, before separators.
    assert(!join::join(ref files).is_ok); // Separators count toward the same cap.
    assert(v::push<syntax::Parsed>(ref mut files,module_at(16,true)).is_ok);
    assert(catalog::build(ref files).is_ok && units::build(ref files).is_ok);
    assert(!join::join(ref files).is_ok); // A single joined owner still has 1 MiB.
    i=17;while(i<32){assert(v::push<syntax::Parsed>(ref mut files,module_at(i,true)).is_ok);i+=1;}
    assert(catalog::build(ref files).is_ok); // Exactly 2 MiB of raw catalog source.
    assert(!units::build(ref files).is_ok); // Its separators would exceed 2 MiB.
    assert(v::push<syntax::Parsed>(ref mut files,module_at(32,false)).is_ok);
    assert(!catalog::build(ref files).is_ok);
    discard v::pop<syntax::Parsed>(ref mut files);discard v::pop<syntax::Parsed>(ref mut files);
    assert(v::push<syntax::Parsed>(ref mut files,module_sized(31,limits::source_bytes()-31)).is_ok);
    let split=switch(units::build(ref files)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(units::source_bytes(ref split)==limits::source_set_bytes() && units::count(ref split)==3);
    assert(catalog::build(ref files).is_ok && !join::join(ref files).is_ok);
    discard v::pop<syntax::Parsed>(ref mut files);
    assert(v::push<syntax::Parsed>(ref mut files,module_sized(31,limits::source_bytes()-30)).is_ok);
    assert(catalog::build(ref files).is_ok && !units::build(ref files).is_ok); // One separator-inclusive byte over.
}
@test @effects(console) fn void large_joined_spans_survive_owned_hir_and_mir(){
    var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut files,module_at(0,true)).is_ok);
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=7};}fn i32 main(){let R r=make();return r.n;}").is_ok);
    assert(v::push<syntax::Parsed>(ref mut files,parsed(source)).is_ok);
    let mir=switch(program::compile(ref files)){case .ok(p):{p}case .err(e):{discard io::write_stderr("program ");discard io::write_stderr_u32(e.phase);discard io::write_stderr_u32(e.code);assert(false);return;}};
    var bytes::Buffer out=bytes::empty();assert(c::render(ref mir,ref mut out)==0 && bytes::len(ref out)>0);
}

@test @effects(console) fn void parser_expression_arena_uses_the_bounded_unit_limit(){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module expressions;fn i32 f(){").is_ok);
    var usize i=0;while(i<1092){assert(bytes::append(ref mut source,b"discard 0+0+0+0+0+0+0+0;").is_ok);i+=1;}
    let accepted=switch(bytes::copy(bytes::view(ref source))){case .ok(b):{b}case .err(_):{assert(false);return;}};
    var bytes::Buffer complete=accepted;assert(bytes::append(ref mut complete,b"return 0;}").is_ok);
    let p=parsed(complete);assert(syntax::expression_count(ref p)==16381 && syntax::expression_count(ref p)<=limits::program_expressions());
    assert(bytes::append(ref mut source,b"discard 0+0+0+0+0+0+0+0;return 0;}").is_ok);
    let stream=switch(scanner::collect(source,false,limits::source_bytes(),1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return;}};
    switch(syntax::parse(stream,512)){case .ok(_):{assert(false);}case .err(code):{assert(code==4);}};
}
