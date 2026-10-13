module module_project_public_extract;
import module_project_public_view as project;
import module_public_interface_extract as ordinary;
import module_public_type_text as coordinates;
import module_signatures as signatures;
import project_path as budget;
import syntax as syntax;
import std::vec as v;

// Accept only a sealed projection of a complete checked project. Authenticate
// source/import/owner/coordinate facts once before rendering the linked graph.
// Descriptor work keeps its independent 1MiB history, folded into the existing
// 16MiB caller budget. No parser, body typing or Flow analysis is rebuilt.
pub fn Result<ordinary::Projection,u32> extract(ref project::View view,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,usize owner,ref mut usize work){
    let current=project::current(view,sources,metadata,table,work);if(current!=0){return .err(current);}var usize measured=0;
    switch(ordinary::extract_records_prepared(sources,metadata,project::type_records(view),project::shape_records(view),project::field_records(view),project::name_records(view),table,project::type_names(view),owner,true,ref mut measured)){case .ok(p):{if(!budget::charge(work,measured)){return .err(10);}return .ok(p);}case .err(code):{discard budget::charge(work,measured);return .err(code);}};return .err(9);
}
