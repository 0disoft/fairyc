module tests::project_cache_body_identity;
import project_cache_body_identity as identity;
import project_identity_bytes as binary;
import project_compiler_core as core;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import scanner as scanner;
import syntax as syntax;
import sha256_stream as sha;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;

fn ref u8[] source(usize mode){if(mode==0){return b"module root;pub fn i32 main(){return 42;}";}if(mode==1){return b"module root;@contract_uid(\"example.empty\")state{}pub fn i32 main(){return 42;}";}if(mode==3){return b"module renamed;pub fn i32 main(){return 99;}@contract_uid(\"example.token\")state{cell u64 total_supply;map bytes20 u64 balances;cell bool paused;}";}if(mode==4){return b"module root;@contract_uid(\"example.token\")state{cell u64 total_supply;map bytes20 u32 balances;cell bool paused;}pub fn i32 main(){return 42;}";}return b"module root;@contract_uid(\"example.token\")state{cell u64 total_supply;map bytes20 u64 balances;cell bool paused;}pub fn i32 main(){return 42;}";}
fn ref u8[] other(){return b"module other;@contract_uid(\"example.other\")state{cell u32 count;}pub fn i32 read(){return 7;}";}
fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);return parsed(raw);}};}
fn coordinates::Coordinates table(usize mode,usize shift){var bytes::Buffer names=bytes::empty();var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();var usize owner=1;while(owner<=2){let start=bytes::len(ref names);assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize at=0;while(at<64){assert(bytes::append_byte(ref mut names,if(owner==2){51}else if(mode==5){50}else{49}).is_ok);at+=1;}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=owner+shift,.start,.end=bytes::len(ref names)}).is_ok);owner+=1;}return coordinates::assemble(names,rows);}
fn core::Complete complete(usize mode,usize shift){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var usize at=0;let usize count=if(mode>=6){2}else{1};while(at<count){let reversed=mode==7 && at==0 || mode==6 && at==1;assert(v::push<syntax::Parsed>(ref mut sources,parsed(if(reversed){other()}else{source(mode)})).is_ok && v::push<usize>(ref mut owners,if(reversed){2+shift}else{1+shift}).is_ok);at+=1;}let imports=v::empty<catalog::Binding>();let coords=table(mode,shift);var usize source_work=0;var usize link_work=0;return switch(core::build_complete_sources(ref sources,ref owners,ref imports,ref coords,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(_):{assert(false);return complete(mode,shift);}};}
fn identity::Identity derive(ref core::Complete p,usize owner,ref mut usize work){return switch(identity::derive(p,owner,work)){case .ok(id):{id}case .err(_):{assert(false);return derive(p,owner,work);}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=4);var usize value=0;var usize i=0;while(i<4){value=value*256+data[*at] as usize;*at+=1;i+=1;}return value;}
@test @effects(filesystem,environment)
fn void checked_packages_match_go_state_body_hashes_and_exact_preimages(){
    var u8[4096] path=[0;4096];let size=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let ref u8[] view=ref path;let fixture=switch(files::load(view[..size])){case .ok(b):{b}case .err(_):{assert(false);return;}};let data=bytes::view(ref fixture);var usize at=0;assert(number(data,ref mut at)==8);var usize mode=0;
    while(mode<8){let p=complete(mode,19);var usize work=0;let id=derive(ref p,20,ref mut work);let count=number(data,ref mut at);assert(at<=data.len && count<=data.len-at && bytes::equal(identity::canonical(ref id),data[at..at+count]));at+=count;assert(at<=data.len && 81<=data.len-at && bytes::equal(identity::digest(ref id),data[at..at+81]));at+=81;var sha::State hash=sha::empty();assert(sha::update(ref mut hash,identity::canonical(ref id)) && sha::matches(ref hash,identity::digest(ref id)[17..]));assert(identity::coordinate(ref id).len==32);mode+=1;}assert(at==data.len);
}
@test
fn void body_identity_ignores_disposable_owners_private_bodies_and_other_packages(){
    var usize work=0;let a={let p=complete(2,0);derive(ref p,1,ref mut work)};let b={let p=complete(3,77);derive(ref p,78,ref mut work)};assert(bytes::equal(identity::digest(ref a),identity::digest(ref b)));let extra={let p=complete(6,0);derive(ref p,1,ref mut work)};let reversed={let p=complete(7,77);derive(ref p,78,ref mut work)};assert(bytes::equal(identity::digest(ref a),identity::digest(ref extra)) && bytes::equal(identity::digest(ref a),identity::digest(ref reversed)));let changed={let p=complete(4,0);derive(ref p,1,ref mut work)};let coordinate={let p=complete(5,0);derive(ref p,1,ref mut work)};assert(!bytes::equal(identity::digest(ref a),identity::digest(ref changed)) && !bytes::equal(identity::digest(ref a),identity::digest(ref coordinate)));
}
@test
fn void unknown_owners_and_exact_phase_work_cannot_publish_partial_hashes(){
    let p=complete(2,0);var usize cost=0;let id=derive(ref p,1,ref mut cost);assert(cost>0 && cost<16777216);var usize work=16777216-cost;let fit=derive(ref p,1,ref mut work);assert(work==16777216 && bytes::equal(identity::canonical(ref id),identity::canonical(ref fit)));work=16777216-cost+1;assert(switch(identity::derive(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=16777216);work=16777217;assert(switch(identity::derive(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});work=0;assert(switch(identity::derive(ref p,2,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(identity::derive(ref p,512,ref mut work)){case .ok(_):{false}case .err(code):{code==9}} && work==0);
}
@test
fn void binary_identity_decoder_rejects_wrong_domains_hex_and_zero_coordinates(){
    let prefix=b"package_coordinate_v1_";var usize work=0;assert(switch(binary::decode(b"package_coordinate_v1_111",prefix,true,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});let raw=b"package_coordinate_v1_0000000000000000000000000000000000000000000000000000000000000000";work=0;assert(switch(binary::decode(raw,prefix,true,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(binary::decode(b"package_coordinate_v1_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",prefix,true,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(binary::decode(b"package_coordinate_v1_gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg",prefix,true,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=16777217;assert(switch(binary::decode(raw,prefix,false,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});
}
