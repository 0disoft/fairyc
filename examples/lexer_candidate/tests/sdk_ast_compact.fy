module tests::sdk_ast_compact;
import sdk_ast_compact as compact;
import sdk_reachability as reach;
import module_signatures as signatures;
import module_ast_join as join;
import module_bind as binding;
import module_program as programs;
import tests::sdk_vector_expand as support;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn usize declaration(ref syntax::Parsed p,ref u8[] name){var usize i=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(bytes::equal(syntax::source_view(p)[d.name_start..d.name_end],name)){return i+1;}i+=1;}return 0;}
fn void snapshot(ref syntax::Parsed a,ref syntax::Parsed b){
    assert(bytes::equal(syntax::source_view(a),syntax::source_view(b)) && syntax::token_count(a)==syntax::token_count(b));
    var usize i=0;while(i<syntax::token_count(a)){let x=syntax::token_info(a,i);let y=syntax::token_info(b,i);assert(x.kind==y.kind && x.start==y.start && x.end==y.end && x.trivia==y.trivia);i+=1;}
    assert(syntax::field_count(a)==syntax::field_count(b) && syntax::attribute_count(a)==syntax::attribute_count(b) && syntax::attribute_argument_count(a)==syntax::attribute_argument_count(b));
}

@test fn void reachability_ranges_preserve_many_repeated_local_names(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize module_index=0;
    while(module_index<22){var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module user").is_ok && bytes::append_usize(ref mut text,module_index).is_ok && bytes::append(ref mut text,b";fn void value(){let i32 n=0;").is_ok);
        var usize i=0;while(i<800){assert(bytes::append(ref mut text,b"discard n;").is_ok);i+=1;}assert(bytes::append(ref mut text,b"}").is_ok);
        assert(v::push<syntax::Parsed>(ref mut sources,support::text(bytes::view(ref text))).is_ok);module_index+=1;
    }
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    var usize work=0;let plan=switch(reach::build(ref sources,ref metadata,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(work<1048576);module_index=0;while(module_index<22){assert(reach::needed(ref plan,module_index,1));module_index+=1;}
    let retained=switch(compact::compact(ref sources,ref metadata)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(v::len<syntax::Parsed>(ref retained)==22);module_index=0;while(module_index<22){let before=v::at<syntax::Parsed>(ref sources,module_index);let after=v::at<syntax::Parsed>(ref retained,module_index);
        snapshot(before,after);assert(syntax::expression_count(before)==syntax::expression_count(after) && syntax::statement_count(before)==syntax::statement_count(after) && syntax::declaration_count(before)==syntax::declaration_count(after));module_index+=1;
    }
}
@effects(filesystem) @test fn void compacted_sdk_keeps_source_and_user_validation(){
    let input=support::sdk_source();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(b"module user;import std::bytes as b;struct Box{i32 value;}fn i32 main(){let b::Buffer x=b::empty();assert(b::len(ref x)==0);let ref u8[] cooked=b\"\\x41\";return 3;}fn i32 invalid_unused(){return true;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(bytes::view(ref input))).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    let output=switch(compact::compact(ref sources,ref metadata)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let original=v::at<syntax::Parsed>(ref sources,1);let sdk=v::at<syntax::Parsed>(ref output,1);snapshot(original,sdk);
    assert(syntax::entry_count(sdk)<syntax::entry_count(original) && syntax::expression_count(sdk)<syntax::expression_count(original) && syntax::statement_count(sdk)<syntax::statement_count(original) && syntax::type_count(sdk)<syntax::type_count(original));
    assert(declaration(sdk,b"empty")!=0 && declaration(sdk,b"empty_buffer")!=0 && declaration(sdk,b"len")!=0 && declaration(sdk,b"with_capacity")==0 && declaration(sdk,b"signed_decimal_digit")==0);
    let before=v::at<syntax::Parsed>(ref sources,0);let after=v::at<syntax::Parsed>(ref output,0);snapshot(before,after);
    assert(syntax::declaration_count(before)==syntax::declaration_count(after) && syntax::expression_count(before)==syntax::expression_count(after) && syntax::statement_count(before)==syntax::statement_count(after) && syntax::entry_count(before)==syntax::entry_count(after) && syntax::type_count(before)==syntax::type_count(after));
    let meta=switch(signatures::build(ref output)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    let combined=switch(join::join(ref output)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref output,ref meta,combined)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(syntax::entry_count(ref bound)==syntax::entry_count(after)+syntax::entry_count(sdk));
    switch(programs::compile(ref output)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==5 && e.code==1 && e.module_index==0 && e.start<syntax::source_view(before).len);}};
    discard v::clear<syntax::Parsed>(ref mut sources);
    switch(compact::compact(ref sources,ref metadata)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==9);}};
}
@effects(filesystem) @test fn void compacted_private_reference_retains_original_diagnostic(){
    let input=support::sdk_source();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(b"module user;import std::bytes as b;fn i32 main(){return b::signed_decimal_digit(1) as i32;}")).is_ok && v::push<syntax::Parsed>(ref mut sources,support::text(bytes::view(ref input))).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    let output=switch(compact::compact(ref sources,ref metadata)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(declaration(v::at<syntax::Parsed>(ref output,1),b"signed_decimal_digit")!=0);
    switch(programs::compile(ref output)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==3 && e.code==6 && e.module_index==0 && e.start==56);}};
}
