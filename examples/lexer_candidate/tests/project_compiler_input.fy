module tests::project_compiler_input;
import project_compiler_input as input;
import project_workspace_lock as loader;
import project_module_visibility as visibility;
import project_source_set as sources;
import module_catalog as catalog;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import file_source as fixture;
import std::bytes as bytes;
import std::process as process;
import std::vec as v;

@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] path=[0;4096];let count=switch(process::arg(1,ref mut path)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(fixture::manifest(view[..count])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer oracle(ref u8[] root){var bytes::Buffer path=bytes::empty();assert(bytes::append(ref mut path,root).is_ok && bytes::append(ref mut path,b"/inputs.bin").is_ok);return switch(fixture::manifest(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let count=number(data,at);assert(*at<=data.len && count<=data.len-*at);let start=*at;*at+=count;return data[start..*at];}
fn ref u8[] name(ref input::Input owned,usize at){let p=input::source(owned,at);let d=syntax::declaration_at(p,0);return syntax::source_view(p)[d.name_start..d.name_end];}
fn usize find(ref input::Input owned,ref u8[] spelling){var usize at=0;while(at<input::count(owned)){if(bytes::equal(name(owned,at),spelling)){return at;}at+=1;}assert(false);return 0;}
@effects(filesystem)
fn input::Input owned(ref u8[] path){var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;let loaded=switch(loader::load(path,true,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return owned(path);}};return switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return owned(path);}};}
@test
@effects(filesystem)
fn void go_exact_snapshots_survive_workspace_close_and_keep_package_type_identity(){let path=root();let data=oracle(bytes::view(ref path));let packet=bytes::view(ref data);assert(packet.len>=16 && bytes::equal(packet[..8],b"FYINPUT1"));var usize offset=8;let count=number(packet,ref mut offset);let projected=owned(bytes::view(ref path));assert(count==6 && input::count(ref projected)==count);
    var v::Vec<bool> seen=v::empty<bool>();var usize at=0;while(at<count){assert(v::push<bool>(ref mut seen,false).is_ok);at+=1;}at=0;
    while(at<count){let id=field(packet,ref mut offset);let file=field(packet,ref mut offset);let module_name=field(packet,ref mut offset);let source=field(packet,ref mut offset);var usize matches=0;var usize i=0;
        while(i<count){if(bytes::equal(input::instance_id(ref projected,i),id) && bytes::equal(input::file(ref projected,i),file) && bytes::equal(name(ref projected,i),module_name)){assert(!*v::at<bool>(ref seen,i));let slot=v::at_mut<bool>(ref mut seen,i);*slot=true;assert(bytes::equal(syntax::source_view(input::source(ref projected,i)),source));matches+=1;}i+=1;}assert(matches==1);at+=1;
    }assert(offset==packet.len && v::len<catalog::Binding>(input::bindings(ref projected))==5);
    assert(switch(catalog::build(input::sources(ref projected))){case .ok(_):{false}case .err(p):{p.code==1}});
    let c=switch(input::declarations(ref projected)){case .ok(value):{value}case .err(_):{assert(false);return;}};at=0;while(at<count){assert(catalog::module_owner(ref c,at)==input::origin(ref projected,at).owner && catalog::source_matches(ref c,at,input::source(ref projected,at)));at+=1;}
    let p=switch(signatures::build_catalog(input::sources(ref projected),c)){case .ok(value):{value}case .err(_):{assert(false);return;}};let main=find(ref projected,b"main");let left=find(ref projected,b"left");let right=find(ref projected,b"right");
    let local=signatures::lookup(ref p,main,b"c::make");let l=signatures::lookup(ref p,main,b"l::make");let r=signatures::lookup(ref p,main,b"r::make");assert(local.code==0 && l.code==0 && r.code==0);
    let local_type=signatures::at(ref p,local.item).result;let left_type=signatures::at(ref p,l.item).result;let right_type=signatures::at(ref p,r.item).result;assert(local_type!=left_type && left_type!=right_type && right_type!=local_type);
    assert(signatures::at(ref p,signatures::lookup(ref p,left,b"c::make").item).result==left_type && signatures::at(ref p,signatures::lookup(ref p,right,b"c::make").item).result==right_type);
    assert(signatures::lookup(ref p,main,b"l::hidden").code==6 && signatures::lookup(ref p,main,b"r::hidden").code==6);
    assert(signatures::type_at(ref p,left_type).module_index!=signatures::type_at(ref p,right_type).module_index);
}
@test
@effects(filesystem)
fn void scoped_bindings_reject_missing_extra_wrong_provider_and_budget_exhaustion(){let path=root();let projected=owned(bytes::view(ref path));let empty=v::empty<catalog::Binding>();assert(switch(catalog::build_scoped(input::sources(ref projected),input::owners(ref projected),ref empty)){case .ok(_):{false}case .err(p):{p.code==9}});
    var v::Vec<catalog::Binding> bad=v::empty<catalog::Binding>();var usize at=0;while(at<v::len<catalog::Binding>(input::bindings(ref projected))){var b=*v::at<catalog::Binding>(input::bindings(ref projected),at);if(at==0){b.target=find(ref projected,b"main");}assert(v::push<catalog::Binding>(ref mut bad,b).is_ok);at+=1;}
    assert(switch(catalog::build_scoped(input::sources(ref projected),input::owners(ref projected),ref bad)){case .ok(_):{false}case .err(p):{p.code==3}});discard v::clear<catalog::Binding>(ref mut bad);at=0;while(at<v::len<catalog::Binding>(input::bindings(ref projected))){assert(v::push<catalog::Binding>(ref mut bad,*v::at<catalog::Binding>(input::bindings(ref projected),at)).is_ok);at+=1;}
    assert(v::push<catalog::Binding>(ref mut bad,catalog::Binding{.source=0,.declaration=1,.target=0}).is_ok);assert(switch(catalog::build_scoped(input::sources(ref projected),input::owners(ref projected),ref bad)){case .ok(_):{false}case .err(p):{p.code==9}});
    var v::Vec<usize> owners=v::empty<usize>();at=0;while(at<input::count(ref projected)){assert(v::push<usize>(ref mut owners,0).is_ok);at+=1;}assert(switch(catalog::build_scoped(input::sources(ref projected),ref owners,input::bindings(ref projected))){case .ok(_):{false}case .err(p):{p.code==1}});
    var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;let loaded=switch(loader::load(bytes::view(ref path),true,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let before=budget;meter=16777216;assert(switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(_):{false}case .err(p):{p.code==1}});assert(budget.files==before.files && budget.bytes==before.bytes && visibility::lookup_count(loader::workspace(ref loaded))==5);
    let c=switch(input::declarations(ref projected)){case .ok(value):{value}case .err(_):{assert(false);return;}};let wrong=v::empty<syntax::Parsed>();assert(switch(signatures::build_catalog(ref wrong,c)){case .ok(_):{false}case .err(p):{p.code==9}});
}
fn syntax::Parsed parsed(ref u8[] input){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,input).is_ok);let s=switch(scanner::collect(b,false,32768,1048576,256,16777216)){case .ok(value):{value}case .err(_):{assert(false);return parsed(b"module a;");}};return switch(syntax::parse(s,512)){case .ok(value):{value}case .err(_):{assert(false);return parsed(b"module a;");}};}
fn syntax::Parsed copied(ref u8[] input){let p=parsed(input);var usize meter=0;return switch(syntax::copy(ref p,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return copied(b"module a;");}};}
@test
fn void exact_parser_copy_keeps_arenas_cooked_bytes_spans_and_errors(){let ref u8[] source=b"module a;struct Item{i32 x;}@must_use fn i32 answer(){let s=b\"a\\n\";let a=[1,2];return a[0];}";let original=parsed(source);let copy=copied(source);assert(syntax::problem_count(ref original)==0 && bytes::equal(syntax::source_view(ref copy),source));
    assert(syntax::source_token_count(ref copy)==syntax::source_token_count(ref original) && syntax::declaration_count(ref copy)==syntax::declaration_count(ref original) && syntax::expression_count(ref copy)==syntax::expression_count(ref original) && syntax::entry_count(ref copy)==syntax::entry_count(ref original) && syntax::statement_count(ref copy)==syntax::statement_count(ref original) && syntax::type_count(ref copy)==syntax::type_count(ref original));
    var usize at=0;while(at<syntax::source_token_count(ref original)){let a=scanner::token_info(syntax::source_stream(ref original),at);let b=scanner::token_info(syntax::source_stream(ref copy),at);assert(a.kind==b.kind && a.start==b.start && a.end==b.end);at+=1;}
    at=1;while(at<=syntax::expression_count(ref original)){let a=syntax::expression_at(ref original,at);let b=syntax::expression_at(ref copy,at);assert(a.kind==b.kind && a.start==b.start && a.end==b.end);if(a.kind==1){assert(bytes::equal(syntax::literal_cooked(ref original,at),syntax::literal_cooked(ref copy,at)));}at+=1;}
    let invalid=parsed(b"module a;fn i32 bad(){return ;");var usize meter=0;let retained=switch(syntax::copy(ref invalid,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref invalid)>0 && syntax::problem_count(ref invalid)==syntax::problem_count(ref retained));let a=syntax::problem_at(ref invalid,0);let b=syntax::problem_at(ref retained,0);assert(a.code==b.code && a.start==b.start && a.end==b.end);
    meter=1048576;assert(switch(syntax::copy(ref original,ref mut meter)){case .ok(_):{false}case .err(code):{code==10}});
}
