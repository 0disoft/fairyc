module tests::module_unit_sdk_sources;
import module_unit_sdk_sources as owned;
import candidate_limits as limits;
import scanner as scanner;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;

fn syntax::Parsed parsed(ref u8[] text){
    let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    assert(syntax::problem_count(ref p)==0);return p;
}
@effects(filesystem) fn bytes::Buffer input(usize index){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);
    let n=switch(process::arg(index,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};
    return switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};
}
fn owned::Sources retained(ref u8[] alloc,ref u8[] io){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn i32 main(){return 0;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(alloc)).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;fn i32 value(){return 1;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(io)).is_ok);var usize work=0;
    return switch(owned::collect(ref sources,ref mut work)){case .ok(s):{s}case .err(_):{assert(false);owned::empty()}};
}
@test @effects(filesystem) fn void exact_sdk_bytes_survive_parser_release_with_original_module_indices(){
    let alloc=input(1);let io=input(2);let p=retained(bytes::view(ref alloc),bytes::view(ref io));
    assert(owned::module_count(ref p)==4 && owned::count(ref p)==2);
    assert(owned::find(ref p,0)==0 && owned::find(ref p,1)==1 && owned::find(ref p,2)==0 && owned::find(ref p,3)==2);
    assert(owned::at(ref p,0).sdk_id==1 && owned::at(ref p,1).sdk_id==4);
    assert(bytes::equal(owned::source_at(ref p,0),bytes::view(ref alloc)) && bytes::equal(owned::source_at(ref p,1),bytes::view(ref io)));
    var usize used=0;assert(owned::check(ref p,ref mut used).code==0 && used==owned::data(ref p).len+2);
    used=1048576;assert(owned::check(ref p,ref mut used).code==1002);
    var v::Vec<syntax::Parsed> duplicate=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut duplicate,parsed(bytes::view(ref alloc))).is_ok && v::push<syntax::Parsed>(ref mut duplicate,parsed(bytes::view(ref alloc))).is_ok);
    used=0;assert(switch(owned::collect(ref duplicate,ref mut used)){case .ok(_):{false}case .err(e):{e.code==1001}});
    var bytes::Buffer changed=bytes::empty();assert(bytes::append(ref mut changed,bytes::view(ref alloc)).is_ok && bytes::append_byte(ref mut changed,32).is_ok);
    assert(rejected(bytes::view(ref changed))==6801);
}
fn u32 altered(ref owned::Sources p,u32 mode){
    var v::Vec<owned::Unit> units=v::empty<owned::Unit>();var usize i=0;
    while(i<owned::count(p)){var u=owned::at(p,i);
        if(i==0){if(mode==2){u.sdk_id=4;}if(mode==3){u.module_index=4;}if(mode==4){u.module_index=3;}if(mode==5){u.start=1;}}
        if(i==1 && mode==4){u.module_index=1;}
        assert(v::push<owned::Unit>(ref mut units,u).is_ok);i+=1;
    }
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,owned::data(p)).is_ok);
    if(mode==1){let usize last=bytes::len(ref source)-1;bytes::view_mut(ref mut source)[last]=32;}
    if(mode==6){assert(bytes::append_byte(ref mut source,10).is_ok);}
    if(mode==7){let first=owned::at(p,0);let start=bytes::len(ref source);assert(bytes::append(ref mut source,owned::source_at(p,0)).is_ok);assert(v::push<owned::Unit>(ref mut units,owned::Unit{.module_index=4,.sdk_id=first.sdk_id,.start,.end=bytes::len(ref source)}).is_ok);}
    return switch(owned::assemble(if(mode==7){5}else if(mode==10){limits::source_files()+1}else{4},units,source,if(mode==8){1048576}else if(mode==9){1048577}else{0})){case .ok(_):{0}case .err(e):{e.code}};
}
@test @effects(filesystem) fn void retained_sdk_source_mutations_cannot_forge_authentication(){
    let alloc=input(1);let io=input(2);let p=retained(bytes::view(ref alloc),bytes::view(ref io));assert(altered(ref p,0)==0);
    var u32 mode=1;while(mode<=10){assert(altered(ref p,mode)==if(mode>=8){1002}else{1001});mode+=1;}
}
fn u32 rejected(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);var usize work=0;return switch(owned::collect(ref sources,ref mut work)){case .ok(_):{0}case .err(e):{e.code}};}
@test fn void reserved_names_and_empty_authority_are_checked(){
    assert(rejected(b"module std::alloc;pub resource Arena{cleanup{}}")==6801);
    assert(rejected(b"module std::unknown;fn i32 f(){return 0;}")==6801);
    let empty=owned::empty();var usize work=0;assert(owned::count(ref empty)==0 && owned::module_count(ref empty)==0 && owned::check(ref empty,ref mut work).code==0);
    work=1048577;assert(owned::check(ref empty,ref mut work).code==1002);
}
