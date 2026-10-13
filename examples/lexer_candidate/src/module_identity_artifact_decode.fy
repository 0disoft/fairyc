module module_identity_artifact_decode;
import module_identity_artifact_model as model;
import module_source_specializations as selection;
import module_public_interface_model as records;
import module_unit_cleanup_mir_model as clean;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import candidate_limits as limits;
import std::bytes as bytes;
import std::vec as v;
fn v::Vec<selection::Row> rows(ref u8[] raw,ref mut wire::Cursor c,usize count,ref mut usize work){var v::Vec<selection::Row> out=v::empty<selection::Row>();var usize i=0;while(i<count){let row=selection::Row{.node=wire::number_from(raw,c,work),.id=records::Text{.start=wire::number_from(raw,c,work),.end=wire::number_from(raw,c,work)}};if(wire::status(c)!=0){return out;}if(!v::push<selection::Row>(ref mut out,row).is_ok){c.code=1003;return out;}i+=1;}return out;}
pub fn Result<model::Snapshot,u32> decode(ref u8[] raw,ref clean::Program context,ref mut usize work){if(!budget::charge(work,0)){return .err(1002);}var c=switch(wire::cursor(raw,b"FYIDS000",work)){case .ok(c):{c}case .err(code):{return .err(code);}};let nt=wire::count_from(raw,ref mut c,4096,24,work);if(wire::status(ref c)!=0){return .err(wire::status(ref c));}if(nt!=clean::type_count(context)){return .err(1001);}let types=rows(raw,ref mut c,nt,work);if(wire::status(ref c)!=0){return .err(wire::status(ref c));}
    let nf=wire::count_from(raw,ref mut c,limits::project_frames(),24,work);if(wire::status(ref c)!=0){return .err(wire::status(ref c));}if(nf!=clean::frame_count(context)){return .err(1001);}let frames=rows(raw,ref mut c,nf,work);if(wire::status(ref c)!=0){return .err(wire::status(ref c));}let n=wire::count_from(raw,ref mut c,1048576,1,work);if(wire::status(ref c)!=0){return .err(wire::status(ref c));}if(!budget::charge(work,n+1)){return .err(1002);}let names=switch(bytes::copy(raw[c.offset..c.offset+n])){case .ok(p):{p}case .err(_):{return .err(1003);}};c.offset+=n;let code=wire::end(raw,ref mut c,work);if(code!=0){return .err(code);}return model::assemble(names,types,frames,context,work);
}
