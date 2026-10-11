module project_lock_fields;
import project_json_tree as json;
import project_json_decode as decode;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

struct Package{usize[10] fields;usize first;usize count;usize capacity;bool null_features;}
struct Dependency{usize[3] fields;}
// An owned decoded draft, not a validated lock Document. Zero string indices
// mean omitted/null-default strings; nullable collection flags stay explicit.
pub resource Fields{json::Tree tree;v::Vec<usize> headers;v::Vec<Package> packages;v::Vec<usize> features;v::Vec<Dependency> dependencies;usize packages_count;usize dependencies_count;bool null_packages;bool null_dependencies;cleanup{}}
fn ref u8[] text(ref json::Tree tree,usize index){return if(index==0){json::source(tree)[..0]}else{json::value(tree,index)};}
pub fn ref u8[] header(ref Fields input,usize field){assert(field<5);return text(ref input.tree,*v::at<usize>(ref input.headers,field));}
pub fn usize package_count(ref Fields input){return input.packages_count;}
pub fn ref u8[] package_field(ref Fields input,usize at,usize field){assert(at<package_count(input) && field<10);return text(ref input.tree,v::at<Package>(ref input.packages,at).fields[field]);}
pub fn bool null_features(ref Fields input,usize at){assert(at<package_count(input));return v::at<Package>(ref input.packages,at).null_features;}
pub fn usize feature_count(ref Fields input,usize at){assert(at<package_count(input));return v::at<Package>(ref input.packages,at).count;}
pub fn ref u8[] feature(ref Fields input,usize at,usize index){assert(index<feature_count(input,at));let package=v::at<Package>(ref input.packages,at);return text(ref input.tree,*v::at<usize>(ref input.features,package.first+index));}
pub fn usize dependency_count(ref Fields input){return input.dependencies_count;}
pub fn ref u8[] dependency_field(ref Fields input,usize at,usize field){assert(at<dependency_count(input) && field<3);return text(ref input.tree,v::at<Dependency>(ref input.dependencies,at).fields[field]);}
pub fn bool null_packages(ref Fields input){return input.null_packages;}
pub fn bool null_dependencies(ref Fields input){return input.null_dependencies;}
fn decode::Problem fail(ref json::Tree tree,u32 code,usize at){let t=json::token(tree,at);return decode::Problem{.code,.start=t.start,.end=t.end};}
fn ref u8[] name(u32 kind,usize field){if(kind==0){if(field==0){return b"schema";}if(field==1){return b"packageInstanceAlgorithm";}if(field==2){return b"sourceDigestAlgorithm";}if(field==3){return b"language";}if(field==4){return b"root";}if(field==5){return b"packages";}return b"dependencies";}if(kind==2){if(field==0){return b"owner";}if(field==1){return b"binding";}return b"target";}if(field==0){return b"id";}if(field==1){return b"role";}if(field==2){return b"packageUid";}if(field==3){return b"sourceKind";}if(field==4){return b"localPath";}if(field==5){return b"sdkIdentity";}if(field==6){return b"runtimeAbi";}if(field==7){return b"sourceIdentity";}if(field==8){return b"sourceDigest";}if(field==9){return b"version";}return b"features";}
// Go encoding/json resolves exact names first, then Unicode simple-fold names.
fn Result<usize,decode::Problem> field(ref json::Tree tree,usize at,u32 kind,ref mut usize meter){let key=json::value(tree,at);let usize count=if(kind==0){7}else if(kind==1){11}else{3};var usize index=0;while(index<count){let canonical=name(kind,index);if(!work::charge(meter,key.len+canonical.len+1)){return .err(fail(tree,3,at));}if(bytes::equal(key,canonical)){return .ok(index);}index+=1;}index=0;while(index<count){let bool equal=switch(work::equal_fold(key,name(kind,index),meter)){case .ok(value):{value}case .err(_):{return .err(fail(tree,3,at));}};if(equal){return .ok(index);}index+=1;}return .err(fail(tree,5,at));}
fn Result<usize,decode::Problem> string(ref json::Tree tree,usize at,usize prior){let u32 kind=json::token(tree,at).kind;if(kind==11){return .ok(prior);}if(kind!=7){return .err(fail(tree,6,at));}return .ok(at);}
fn usize following(ref json::Tree tree,usize at){let usize end=json::next(tree,at);return if(json::token(tree,end).kind==6){end+1}else{end};}
fn Package empty_package(){return Package{.fields=[0;10],.first=0,.count=0,.capacity=0,.null_features=true};}
fn Result<Package,decode::Problem> feature_values(ref json::Tree tree,usize at,Package prior,ref mut v::Vec<usize> pool,ref mut usize meter){
    var Package record=prior;let u32 kind=json::token(tree,at).kind;
    if(kind!=3 && kind!=11){return .err(fail(tree,6,at));}
    record.first=v::len<usize>(pool);record.count=0;record.capacity=0;record.null_features=kind==11;
    if(kind==11 || json::token(tree,at+1).kind==4){return .ok(record);}
    // Go resets slice length while reusing earlier elements, including a tail
    // beyond its current length. Clone the retained capacity into our own pool.
    var usize index=0;while(index<prior.capacity){
        if(!work::charge(meter,1) || v::len<usize>(pool)>=1048576){return .err(fail(tree,3,at));}
        let usize value=*v::at<usize>(pool,prior.first+index);
        if(!v::push<usize>(pool,value).is_ok){return .err(fail(tree,2,at));}index+=1;
    }record.capacity=prior.capacity;
    var usize item=at+1;while(json::token(tree,item).kind!=4){
        if(!work::charge(meter,1)){return .err(fail(tree,3,item));}
        let usize previous=if(record.count<record.capacity){*v::at<usize>(pool,record.first+record.count)}else{0};
        let usize value=switch(string(tree,item,previous)){case .ok(value):{value}case .err(p):{return .err(p);}};
        if(record.count<record.capacity){let ref mut usize slot=v::at_mut<usize>(pool,record.first+record.count);*slot=value;}
        else{if(v::len<usize>(pool)>=1048576){return .err(fail(tree,3,item));}if(!v::push<usize>(pool,value).is_ok){return .err(fail(tree,2,item));}record.capacity+=1;}
        record.count+=1;item=following(tree,item);
    }return .ok(record);
}
fn Result<Package,decode::Problem> package(ref json::Tree tree,usize at,Package prior,ref mut v::Vec<usize> features,ref mut usize meter){
    var Package record=prior;let u32 kind=json::token(tree,at).kind;if(kind==11){return .ok(record);}if(kind!=1){return .err(fail(tree,6,at));}
    var usize key=at+1;while(json::token(tree,key).kind!=2){
        let usize id=switch(field(tree,key,1,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};let usize value=key+2;
        if(id<10){let usize previous=record.fields[id];let usize index=switch(string(tree,value,previous)){case .ok(value):{value}case .err(p):{return .err(p);}};record.fields[id]=index;}
        else{record=switch(feature_values(tree,value,record,features,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};}
        key=following(tree,value);
    }return .ok(record);
}
fn Result<Dependency,decode::Problem> dependency(ref json::Tree tree,usize at,Dependency prior,ref mut usize meter){
    var Dependency record=prior;let u32 kind=json::token(tree,at).kind;if(kind==11){return .ok(record);}if(kind!=1){return .err(fail(tree,6,at));}
    var usize key=at+1;while(json::token(tree,key).kind!=2){let usize id=switch(field(tree,key,2,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};let usize value=key+2;let usize previous=record.fields[id];let usize index=switch(string(tree,value,previous)){case .ok(value):{value}case .err(p):{return .err(p);}};record.fields[id]=index;key=following(tree,value);}return .ok(record);
}
fn Result<Fields,decode::Problem> read(json::Tree tree,ref mut usize meter){
    var v::Vec<usize> headers=v::empty<usize>();var usize h=0;while(h<5){if(!v::push<usize>(ref mut headers,0).is_ok){return .err(fail(ref tree,2,0));}h+=1;}
    var v::Vec<Package> packages=v::empty<Package>();var v::Vec<usize> features=v::empty<usize>();var v::Vec<Dependency> dependencies=v::empty<Dependency>();
    var usize packages_count=0;var usize dependencies_count=0;var bool null_packages=true;var bool null_dependencies=true;
    let u32 kind=json::token(ref tree,0).kind;if(kind!=1 && kind!=11){return .err(fail(ref tree,6,0));}
    if(kind==1){var usize key=1;while(json::token(ref tree,key).kind!=2){
        let usize id=switch(field(ref tree,key,0,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};let usize value=key+2;
        if(id<5){let usize index=switch(string(ref tree,value,*v::at<usize>(ref headers,id))){case .ok(value):{value}case .err(p):{return .err(p);}};let ref mut usize slot=v::at_mut<usize>(ref mut headers,id);*slot=index;}
        else{let u32 kind=json::token(ref tree,value).kind;if(kind!=3 && kind!=11){return .err(fail(ref tree,6,value));}
            let bool empty=kind==11 || json::token(ref tree,value+1).kind==4;
            if(id==5){packages_count=0;null_packages=kind==11;if(empty){discard v::truncate<Package>(ref mut packages,0);}}
            else{dependencies_count=0;null_dependencies=kind==11;if(empty){discard v::truncate<Dependency>(ref mut dependencies,0);}}
            if(kind==3){var usize item=value+1;while(json::token(ref tree,item).kind!=4){
                if(!work::charge(meter,1)){return .err(fail(ref tree,3,item));}
                if(id==5){let Package prior=if(packages_count<v::len<Package>(ref packages)){*v::at<Package>(ref packages,packages_count)}else{empty_package()};
                    let Package record=switch(package(ref tree,item,prior,ref mut features,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};
                    if(packages_count<v::len<Package>(ref packages)){let ref mut Package slot=v::at_mut<Package>(ref mut packages,packages_count);*slot=record;}
                    else if(!v::push<Package>(ref mut packages,record).is_ok){return .err(fail(ref tree,2,item));}packages_count+=1;
                }else{let Dependency prior=if(dependencies_count<v::len<Dependency>(ref dependencies)){*v::at<Dependency>(ref dependencies,dependencies_count)}else{Dependency{.fields=[0;3]}};
                    let Dependency record=switch(dependency(ref tree,item,prior,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};
                    if(dependencies_count<v::len<Dependency>(ref dependencies)){let ref mut Dependency slot=v::at_mut<Dependency>(ref mut dependencies,dependencies_count);*slot=record;}
                    else if(!v::push<Dependency>(ref mut dependencies,record).is_ok){return .err(fail(ref tree,2,item));}dependencies_count+=1;
                }item=following(ref tree,item);
            }}
        }key=following(ref tree,value);
    }}return .ok(Fields{.tree,.headers,.packages,.features,.dependencies,.packages_count,.dependencies_count,.null_packages,.null_dependencies});
}
// 1-4 generic JSON errors, 5 unknown field, 6 incompatible value type.
// Missing/default values and null collections require later semantic validation.
pub fn Result<Fields,decode::Problem> parse(bytes::Buffer source,ref mut usize meter){let tree=switch(json::parse(source,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};return read(tree,meter);}
