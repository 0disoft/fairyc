module generic_source_requests;
import generic_source_dependency_model as model;
import generic_source_dependency_walk as walk;
import generic_source_catalog as catalog;
import generic_package_bindings as bindings;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Detached direct requests precede dependency expansion and executable Flow.
// Generic bodies contribute dependencies, never direct roots. Ordinary function
// signatures, aggregate fields and cleanup match the original Go source adapter.
pub resource Requests{catalog::Catalog templates;model::Data data;cleanup{}}
pub fn ref catalog::Catalog templates(ref Requests p){return ref p.templates;}
pub fn ref model::Data data(ref Requests p){return ref p.data;}
pub fn Result<Requests,model::Problem> build(catalog::Catalog templates,ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    let resolved=catalog::packages(ref templates);if(bindings::count(resolved)!=v::len<syntax::Parsed>(sources)){return .err(model::Problem{.code=9,.module_index=0,.declaration=0,.start=0,.end=0});}var usize m=0;
    while(m<v::len<syntax::Parsed>(sources)){let code=catalog::current(resolved,m,v::at<syntax::Parsed>(sources,m),work);if(code!=0){return .err(model::Problem{.code,.module_index=m,.declaration=0,.start=0,.end=0});}m+=1;}
    var bytes::Buffer names=bytes::empty();var v::Vec<model::Argument> arguments=v::empty<model::Argument>();var v::Vec<model::Edge> edges=v::empty<model::Edge>();var v::Vec<model::Range> ranges=v::empty<model::Range>();m=0;
    while(m<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,m);let owned_module=bindings::module_at(resolved,m);let vector=bytes::equal(bindings::text(resolved,owned_module.path),b"std::vec");var usize d=1;
        while(d<syntax::declaration_count(p)){if(!budget::charge(work,1)){return .err(model::Problem{.code=10,.module_index=m,.declaration=d,.start=0,.end=0});}let row=syntax::declaration_at(p,d);if(row.generic_parameters==0 && (row.kind==3 || row.kind==4 || row.kind==6)){let problem=walk::collect_root(ref templates,sources,m,d,vector,ref mut names,ref mut arguments,ref mut edges,ref mut ranges,work);if(problem.code!=0){return .err(problem);}}d+=1;
        }m+=1;
    }let code=model::finish(ref mut edges,ref names,ref mut ranges,0,work);if(code!=0){return .err(model::Problem{.code,.module_index=0,.declaration=0,.start=0,.end=0});}return .ok(Requests{.templates,.data=model::assemble_data(names,arguments,edges,ranges)});
}
