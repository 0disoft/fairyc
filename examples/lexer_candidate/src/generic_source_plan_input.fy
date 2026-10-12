module generic_source_plan_input;
import generic_source_requests as requests;
import generic_source_dependencies as collector;
import generic_source_dependency_model as model;
import generic_source_catalog as catalog;
import generic_source_arguments as arguments;
import generic_concrete_types as concrete;
import generic_package_context as context;
import project_path as budget;
import syntax as syntax;
import std::vec as v;

// Only source-admitted parameter abilities are represented. No source
// capability providers are invented, and no body or specialization is admitted.
pub struct Argument{usize parameter;usize concrete;}
pub struct Problem{u32 phase;model::Problem source;}
pub resource Input{requests::Requests source;model::Data dependencies;concrete::Catalog types;v::Vec<usize> roots;v::Vec<Argument> arguments;cleanup{}}
pub fn ref catalog::Catalog templates(ref Input p){return requests::templates(ref p.source);}
pub fn ref model::Data roots(ref Input p){return requests::data(ref p.source);}
pub fn ref model::Data dependencies(ref Input p){return ref p.dependencies;}
pub fn ref concrete::Catalog types(ref Input p){return ref p.types;}
pub fn usize root_argument_count(ref Input p){return v::len<usize>(ref p.roots);}
pub fn usize root_argument(ref Input p,usize at){assert(at<root_argument_count(p));return *v::at<usize>(ref p.roots,at);}
pub fn usize argument_count(ref Input p){return v::len<Argument>(ref p.arguments);}
pub fn Argument argument_at(ref Input p,usize at){assert(at<argument_count(p));return *v::at<Argument>(ref p.arguments,at);}
fn Problem fail(u32 phase,u32 code,usize module_index,usize declaration,usize start,usize end,ref mut context::Budget b){if(b.code==0){b.code=code;}return Problem{.phase,.source=model::Problem{.code=b.code,.module_index,.declaration,.start,.end}};}
fn Result<v::Vec<usize>,u32> root_arguments(ref catalog::Catalog source,ref model::Data roots,ref concrete::Catalog types,ref mut usize work){
    if(model::range_count(roots)!=1 || model::range_at(roots,0).first!=0 || model::range_at(roots,0).count!=model::edge_count(roots)){return .err(9);}var v::Vec<usize> out=v::empty<usize>();var usize i=0;
    while(i<model::argument_count(roots)){if(!budget::charge(work,1)){return .err(10);}let a=model::argument_at(roots,i);if(a.parameter || !model::valid_text(roots,a.name)){return .err(9);}let selected=switch(concrete::lookup_id(types,model::text(roots,a.name),work)){case .ok(value):{value}case .err(code):{return .err(code);}};if(!v::push<usize>(ref mut out,selected).is_ok){return .err(11);}i+=1;}
    i=0;while(i<model::edge_count(roots)){if(!budget::charge(work,1)){return .err(10);}let e=model::edge_at(roots,i);if(e.target==0 || e.target>catalog::count(source) || e.first>v::len<usize>(ref out) || e.count>v::len<usize>(ref out)-e.first || e.count!=catalog::template_at(source,e.target-1).count){return .err(9);}i+=1;}return .ok(out);
}
fn model::Problem at_template(ref catalog::Catalog source,usize at,u32 code){let row=catalog::template_at(source,at);return model::Problem{.code,.module_index=row.module_index,.declaration=row.declaration,.start=row.start,.end=row.end};}
fn Result<v::Vec<Argument>,model::Problem> dependency_arguments(ref catalog::Catalog source,ref model::Data graph,ref concrete::Catalog types,ref mut usize work){
    if(model::range_count(graph)!=catalog::count(source)){return .err(model::Problem{.code=9,.module_index=0,.declaration=0,.start=0,.end=0});}var v::Vec<Argument> out=v::empty<Argument>();var usize i=0;
    while(i<model::argument_count(graph)){if(!budget::charge(work,1)){return .err(model::Problem{.code=10,.module_index=0,.declaration=0,.start=0,.end=0});}if(!v::push<Argument>(ref mut out,Argument{.parameter=0,.concrete=0}).is_ok){return .err(model::Problem{.code=11,.module_index=0,.declaration=0,.start=0,.end=0});}i+=1;}
    // Each parameter belongs to the originating declaration, not its target.
    // Bind directly to original argument slots; sorted edges need no owner scan.
    i=0;while(i<catalog::count(source)){if(!budget::charge(work,1)){return .err(at_template(source,i,10));}let range=model::range_at(graph,i);if(range.first>model::edge_count(graph) || range.count>model::edge_count(graph)-range.first){return .err(at_template(source,i,9));}var usize j=0;
        while(j<range.count){if(!budget::charge(work,1)){return .err(at_template(source,i,10));}let e=model::edge_at(graph,range.first+j);if(e.target==0 || e.target>catalog::count(source) || e.first>model::argument_count(graph) || e.count>model::argument_count(graph)-e.first || e.count!=catalog::template_at(source,e.target-1).count){return .err(at_template(source,i,9));}var usize k=0;
            while(k<e.count){if(!budget::charge(work,1)){return .err(at_template(source,i,10));}let a=model::argument_at(graph,e.first+k);let old=*v::at<Argument>(ref out,e.first+k);if(!model::valid_text(graph,a.name) || old.parameter!=0 || old.concrete!=0){return .err(at_template(source,i,9));}var usize parameter=0;var usize selected=0;
                if(a.parameter){parameter=switch(arguments::parameter(source,i,model::text(graph,a.name),work)){case .ok(value):{value}case .err(code):{return .err(at_template(source,i,code));}};if(parameter==0){return .err(at_template(source,i,9));}}
                else{selected=switch(concrete::lookup_id(types,model::text(graph,a.name),work)){case .ok(value):{value}case .err(code):{return .err(at_template(source,i,code));}};}
                let slot=v::at_mut<Argument>(ref mut out,e.first+k);*slot=Argument{.parameter,.concrete=selected};k+=1;
            }j+=1;
        }i+=1;
    }i=0;while(i<v::len<Argument>(ref out)){if(!budget::charge(work,1)){return .err(model::Problem{.code=10,.module_index=0,.declaration=0,.start=0,.end=0});}let a=*v::at<Argument>(ref out,i);if((a.parameter==0)==(a.concrete==0)){return .err(model::Problem{.code=9,.module_index=0,.declaration=0,.start=0,.end=0});}i+=1;}return .ok(out);
}
pub fn Result<Input,Problem> build(requests::Requests source,ref v::Vec<syntax::Parsed> sources,ref mut context::Budget b){
    if(b.code!=0){return .err(fail(0,b.code,0,0,0,0,b));}let catalog=requests::templates(ref source);
    let graph=switch(collector::collect(catalog,sources,ref mut b.work)){case .ok(value):{value}case .err(p):{return .err(fail(1,p.code,p.module_index,p.declaration,p.start,p.end,b));}};
    let types=switch(concrete::build(catalog,sources,b)){case .ok(value):{value}case .err(code):{return .err(fail(2,code,0,0,0,0,b));}};
    let roots=switch(root_arguments(catalog,requests::data(ref source),ref types,ref mut b.work)){case .ok(value):{value}case .err(code):{return .err(fail(3,code,0,0,0,0,b));}};
    let bound=switch(dependency_arguments(catalog,ref graph,ref types,ref mut b.work)){case .ok(value):{value}case .err(p):{return .err(fail(4,p.code,p.module_index,p.declaration,p.start,p.end,b));}};
    return .ok(Input{.source,.dependencies=graph,.types,.roots,.arguments=bound});
}
