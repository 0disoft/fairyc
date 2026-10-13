module module_public_source_types;
import module_public_type_ids as ids;
import module_public_interface_model as model;
import module_source_specializations as selection;
import generic_source_plan as plan;
import module_unit_interfaces as typed;
import module_signatures as signatures;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub fn Result<ids::Table,selection::Problem> build(ref plan::Plan source,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref mut usize work){
    let selected=switch(selection::build(source,sources,metadata,graph,work)){case .ok(p):{p}case .err(e):{return .err(e);}};var bytes::Buffer names=bytes::empty();var v::Vec<ids::Row> rows=v::empty<ids::Row>();var v::Vec<ids::Stamp> stamps=v::empty<ids::Stamp>();var usize i=0;
    while(i<signatures::module_count(metadata)){if(!budget::charge(work,1)){return .err(selection::Problem{.code=10,.kind=0,.id=0});}if(!v::push<ids::Stamp>(ref mut stamps,ids::Stamp{.owner=signatures::module_owner(metadata,i),.digest=signatures::source_digest(metadata,i)}).is_ok){return .err(selection::Problem{.code=11,.kind=0,.id=0});}i+=1;}
    i=1;while(i<=typed::type_count(graph)){if(!budget::charge(work,1)){return .err(selection::Problem{.code=10,.kind=2,.id=i});}let type=typed::type_at(graph,i);var model::Text text=model::Text{.start=0,.end=0};var usize owner=0;
        if(type.kind==13){let row=selection::type_at(ref selected,i);let id=selection::identity(ref selected,row);let prefix=b"std::vec::_fy_generic_";if(row.node==0 || id.len!=67 || type.module_index>=v::len<syntax::Parsed>(sources)){return .err(selection::Problem{.code=9,.kind=2,.id=i});}if(bytes::len(ref names)>1048576-prefix.len-id.len || !budget::charge(work,prefix.len+id.len+1)){return .err(selection::Problem{.code=10,.kind=2,.id=i});}let start=bytes::len(ref names);if(!bytes::append(ref mut names,prefix).is_ok || !bytes::append(ref mut names,id).is_ok){return .err(selection::Problem{.code=11,.kind=2,.id=i});}text=model::Text{.start,.end=bytes::len(ref names)};if(!ids::valid(bytes::view(ref names)[text.start..text.end])){return .err(selection::Problem{.code=9,.kind=2,.id=i});}owner=signatures::module_owner(metadata,type.module_index);}
        if(!v::push<ids::Row>(ref mut rows,ids::Row{.type,.owner,.name=text}).is_ok){return .err(selection::Problem{.code=11,.kind=2,.id=i});}i+=1;
    }return .ok(ids::assemble(names,rows,stamps));
}
