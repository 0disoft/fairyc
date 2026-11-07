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
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module m").is_ok && bytes::append_usize(ref mut source,index).is_ok && bytes::append(ref mut source,b";").is_ok);
    if(size!=0){
        assert(size>=bytes::len(ref source)+4 && size<=limits::source_bytes());assert(bytes::append(ref mut source,b"/*").is_ok);
        var u8[4096] spaces=[32;4096];let ref u8[] padding=ref spaces;
        while(bytes::len(ref source)<size-2){let usize remaining=size-2-bytes::len(ref source);let usize count=if(remaining<4096){remaining}else{4096};assert(bytes::append(ref mut source,padding[..count]).is_ok);}
        assert(bytes::append(ref mut source,b"*/").is_ok);
    }
    return parsed(source);
}
@effects(console) fn syntax::Parsed module_at(usize index,bool padded){return module_sized(index,if(padded){limits::source_bytes()}else{0});}
@test @effects(console) fn void source_set_boundaries_are_consistent(){
    assert(limits::source_files()==1024 && limits::source_set_bytes()==8388608);
    assert(limits::source_bytes()==65536 && limits::program_bytes()==1048576);
    var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();var usize i=0;
    while(i<limits::source_files()){assert(v::push<syntax::Parsed>(ref mut files,module_at(i,false)).is_ok);i+=1;}
    let owned=switch(catalog::build(ref files)){case .ok(c):{c}case .err(e):{discard io::write_stderr("catalog ");discard io::write_stderr_u32(e.code);assert(false);return;}};assert(catalog::module_count(ref owned)==limits::source_files());
    assert(catalog::source_matches(ref owned,1023,v::at<syntax::Parsed>(ref files,1023)));
    let grouped=switch(units::build(ref files)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(units::file_count(ref grouped)==1024 && units::count(ref grouped)==2);
    i=0;while(i<2){
        let unit=units::at(ref grouped,i);assert(unit.first==i*512 && unit.stop==(i+1)*512 && unit.usage.declarations==512);
        let joined=switch(units::joined(ref grouped,ref files,i)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(syntax::declaration_count(ref joined)==512);i+=1;
    }
    switch(join::join_range(ref files,0,513)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10 && e.module_index==512);}};
    switch(join::join(ref files)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10 && e.module_index==512);}};
    assert(v::push<syntax::Parsed>(ref mut files,module_at(limits::source_files(),false)).is_ok);
    switch(catalog::build(ref files)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10);}};
    switch(units::build(ref files)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10);}};
    switch(join::join(ref files)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10);}};
    discard v::clear<syntax::Parsed>(ref mut files);i=0;
    while(i<16){assert(v::push<syntax::Parsed>(ref mut files,module_at(i,true)).is_ok);i+=1;}
    assert(catalog::build(ref files).is_ok); // Exactly 1 MiB, before separators.
    assert(!join::join(ref files).is_ok); // Separators count toward the same cap.
    assert(v::push<syntax::Parsed>(ref mut files,module_at(16,true)).is_ok);
    assert(catalog::build(ref files).is_ok && units::build(ref files).is_ok);
    assert(!join::join(ref files).is_ok); // A single joined owner still has 1 MiB.
    let usize full_files=limits::source_set_bytes()/limits::source_bytes();assert(full_files==128);
    i=17;while(i<full_files){assert(v::push<syntax::Parsed>(ref mut files,module_at(i,true)).is_ok);i+=1;}
    assert(catalog::build(ref files).is_ok); // Exactly the raw catalog byte cap.
    assert(!units::build(ref files).is_ok); // Separators exceed the aggregate cap.
    discard v::pop<syntax::Parsed>(ref mut files);
    let extra=module_at(full_files,false);let usize extra_bytes=syntax::source_view(ref extra).len;
    assert(v::push<syntax::Parsed>(ref mut files,module_sized(full_files-1,limits::source_bytes()-extra_bytes+1)).is_ok);
    assert(v::push<syntax::Parsed>(ref mut files,extra).is_ok);
    switch(catalog::build(ref files)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10 && e.module_index==full_files);}}; // Exactly one raw byte over.
    discard v::pop<syntax::Parsed>(ref mut files);discard v::pop<syntax::Parsed>(ref mut files);
    assert(v::push<syntax::Parsed>(ref mut files,module_sized(full_files-1,limits::source_bytes()-(full_files-1))).is_ok);
    let split=switch(units::build(ref files)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(units::source_bytes(ref split)==limits::source_set_bytes() && units::count(ref split)==(full_files+14)/15);
    assert(catalog::build(ref files).is_ok && !join::join(ref files).is_ok);
    discard v::pop<syntax::Parsed>(ref mut files);
    assert(v::push<syntax::Parsed>(ref mut files,module_sized(full_files-1,limits::source_bytes()-(full_files-2))).is_ok);
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
