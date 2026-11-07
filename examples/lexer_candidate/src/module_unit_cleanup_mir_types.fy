module module_unit_cleanup_mir_types;
import generic_transport_scalars as transport;
import module_unit_cleanup_mir_model as ir;
import module_types as types;
import c_abi_types as abi;
import pointer_types as pointers;
import scalar_views as views;
import candidate_limits as limits;

pub fn bool same(types::Type a,types::Type b){return a.kind==b.kind && a.scalar==b.scalar && a.module_index==b.module_index && a.declaration==b.declaration && a.first==b.first && a.second==b.second;}
pub fn usize bucket(types::Type t){return (t.kind as usize*31 ^ t.scalar as usize*2246822519 ^ t.module_index*104729 ^ t.declaration*3266489917 ^ t.first*2654435761 ^ t.second*668265263)&8191;}

// The catalog has been released. Validate the owned type descriptors without
// looking up declarations or replacing their canonical module coordinates.
pub fn bool valid(ref ir::Program p,usize id){
    let t=ir::type_at(p,id);
    // Reject untrusted coordinates before the canonical bucket arithmetic.
    if(t.module_index>=limits::source_files() || t.declaration>1048576 || t.first>4096 || t.second>4096){return false;}
    if(t.kind==20 || t.kind==21){return ir::aggregates(p) && t.declaration!=0 && t.scalar==0 && t.second==0 && t.first!=0 && t.first<id && ir::type_at(p,t.first).kind==22;}
    if(t.kind==2 || t.kind==3 || t.kind==4){return t.module_index<limits::source_files() && t.declaration!=0 && t.scalar==0 && t.first==0 && t.second==0;}
    if(t.kind==13){return t.module_index<limits::source_files() && t.declaration!=0 && t.scalar==0 && t.first!=0 && t.first<id && t.second==0 && ir::type_at(p,t.first).kind!=23;}
    if(t.module_index!=0 || t.declaration!=0){return false;}
    if(t.kind==22){if(!ir::aggregates(p) || t.scalar!=0 || t.first==0 || t.first>=id || ir::type_at(p,t.first).kind!=1 || !transport::valid(ir::type_at(p,t.first).scalar)){return false;}var usize tail=t.second;var usize length=1;while(tail!=0){if(tail>=id || ir::type_at(p,tail).kind!=22 || length>=8){return false;}length+=1;tail=ir::type_at(p,tail).second;}return true;}
    // The closure leaf has no descriptor payload and no wrapper admission.
    if(t.kind==23){return ir::closures(p) && t.scalar==0 && t.first==0 && t.second==0;}
    if(t.kind==1){return t.scalar!=0 && (t.scalar<=7 || transport::extended(t.scalar) || abi::valid(t.scalar)) && t.first==0 && t.second==0;}
    if(t.kind==15){return views::extended(t.scalar) && t.first==0 && t.second==0;}
    if(t.kind==17 || t.kind==18){return t.scalar==0 && t.first!=0 && t.first<id && t.second==0 && ir::type_at(p,t.first).kind==2;}
    if(t.kind==16){return t.scalar==0 && t.first!=0 && t.first<id && t.second<=4096 && ir::type_at(p,t.first).kind!=22 && ir::type_at(p,t.first).kind!=23;}
    if(t.kind==12){if(t.second!=0){return false;}return if(t.first==0){pointers::valid(t.scalar) && !(pointers::inner(t.scalar)>=256 && pointers::inner(t.scalar)<=767)}else{t.scalar<=1 && t.first<id && ir::type_at(p,t.first).kind==2};}
    if(t.kind>=9 && t.kind<=11 || t.kind==14 || t.kind==19){return t.scalar==0 && t.first==0 && t.second==0;}
    if(t.kind>=5 && t.kind<=8){return t.scalar==0 && t.first!=0 && t.first<id && ir::type_at(p,t.first).kind!=19 && ir::type_at(p,t.first).kind!=22 && ir::type_at(p,t.first).kind!=23 && if(t.kind==6){t.second!=0 && t.second<id && ir::type_at(p,t.second).kind!=19 && ir::type_at(p,t.second).kind!=22 && ir::type_at(p,t.second).kind!=23}else{t.second==0};}
    return false;
}
