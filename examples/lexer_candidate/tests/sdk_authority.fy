module tests::sdk_authority;
import sdk_authority as sdk;
import sha256 as hash;
import module_catalog as catalog;
import module_types as types;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::fs as fs;
import std::process as process;
import std::text as utf8;

@effects(filesystem)
fn bytes::Buffer load(ref str path){
    var fs::File file=switch(fs::open_read_utf8(path)){case .ok(f):{f}case .err(_):{assert(false);return bytes::empty();}};
    var bytes::Buffer b=bytes::empty();assert(bytes::resize_zeroed(ref mut b,65537).is_ok);var usize used=0;
    while(true){let usize n=switch(fs::read(ref mut file,bytes::view_mut(ref mut b)[used..])){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};used+=n;assert(used<=65536);if(n==0){break;}}
    assert(fs::close(file).is_ok && bytes::resize_zeroed(ref mut b,used).is_ok);return b;
}
fn syntax::Parsed parsed(bytes::Buffer input){let s=switch(scanner::collect(input,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}
fn syntax::Parsed text(ref u8[] source){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,source).is_ok);return parsed(b);}
fn usize result_root(ref syntax::Parsed p,usize ordinal){var usize i=0;var usize count=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==3){if(count==ordinal){return d.result_type;}count+=1;}i+=1;}assert(false);return 0;}
@test fn void sha256_standard_padding_and_modular_arithmetic(){
    assert(hash::matches(b"",b"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"));
    assert(hash::matches(b"abc",b"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"));
    assert(hash::matches(b"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq",b"248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1"));
    assert(!hash::matches(b"abc",b"00") && !sdk::authentic(b"std::vec",b"module std::vec;pub resource Vec<T>{cleanup{}}"));
}
@effects(filesystem)
@test fn void authenticated_vec_graph_retains_element_and_provider(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let filename=switch(utf8::validate_utf8(bytes::view(ref path)[..n])){case .ok(s):{s}case .err(_):{assert(false);return;}};let source=load(filename);assert(sdk::authentic(b"std::vec",bytes::view(ref source)) && !sdk::authentic(b"std::bytes",bytes::view(ref source)));
    var bytes::Buffer altered=bytes::empty();assert(bytes::append(ref mut altered,bytes::view(ref source)).is_ok && bytes::append_byte(ref mut altered,10).is_ok);assert(!sdk::authentic(b"std::vec",bytes::view(ref altered)));
    let ref u8[] user=b"module user;import std::vec as v;import std::vec as same;resource Token{i32 n;cleanup{}}fn v::Vec<i64> a(){}fn same::Vec<i64> b(){}fn v::Vec<Token> c(){}fn ref mut v::Vec<i64> d(){}fn v::Vec<ref i32> invalid(){}fn v::Vec<Option<ref i32>> nested_invalid(){}fn v::Vec bare(){}";
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,text(user)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(source)).is_ok);
    let c=switch(catalog::build(ref sources)){case .ok(c):{c}case .err(_):{assert(false);return;}};assert(catalog::sdk_vector_source(ref c,1) && !catalog::sdk_vector_source(ref c,0) && !catalog::sdk_vector_source(ref c,2));let p=v::at<syntax::Parsed>(ref sources,0);var v::Vec<types::Type> graph=v::empty<types::Type>();
    let a=types::resolve(ref c,p,0,result_root(p,0),ref mut graph);let b=types::resolve(ref c,p,0,result_root(p,1),ref mut graph);let token=types::resolve(ref c,p,0,result_root(p,2),ref mut graph);let borrowed=types::resolve(ref c,p,0,result_root(p,3),ref mut graph);
    assert(a.problem.code==0 && b.type==a.type && token.problem.code==0 && token.type!=a.type && borrowed.problem.code==0);
    let t=types::at(ref graph,a.type);assert(t.kind==13 && t.module_index==1 && types::at(ref graph,t.first).scalar==2 && types::at(ref graph,borrowed.type).first==a.type);
    let usize before=types::count(ref graph);assert(types::resolve(ref c,p,0,result_root(p,4),ref mut graph).problem.code==9 && types::count(ref graph)==before);
    assert(types::resolve(ref c,p,0,result_root(p,5),ref mut graph).problem.code==9 && types::count(ref graph)==before);assert(types::resolve(ref c,p,0,result_root(p,6),ref mut graph).problem.code==9 && types::count(ref graph)==before);
    var usize i=0;while(i<6){assert(sdk::vector_field(i)==if(i==0){1048591}else{5});i+=1;}assert(sdk::vector_field(6)==0 && types::validate(ref c,ref graph)==0);
    {let ref mut types::Type forged=v::at_mut<types::Type>(ref mut graph,a.type-1);forged.module_index=0;}assert(types::validate(ref c,ref graph)==9);
    var v::Vec<syntax::Parsed> fake=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut fake,text(b"module std::vec;pub resource Vec<T>{cleanup{}}" )).is_ok);assert(switch(catalog::build(ref fake)){case .ok(_):{false}case .err(p):{p.code==6801}});
}
