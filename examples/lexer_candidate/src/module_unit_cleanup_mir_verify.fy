module module_unit_cleanup_mir_verify;
import module_unit_cleanup_mir_model as m;
import candidate_limits as limits;
import module_unit_value_model as budget;
import module_unit_cleanup_mir_types as type_proof;
import std::vec as v;

struct Requirement {bool owned;bool droppable;u32 effects;}
fn m::Problem fail(u32 code,usize module_index,usize start,usize end){return m::Problem{.code,.module_index,.start,.end};}
fn Result<Requirement,m::Problem> requirement(ref m::Program p,usize type,ref mut v::Vec<u8> state,ref mut v::Vec<Requirement> cache,ref mut usize work,usize depth){
    if(type==0 || type>m::type_count(p)){return .err(fail(1001,0,0,0));}if(depth>=64 || !budget::charge(work,1)){return .err(fail(1002,0,0,0));}let mark=*v::at<u8>(state,type-1);if(mark==1){return .err(fail(1001,0,0,0));}if(mark==2){return .ok(*v::at<Requirement>(cache,type-1));}
    let slot=v::at_mut<u8>(state,type-1);*slot=1;let t=m::type_at(p,type);let g=m::glue_at(p,type);var result=Requirement{.owned=g.kind==1,.droppable=g.kind!=1 || (g.abilities&1)!=0,.effects=g.effects};
    if(g.kind==2 || g.kind==3){let a=switch(requirement(p,t.first,state,cache,work,depth+1)){case .ok(a):{a}case .err(e):{return .err(e);}};result.owned=a.owned;result.droppable=a.droppable;result.effects=a.effects;
        if(g.kind==3){let b=switch(requirement(p,t.second,state,cache,work,depth+1)){case .ok(b):{b}case .err(e):{return .err(e);}};result.owned=result.owned || b.owned;result.droppable=result.droppable && b.droppable;result.effects|=b.effects;}
    }else if(g.kind==1){var usize i=0;while(i<g.field_count){let f=m::field_at(p,g.field_start+i);let child=switch(requirement(p,f.type,state,cache,work,depth+1)){case .ok(a):{a}case .err(e):{return .err(e);}};result.droppable=result.droppable && child.droppable;result.effects|=child.effects;i+=1;}}
    let memo=v::at_mut<Requirement>(cache,type-1);*memo=result;let done=v::at_mut<u8>(state,type-1);*done=2;return .ok(result);
}
pub fn m::Problem check(ref m::Program p){
    let nt=m::type_count(p);let nf=m::frame_count(p);let no=m::operand_count(p);if(nt==0 || nt>4096 || nf>limits::project_frames() || no>16384 || m::drop_count(p)>16384 || m::field_count(p)>limits::metadata_fields() || m::work(p)>1048576){return fail(1002,0,0,0);}
    var usize work=m::work(p);var v::Vec<u8> state=v::empty<u8>();var v::Vec<Requirement> cache=v::empty<Requirement>();var v::Vec<usize> canonical=v::empty<usize>();var usize i=0;
    while(i<8192){if(!v::push<usize>(ref mut canonical,0).is_ok){return fail(1003,0,0,0);}i+=1;}if(!budget::charge(ref mut work,8192)){return fail(1002,0,0,0);}i=1;
    while(i<=nt){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let t=m::type_at(p,i);let g=m::glue_at(p,i);let u32 kind=if(t.kind==3 || t.kind==13){1}else if(t.kind==5){2}else if(t.kind==6){3}else{0};
        if(!type_proof::valid(p,i) || g.kind!=kind || g.effects>4095 || g.abilities>7 || (g.abilities&4)!=0 && (g.abilities&2)==0 || g.field_start>m::field_count(p) || g.field_count>m::field_count(p)-g.field_start){return fail(1001,t.module_index,0,0);}
        var usize bucket=type_proof::bucket(t);while(*v::at<usize>(ref canonical,bucket)!=0){if(!budget::charge(ref mut work,1)){return fail(1002,t.module_index,0,0);}if(type_proof::same(t,m::type_at(p,*v::at<usize>(ref canonical,bucket)))){return fail(1001,t.module_index,0,0);}bucket=(bucket+1)&8191;}let identity=v::at_mut<usize>(ref mut canonical,bucket);*identity=i;
        if(g.kind!=1 && (g.frame!=0 || g.nonempty || g.effects!=0 || g.abilities!=0 || g.field_start!=0 || g.field_count!=0)){return fail(1001,t.module_index,0,0);}
        if(g.kind==1 && g.nonempty!=(g.frame!=0)){return fail(1001,t.module_index,0,0);}
        if(g.kind==1 && g.frame!=0){if(g.frame>nf){return fail(1001,t.module_index,0,0);}let f=m::frame_at(p,g.frame);if(f.kind!=2 || f.module_index!=t.module_index || f.declaration!=t.declaration || f.receiver_type!=i || f.effects!=g.effects){return fail(1001,t.module_index,0,0);}}
        if(g.kind==1 && g.effects!=0 && g.frame==0){return fail(1001,t.module_index,0,0);}
        var usize field=0;while(field<g.field_count){let f=m::field_at(p,g.field_start+field);if(f.owner!=i || f.ordinal!=field || f.type==0 || f.type>nt || !budget::charge(ref mut work,1)){return fail(1001,t.module_index,0,0);}field+=1;}
        if(!v::push<u8>(ref mut state,0).is_ok || !v::push<Requirement>(ref mut cache,Requirement{.owned=false,.droppable=true,.effects=0}).is_ok){return fail(1003,0,0,0);}i+=1;
    }
    i=1;while(i<=nf){let f=m::frame_at(p,i);if(f.kind<1 || f.kind>3 || f.context==0 || f.module_index>=256 || f.declaration==0 || f.result==0 || f.result>nt || f.effects>4095 || (f.kind==2)!=(f.receiver_type!=0) || (f.kind==2)!=(f.receiver!=0) || f.receiver_type>nt){return fail(1001,f.module_index,0,0);}let result=m::type_at(p,f.result);if(f.kind==2 && (result.kind!=1 || result.scalar!=7)){return fail(1001,f.module_index,0,0);}if(!budget::charge(ref mut work,1)){return fail(1002,f.module_index,0,0);}i+=1;}
    i=0;while(i<m::field_count(p)){let f=m::field_at(p,i);if(f.owner==0 || f.owner>nt){return fail(1001,0,0,0);}let g=m::glue_at(p,f.owner);if(g.kind!=1 || f.ordinal>=g.field_count || g.field_start+f.ordinal!=i){return fail(1001,0,0,0);}i+=1;}
    // Validate every recipe, even one not used by a drop, so dead metadata cannot
    // hide cycles or an invalid tagged alternative from a later C handoff.
    i=1;while(i<=nt){switch(requirement(p,i,ref mut state,ref mut cache,ref mut work,0)){case .ok(_):{}case .err(e):{return e;}};i+=1;}
    var v::Vec<usize> seen=v::empty<usize>();i=0;while(i<32768){if(!v::push<usize>(ref mut seen,0).is_ok){return fail(1003,0,0,0);}i+=1;}if(!budget::charge(ref mut work,32768)){return fail(1002,0,0,0);}
    i=1;while(i<=no){let o=m::operand_at(p,i);if(o.frame==0 || o.frame>nf || o.kind<1 || o.kind>2 || o.id==0 || o.id>16384 || o.type==0 || o.type>nt || o.start>=o.end){return fail(1001,0,o.start,o.end);}var usize slot=(o.frame*104729 ^ o.id*2246822519 ^ o.kind as usize*3266489917)&32767;
        while(*v::at<usize>(ref seen,slot)!=0){if(!budget::charge(ref mut work,1)){return fail(1002,0,o.start,o.end);}let before=m::operand_at(p,*v::at<usize>(ref seen,slot));if(before.frame==o.frame && before.kind==o.kind && before.id==o.id){return fail(1001,m::frame_at(p,o.frame).module_index,o.start,o.end);}slot=(slot+1)&32767;}let cell=v::at_mut<usize>(ref mut seen,slot);*cell=i;i+=1;
    }
    i=0;while(i<m::drop_count(p)){let d=m::drop_at(p,i);if(d.operand==0 || d.operand>no || d.exit==0 || d.exit>9 || d.start>=d.end){return fail(1001,0,d.start,d.end);}let o=m::operand_at(p,d.operand);let f=m::frame_at(p,o.frame);let needs=*v::at<Requirement>(ref cache,o.type-1);if(!needs.owned || !needs.droppable || (f.effects&needs.effects)!=needs.effects || o.kind==1 && f.kind==2 && o.id==f.receiver){return fail(1001,f.module_index,d.start,d.end);}if(!budget::charge(ref mut work,1)){return fail(1002,f.module_index,d.start,d.end);}i+=1;}
    return fail(0,0,0,0);
}
