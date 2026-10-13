module module_program_artifact_encode;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_cleanup_artifact_encode as cleanup_codec;
import module_sdk_artifact as sdk;
import module_metadata_artifact_encode as metadata;
import module_state_artifact as state;
import module_body_artifact_encode as body;
import candidate_artifact_wire as component;
import module_program_artifact_wire as wire;

pub fn Result<component::Payload,u32> encode(ref ir::Program p,ref mut usize work){
    if(!wire::charge(work,0)){return .err(1002);}var usize measured=0;let checked=proof::check_metered(p,ref mut measured);if(!wire::charge(work,measured)){return .err(1002);}if(checked.code!=0){return .err(checked.code);}
    var wire::Status s=wire::Status{.code=0};var out=wire::writer(ref mut s,work);
    measured=0;let clean_data=switch(cleanup_codec::encode(ir::cleanup_ir(p),ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}wire::member(ref mut out,component::content(ref clean_data),ref mut s,work);if(s.code!=0){return .err(s.code);}
    measured=0;let sdk_data=switch(sdk::encode(ir::sdk_units(p),ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}wire::member(ref mut out,component::content(ref sdk_data),ref mut s,work);if(s.code!=0){return .err(s.code);}
    measured=0;let meta_data=switch(metadata::program(p,ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}wire::member(ref mut out,component::content(ref meta_data),ref mut s,work);if(s.code!=0){return .err(s.code);}
    measured=0;let state_data=switch(state::encode(ir::state_layouts(p),ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}wire::member(ref mut out,component::content(ref state_data),ref mut s,work);if(s.code!=0){return .err(s.code);}
    wire::number(ref mut out,ir::count(p),ref mut s,work);if(s.code!=0){return .err(s.code);}var usize frame=1;
    while(frame<=ir::count(p)){measured=0;let data=switch(body::encode(p,frame,ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}wire::member(ref mut out,component::content(ref data),ref mut s,work);if(s.code!=0){return .err(s.code);}frame+=1;}
    return wire::finish(out,s);
}
