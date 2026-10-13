module module_project_artifact_plan_index;
import generic_source_plan as source;
import project_path as budget;
import std::bytes as bytes;
import std::vec as v;

pub resource Index{v::Vec<usize> nodes;cleanup{}}
fn usize bucket(ref u8[] key,usize mask){var usize hash=0;var usize i=3;while(i<key.len){hash=(hash*33+(key[i] as usize))&mask;i+=1;}return hash;}
pub fn Result<Index,u32> build(ref source::Plan p,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}if(source::count(p)>4096){return .err(9);}var usize size=1;while(size<source::count(p)*2){size*=2;}var v::Vec<usize> nodes=v::empty<usize>();var usize i=0;
    while(i<size){if(!budget::charge(work,1)){return .err(10);}if(!v::push<usize>(ref mut nodes,0).is_ok){return .err(11);}i+=1;}i=0;
    while(i<source::count(p)){let row=source::at(p,i);let key=source::text(p,row.id);if(row.state!=2 || key.len!=67 || !bytes::equal(key[..3],b"g0_")){return .err(9);}if(!budget::charge(work,key.len+1)){return .err(10);}var usize slot=bucket(key,size-1);
        while(true){if(!budget::charge(work,1)){return .err(10);}let old=*v::at<usize>(ref nodes,slot);if(old==0){let cell=v::at_mut<usize>(ref mut nodes,slot);*cell=i+1;break;}let other=source::text(p,source::at(p,old-1).id);if(!budget::charge(work,other.len+1)){return .err(10);}if(bytes::equal(key,other)){return .err(9);}slot=(slot+1)&(size-1);}i+=1;
    }return .ok(Index{.nodes});
}
pub fn Result<usize,u32> find(ref Index index,ref source::Plan p,ref u8[] key,ref mut usize work){
    if(!budget::charge(work,key.len+1)){return .err(10);}if(key.len!=67 || !bytes::equal(key[..3],b"g0_")){return .err(9);}let count=v::len<usize>(ref index.nodes);var usize slot=bucket(key,count-1);var usize probes=0;
    while(probes<count){if(!budget::charge(work,1)){return .err(10);}let selected=*v::at<usize>(ref index.nodes,slot);if(selected==0){return .err(9);}if(selected>source::count(p)){return .err(9);}let other=source::text(p,source::at(p,selected-1).id);if(!budget::charge(work,other.len+1)){return .err(10);}if(bytes::equal(key,other)){return .ok(selected);}slot=(slot+1)&(count-1);probes+=1;}return .err(9);
}
