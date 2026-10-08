module tests::module_units;
import module_units as units;
import module_ast_join as join;
import scanner as scanner;
import syntax as syntax;
import std::vec as v;
import std::bytes as bytes;

fn syntax::Parsed parsed(bytes::Buffer input){
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(bytes::empty());}};
    assert(scanner::diagnostic_count(ref stream)==0);let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;
}
fn bytes::Buffer copied(ref u8[] text){return switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);copied(b"")}};}
fn bytes::Buffer generated(usize module_index){
    var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"module user").is_ok && bytes::append_usize(ref mut out,module_index).is_ok && bytes::append(ref mut out,b";fn i32 f(){").is_ok);
    var usize i=0;while(i<200){assert(bytes::append(ref mut out,b"let i32 value").is_ok && bytes::append_usize(ref mut out,i).is_ok && bytes::append(ref mut out,b"=7;").is_ok);i+=1;}assert(bytes::append(ref mut out,b"return 7;}").is_ok);return out;
}

@test fn void separate_joined_owners_preserve_all_entries_under_the_existing_cap(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;while(i<6){assert(v::push<syntax::Parsed>(ref mut sources,parsed(generated(i))).is_ok);i+=1;}
    switch(join::join(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==10 && p.module_index==5);}};
    let plan=switch(units::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(units::count(ref plan)==2 && units::file_count(ref plan)==6 && units::at(ref plan,0).first==0 && units::at(ref plan,0).stop==5 && units::at(ref plan,1).first==5 && units::at(ref plan,1).stop==6);
    var usize total_entries=0;var usize total_bytes=0;i=0;
    while(i<units::count(ref plan)){let unit=units::at(ref plan,i);let p=switch(units::joined(ref plan,ref sources,i)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(syntax::entry_count(ref p)==unit.usage.entries && unit.usage.entries<=1024 && syntax::module_count(ref p)==unit.stop-unit.first);
        assert(syntax::declaration_count(ref p)==unit.usage.declarations && syntax::expression_count(ref p)==unit.usage.expressions && syntax::statement_count(ref p)==unit.usage.statements && syntax::type_count(ref p)==unit.usage.types);
        var bytes::Buffer exact=bytes::empty();var usize module_index=unit.first;
        while(module_index<unit.stop){if(module_index!=unit.first){assert(bytes::append(ref mut exact,b"\n").is_ok);}assert(bytes::append(ref mut exact,syntax::source_view(v::at<syntax::Parsed>(ref sources,module_index))).is_ok);module_index+=1;}
        assert(bytes::equal(syntax::source_view(ref p),bytes::view(ref exact)) && bytes::len(ref exact)==unit.usage.bytes);
        var usize entry_index=0;while(entry_index<syntax::entry_count(ref p)){let e=syntax::entry_at(ref p,entry_index);assert(e.kind==2 || e.kind==4);assert(e.start<e.end && e.end<=syntax::source_view(ref p).len);entry_index+=1;}
        total_entries+=syntax::entry_count(ref p);total_bytes+=syntax::source_view(ref p).len;i+=1;
    }
    assert(total_entries==1206 && total_bytes+units::count(ref plan)-1==units::source_bytes(ref plan));
}

@test fn void ranges_keep_original_error_coordinates_and_reject_invalid_ranges(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(copied(b"module skipped;"))).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(copied(b"module good;fn i32 f(){return 7;}"))).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(copied(b"module bad;struct S<U>{U value;}"))).is_ok);
    let p=switch(join::join_range(ref sources,1,2)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::module_count(ref p)==1 && bytes::equal(syntax::source_view(ref p),syntax::source_view(v::at<syntax::Parsed>(ref sources,1))));
    switch(join::join_range(ref sources,1,3)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4501 && p.module_index==2 && p.start<p.end);}};
    switch(join::join_range(ref sources,2,1)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==9);}};
    switch(join::join_range(ref sources,0,4)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==9);}};
    var v::Vec<syntax::Parsed> empty=v::empty<syntax::Parsed>();switch(units::build(ref empty)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==10);}};
}
