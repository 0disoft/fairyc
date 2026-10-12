module project_compiler_entry;
import project_compiler_input as input;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_scalar as scalar;
import module_unit_value_model as budget;
import syntax as syntax;
import std::bytes as bytes;

// Callers retain the original projected input with its checked owned program.
// Root instance zero is established by project_instance_graph::expand. Only
// its exact module main owns the canonical entry; provider/helper names do not.
// 1 missing root main module, 2 missing ordinary main, 3 invalid signature,
// 4 invalid IR/source coordinate, 5 exhausted entry-selection work.
pub struct Problem {u32 kind;usize module_index;usize start;usize end;}
fn Problem fail(u32 kind,usize module_index,usize start,usize end){return Problem{.kind,.module_index,.start,.end};}
pub fn Result<usize,Problem> select(ref input::Input sources,ref ir::Program p,ref mut usize work){
    var usize module_id=0;var usize at=0;
    while(at<input::count(sources)){
        if(!budget::charge(work,1)){return .err(fail(5,0,0,0));}
        if(input::origin(sources,at).owner==0){let parsed=input::source(sources,at);if(syntax::declaration_count(parsed)==0){return .err(fail(4,at,0,0));}let d=syntax::declaration_at(parsed,0);
            if(d.kind==1 && bytes::equal(syntax::source_view(parsed)[d.name_start..d.name_end],b"main")){if(module_id!=0){return .err(fail(4,at,d.start,d.end));}module_id=at+1;}
        }at+=1;
    }
    if(module_id==0){return .err(fail(1,0,0,0));}let module_index=module_id-1;let parsed=input::source(sources,module_index);var usize selected=0;var usize frame=1;
    while(frame<=ir::count(p)){
        if(!budget::charge(work,1)){return .err(fail(5,module_index,0,0));}let f=clean::frame_at(ir::cleanup_ir(p),frame);
        if(f.module_index==module_index && f.kind==3){if(f.declaration==0 || f.declaration>syntax::declaration_count(parsed)){return .err(fail(4,module_index,0,0));}let d=syntax::declaration_at(parsed,f.declaration-1);
            if(bytes::equal(syntax::source_view(parsed)[d.name_start..d.name_end],b"main")){
                if(selected!=0 || d.kind!=3){return .err(fail(4,module_index,d.start,d.end));}let b=ir::at(p,frame);if(ir::source_bytes(b)!=syntax::source_view(parsed).len || ir::start(b)!=d.start || ir::end(b)!=d.end){return .err(fail(4,module_index,d.start,d.end));}if(scalar::scalar(p,f.result)!=1){return .err(fail(3,module_index,d.start,d.end));}var usize local=1;
                while(local<=ir::local_count(b)){if(!budget::charge(work,1)){return .err(fail(5,module_index,d.start,d.end));}if(ir::local_at(b,local).kind==1){return .err(fail(3,module_index,d.start,d.end));}local+=1;}selected=frame;
            }
        }frame+=1;
    }
    return if(selected==0){.err(fail(2,module_index,0,0))}else{.ok(selected)};
}
