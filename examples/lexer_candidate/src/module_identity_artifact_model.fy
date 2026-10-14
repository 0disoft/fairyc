module module_identity_artifact_model;
import module_source_specializations as selection;
import module_public_interface_model as records;
import module_unit_cleanup_mir_model as clean;
import candidate_limits as limits;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// Snapshot rows and planner handles confer no checked source-owner authority.
pub resource Snapshot{bytes::Buffer names;v::Vec<selection::Row> types;v::Vec<selection::Row> frames;cleanup{}}
pub fn usize type_count(ref Snapshot p){return v::len<selection::Row>(ref p.types);}
pub fn usize frame_count(ref Snapshot p){return v::len<selection::Row>(ref p.frames);}
pub fn selection::Row type_at(ref Snapshot p,usize id){return *v::at<selection::Row>(ref p.types,id-1);}
pub fn selection::Row frame_at(ref Snapshot p,usize id){return *v::at<selection::Row>(ref p.frames,id-1);}
pub fn ref u8[] names(ref Snapshot p){return bytes::view(ref p.names);}
pub fn ref u8[] identity(ref Snapshot p,selection::Row row){return names(p)[row.id.start..row.id.end];}
fn selection::Row at(ref Snapshot p,usize id){return if(id<=type_count(p)){type_at(p,id)}else{frame_at(p,id-type_count(p))};}
fn u32 category(ref Snapshot p,ref clean::Program context,usize id){return if(id<=type_count(p)){1}else if(clean::frame_at(context,id-type_count(p)).kind==2){3}else{2};}
fn usize bucket(ref u8[] id,u32 kind,usize mask){var usize hash=kind as usize;var usize i=3;while(i<id.len){hash=(hash*33+(id[i] as usize))&mask;i+=1;}return hash;}
fn u32 row(ref Snapshot p,selection::Row value,ref mut bytes::Buffer coverage,ref mut usize work){if(!budget::charge(work,1)){return 1002;}if(value.node==0){return if(value.id.start==0 && value.id.end==0){0}else{1001};}if(value.node>4096 || value.id.start>value.id.end || value.id.end>names(p).len || value.id.end-value.id.start!=67){return 1001;}if(!budget::charge(work,68)){return 1002;}let id=identity(p,value);if(!bytes::equal(id[..3],b"g0_")){return 1001;}var usize i=3;while(i<67){let c=id[i];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return 1001;}i+=1;}let seen=bytes::view_mut(coverage);i=value.id.start;while(i<value.id.end){if(seen[i]!=0){return 1001;}seen[i]=1;i+=1;}return 0;}
pub fn u32 check(ref Snapshot p,ref clean::Program context,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(type_count(p)>4096 || frame_count(p)>limits::project_frames() || names(p).len>1048576){return 1002;}if(type_count(p)!=clean::type_count(context) || frame_count(p)!=clean::frame_count(context)){return 1001;}
    if(!budget::charge(work,names(p).len+1)){return 1002;}var bytes::Buffer coverage=bytes::empty();if(!bytes::resize_zeroed(ref mut coverage,names(p).len).is_ok){return 1003;}var usize size=1;while(size<2*(type_count(p)+frame_count(p))){size*=2;}var v::Vec<usize> seen=v::empty<usize>();var usize i=0;while(i<size){if(!budget::charge(work,1)){return 1002;}if(!v::push<usize>(ref mut seen,0).is_ok){return 1003;}i+=1;}
    i=1;while(i<=type_count(p)+frame_count(p)){let value=at(p,i);let checked=row(p,value,ref mut coverage,work);if(checked!=0){return checked;}
        if(i<=type_count(p)){if((value.node!=0)!=(clean::type_at(context,i).kind==13)){return 1001;}}
        else{let f=clean::frame_at(context,i-type_count(p));if(value.node!=0 && f.kind!=1 && f.kind!=2){return 1001;}if(f.kind==2){if(f.receiver_type==0 || f.receiver_type>type_count(p)){return 1001;}let receiver=type_at(p,f.receiver_type);if((receiver.node!=0)!=(value.node!=0)){return 1001;}if(value.node!=0 && !bytes::equal(identity(p,receiver),identity(p,value))){return 1001;}}}
        if(value.node!=0){let kind=category(p,context,i);let id=identity(p,value);var usize slot=bucket(id,kind,size-1);while(true){if(!budget::charge(work,1)){return 1002;}let old=*v::at<usize>(ref seen,slot);if(old==0){let cell=v::at_mut<usize>(ref mut seen,slot);*cell=i;break;}if(category(p,context,old)==kind){if(!budget::charge(work,68)){return 1002;}if(bytes::equal(identity(p,at(p,old)),id)){return 1001;}}slot=(slot+1)&(size-1);} }
        i+=1;
    }if(!budget::charge(work,names(p).len+1)){return 1002;}let covered=bytes::view(ref coverage);i=0;while(i<covered.len){if(covered[i]!=1){return 1001;}i+=1;}return 0;
}
pub fn Result<Snapshot,u32> assemble(bytes::Buffer names,v::Vec<selection::Row> types,v::Vec<selection::Row> frames,ref clean::Program context,ref mut usize work){let p=Snapshot{.names,.types,.frames};let code=check(ref p,context,work);if(code!=0){return .err(code);}return .ok(p);}
