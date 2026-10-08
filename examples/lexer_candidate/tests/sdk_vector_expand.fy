module tests::sdk_vector_expand;
import module_ast_join as join;
import module_bind as binding;
import module_signatures as signatures;
import generic_expand as expand;
import syntax as syntax;
import scanner as scanner;
import nominal_types as nominal;
import storage_intrinsics as storage;
import std::bytes as bytes;
import std::vec as v;
import std::fs as fs;
import std::process as process;
import std::text as utf8;

fn syntax::Parsed parsed(bytes::Buffer input){
    let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(bytes::empty());}};
    let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;
}
fn syntax::Parsed text(ref u8[] source){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,source).is_ok);return parsed(b);}
@effects(filesystem)
fn bytes::Buffer sdk_source(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);
    let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};
    let filename=switch(utf8::validate_utf8(bytes::view(ref path)[..n])){case .ok(s):{s}case .err(_):{assert(false);return bytes::empty();}};
    var fs::File file=switch(fs::open_read_utf8(filename)){case .ok(f):{f}case .err(_):{assert(false);return bytes::empty();}};
    var bytes::Buffer input=bytes::empty();assert(bytes::resize_zeroed(ref mut input,65537).is_ok);var usize used=0;
    while(true){let usize read=switch(fs::read(ref mut file,bytes::view_mut(ref mut input)[used..])){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};used+=read;assert(used<=65536);if(read==0){break;}}
    assert(fs::close(file).is_ok && bytes::resize_zeroed(ref mut input,used).is_ok);return input;
}
fn syntax::Parsed expanded(ref u8[] user,ref u8[] sdk){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,text(user)).is_ok && v::push<syntax::Parsed>(ref mut sources,text(sdk)).is_ok);
    let meta=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(e):{assert(false);return text(b"module failed;");}};
    let combined=switch(join::join(ref sources)){case .ok(p):{p}case .err(e):{assert(false);return text(b"module failed;");}};
    let p=switch(binding::bind(ref sources,ref meta,combined)){case .ok(p):{p}case .err(e):{assert(false);return text(b"module failed;");}};
    return expand::expand(p);
}
@effects(filesystem)
@test fn void sdk_vec_expansion_retains_owned_templates(){
    let sdk=sdk_source();let p=expanded(b"module user;import std::vec as v;import std::vec as alias;fn usize main(){var v::Vec<i32> a=v::empty<i32>();var alias::Vec<i32> b=alias::empty<i32>();return v::len<i32>(ref a)+alias::len<i32>(ref b);}",bytes::view(ref sdk));
    assert(syntax::problem_count(ref p)==0);var usize vectors=0;var usize functions=0;var usize storage_calls=0;var usize i=0;
    while(i<syntax::declaration_count(ref p)){let d=syntax::declaration_at(ref p,i);
        if(d.kind==6){vectors+=1;assert(d.generic_parameters==0 && d.generic_start!=0 && d.field_count==6 && d.cleanup_body!=0);
            var usize j=0;while(j<6){let f=syntax::field_at(ref p,d.field_start+j);assert(nominal::resolve(ref p,f.type_root)==if(j==0){1048591}else{5});j+=1;}
        }
        if(d.kind==3 && d.generic_start!=0){functions+=1;}i+=1;
    }
    i=1;while(i<=syntax::expression_count(ref p)){if(storage::call(ref p,i)!=0 && storage::element(ref p,i)!=0){assert(storage::allowed(ref p,i));storage_calls+=1;}i+=1;}
    assert(vectors==1 && functions==3 && storage_calls==3);
    let ref u8[] source=syntax::source_view(ref p);assert(bytes::equal(source[source.len-bytes::len(ref sdk)..],bytes::view(ref sdk)));
    rejected_elements_and_spoofing(bytes::view(ref sdk));
}
fn void rejected_elements_and_spoofing(ref u8[] sdk){
    let p=expanded(b"module user;import std::vec as v;resource Token{i32 n;cleanup{}}fn v::Vec<Token> main(){return v::empty<Token>();}",sdk);
    assert(syntax::problem_count(ref p)==1 && syntax::problem_at(ref p,0).code==4502);
    let spoof=expand::expand(text(b"module std::vec;pub resource Vec<T>{cleanup{}}pub fn Vec<T> empty<T>(){}"));assert(syntax::problem_count(ref spoof)==1 && syntax::problem_at(ref spoof,0).code==6801);
    let shadow=expand::expand(text(b"module user;fn T id<T>(T x){return x;}fn i32 other(i32 id){return id;}fn i32 main(){return id<i32>(7);}"));assert(syntax::problem_count(ref shadow)==0);
    let blocked=expand::expand(text(b"module user;fn T id<T>(T x){return x;}fn i32 main(){let i32 id=7;return id<i32>(7);}"));assert(syntax::problem_count(ref blocked)==1 && syntax::problem_at(ref blocked,0).code==4501);
    let ordinary=expand::expand(text(b"module user;resource Vec<T>{cleanup{}}fn Vec<i32> main(){}"));assert(syntax::problem_count(ref ordinary)==1 && syntax::problem_at(ref ordinary,0).code==4501);
}
