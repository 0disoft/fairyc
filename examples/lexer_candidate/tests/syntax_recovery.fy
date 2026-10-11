module tests::syntax_recovery;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

fn scanner::Stream tokens(ref u8[] source){
    let input=switch(bytes::copy(source)){case .ok(value):{value}case .err(_):{assert(false);return tokens(b"module fallback;");}};
    return switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(value):{value}case .err(_):{assert(false);tokens(b"module fallback;")}};
}

@test fn void declarations_keep_attributes_generics_cleanup_and_original_source(){
    let source=b"module main;import other as p;@repr(C) pub struct Pair{i32 value;}resource Token{i32 value;cleanup{}}fn T id<T>(T value){return value;}pub unsafe fn i32 read(ptr i32 value){return 0;}";
    let parsed=switch(syntax::parse(tokens(source),512)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(syntax::problem_count(ref parsed)==0 && syntax::declaration_count(ref parsed)==6);
    assert(bytes::equal(syntax::source_view(ref parsed),source));
    let pair=syntax::declaration_at(ref parsed,2);assert(pair.kind==4 && pair.public && pair.attribute_count==1 && pair.field_count==1);
    let token=syntax::declaration_at(ref parsed,3);assert(token.kind==6 && token.cleanup_body!=0);
    let generic=syntax::declaration_at(ref parsed,4);assert(generic.kind==3 && syntax::generic_parameter_count(ref parsed,4)==1);
    let unsafe_function=syntax::declaration_at(ref parsed,5);assert(unsafe_function.kind==3 && unsafe_function.public && unsafe_function.unsafe_end>unsafe_function.unsafe_start);
}

@test fn void invalid_body_rolls_back_arenas_and_recovers_the_next_declaration(){
    let source=b"module main;fn i32 broken(){let i32 value=;}fn i32 good(){return 7;}";
    let parsed=switch(syntax::parse(tokens(source),512)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(syntax::problem_count(ref parsed)==1 && syntax::declaration_count(ref parsed)==2);
    let clean=switch(syntax::parse(tokens(b"module main;fn i32 good(){return 7;}"),512)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(syntax::entry_count(ref parsed)==syntax::entry_count(ref clean) && syntax::expression_count(ref parsed)==syntax::expression_count(ref clean));
    assert(syntax::statement_count(ref parsed)==syntax::statement_count(ref clean) && syntax::type_count(ref parsed)==syntax::type_count(ref clean));
    let good=syntax::declaration_at(ref parsed,1);assert(bytes::equal(syntax::source_view(ref parsed)[good.name_start..good.name_end],b"good"));
    let problem=syntax::problem_at(ref parsed,0);assert(problem.end>=problem.start && problem.start>0 && problem.end<=source.len);
}

@test fn void diagnostic_parse_retains_lexical_failures_and_declaration_limits(){
    let source=b"module main;fn i32 good(){return 0;} $";
    switch(syntax::parse(tokens(source),512)){case .ok(_):{assert(false);}case .err(code):{assert(code==5);}};
    let parsed=switch(syntax::parse_diagnostics(tokens(source),512)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(scanner::diagnostic_count(syntax::source_stream(ref parsed))!=0);
    assert(syntax::declaration_count(ref parsed)==2 && bytes::equal(syntax::source_view(ref parsed),source));
    switch(syntax::parse(tokens(b"module main;fn i32 good(){return 0;}"),1)){case .ok(_):{assert(false);}case .err(code):{assert(code==4);}};
    switch(syntax::parse_diagnostics(tokens(b"module main;"),513)){case .ok(_):{assert(false);}case .err(code):{assert(code==4);}};
}
