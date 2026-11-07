module tests::closure_value_interfaces;
import module_signatures as signatures;
import module_catalog as catalog;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import syntax_parse_driver as parser;
import scanner as scanner;
import file_source as file;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){
    let input=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{assert(false);return parsed(raw);}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(raw);}};
    let p=switch(parser::parse_diagnostics(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};
    assert(syntax::problem_count(ref p)==0);return p;
}
@effects(filesystem) fn bytes::Buffer fixture(){
    var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;
    return switch(file::manifest(view[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};
}
fn usize number(ref u8[] wire,ref mut usize at){assert(*at<=wire.len && wire.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(wire[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] wire,ref mut usize at){let n=number(wire,at);assert(*at<=wire.len && n<=wire.len-*at);let start=*at;*at+=n;return wire[start..*at];}
fn void leaf(types::Type t){assert(t.kind==23 && t.scalar==0 && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0);}
fn Result<signatures::Program,catalog::Problem> metadata(ref v::Vec<syntax::Parsed> sources,bool explicit_catalog){
    if(!explicit_catalog){return signatures::build(sources);}
    let c=switch(catalog::build(sources)){case .ok(c):{c}case .err(p):{return .err(p);}};return signatures::build_catalog(sources,c);
}
@effects(console) fn void failure(ref u8[] name,usize stage,u32 code,usize module_index,usize start,usize end){
    discard io::write_bytes(name);discard io::write(" stage=");discard io::write_usize(stage);discard io::write(" code=");discard io::write_u32(code);
    discard io::write(" module=");discard io::write_usize(module_index);discard io::write(" span=");discard io::write_usize(start);discard io::write("..");discard io::write_usize(end);discard io::write("\n");assert(false);
}
fn void signature_facts(ref v::Vec<syntax::Parsed> sources,ref signatures::Program p){
    assert(signatures::module_count(p)==v::len<syntax::Parsed>(sources));var usize i=0;
    while(i<v::len<syntax::Parsed>(sources)){assert(signatures::source_matches(p,i,v::at<syntax::Parsed>(sources,i)));i+=1;}
    i=1;while(i<=signatures::count(p)){let s=signatures::at(p,i);let source=v::at<syntax::Parsed>(sources,s.module_index);let d=syntax::declaration_at(source,s.declaration-1);
        var usize parameters=0;var usize j=0;while(j<d.entry_count){if(syntax::entry_at(source,d.entry_start+j).kind==1){parameters+=1;}j+=1;}assert(signatures::identity(p,s.module_index,s.declaration)==i && s.parameter_count==parameters);
        if(syntax::type_at(source,d.result_type).kind==17){leaf(signatures::type_at(p,s.result));assert(s.parameter_count==1 && signatures::parameter_at(p,s.parameter_start)==s.result);}
        i+=1;
    }
}
fn void interface_facts(ref u8[] name,ref v::Vec<syntax::Parsed> sources,ref interfaces::Interfaces p){
    assert(interfaces::work(p)<1048576 && interfaces::first(p)==1 && interfaces::stop(p)==2);var usize leaves=0;var usize i=1;
    while(i<=interfaces::type_count(p)){let t=interfaces::type_at(p,i);if(t.kind==23){leaf(t);leaves+=1;}i+=1;}assert(leaves<=1);
    var usize expected_leaves=0;i=0;while(i<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,i);var usize j=1;while(j<=syntax::type_count(source)){if(syntax::type_at(source,j).kind==17){expected_leaves=1;}j+=1;}i+=1;}assert(leaves==expected_leaves);
    var usize left=0;var usize right=0;i=0;
    while(i<interfaces::shape_count(p)){let s=interfaces::shape_at(p,i);let t=interfaces::type_at(p,s.type);assert(t.module_index==s.module_index && t.declaration==s.declaration);
        let source=v::at<syntax::Parsed>(sources,s.module_index);let d=syntax::declaration_at(source,s.declaration-1);assert(s.start==d.start && s.end==d.end);
        var usize j=0;while(j<s.field_count){let f=interfaces::field_at(p,s.field_start+j);assert(f.module_index==s.module_index && f.declaration==s.declaration);let original=syntax::field_at(source,f.source_index);assert(f.start==original.start && f.end==original.end);
            if(f.type!=0 && interfaces::type_at(p,f.type).kind==23){leaf(interfaces::type_at(p,f.type));assert(t.kind==2 && !s.repr_c && syntax::type_at(source,original.type_root).kind==17);if(s.module_index==2){left=s.type;}if(s.module_index==3){right=s.type;}}
            j+=1;
        }i+=1;
    }
    // Identical written Holder fields never merge distinct provider identities.
    if(bytes::equal(name,b"distinct-provider-identity")){assert(left!=0 && right!=0 && left!=right && interfaces::type_at(p,left).module_index==2 && interfaces::type_at(p,right).module_index==3);}
    if(bytes::equal(name,b"imported-plain-copy")){assert(left!=0 && interfaces::type_at(p,left).module_index==2);}
}

@effects(console) fn void check_sources(ref u8[] name,ref v::Vec<syntax::Parsed> sources,bool explicit_catalog,usize accepted,usize bad_module,usize bad_code,usize bad_start,usize bad_end){
    let p=switch(metadata(sources,explicit_catalog)){case .ok(p):{if(accepted==0){failure(name,1,0,0,0,0);}p}case .err(p):{
        if(accepted!=0 || p.module_index!=bad_module || p.start!=bad_start || p.end!=bad_end || (p.code as usize)!=bad_code){failure(name,1,p.code,p.module_index,p.start,p.end);}return;
    }};
    signature_facts(sources,ref p);let bound=switch(binding::bind(sources,ref p,1,2)){case .ok(b):{b}case .err(e):{failure(name,2,e.code,e.module_index,e.start,e.end);return;}};
    let retained=switch(interfaces::build(sources,ref p,ref bound)){case .ok(p):{p}case .err(e):{failure(name,3,e.code,e.module_index,e.start,e.end);return;}};
    interface_facts(name,sources,ref retained);let seeds=v::empty<binding::Coordinate>();
    let unit=switch(interfaces::build_unit_seeded(sources,ref p,ref bound,ref seeds)){case .ok(p):{p}case .err(e):{failure(name,4,e.code,e.module_index,e.start,e.end);return;}};
    interface_facts(name,sources,ref unit);assert(interfaces::shape_count(ref retained)==interfaces::shape_count(ref unit) && interfaces::field_count(ref retained)==interfaces::field_count(ref unit));
}
@test @effects(filesystem,console) fn void closure_interfaces_match_original_go_sources(){
    let input=fixture();let wire=bytes::view(ref input);assert(wire.len>=16 && bytes::equal(wire[..8],b"FYCVIF01"));var usize at=8;let cases=number(wire,ref mut at);assert(cases==31);var usize i=0;
    while(i<cases){let name=field(wire,ref mut at);let files=number(wire,ref mut at);var sources=v::empty<syntax::Parsed>();var usize j=0;while(j<files){let raw=field(wire,ref mut at);let p=parsed(raw);assert(bytes::equal(syntax::source_view(ref p),raw));assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);j+=1;}
        let accepted=number(wire,ref mut at);let bad_module=number(wire,ref mut at);let bad_code=number(wire,ref mut at);let bad_start=number(wire,ref mut at);let bad_end=number(wire,ref mut at);
        check_sources(name,ref sources,false,accepted,bad_module,bad_code,bad_start,bad_end);check_sources(name,ref sources,true,accepted,bad_module,bad_code,bad_start,bad_end);i+=1;
    }assert(at==wire.len);
}

fn v::Vec<syntax::Parsed> pair(ref u8[] caller,ref u8[] provider){var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(caller)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);return sources;}
@test fn void closure_interfaces_preserve_source_authority(){
    let caller=b"module main;import provider as p;fn p::Holder keep(p::Holder value){let p::Holder copy=value;return copy;}";
    let provider=b"module provider;pub struct Holder{fn i32(i32) action;}";
    let original=pair(caller,provider);let p=switch(signatures::build(ref original)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref original,ref p,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    // Same declarations and graph leaf, changed original bytes: owner authentication must win.
    let changed=pair(caller,b"module provider; // changed original bytes\r\npub struct Holder{fn i32(i32) action;}");
    let stale=switch(catalog::build(ref original)){case .ok(c):{c}case .err(_):{assert(false);return;}};
    switch(signatures::build_catalog(ref changed,stale)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==9 && e.module_index==1 && e.start==0 && e.end==0);}};
    assert(!signatures::source_matches(ref p,1,v::at<syntax::Parsed>(ref changed,1)));
    switch(binding::bind(ref changed,ref p,0,1)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==12 && e.module_index==1);}};
    switch(interfaces::build(ref changed,ref p,ref bound)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==12 && e.module_index==1);}};
    let seeds=v::empty<binding::Coordinate>();switch(interfaces::build_unit_seeded(ref changed,ref p,ref bound,ref seeds)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==12 && e.module_index==1);}};
    let fresh=switch(signatures::build(ref changed)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let rebound=switch(binding::bind(ref changed,ref fresh,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    assert(interfaces::build(ref changed,ref fresh,ref rebound).is_ok && interfaces::build_unit_seeded(ref changed,ref fresh,ref rebound,ref seeds).is_ok);
    let swapped=pair(provider,caller);let c=switch(catalog::build(ref original)){case .ok(c):{c}case .err(_):{assert(false);return;}};
    assert(switch(signatures::build_catalog(ref swapped,c)){case .ok(_):{false}case .err(e):{e.code==9 && e.module_index==0}});
    let changed_caller=pair(b"module main;import provider as p;fn p::Holder keep(p::Holder value){return value;}",provider);
    let current=switch(signatures::build(ref changed_caller)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(interfaces::build(ref changed_caller,ref current,ref bound)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==12 && e.module_index==0);}};
    switch(interfaces::build_unit_seeded(ref changed_caller,ref current,ref bound,ref seeds)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==12 && e.module_index==0);}};
}
