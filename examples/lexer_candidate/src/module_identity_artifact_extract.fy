module module_identity_artifact_extract;
import module_identity_artifact_model as model;
import module_project_identities as identities;
import module_source_specializations as selection;
import module_unit_cleanup_mir_model as clean;
import candidate_limits as limits;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// Cold-owner extraction is separate from detached snapshot validation.
pub fn Result<model::Snapshot,u32> extract(ref identities::Table input,ref clean::Program context,ref mut usize work){if(!budget::charge(work,1)){return .err(1002);}if(identities::type_count(input)>4096 || identities::frame_count(input)>limits::project_frames() || identities::names_view(input).len>1048576){return .err(1002);}if(!budget::charge(work,identities::names_view(input).len+1)){return .err(1002);}let names=switch(bytes::copy(identities::names_view(input))){case .ok(p):{p}case .err(_):{return .err(1003);}};
    var v::Vec<selection::Row> types=v::empty<selection::Row>();var v::Vec<selection::Row> frames=v::empty<selection::Row>();var usize i=1;while(i<=identities::type_count(input)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<selection::Row>(ref mut types,identities::type_at(input,i)).is_ok){return .err(1003);}i+=1;}i=1;while(i<=identities::frame_count(input)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<selection::Row>(ref mut frames,identities::frame_at(input,i)).is_ok){return .err(1003);}i+=1;}return model::assemble(names,types,frames,context,work);
}
