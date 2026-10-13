module module_program_artifact_decode;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_cleanup_artifact_decode as cleanup_codec;
import module_unit_sdk_sources as sources;
import module_sdk_artifact as sdk;
import module_metadata_artifact_decode as metadata;
import module_metadata_artifact_model as model;
import module_state_artifact as state;
import module_body_artifact_decode as body;
import module_program_artifact_wire as wire;
import std::vec as v;

// Returns a freshly verified Program, not a source-bound Complete/cache hit.
pub fn Result<ir::Program,u32> decode(ref u8[] raw,ref mut usize work){
    let framed=wire::preflight(raw,work);if(framed!=0){return .err(framed);}var c=switch(wire::cursor(raw,work)){case .ok(c):{c}case .err(code):{return .err(code);}};
    let clean_span=wire::member_from(raw,ref mut c,8388608,b"FYCLEAN0",work);let sdk_span=wire::member_from(raw,ref mut c,8388608,b"FYSDK000",work);let meta_span=wire::member_from(raw,ref mut c,8388608,b"FYMETA00",work);let state_span=wire::member_from(raw,ref mut c,8388608,b"FYSTATE0",work);let count=wire::bodies_from(raw,ref mut c,work);if(wire::status(ref c)!=0){return .err(wire::status(ref c));}
    var usize measured=0;let cleaned=switch(cleanup_codec::decode(raw[clean_span.start..clean_span.end],ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}if(count!=clean::frame_count(ref cleaned)){return .err(1001);}
    measured=0;let sdk_sources=switch(sdk::decode(raw[sdk_span.start..sdk_span.end],ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}
    measured=0;let meta=switch(metadata::decode(raw[meta_span.start..meta_span.end],ref cleaned,sources::module_count(ref sdk_sources),ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}
    measured=0;let states=switch(state::decode(raw[state_span.start..state_span.end],ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}
    var v::Vec<ir::Body> bodies=v::empty<ir::Body>();var usize frame=1;
    while(frame<=count){let part=wire::member_from(raw,ref mut c,16777216,b"FYBODY00",work);if(wire::status(ref c)!=0){return .err(wire::status(ref c));}measured=0;let b=switch(body::decode(raw[part.start..part.end],ref cleaned,frame,ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}if(!v::push<ir::Body>(ref mut bodies,b).is_ok){return .err(1003);}frame+=1;}
    let end=wire::end(raw,ref mut c,work);if(end!=0){return .err(end);}measured=0;let restored=switch(model::restore(ref meta,states,sdk_sources,cleaned,bodies,ref mut measured)){case .ok(p):{p}case .err(code):{if(!wire::charge(work,measured)){return .err(1002);}return .err(code);}};if(!wire::charge(work,measured)){return .err(1002);}return .ok(restored);
}
