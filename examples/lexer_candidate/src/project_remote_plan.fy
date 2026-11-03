module project_remote_plan;
import project_json_tree as json;
import project_json_decode as decode;
import project_manifest_values as rules;
import project_source_url as url;
import project_path as work;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

struct Pin{usize path;usize uid;usize version;usize url;usize digest;}
pub resource Plan{json::Tree tree;v::Vec<Pin> pins;cleanup{}}
// Private failures: 1 JSON/shape, 2 allocation, 3 input/work bound,
// 4 duplicate key, 5 invalid pin/schema or duplicate path/UID.
fn decode::Problem fail(ref json::Tree tree,u32 code,usize at){let t=json::token(tree,at);return decode::Problem{.code,.start=t.start,.end=t.end};}
pub fn usize count(ref Plan plan){return v::len<Pin>(ref plan.pins);}
pub fn ref u8[] value(ref Plan plan,usize index,u32 field){
    let pin=*v::at<Pin>(ref plan.pins,index);
    let usize at=if(field==0){pin.path}else if(field==1){pin.uid}else if(field==2){pin.version}else if(field==3){pin.url}else{pin.digest};
    return json::value(ref plan.tree,at);
}
fn usize field(ref json::Tree tree,usize object,ref u8[] name){
    var usize at=object+1;while(json::token(tree,at).kind!=2){if(bytes::equal(json::value(tree,at),name)){return at+2;}at=json::next(tree,at+2);if(json::token(tree,at).kind==6){at+=1;}}
    return 0;
}
fn ref u8[] key(u32 id){if(id==0){return b"path";}if(id==1){return b"packageUid";}if(id==2){return b"version";}if(id==3){return b"url";}return b"sha256";}
fn bool shape(ref json::Tree tree,usize object,bool root){
    if(json::token(tree,object).kind!=1){return false;}var usize at=object+1;var usize fields=0;
    while(json::token(tree,at).kind!=2){
        let ref u8[] name=json::value(tree,at);var bool known=false;
        if(root){known=bytes::equal(name,b"schema") || bytes::equal(name,b"packages");}
        else{var u32 i=0;while(i<5){if(bytes::equal(name,key(i))){known=true;}i+=1;}if(json::token(tree,at+2).kind!=7){return false;}}
        if(!known){return false;}fields+=1;at=json::next(tree,at+2);if(json::token(tree,at).kind==6){at+=1;}
    }return fields==if(root){2}else{5};
}
fn bool valid_path(ref u8[] value){return value.len>7 && bytes::equal(value[..7],b"vendor/") && rules::name(value[7..]) && rules::segment(value[7..]);}
fn bool digest(ref u8[] value){if(value.len!=64){return false;}var usize i=0;while(i<value.len){let u8 c=value[i];if(!(c>=48 && c<=57) && !(c>=97 && c<=102)){return false;}i+=1;}return true;}
fn void put(ref mut v::Vec<Pin> pins,usize index,Pin pin){let ref mut Pin slot=v::at_mut<Pin>(pins,index);*slot=pin;}
pub fn Result<Plan,decode::Problem> parse(bytes::Buffer source,ref mut usize meter){
    let usize size=bytes::len(ref source);
    if(size>65536){return .err(decode::Problem{.code=3,.start=0,.end=size});}
    if(!text::validate_utf8(bytes::view(ref source)).is_ok){return .err(decode::Problem{.code=1,.start=0,.end=size});}
    let tree=switch(json::parse(source,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};
    if(!shape(ref tree,0,true)){return .err(fail(ref tree,1,0));}
    let usize schema=field(ref tree,0,b"schema");let usize packages=field(ref tree,0,b"packages");
    if(json::token(ref tree,schema).kind!=7 || !bytes::equal(json::value(ref tree,schema),b"fairyc.sources/v1") || json::token(ref tree,packages).kind!=3){return .err(fail(ref tree,5,0));}
    var v::Vec<Pin> pins=v::empty<Pin>();var usize at=packages+1;
    while(json::token(ref tree,at).kind!=4){
        if(v::len<Pin>(ref pins)>=32 || !shape(ref tree,at,false)){return .err(fail(ref tree,1,at));}
        let pin=Pin{.path=field(ref tree,at,b"path"),.uid=field(ref tree,at,b"packageUid"),.version=field(ref tree,at,b"version"),.url=field(ref tree,at,b"url"),.digest=field(ref tree,at,b"sha256")};
        let ref u8[] path=json::value(ref tree,pin.path);let ref u8[] uid=json::value(ref tree,pin.uid);
        if(!work::charge(meter,size+1)){return .err(fail(ref tree,3,at));}
        if(!valid_path(path) || !rules::uid(uid) || !rules::version(json::value(ref tree,pin.version)) || !url::valid(json::value(ref tree,pin.url)) || !digest(json::value(ref tree,pin.digest))){return .err(fail(ref tree,5,at));}
        var usize index=0;while(index<v::len<Pin>(ref pins)){let previous=*v::at<Pin>(ref pins,index);if(bytes::equal(path,json::value(ref tree,previous.path)) || bytes::equal(uid,json::value(ref tree,previous.uid))){return .err(fail(ref tree,5,at));}index+=1;}
        if(!v::push<Pin>(ref mut pins,pin).is_ok){return .err(fail(ref tree,2,at));}
        index=v::len<Pin>(ref pins)-1;
        while(index>0){let previous=*v::at<Pin>(ref pins,index-1);if(!rules::before(path,json::value(ref tree,previous.path))){break;}put(ref mut pins,index,previous);index-=1;}
        put(ref mut pins,index,pin);at=json::next(ref tree,at);if(json::token(ref tree,at).kind==6){at+=1;}
    }
    if(v::len<Pin>(ref pins)==0){return .err(fail(ref tree,5,packages));}
    return .ok(Plan{.tree,.pins});
}
