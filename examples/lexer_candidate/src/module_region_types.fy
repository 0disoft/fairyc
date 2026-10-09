module module_region_types;
import module_signatures as signatures;
import module_types as types;
import module_ast_join as join;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Catalog construction already authenticates the reserved SDK source domain.
// A user module with a same-spelled AllocError cannot supply this intrinsic.
fn Result<usize,join::Problem> intrinsic_type(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut v::Vec<types::Type> graph,usize caller,usize start,usize end,ref u8[] name,u32 kind,ref mut usize work){
    if(v::len<syntax::Parsed>(sources)!=signatures::module_count(metadata)){return .err(join::Problem{.code=12,.module_index=caller,.start,.end});}var usize i=0;while(i<v::len<syntax::Parsed>(sources)){if(*work>=1048576){return .err(join::Problem{.code=10,.module_index=caller,.start,.end});};*work=*work+1;let p=v::at<syntax::Parsed>(sources,i);let d=syntax::declaration_at(p,0);
        if(bytes::equal(syntax::source_view(p)[d.name_start..d.name_end],b"std::alloc")){if(!signatures::source_matches(metadata,i,p)){return .err(join::Problem{.code=12,.module_index=caller,.start,.end});}let found=signatures::declaration_lookup_measured(metadata,i,name,work);if(*work>=1048576){return .err(join::Problem{.code=10,.module_index=caller,.start,.end});}if(found.code!=0){return .err(join::Problem{.code=2,.module_index=caller,.start,.end});}let item=signatures::declaration_at(metadata,found.item);if(item.kind!=kind || !item.public){return .err(join::Problem{.code=2,.module_index=caller,.start,.end});}let r=signatures::declaration_type(metadata,i,item.declaration,graph);if(r.problem.code!=0){return .err(join::Problem{.code=r.problem.code,.module_index=caller,.start,.end});}return .ok(r.type);}
        i+=1;
    }return .err(join::Problem{.code=2,.module_index=caller,.start,.end});
}

pub fn Result<usize,join::Problem> error_type(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut v::Vec<types::Type> graph,usize caller,usize start,usize end,ref mut usize work){return intrinsic_type(sources,metadata,graph,caller,start,end,b"AllocError",4,work);}
pub fn Result<usize,join::Problem> arena_type(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut v::Vec<types::Type> graph,usize caller,usize start,usize end,ref mut usize work){return intrinsic_type(sources,metadata,graph,caller,start,end,b"Arena",6,work);}
