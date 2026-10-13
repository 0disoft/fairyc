module tests::project_generic_cache_key;
import project_generic_cache_key as generic;
import project_cache_key as key;
import sha256_stream as sha;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;

fn ref u8[] env_value(){return b"fixture";}
fn ref u8[] changed_env(){return b"other";}
fn bytes::Buffer copied(ref u8[] input){return switch(bytes::copy(input)){case .ok(b):{b}case .err(_):{assert(false);return copied(input);}};}
fn u8[32] hash(usize seed){var u8[32] result=[0;32];let table=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09";var usize at=0;while(at<32){result[at]=table[seed];at+=1;}return result;}
fn key::Inputs ordinary(usize mode){
    let env=key::environment_of(copied(if(mode==7){changed_env()}else{env_value()}),copied(env_value()),copied(env_value()),copied(env_value()),copied(env_value()),copied(env_value()),copied(env_value()),copied(env_value()));
    let facts=key::Facts{.standard_library=hash(1),.package=hash(2),.source=hash(3),.current_interface=hash(4),.current_body=hash(if(mode==20){0}else if(mode==6){6}else{5})};var v::Vec<bytes::Buffer> flags=v::empty<bytes::Buffer>();assert(v::push<bytes::Buffer>(ref mut flags,copied(b"-O2")).is_ok);let features=v::empty<bytes::Buffer>();var v::Vec<key::Dependency> deps=v::empty<key::Dependency>();
    if(mode!=5){assert(v::push<key::Dependency>(ref mut deps,key::Dependency{.coordinate=hash(1),.interface_hash=hash(2)}).is_ok);}return key::assemble(env,flags,facts,deps,features);
}
fn ref u8[] specialization(usize id){if(id==0){return b"g0_1111111111111111111111111111111111111111111111111111111111111111";}if(id==1){return b"g0_2222222222222222222222222222222222222222222222222222222222222222";}if(id==2){return b"g0_3333333333333333333333333333333333333333333333333333333333333333";}if(id==3){return b"g0_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA";}if(id==4){return b"g0_gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg";}return b"g0_1";}
fn void push(ref mut v::Vec<bytes::Buffer> ids,usize id){assert(v::push<bytes::Buffer>(ids,copied(specialization(id))).is_ok);}
fn generic::Inputs input(usize mode){
    let base=ordinary(mode);let current=hash(if(mode==21){0}else if(mode==1){7}else{6});var v::Vec<generic::Dependency> deps=v::empty<generic::Dependency>();
    if(mode!=5 && mode!=22){assert(v::push<generic::Dependency>(ref mut deps,generic::Dependency{.coordinate=hash(if(mode==23){2}else{1}),.interface_hash=hash(if(mode==24){0}else if(mode==2){8}else{7})}).is_ok);}
    if(mode==25){assert(v::push<generic::Dependency>(ref mut deps,generic::Dependency{.coordinate=hash(2),.interface_hash=hash(7)}).is_ok);}
    var v::Vec<bytes::Buffer> ids=v::empty<bytes::Buffer>();if(mode!=4 && mode!=5){push(ref mut ids,if(mode==3){1}else if(mode==29){3}else if(mode==30){4}else if(mode==31){5}else{0});push(ref mut ids,if(mode==26){0}else{2});}
    if(mode==27){push(ref mut ids,1);}return generic::assemble(base,current,deps,ids);
}
fn generic::Identity compute(ref generic::Inputs p,ref mut usize work){return switch(generic::compute(p,work)){case .ok(id):{id}case .err(_):{assert(false);return compute(p,work);}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=4);var usize value=0;var usize i=0;while(i<4){value=value*256+data[*at] as usize;*at+=1;i+=1;}return value;}
@test
@effects(filesystem,environment)
fn void matches_actual_go_generic_keys_and_complete_encoding(){
    var u8[4096] path=[0;4096];let length=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let ref u8[] view=ref path;let fixture=switch(files::load(view[..length])){case .ok(b):{b}case .err(_):{assert(false);return;}};let data=bytes::view(ref fixture);var usize at=0;assert(number(data,ref mut at)==8);var usize mode=0;let baseline=input(0);var usize base_work=0;let base=compute(ref baseline,ref mut base_work);
    while(mode<8){let n=number(data,ref mut at);assert(at<=data.len && n<=data.len-at);let p=input(mode);var usize work=0;let id=compute(ref p,ref mut work);assert(bytes::equal(generic::canonical(ref id),data[at..at+n]));at+=n;assert(at<=data.len && 81<=data.len-at);assert(bytes::equal(generic::digest(ref id),data[at..at+81]) && generic::valid_text(generic::digest(ref id)));at+=81;if(mode!=0){assert(!bytes::equal(generic::digest(ref base),generic::digest(ref id)));}var sha::State state=sha::empty();assert(sha::update(ref mut state,generic::canonical(ref id)) && sha::matches(ref state,generic::digest(ref id)[17..]));mode+=1;}assert(at==data.len);
}
@test
fn void rejects_incomplete_environment_dependency_coverage_and_noncanonical_ids(){
    var usize mode=20;while(mode<28){let p=input(mode);var usize work=0;let u32 expected=if(mode==20 || mode==21 || mode==24){2}else{3};assert(switch(generic::compute(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==expected}});mode+=1;}
    mode=29;while(mode<32){let p=input(mode);var usize work=0;assert(switch(generic::compute(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==1}});mode+=1;}
}
@test
fn void retains_detached_keys_and_respects_shared_phase_budget(){
    var usize cost=0;let id={let p=input(0);compute(ref p,ref mut cost)};assert(cost>0 && cost<16777216 && generic::valid_text(generic::digest(ref id)));let p=input(0);var usize work=16777216-cost;let fit=compute(ref p,ref mut work);assert(work==16777216 && bytes::equal(generic::canonical(ref id),generic::canonical(ref fit)) && bytes::equal(generic::digest(ref id),generic::digest(ref fit)));work=16777216-cost+1;assert(switch(generic::compute(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==5}});assert(work<=16777216);work=16777217;assert(switch(generic::compute(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==5}});assert(!generic::valid_text(generic::digest(ref id)[..80]) && !generic::specialization_text(b"g0_111") && !generic::specialization_text(specialization(3)) && !generic::specialization_text(specialization(4)));
}
