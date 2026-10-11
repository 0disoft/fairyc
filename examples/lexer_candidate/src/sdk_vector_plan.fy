module sdk_vector_plan;
import module_catalog as catalog;
import module_types as types;
import sdk_authority as sdk;
import syntax as syntax;
import std::vec as v;

pub struct Instance {usize type;usize element;usize module_index;usize declaration;}
pub resource Plan {catalog::Catalog declarations;types::Graph graph;v::Vec<Instance> instances;cleanup{}}
// Source-wide nominal descriptors are separate from per-unit executable
// generic specialization and layout admission budgets.
pub fn usize limit(){return 256;}

pub fn u32 validate(ref catalog::Catalog c,ref types::Graph graph,ref v::Vec<Instance> instances){
    let u32 invalid=types::validate_graph(c,graph);if(invalid!=0){return invalid;}
    if(v::len<Instance>(instances)>limit()){return 4503;}
    var usize slot=0;var usize id=1;
    while(id<=types::graph_count(graph)){let t=types::graph_at(graph,id);
        if(t.kind==13){
            if(slot>=v::len<Instance>(instances)){return 9;}let item=*v::at<Instance>(instances,slot);
            if(item.type!=id || item.element!=t.first || item.module_index!=t.module_index || item.declaration!=t.declaration){return 9;}
            var usize previous=0;while(previous<slot){let old=*v::at<Instance>(instances,previous);
                if(old.module_index==item.module_index && old.declaration==item.declaration && old.element==item.element){return 9;}previous+=1;
            }slot+=1;
        }id+=1;
    }
    return if(slot==v::len<Instance>(instances)){0}else{9};
}
pub fn Result<Plan,u32> assemble(catalog::Catalog c,types::Graph graph,v::Vec<Instance> instances){
    let u32 code=validate(ref c,ref graph,ref instances);if(code!=0){return .err(code);}
    return .ok(Plan{.declarations=c,.graph,.instances});
}
pub fn Result<Plan,u32> build(catalog::Catalog c,types::Graph graph){
    var v::Vec<Instance> instances=v::empty<Instance>();var usize id=1;
    while(id<=types::graph_count(ref graph)){let t=types::graph_at(ref graph,id);if(t.kind==13){
        if(v::len<Instance>(ref instances)>=limit()){return .err(4503);}
        if(!v::push<Instance>(ref mut instances,Instance{.type=id,.element=t.first,.module_index=t.module_index,.declaration=t.declaration}).is_ok){return .err(11);}
    }id+=1;}
    return assemble(c,graph,instances);
}
pub fn usize count(ref Plan p){return v::len<Instance>(ref p.instances);}
pub fn Instance at(ref Plan p,usize index){assert(index<count(p));return *v::at<Instance>(ref p.instances,index);}
pub fn usize find(ref Plan p,usize type){var usize i=0;while(i<count(p)){if(at(p,i).type==type){return i+1;}i+=1;}return 0;}
pub fn types::Type type_at(ref Plan p,usize id){return types::graph_at(ref p.graph,id);}
pub fn usize type_count(ref Plan p){return types::graph_count(ref p.graph);}
pub fn types::Resolution resolve_type(ref Plan p,ref syntax::Parsed source,usize module_index,usize root,ref mut v::Vec<types::Type> graph){return types::resolve(ref p.declarations,source,module_index,root,graph);}
pub fn types::Resolution declaration_type(ref Plan p,usize module_index,usize declaration,ref mut v::Vec<types::Type> graph){return types::declaration(ref p.declarations,module_index,declaration,graph);}
pub fn Result<types::Graph,u32> own_types(ref Plan p,v::Vec<types::Type> graph){return types::assemble(ref p.declarations,graph);}
pub fn usize module_count(ref Plan p){return catalog::module_count(ref p.declarations);}
pub fn usize declaration_count(ref Plan p){return catalog::declaration_count(ref p.declarations);}
pub fn bool source_matches(ref Plan p,usize module_index,ref syntax::Parsed source){return catalog::source_matches(ref p.declarations,module_index,source);}
pub fn u32[8] source_digest(ref Plan p,usize module_index){return catalog::source_digest(ref p.declarations,module_index);}
pub fn catalog::Lookup declaration_lookup(ref Plan p,usize module_index,ref u8[] spelling){return catalog::lookup(ref p.declarations,module_index,spelling);}
pub fn catalog::Lookup declaration_lookup_measured(ref Plan p,usize module_index,ref u8[] spelling,ref mut usize probes){return catalog::lookup_measured(ref p.declarations,module_index,spelling,probes);}
pub fn catalog::Declaration declaration_at(ref Plan p,usize id){return catalog::declaration_at(ref p.declarations,id);}

pub fn usize field_count(){return 6;}
pub fn u32 field_type(usize index){return sdk::vector_field(index);}
pub fn usize field_offset(usize index){assert(index<field_count());return index*8;}
pub fn usize storage_size(){return 48;}
pub fn usize storage_alignment(){return 8;}

pub fn types::Resolution specialize_type(ref Plan p,ref syntax::Parsed source,usize module_index,usize declaration,usize root,ref v::Vec<usize> args,ref mut v::Vec<types::Type> graph){return types::specialize(ref p.declarations,source,module_index,declaration,root,args,graph);}
