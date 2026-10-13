module project_cache_keys;
import project_cache_workspace as workspace;
import project_cache_environment as environment;
import project_compiler_core as core;
import project_module_visibility as visibility;
import module_project_interfaces as interfaces;
import project_cache_body_identity as body;
import project_cache_specializations as specializations;
import project_identity_bytes as binary;
import project_cache_key as ordinary;
import project_generic_cache_key as generic;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

pub resource Record{usize owner;bytes::Buffer instance;bytes::Buffer coordinate;ordinary::Identity ordinary;generic::Identity generic;cleanup{}}
pub resource Keys{v::Vec<Record> packages;cleanup{}}
struct Dependency{ordinary::Dependency ordinary;generic::Dependency generic;}
pub fn usize count(ref Keys p){return v::len<Record>(ref p.packages);}
pub fn ref Record at(ref Keys p,usize index){return v::at<Record>(ref p.packages,index);}
pub fn usize owner(ref Record p){return p.owner;}
pub fn ref u8[] instance(ref Record p){return bytes::view(ref p.instance);}
pub fn ref u8[] coordinate(ref Record p){return bytes::view(ref p.coordinate);}
pub fn ref ordinary::Identity ordinary_key(ref Record p){return ref p.ordinary;}
pub fn ref generic::Identity generic_key(ref Record p){return ref p.generic;}
fn u32 mapped(u32 code){return if(code==5){10}else if(code==4){11}else{9};}
fn Result<u8[32],u32> digest(ref u8[] text,ref u8[] prefix,ref mut usize work){let raw=switch(binary::decode(text,prefix,true,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(!budget::charge(work,33)){return .err(10);}var u8[32] out=[0;32];let data=bytes::view(ref raw);var usize i=0;while(i<32){out[i]=data[i];i+=1;}return .ok(out);}
fn Result<usize,u32> record(ref interfaces::Records p,usize owner,ref mut usize work){var usize found=interfaces::count(p);var usize i=0;while(i<interfaces::count(p)){if(!budget::charge(work,1)){return .err(10);}if(interfaces::owner(interfaces::at(p,i))==owner){if(found!=interfaces::count(p)){return .err(9);}found=i;}i+=1;}if(found==interfaces::count(p)){return .err(9);}return .ok(found);}
fn Result<bytes::Buffer,u32> copy(ref u8[] value,ref mut usize work){if(!budget::charge(work,value.len+1)){return .err(10);}return switch(bytes::copy(value)){case .ok(p):{.ok(p)}case .err(_):{.err(11)}};}
fn u32 sort(ref mut v::Vec<Dependency> rows,ref mut usize work){var usize i=1;while(i<v::len<Dependency>(rows)){let current=*v::at<Dependency>(rows,i);var usize at=i;while(at>0){if(!budget::charge(work,65)){return 10;}let previous=*v::at<Dependency>(rows,at-1);if(!budget::before(ref current.ordinary.coordinate,ref previous.ordinary.coordinate)){break;}let slot=v::at_mut<Dependency>(rows,at);*slot=previous;at-=1;}let slot=v::at_mut<Dependency>(rows,at);*slot=current;i+=1;}i=1;while(i<v::len<Dependency>(rows)){if(!budget::charge(work,65)){return 10;}let a=*v::at<Dependency>(rows,i-1);let b=*v::at<Dependency>(rows,i);if(!budget::before(ref a.ordinary.coordinate,ref b.ordinary.coordinate)){return 9;}i+=1;}return 0;}
fn Result<v::Vec<Dependency>,u32> dependencies(ref workspace::Facts facts,ref interfaces::Records records,usize owner,ref mut usize work){let p=workspace::at(facts,owner);var v::Vec<Dependency> out=v::empty<Dependency>();var usize i=0;
    while(i<workspace::dependency_count(p)){if(!budget::charge(work,1)){return .err(10);}let target=workspace::dependency(p,i);if(target>=workspace::count(facts) || target==owner){return .err(9);}let index=switch(record(records,target,work)){case .ok(n):{n}case .err(code):{return .err(code);}};let r=interfaces::at(records,index);
        let coordinate=switch(digest(workspace::coordinate(workspace::at(facts,target)),b"fypc1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        let ordinary_hash=switch(digest(interfaces::ordinary_hash(r),b"fyif5-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        let generic_hash=switch(digest(interfaces::source_hash(r),b"fygif0-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(!v::push<Dependency>(ref mut out,Dependency{.ordinary=ordinary::Dependency{.coordinate,.interface_hash=ordinary_hash},.generic=generic::Dependency{.coordinate,.interface_hash=generic_hash}}).is_ok){return .err(11);}i+=1;
    }let code=sort(ref mut out,work);if(code!=0){return .err(code);}return .ok(out);
}
fn Result<Record,u32> package(ref workspace::Facts facts,ref core::Complete complete,ref environment::Environment env,u8[32] standard_library,usize owner,ref mut usize work){
    let p=workspace::at(facts,owner);let records=core::public_interfaces(complete);let index=switch(record(records,owner,work)){case .ok(n):{n}case .err(code):{return .err(code);}};let r=interfaces::at(records,index);
    let package_id=switch(digest(workspace::instance(p),b"fypi1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let source=switch(digest(workspace::source(p),b"fysrc1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let current_interface=switch(digest(interfaces::ordinary_hash(r),b"fyif5-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let current_generic=switch(digest(interfaces::source_hash(r),b"fygif0-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let state=switch(body::derive(complete,owner,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let coordinate=switch(binary::decode(workspace::coordinate(p),b"fypc1-sha256-",true,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(!budget::charge(work,65)){return .err(10);}if(!bytes::equal(bytes::view(ref coordinate),body::coordinate(ref state))){return .err(9);}
    let current_body=switch(digest(body::digest(ref state),b"fycsbody1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let deps=switch(dependencies(facts,records,owner,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var v::Vec<ordinary::Dependency> ordinary_deps=v::empty<ordinary::Dependency>();var v::Vec<generic::Dependency> generic_deps=v::empty<generic::Dependency>();var usize i=0;
    while(i<v::len<Dependency>(ref deps)){if(!budget::charge(work,1)){return .err(10);}let d=*v::at<Dependency>(ref deps,i);if(!v::push<ordinary::Dependency>(ref mut ordinary_deps,d.ordinary).is_ok || !v::push<generic::Dependency>(ref mut generic_deps,d.generic).is_ok){return .err(11);}i+=1;}
    var v::Vec<bytes::Buffer> features=v::empty<bytes::Buffer>();i=0;while(i<workspace::feature_count(p)){let value=switch(copy(workspace::feature(p,i),work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(!v::push<bytes::Buffer>(ref mut features,value).is_ok){return .err(11);}i+=1;}
    let e=switch(environment::key_environment(env,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let base=ordinary::assemble(e,v::empty<bytes::Buffer>(),ordinary::Facts{.standard_library,.package=package_id,.source,.current_interface,.current_body},ordinary_deps,features);
    let ordinary_key=switch(ordinary::compute(ref base,work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};
    let selected=switch(specializations::collect(complete,owner,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let ids=switch(specializations::copy_ids(ref selected,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let source_input=generic::assemble(base,current_generic,generic_deps,ids);let generic_key=switch(generic::compute(ref source_input,work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};
    let instance=switch(copy(workspace::instance(p),work)){case .ok(p):{p}case .err(code):{return .err(code);}};let namespace=switch(copy(workspace::coordinate(p),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    return .ok(Record{.owner,.instance,.coordinate=namespace,.ordinary=ordinary_key,.generic=generic_key});
}
// Derive all package keys atomically. Authenticate the loaded graph once before
// any package facts are consumed; no independent caller-provided hashes, IDs,
// specialization sets, environment strings or partial key vector are admitted.
// The keys are compatibility facts, not artifact verification or cache hits.
pub fn Result<Keys,u32> collect(ref visibility::Resolved loaded,ref core::Complete complete,ref environment::Environment env,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}let facts=switch(workspace::collect(loaded,complete,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(interfaces::count(core::public_interfaces(complete))!=workspace::count(ref facts)){return .err(9);}
    let standard_library=switch(digest(environment::standard_library(env),b"fypi1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};var v::Vec<Record> packages=v::empty<Record>();var usize owner=0;
    while(owner<workspace::count(ref facts)){if(!budget::charge(work,1)){return .err(10);}let p=switch(package(ref facts,complete,env,standard_library,owner,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(!v::push<Record>(ref mut packages,p).is_ok){return .err(11);}owner+=1;}return .ok(Keys{.packages});
}
