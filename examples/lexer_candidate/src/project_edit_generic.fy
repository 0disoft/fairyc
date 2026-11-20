module project_edit_generic;
import project_compiler_input as input;
import project_atlas_declarations as declarations;
import project_source_plan as planner;
import generic_source_plan as plan;
import generic_source_plan_input as planned;
import generic_source_catalog as catalog;
import module_public_type_text as coordinates;
import module_project_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub resource Coverage {
    v::Vec<bytes::Buffer> keys;
    cleanup {}
}

pub fn Coverage empty() {
    return Coverage{.keys=v::empty<bytes::Buffer>()};
}

pub fn usize count(ref Coverage coverage) {
    return v::len<bytes::Buffer>(ref coverage.keys);
}

// Match the selected declaration's checked frames to the exact source plan.
// The ordered argument key survives changes to the generated body identity.
pub fn Result<Coverage,u32> collect(ref input::Input source,ref coordinates::Coordinates table,ref identified::Program checked,declarations::Row selected,bool aggregates,ref mut usize meter) {
    let original=switch(if(aggregates){
        planner::build_aggregate_sources(input::sources(source),input::owners(source),input::bindings(source),table,meter)
    }else{planner::build(source,table,meter)}){
        case .ok(p):{p}
        case .err(e):{return .err(e.code);}
    };
    let templates=planned::templates(plan::input(ref original));
    let context=ir::cleanup_ir(identified::ir_program(checked));
    var keys=v::empty<bytes::Buffer>();
    var usize frame=1;
    while(frame<=identified::frame_count(checked)){
        if(!work::charge(meter,1)){return .err(3);}
        let f=clean::frame_at(context,frame);
        let row=identified::frame_at(checked,frame);
        frame+=1;
        if(f.kind!=1 || f.module_index!=selected.module_index || f.declaration!=selected.declaration+1){continue;}
        if(row.node==0 || row.node>plan::count(ref original)){return .err(7);}
        let node=plan::at(ref original,row.node-1);
        let t=catalog::template_at(templates,node.template_index);
        let id=plan::text(ref original,node.id);
        if(!work::charge(meter,id.len+1)){return .err(3);}
        if(t.kind!=3 || t.module_index!=selected.module_index || t.declaration!=selected.declaration ||
            !bytes::equal(id,identified::identity(checked,row))){return .err(7);}
        let key=plan::text(ref original,node.key);
        if(!work::charge(meter,key.len+1)){return .err(3);}
        let copy=switch(bytes::copy(key)){case .ok(p):{p}case .err(_):{return .err(2);}};
        if(!v::push<bytes::Buffer>(ref mut keys,copy).is_ok){return .err(2);}
    }
    return .ok(Coverage{.keys});
}

pub fn Result<bool,u32> preserves(ref Coverage before,ref Coverage after,ref mut usize meter) {
    var usize i=0;
    while(i<count(before)){
        let key=bytes::view(v::at<bytes::Buffer>(ref before.keys,i));
        var usize j=0;
        while(j<count(after)){
            let candidate=bytes::view(v::at<bytes::Buffer>(ref after.keys,j));
            if(!work::charge(meter,key.len+candidate.len+1)){return .err(3);}
            if(bytes::equal(key,candidate)){break;}
            j+=1;
        }
        if(j==count(after)){return .ok(false);}
        i+=1;
    }
    return .ok(true);
}