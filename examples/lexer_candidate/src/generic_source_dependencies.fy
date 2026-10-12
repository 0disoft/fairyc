module generic_source_dependencies;
import generic_source_dependency_model as model;
import generic_source_dependency_walk as walk;
import generic_source_arguments as arguments;
import generic_source_catalog as catalog;
import generic_package_bindings as bindings;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Source catalog facts become a detached direct-dependency graph, not an
// executable specialization plan. Unused recursive templates remain finite.
pub fn Result<model::Graph,model::Problem> build(catalog::Catalog templates,ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    let resolved=catalog::packages(ref templates);if(bindings::count(resolved)!=v::len<syntax::Parsed>(sources)){return .err(model::Problem{.code=9,.module_index=0,.declaration=0,.start=0,.end=0});}var usize i=0;
    while(i<v::len<syntax::Parsed>(sources)){let code=catalog::current(resolved,i,v::at<syntax::Parsed>(sources,i),work);if(code!=0){return .err(model::Problem{.code,.module_index=i,.declaration=0,.start=0,.end=0});}i+=1;}
    var bytes::Buffer names=bytes::empty();var v::Vec<model::Argument> args=v::empty<model::Argument>();var v::Vec<model::Edge> edges=v::empty<model::Edge>();var v::Vec<model::Range> ranges=v::empty<model::Range>();i=0;while(i<catalog::count(ref templates)){let row=catalog::template_at(ref templates,i);if(!budget::charge(work,row.count+1)){return .err(model::Problem{.code=10,.module_index=row.module_index,.declaration=row.declaration,.start=row.start,.end=row.end});}var usize j=0;
        while(j<row.count){let parameter=catalog::parameter_at(ref templates,row.first+j);let name=catalog::text(ref templates,parameter.name);let nominal=arguments::nominal(ref templates,sources,row.module_index,name,work);if(switch(nominal){case .ok(_):{true}case .err(code):{if(code==10 || code==11){return .err(model::Problem{.code,.module_index=row.module_index,.declaration=row.declaration,.start=row.start,.end=row.end});}false}}){return .err(model::Problem{.code=4501,.module_index=row.module_index,.declaration=row.declaration,.start=row.start,.end=row.end});}j+=1;}
        let problem=walk::collect(ref templates,sources,i,ref mut names,ref mut args,ref mut edges,ref mut ranges,work);if(problem.code!=0){return .err(problem);}i+=1;
    }return .ok(model::assemble(templates,model::assemble_data(names,args,edges,ranges)));
}
