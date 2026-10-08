module tests::sdk_reachability;
import sdk_reachability as reach;
import module_signatures as signatures;
import tests::sdk_vector_expand as support;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn usize declaration(ref syntax::Parsed p,ref u8[] name){var usize i=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(bytes::equal(syntax::source_view(p)[d.name_start..d.name_end],name)){return i;}i+=1;}assert(false);return 0;}
@effects(filesystem) @test fn void sdk_dependencies_aliases_and_all_user_bodies_are_retained(){
    let input=support::sdk_source();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(b"module user;import std::bytes as b;import std::bytes as alias;fn usize main(){let b::Buffer x=alias::empty();return b::len(ref x);}fn i32 invalid_unused(){return true;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(bytes::view(ref input))).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};var usize work=0;
    let plan=switch(reach::build(ref sources,ref metadata,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let sdk=v::at<syntax::Parsed>(ref sources,1);assert(reach::needed(ref plan,0,3) && reach::needed(ref plan,0,4));
    assert(reach::needed(ref plan,1,declaration(sdk,b"empty")) && reach::needed(ref plan,1,declaration(sdk,b"empty_buffer")) && reach::needed(ref plan,1,declaration(sdk,b"len")));
    assert(!reach::needed(ref plan,1,declaration(sdk,b"with_capacity")) && !reach::needed(ref plan,1,declaration(sdk,b"signed_decimal_digit")));
    work=1048576;switch(reach::build(ref sources,ref metadata,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10);}};
    discard v::clear<syntax::Parsed>(ref mut sources);work=0;switch(reach::build(ref sources,ref metadata,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==9);}};
}
@effects(filesystem) @test fn void private_sdk_reference_preserves_the_privacy_target(){
    let input=support::sdk_source();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(b"module user;import std::bytes as b;fn i32 main(){return b::signed_decimal_digit(1) as i32;}")).is_ok && v::push<syntax::Parsed>(ref mut sources,support::text(bytes::view(ref input))).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};var usize work=0;
    let plan=switch(reach::build(ref sources,ref metadata,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(reach::needed(ref plan,1,declaration(v::at<syntax::Parsed>(ref sources,1),b"signed_decimal_digit")) && signatures::declaration_lookup(ref metadata,0,b"b::signed_decimal_digit").code==6);
}
@test fn void scanner_snapshot_preserves_revision_cooked_and_diagnostics(){
    var bool previous=false;var usize round=0;while(round<2){
        let input=switch(bytes::copy(b"/// doc\r\nb\"\\x41\" /* open")){case .ok(b):{b}case .err(_):{assert(false);return;}};
        let original=switch(scanner::collect(input,previous,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return;}};var usize work=0;
        let snapshot=switch(scanner::copy(ref original,ref mut work)){case .ok(s):{s}case .err(_):{assert(false);return;}};
        assert(bytes::equal(scanner::source_view(ref original),scanner::source_view(ref snapshot)) && scanner::token_count(ref original)==scanner::token_count(ref snapshot) && scanner::diagnostic_count(ref original)==scanner::diagnostic_count(ref snapshot));
        var usize i=0;while(i<scanner::token_count(ref original)){let a=scanner::token_info(ref original,i);let b=scanner::token_info(ref snapshot,i);assert(a.kind==b.kind && a.trivia==b.trivia && a.start==b.start && a.end==b.end && bytes::equal(scanner::token_cooked(ref original,i),scanner::token_cooked(ref snapshot,i)));i+=1;}
        i=0;while(i<scanner::diagnostic_count(ref original)){let a=scanner::diagnostic_info(ref original,i);let b=scanner::diagnostic_info(ref snapshot,i);assert(a.code==b.code && a.start==b.start && a.end==b.end && a.exit_repair==b.exit_repair && bytes::equal(scanner::diagnostic_token(ref original,i),scanner::diagnostic_token(ref snapshot,i)));i+=1;}
        work=1048576;switch(scanner::copy(ref original,ref mut work)){case .ok(_):{assert(false);}case .err(code):{assert(code==10);}};
        previous=true;round+=1;
    }
}
