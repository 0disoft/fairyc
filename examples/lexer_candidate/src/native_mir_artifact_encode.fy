module native_mir_artifact_encode;
import native_mir_artifact_wire as private;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import mir_model as m;
import mir_verify as proof;
import hir_shapes as shapes;
import sdk_scopes as sdk;
import candidate_limits as limits;
import std::bytes as bytes;

// Detached transport only. Current-source rebinding is a separate owner.
pub fn Result<wire::Payload,u32> encode(ref m::Program p,ref mut usize work){
    if(!budget::charge(work,m::operation_count(p)+m::block_count(p)+m::local_count(p)+m::edge_count(p)+1)){return .err(1002);}
    if(m::module_index(p)>=limits::source_files()){return .err(1001);}
    let checked=proof::check(p);if(checked.code!=0){return .err(if(checked.code==2){1002}else if(checked.code==3){1003}else{1001});}
    var wire::Status s=wire::Status{.code=0};let family=m::has_closures(p);
    var out=wire::writer(if(family){b"FYNMIR01"}else{b"FYNMIR00"},ref mut s,work);
    wire::number(ref mut out,m::module_index(p),ref mut s,work);
    private::blob(ref mut out,m::source_view(p),ref mut s,work);
    private::blob(ref mut out,m::payload_view(p),ref mut s,work);
    var usize i=0;
    wire::number(ref mut out,shapes::count(m::nominal(p)),ref mut s,work);
    i=0;while(i<shapes::count(m::nominal(p)) && s.code==0){let row=shapes::at(m::nominal(p),i);
        wire::number(ref mut out,row.type as usize,ref mut s,work);
        wire::number(ref mut out,row.binding,ref mut s,work);
        wire::number(ref mut out,row.field_start,ref mut s,work);
        wire::number(ref mut out,row.field_count,ref mut s,work);
        wire::number(ref mut out,if(row.public){1}else{0},ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,shapes::field_count(m::nominal(p)),ref mut s,work);
    i=0;while(i<shapes::field_count(m::nominal(p)) && s.code==0){let row=shapes::field_at(m::nominal(p),i+1);
        wire::number(ref mut out,row.owner as usize,ref mut s,work);
        wire::number(ref mut out,row.type as usize,ref mut s,work);
        wire::number(ref mut out,row.start,ref mut s,work);
        wire::number(ref mut out,row.end,ref mut s,work);
        wire::number(ref mut out,row.payload_start,ref mut s,work);
        wire::number(ref mut out,row.payload_count,ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,shapes::abi_count(m::nominal(p)),ref mut s,work);
    i=0;while(i<shapes::abi_count(m::nominal(p)) && s.code==0){let row=shapes::abi_at(m::nominal(p),i);
        wire::number(ref mut out,row.type as usize,ref mut s,work);
        wire::number(ref mut out,row.declaration,ref mut s,work);
        wire::number(ref mut out,row.name_start,ref mut s,work);
        wire::number(ref mut out,row.name_end,ref mut s,work);
        wire::number(ref mut out,row.marker_start,ref mut s,work);
        wire::number(ref mut out,row.marker_end,ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,shapes::sdk_count(m::nominal(p)),ref mut s,work);
    i=0;while(i<shapes::sdk_count(m::nominal(p)) && s.code==0){let row=shapes::sdk_at(m::nominal(p),i);
        wire::number(ref mut out,row.sdk_id,ref mut s,work);
        wire::number(ref mut out,row.start,ref mut s,work);
        wire::number(ref mut out,row.end,ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,m::drop_count(p),ref mut s,work);
    i=0;while(i<m::drop_count(p) && s.code==0){let row=m::drop_at(p,i);
        wire::number(ref mut out,row.abilities as usize,ref mut s,work);
        wire::number(ref mut out,row.type as usize,ref mut s,work);
        wire::number(ref mut out,row.target,ref mut s,work);
        wire::number(ref mut out,if(row.nonempty){1}else{0},ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,m::target_count(p),ref mut s,work);
    i=0;while(i<m::target_count(p) && s.code==0){let row=m::target_at(p,i+1);
        wire::number(ref mut out,if(row.foreign){1}else{0},ref mut s,work);
        wire::number(ref mut out,row.link_start,ref mut s,work);
        wire::number(ref mut out,row.link_end,ref mut s,work);
        wire::number(ref mut out,if(row.unsafe_call){1}else{0},ref mut s,work);
        wire::number(ref mut out,row.effects as usize,ref mut s,work);
        wire::number(ref mut out,row.binding,ref mut s,work);
        wire::number(ref mut out,row.module_index,ref mut s,work);
        wire::number(ref mut out,row.declaration,ref mut s,work);
        wire::number(ref mut out,row.result as usize,ref mut s,work);
        wire::number(ref mut out,row.signature_start,ref mut s,work);
        wire::number(ref mut out,row.signature_count,ref mut s,work);
        wire::number(ref mut out,row.start,ref mut s,work);
        wire::number(ref mut out,row.end,ref mut s,work);
        wire::number(ref mut out,if(row.external){1}else{0},ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,m::signature_count(p),ref mut s,work);
    i=0;while(i<m::signature_count(p) && s.code==0){let row=m::Signature{.type=m::signature_at(p,i)};
        wire::number(ref mut out,row.type as usize,ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,m::function_count(p),ref mut s,work);
    i=0;while(i<m::function_count(p) && s.code==0){let row=m::function_at(p,i+1);
        wire::number(ref mut out,row.target,ref mut s,work);
        wire::number(ref mut out,row.entry,ref mut s,work);
        wire::number(ref mut out,row.local_start,ref mut s,work);
        wire::number(ref mut out,row.local_count,ref mut s,work);
        wire::number(ref mut out,row.block_start,ref mut s,work);
        wire::number(ref mut out,row.block_count,ref mut s,work);
        wire::number(ref mut out,row.parameter_start,ref mut s,work);
        wire::number(ref mut out,row.parameter_count,ref mut s,work);
        wire::number(ref mut out,row.start,ref mut s,work);
        wire::number(ref mut out,row.end,ref mut s,work);
        wire::number(ref mut out,if(row.public){1}else{0},ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,m::local_count(p),ref mut s,work);
    i=0;while(i<m::local_count(p) && s.code==0){let row=m::local_at(p,i+1);
        wire::number(ref mut out,row.function,ref mut s,work);
        wire::number(ref mut out,row.binding,ref mut s,work);
        wire::number(ref mut out,row.type as usize,ref mut s,work);
        wire::number(ref mut out,if(row.parameter){1}else{0},ref mut s,work);
        wire::number(ref mut out,if(row.mutable){1}else{0},ref mut s,work);
        wire::number(ref mut out,row.start,ref mut s,work);
        wire::number(ref mut out,row.end,ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,m::block_count(p),ref mut s,work);
    i=0;while(i<m::block_count(p) && s.code==0){let row=m::block_at(p,i+1);
        wire::number(ref mut out,if(row.unsafe_call){1}else{0},ref mut s,work);
        wire::number(ref mut out,row.function,ref mut s,work);
        wire::number(ref mut out,row.first,ref mut s,work);
        wire::number(ref mut out,row.last,ref mut s,work);
        wire::number(ref mut out,row.kind as usize,ref mut s,work);
        wire::number(ref mut out,row.value,ref mut s,work);
        wire::number(ref mut out,row.left,ref mut s,work);
        wire::number(ref mut out,row.right,ref mut s,work);
        wire::number(ref mut out,row.operator as usize,ref mut s,work);
        wire::number(ref mut out,row.type as usize,ref mut s,work);
        wire::number(ref mut out,row.target,ref mut s,work);
        wire::number(ref mut out,row.alternative,ref mut s,work);
        wire::number(ref mut out,row.check,ref mut s,work);
        wire::number(ref mut out,row.callee,ref mut s,work);
        wire::number(ref mut out,row.destination,ref mut s,work);
        wire::number(ref mut out,row.argument_start,ref mut s,work);
        wire::number(ref mut out,row.argument_count,ref mut s,work);
        wire::number(ref mut out,row.trap_mask as usize,ref mut s,work);
        wire::number(ref mut out,row.start,ref mut s,work);
        wire::number(ref mut out,row.end,ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,m::operation_count(p),ref mut s,work);
    i=0;while(i<m::operation_count(p) && s.code==0){let row=m::operation_at(p,i+1);
        wire::number(ref mut out,row.function,ref mut s,work);
        wire::number(ref mut out,row.block,ref mut s,work);
        wire::number(ref mut out,row.kind as usize,ref mut s,work);
        wire::number(ref mut out,row.result,ref mut s,work);
        wire::number(ref mut out,row.type as usize,ref mut s,work);
        wire::number(ref mut out,row.owner_type as usize,ref mut s,work);
        wire::number(ref mut out,row.operator as usize,ref mut s,work);
        wire::number(ref mut out,row.left,ref mut s,work);
        wire::number(ref mut out,row.right,ref mut s,work);
        wire::number(ref mut out,row.next,ref mut s,work);
        wire::number(ref mut out,row.payload_start,ref mut s,work);
        wire::number(ref mut out,row.payload_end,ref mut s,work);
        wire::number(ref mut out,if(row.boolean){1}else{0},ref mut s,work);
        wire::number(ref mut out,row.start,ref mut s,work);
        wire::number(ref mut out,row.end,ref mut s,work);
        wire::number(ref mut out,row.argument_start,ref mut s,work);
        wire::number(ref mut out,row.argument_count,ref mut s,work);
        i+=1;
    }
    wire::number(ref mut out,m::edge_count(p),ref mut s,work);
    i=0;while(i<m::edge_count(p) && s.code==0){let row=m::edge_at(p,i);
        wire::number(ref mut out,row,ref mut s,work);
        i+=1;
    }
    if(family){
    wire::number(ref mut out,m::closure_count(p),ref mut s,work);
    i=0;while(i<m::closure_count(p) && s.code==0){let row=m::closure_at(p,i);
        wire::number(ref mut out,row.target,ref mut s,work);
        wire::number(ref mut out,row.captures,ref mut s,work);
        i+=1;
    }
    }
    return wire::finish(out,s);
}
