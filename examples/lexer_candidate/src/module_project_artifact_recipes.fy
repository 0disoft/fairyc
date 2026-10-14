module module_project_artifact_recipes;
import module_project_artifact_drops as drops;
import module_project_artifact_frames as frames;
import module_project_artifact_types as types;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_value_model as body;
import module_unit_interfaces as interfaces;
import module_unit_cleanup_liveness as liveness;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_identity_artifact_model as identities;
import generic_source_plan as source;
import module_signatures as signatures;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::vec as v;

// Compare source-derived recipes directly; no replacement cleanup MIR is built.
// Signature-only unused resources may have no shape in this source unit. Their
// recipes belong to other checked units; whole-project coverage remains later.
fn Result<v::Vec<usize>,u32> shapes(ref values::Plan typed,ref interfaces::Interfaces imported,ref mut usize work){
    var v::Vec<usize> slots=v::empty<usize>();var usize i=0;
    while(i<=values::type_count(typed)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut slots,0).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<interfaces::shape_count(imported)){if(!budget::charge(work,1)){return .err(1002);}let s=interfaces::shape_at(imported,i);if(s.type==0 || s.type>values::type_count(typed) || *v::at<usize>(ref slots,s.type)!=0){return .err(1001);}let cell=v::at_mut<usize>(ref mut slots,s.type);*cell=i+1;i+=1;}return .ok(slots);
}
fn Result<v::Vec<usize>,u32> destructors(ref clean::Program saved,ref values::Plan typed,ref frames::Mapping map,ref mut usize work){
    var v::Vec<usize> slots=v::empty<usize>();var usize i=0;
    while(i<=clean::type_count(saved)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut slots,0).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<values::count(typed)){if(!budget::charge(work,1)){return .err(1002);}if(body::frame(values::at(typed,i)).kind==2){let id=frames::mapped(map,i+1);if(id==0 || id>clean::frame_count(saved)){return .err(1001);}let f=clean::frame_at(saved,id);if(f.kind!=2 || f.receiver_type==0 || f.receiver_type>clean::type_count(saved) || *v::at<usize>(ref slots,f.receiver_type)!=0){return .err(1001);}let cell=v::at_mut<usize>(ref mut slots,f.receiver_type);*cell=id;}i+=1;}return .ok(slots);
}
pub fn Result<frames::Mapping,u32> match(ref ir::Program program,ref identities::Snapshot ids,ref checking::Plan checked,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){return match_mode(program,ids,checked,plan,sources,metadata,false,work);}
pub fn Result<frames::Mapping,u32> match_unit(ref ir::Program program,ref identities::Snapshot ids,ref checking::Plan checked,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){return match_mode(program,ids,checked,plan,sources,metadata,true,work);}
fn Result<frames::Mapping,u32> match_mode(ref ir::Program program,ref identities::Snapshot ids,ref checking::Plan checked,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,bool partial,ref mut usize work){
    let map=switch(if(partial){drops::match_unit(program,ids,checked,plan,sources,metadata,work)}else{drops::match(program,ids,checked,plan,sources,metadata,work)}){case .ok(p):{p}case .err(code):{return .err(code);}};
    let typed=checking::checked_values(checked);let imported=checking::checked_interfaces(checked);let saved=ir::cleanup_ir(program);
    let indexed=switch(shapes(typed,imported,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let bodies=switch(destructors(saved,typed,ref map,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var usize measured=0;let used=switch(liveness::build(typed,ref mut measured)){case .ok(p):{p}case .err(e):{if(!budget::charge(work,measured)){return .err(1002);}return .err(e.code);}};if(!budget::charge(work,measured)){return .err(1002);}
    var usize i=1;while(i<=values::type_count(typed)){if(!budget::charge(work,1)){return .err(1002);}let t=values::type_at(typed,i);
        if(t.kind==3 || t.kind==13){let at=*v::at<usize>(ref indexed,i);if(at==0){if(*v::at<u8>(ref used,i)!=0){return .err(1001);}}
            else{let s=interfaces::shape_at(imported,at-1);let target=types::mapped(frames::type_mapping(ref map),i);if(target==0 || target>clean::type_count(saved) || s.module_index!=t.module_index || s.declaration!=t.declaration){return .err(1001);}let g=clean::glue_at(saved,target);
                if(g.kind!=1 || g.nonempty!=s.cleanup_nonempty || g.effects!=s.cleanup_effects || g.abilities!=s.abilities || g.field_count!=s.field_count || g.frame!=if(s.cleanup_nonempty){*v::at<usize>(ref bodies,target)}else{0} || s.cleanup_nonempty && g.frame==0){return .err(1001);}
                var usize j=0;while(j<s.field_count){if(!budget::charge(work,1)){return .err(1002);}let fresh=interfaces::field_at(imported,s.field_start+j);let old=clean::field_at(saved,g.field_start+j);let type=types::mapped(frames::type_mapping(ref map),fresh.type);if(type==0 || old.owner!=target || old.ordinal!=j || old.type!=type){return .err(1001);}j+=1;}
            }
        }i+=1;
    }return .ok(map);
}
