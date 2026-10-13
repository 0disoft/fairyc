module tests::project_cache_key;
import project_cache_key as key;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;

fn bytes::Buffer copied(ref u8[] input){return switch(bytes::copy(input)){case .ok(b):{b}case .err(_):{assert(false);return copied(input);}};}
fn ref u8[] environment_value(usize index){if(index==0){return b"compiler-build";}if(index==1){return b"fairyc-language/20";}if(index==2){return b"mir/1";}if(index==3){return b"target/1";}if(index==4){return b"msvc-x64";}if(index==5){return b"toolchain";}if(index==6){return b"c11";}if(index==7){return b"runtime/0";}return b"";}
fn bytes::Buffer env(usize mode,usize index){if(mode==30+index){return bytes::empty();}if(mode==50 && index==0){return copied(b"\xff");}var b=copied(environment_value(index));if(mode==index+1){assert(bytes::append(ref mut b,b"!").is_ok);}if(mode==19 && index==0){assert(bytes::append(ref mut b,b"\x00\xec\xa0\x9c\xeb\xa1\x9c\xeb\x94\x94").is_ok);}return b;}
fn u8[32] digest(usize seed,bool changed){var u8[32] result=[0;32];let table=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09";var usize at=0;while(at<32){result[at]=table[seed];at+=1;}if(changed){result[0]=9;}return result;}
fn void push(ref mut v::Vec<bytes::Buffer> values,ref u8[] value){assert(v::push<bytes::Buffer>(values,copied(value)).is_ok);}
fn key::Inputs input(usize mode){
    let environment=key::environment_of(env(mode,0),env(mode,1),env(mode,2),env(mode,3),env(mode,4),env(mode,5),env(mode,6),env(mode,7));
    let facts=key::Facts{.standard_library=digest(if(mode==40){0}else{1},mode==9),.package=digest(if(mode==41){0}else{2},mode==10),.source=digest(if(mode==42){0}else{3},mode==11),.current_interface=digest(if(mode==43){0}else{4},mode==12),.current_body=digest(if(mode==44){0}else{5},mode==13)};
    var v::Vec<bytes::Buffer> profile_flags=v::empty<bytes::Buffer>();var v::Vec<bytes::Buffer> features=v::empty<bytes::Buffer>();var v::Vec<key::Dependency> dependencies=v::empty<key::Dependency>();
    if(mode!=18){push(ref mut profile_flags,b"-O2");push(ref mut features,b"a");assert(v::push<key::Dependency>(ref mut dependencies,key::Dependency{.coordinate=digest(if(mode==16){2}else{1},false),.interface_hash=digest(if(mode==17){8}else{2},false)}).is_ok);assert(v::push<key::Dependency>(ref mut dependencies,key::Dependency{.coordinate=digest(3,false),.interface_hash=digest(4,false)}).is_ok);}
    if(mode==14){push(ref mut profile_flags,b"-g");}if(mode==15){push(ref mut features,b"z");}
    if(mode>=60 && mode<63){push(ref mut profile_flags,if(mode==60){b"-O2"}else if(mode==61){b"-A"}else{b""});}if(mode>=63 && mode<66){push(ref mut features,if(mode==63){b"a"}else if(mode==64){b"A"}else{b"\xff"});}
    if(mode>=70 && mode<74){let first=key::Dependency{.coordinate=digest(if(mode==70){0}else if(mode==72){3}else if(mode==73){4}else{1},false),.interface_hash=digest(if(mode==71){0}else{2},false)};let slot=v::at_mut<key::Dependency>(ref mut dependencies,0);*slot=first;}
    return key::assemble(environment,profile_flags,facts,dependencies,features);
}
fn key::Identity compute(ref key::Inputs p,ref mut usize work){return switch(key::compute(p,work)){case .ok(id):{id}case .err(_):{assert(false);return compute(p,work);}};}
fn bool rejected(ref key::Inputs p,u32 code){var usize work=0;return switch(key::compute(p,ref mut work)){case .ok(_):{false}case .err(c):{c==code}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=4);var usize value=0;var usize i=0;while(i<4){value=value*256+data[*at] as usize;*at+=1;i+=1;}return value;}
@test
@effects(filesystem,environment)
fn void matches_actual_go_v4_bytes_and_all_invalidation_members(){
    var u8[4096] path=[0;4096];let length=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let ref u8[] view=ref path;
    let fixture=switch(files::load(view[..length])){case .ok(b):{b}case .err(_):{assert(false);return;}};let data=bytes::view(ref fixture);var usize at=0;assert(number(data,ref mut at)==20);var usize mode=0;
    let baseline=input(0);var usize baseline_work=0;let base=compute(ref baseline,ref mut baseline_work);
    while(mode<20){let n=number(data,ref mut at);assert(at<=data.len && n<=data.len-at);let p=input(mode);var usize work=0;let id=compute(ref p,ref mut work);assert(bytes::equal(key::canonical(ref id),data[at..at+n]));at+=n;assert(at<=data.len && 80<=data.len-at);assert(bytes::equal(key::digest(ref id),data[at..at+80]) && key::valid_text(key::digest(ref id)));at+=80;if(mode!=0){assert(!bytes::equal(key::digest(ref base),key::digest(ref id)));}mode+=1;}assert(at==data.len);
}
@test
fn void rejects_missing_environment_and_body_or_package_facts(){
    var usize field=0;while(field<8){let p=input(30+field);assert(rejected(ref p,1));field+=1;}
    field=0;while(field<5){let p=input(40+field);assert(rejected(ref p,2));field+=1;}
    let p=input(50);assert(rejected(ref p,1));
}
@test
fn void rejects_unsorted_duplicate_empty_and_malformed_sets(){
    var usize mode=0;while(mode<6){let p=input(60+mode);assert(rejected(ref p,if(mode==2 || mode==5){1}else{3}));mode+=1;}
    mode=0;while(mode<4){let p=input(70+mode);assert(rejected(ref p,if(mode<2){2}else{3}));mode+=1;}
}
@test
fn void retains_detached_key_and_obeys_exact_phase_budget(){
    var usize cost=0;let id={let p=input(19);compute(ref p,ref mut cost)};assert(key::valid_text(key::digest(ref id)) && cost>0 && cost<16777216);let p=input(19);var usize work=16777216-cost;let fit=compute(ref p,ref mut work);assert(work==16777216 && bytes::equal(key::canonical(ref id),key::canonical(ref fit)) && bytes::equal(key::digest(ref id),key::digest(ref fit)));work=16777216-cost+1;assert(switch(key::compute(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==5}});assert(work<=16777216);work=16777217;assert(switch(key::compute(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==5}});
    assert(!key::valid_text(key::digest(ref id)[..78]) && !key::valid_text(b"fycache4-sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA") && !key::valid_text(b"fycache4-sha256-gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"));
}
