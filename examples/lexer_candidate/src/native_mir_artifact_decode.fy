module native_mir_artifact_decode;
import native_mir_artifact_wire as private;
import candidate_artifact_wire as wire;
import mir_model as m;
import mir_verify as proof;
import hir_shapes as shapes;
import sdk_scopes as sdk;
import candidate_limits as limits;
import std::vec as v;

// Re-enter the explicit family assembler and full verifier. A detached result
// is never itself evidence of current source, a Complete owner or a cache hit.
pub fn Result<m::Program,u32> decode(ref u8[] raw,ref mut usize work){
    let family=switch(private::family(raw)){case .ok(v):{v}case .err(code):{return .err(code);}};
    var c=switch(wire::cursor(raw,if(family){b"FYNMIR01"}else{b"FYNMIR00"},work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let module_index=wire::number_from(raw,ref mut c,work);if(c.code!=0){return .err(c.code);}if(module_index>=limits::source_files()){return .err(1001);}
    let source=switch(private::blob_from(raw,ref mut c,limits::program_bytes(),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let payload=switch(private::blob_from(raw,ref mut c,688128,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var usize i=0;
    let shapes_count=wire::count_from(raw,ref mut c,512,40,work);
    if(c.code!=0){return .err(c.code);}var shapes=v::empty<shapes::Shape>();
    i=0;while(i<shapes_count){
        let row=shapes::Shape{
            .type=wire::word_from(raw,ref mut c,work),
            .binding=wire::number_from(raw,ref mut c,work),
            .field_start=wire::number_from(raw,ref mut c,work),
            .field_count=wire::number_from(raw,ref mut c,work),
            .public=wire::flag_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<shapes::Shape>(ref mut shapes,row).is_ok){return .err(1003);}i+=1;
    }
    let fields_count=wire::count_from(raw,ref mut c,1024,48,work);
    if(c.code!=0){return .err(c.code);}var fields=v::empty<shapes::Field>();
    i=0;while(i<fields_count){
        let row=shapes::Field{
            .owner=wire::word_from(raw,ref mut c,work),
            .type=wire::word_from(raw,ref mut c,work),
            .start=wire::number_from(raw,ref mut c,work),
            .end=wire::number_from(raw,ref mut c,work),
            .payload_start=wire::number_from(raw,ref mut c,work),
            .payload_count=wire::number_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<shapes::Field>(ref mut fields,row).is_ok){return .err(1003);}i+=1;
    }
    let abi_count=wire::count_from(raw,ref mut c,512,48,work);
    if(c.code!=0){return .err(c.code);}var abi=v::empty<shapes::Abi>();
    i=0;while(i<abi_count){
        let row=shapes::Abi{
            .type=wire::word_from(raw,ref mut c,work),
            .declaration=wire::number_from(raw,ref mut c,work),
            .name_start=wire::number_from(raw,ref mut c,work),
            .name_end=wire::number_from(raw,ref mut c,work),
            .marker_start=wire::number_from(raw,ref mut c,work),
            .marker_end=wire::number_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<shapes::Abi>(ref mut abi,row).is_ok){return .err(1003);}i+=1;
    }
    let sdk_count=wire::count_from(raw,ref mut c,8,24,work);
    if(c.code!=0){return .err(c.code);}var sdk=v::empty<sdk::Scope>();
    i=0;while(i<sdk_count){
        let row=sdk::Scope{
            .sdk_id=wire::number_from(raw,ref mut c,work),
            .start=wire::number_from(raw,ref mut c,work),
            .end=wire::number_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<sdk::Scope>(ref mut sdk,row).is_ok){return .err(1003);}i+=1;
    }
    let drops_count=wire::count_from(raw,ref mut c,512,32,work);
    if(c.code!=0){return .err(c.code);}var drops=v::empty<m::Drop>();
    i=0;while(i<drops_count){
        let row=m::Drop{
            .abilities=wire::word_from(raw,ref mut c,work),
            .type=wire::word_from(raw,ref mut c,work),
            .target=wire::number_from(raw,ref mut c,work),
            .nonempty=wire::flag_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<m::Drop>(ref mut drops,row).is_ok){return .err(1003);}i+=1;
    }
    let targets_count=wire::count_from(raw,ref mut c,512,112,work);
    if(c.code!=0){return .err(c.code);}var targets=v::empty<m::Target>();
    i=0;while(i<targets_count){
        let row=m::Target{
            .foreign=wire::flag_from(raw,ref mut c,work),
            .link_start=wire::number_from(raw,ref mut c,work),
            .link_end=wire::number_from(raw,ref mut c,work),
            .unsafe_call=wire::flag_from(raw,ref mut c,work),
            .effects=wire::word_from(raw,ref mut c,work),
            .binding=wire::number_from(raw,ref mut c,work),
            .module_index=wire::number_from(raw,ref mut c,work),
            .declaration=wire::number_from(raw,ref mut c,work),
            .result=wire::word_from(raw,ref mut c,work),
            .signature_start=wire::number_from(raw,ref mut c,work),
            .signature_count=wire::number_from(raw,ref mut c,work),
            .start=wire::number_from(raw,ref mut c,work),
            .end=wire::number_from(raw,ref mut c,work),
            .external=wire::flag_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<m::Target>(ref mut targets,row).is_ok){return .err(1003);}i+=1;
    }
    let signatures_count=wire::count_from(raw,ref mut c,65536,8,work);
    if(c.code!=0){return .err(c.code);}var signatures=v::empty<m::Signature>();
    i=0;while(i<signatures_count){
        let row=m::Signature{
            .type=wire::word_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<m::Signature>(ref mut signatures,row).is_ok){return .err(1003);}i+=1;
    }
    let functions_count=wire::count_from(raw,ref mut c,512,88,work);
    if(c.code!=0){return .err(c.code);}var functions=v::empty<m::Function>();
    i=0;while(i<functions_count){
        let row=m::Function{
            .target=wire::number_from(raw,ref mut c,work),
            .entry=wire::number_from(raw,ref mut c,work),
            .local_start=wire::number_from(raw,ref mut c,work),
            .local_count=wire::number_from(raw,ref mut c,work),
            .block_start=wire::number_from(raw,ref mut c,work),
            .block_count=wire::number_from(raw,ref mut c,work),
            .parameter_start=wire::number_from(raw,ref mut c,work),
            .parameter_count=wire::number_from(raw,ref mut c,work),
            .start=wire::number_from(raw,ref mut c,work),
            .end=wire::number_from(raw,ref mut c,work),
            .public=wire::flag_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<m::Function>(ref mut functions,row).is_ok){return .err(1003);}i+=1;
    }
    let locals_count=wire::count_from(raw,ref mut c,16384,56,work);
    if(c.code!=0){return .err(c.code);}var locals=v::empty<m::Local>();
    i=0;while(i<locals_count){
        let row=m::Local{
            .function=wire::number_from(raw,ref mut c,work),
            .binding=wire::number_from(raw,ref mut c,work),
            .type=wire::word_from(raw,ref mut c,work),
            .parameter=wire::flag_from(raw,ref mut c,work),
            .mutable=wire::flag_from(raw,ref mut c,work),
            .start=wire::number_from(raw,ref mut c,work),
            .end=wire::number_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<m::Local>(ref mut locals,row).is_ok){return .err(1003);}i+=1;
    }
    let blocks_count=wire::count_from(raw,ref mut c,16384,160,work);
    if(c.code!=0){return .err(c.code);}var blocks=v::empty<m::Block>();
    i=0;while(i<blocks_count){
        let row=m::Block{
            .unsafe_call=wire::flag_from(raw,ref mut c,work),
            .function=wire::number_from(raw,ref mut c,work),
            .first=wire::number_from(raw,ref mut c,work),
            .last=wire::number_from(raw,ref mut c,work),
            .kind=wire::word_from(raw,ref mut c,work),
            .value=wire::number_from(raw,ref mut c,work),
            .left=wire::number_from(raw,ref mut c,work),
            .right=wire::number_from(raw,ref mut c,work),
            .operator=wire::word_from(raw,ref mut c,work),
            .type=wire::word_from(raw,ref mut c,work),
            .target=wire::number_from(raw,ref mut c,work),
            .alternative=wire::number_from(raw,ref mut c,work),
            .check=wire::number_from(raw,ref mut c,work),
            .callee=wire::number_from(raw,ref mut c,work),
            .destination=wire::number_from(raw,ref mut c,work),
            .argument_start=wire::number_from(raw,ref mut c,work),
            .argument_count=wire::number_from(raw,ref mut c,work),
            .trap_mask=wire::word_from(raw,ref mut c,work),
            .start=wire::number_from(raw,ref mut c,work),
            .end=wire::number_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<m::Block>(ref mut blocks,row).is_ok){return .err(1003);}i+=1;
    }
    let operations_count=wire::count_from(raw,ref mut c,32768,136,work);
    if(c.code!=0){return .err(c.code);}var operations=v::empty<m::Operation>();
    i=0;while(i<operations_count){
        let row=m::Operation{
            .function=wire::number_from(raw,ref mut c,work),
            .block=wire::number_from(raw,ref mut c,work),
            .kind=wire::word_from(raw,ref mut c,work),
            .result=wire::number_from(raw,ref mut c,work),
            .type=wire::word_from(raw,ref mut c,work),
            .owner_type=wire::word_from(raw,ref mut c,work),
            .operator=wire::word_from(raw,ref mut c,work),
            .left=wire::number_from(raw,ref mut c,work),
            .right=wire::number_from(raw,ref mut c,work),
            .next=wire::number_from(raw,ref mut c,work),
            .payload_start=wire::number_from(raw,ref mut c,work),
            .payload_end=wire::number_from(raw,ref mut c,work),
            .boolean=wire::flag_from(raw,ref mut c,work),
            .start=wire::number_from(raw,ref mut c,work),
            .end=wire::number_from(raw,ref mut c,work),
            .argument_start=wire::number_from(raw,ref mut c,work),
            .argument_count=wire::number_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<m::Operation>(ref mut operations,row).is_ok){return .err(1003);}i+=1;
    }
    let edges_count=wire::count_from(raw,ref mut c,65536,8,work);
    if(c.code!=0){return .err(c.code);}var edges=v::empty<usize>();
    i=0;while(i<edges_count){
        let row=wire::number_from(raw,ref mut c,work);
        if(c.code!=0){return .err(c.code);}if(!v::push<usize>(ref mut edges,row).is_ok){return .err(1003);}i+=1;
    }
    let usize closures_count=if(family){wire::count_from(raw,ref mut c,512,16,work)}else{0};
    if(c.code!=0){return .err(c.code);}var closures=v::empty<m::ClosureTarget>();
    i=0;while(i<closures_count){
        let row=m::ClosureTarget{
            .target=wire::number_from(raw,ref mut c,work),
            .captures=wire::number_from(raw,ref mut c,work)
        };
        if(c.code!=0){return .err(c.code);}if(!v::push<m::ClosureTarget>(ref mut closures,row).is_ok){return .err(1003);}i+=1;
    }
    let end=wire::end(raw,ref mut c,work);if(end!=0){return .err(end);}
    let nominal=shapes::assemble_sdk(shapes,fields,abi,sdk);
    let p=switch(m::assemble_native_artifact(source,payload,targets,signatures,functions,locals,blocks,operations,edges,module_index,nominal,drops,closures,family)){case .ok(p):{p}case .err(e):{return .err(if(e.code==2){1002}else if(e.code==3){1003}else{1001});}};
    if(m::has_closures(ref p)!=family){return .err(1001);}
    let checked=proof::check(ref p);if(checked.code!=0){return .err(if(checked.code==2){1002}else if(checked.code==3){1003}else{1001});}
    return .ok(p);
}
