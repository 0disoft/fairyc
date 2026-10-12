module module_unit_interfaces;
import module_unit_interface_work as meter;
import module_unit_interface_annotations as annotation_builder;
import module_unit_interface_declarations as declarations;
import module_unit_bind as binding;
import module_signatures as signatures;
import module_types as types;
import module_unit_generics as generics;
import module_unit_body_types as body_types;
import syntax as syntax;
import module_ast_join as join;
import std::bytes as bytes;
import std::vec as v;
import module_unit_shape_contracts as contracts;
import module_unit_region_demands as regions;
import sdk_vector_plan as vector_plan;

pub struct Shape {usize type;usize module_index;usize declaration;usize field_start;usize field_count;u32 abilities;bool repr_c;usize start;usize end;bool cleanup_nonempty;u32 cleanup_effects;}
pub struct Field {u32 kind;usize type;usize module_index;usize declaration;usize source_index;usize start;usize end;usize name_start;usize name_end;usize payload_start;usize payload_count;}
pub struct Callable {usize module_index;usize declaration;bool template;usize result;usize parameter_start;usize parameter_count;u32 effects;bool must_use;bool unsafe_call;bool foreign;usize link_start;usize link_end;}
pub struct Annotation {usize type;bool required;bool deferred;}
pub resource Interfaces {usize region_error;usize region_arena;types::Graph graph;v::Vec<Shape> shapes;v::Vec<Field> fields;v::Vec<Callable> callables;v::Vec<usize> parameters;v::Vec<Annotation> annotations;generics::Plan specializations;v::Vec<body_types::Request> body_types;bytes::Buffer names;usize work;usize first;usize stop;cleanup{}}
fn join::Problem body_demands(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref generics::Plan specializations,u32 kind,usize id,ref mut v::Vec<types::Type> graph,ref mut v::Vec<body_types::Request> requests,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,ref mut meter::Budget work){
    var usize module_index=0;var usize declaration=0;var usize argument_start=0;var usize argument_count=0;
    if(kind==1){let c=generics::at(specializations,id);module_index=c.module_index;declaration=c.declaration;argument_start=c.argument_start;argument_count=c.argument_count;}
    else{let c=generics::cleanup_at(specializations,id);module_index=c.module_index;declaration=c.declaration;argument_start=c.argument_start;argument_count=c.argument_count;}
    var v::Vec<usize> arguments=v::empty<usize>();var usize i=0;while(i<argument_count){if(!v::push<usize>(ref mut arguments,generics::argument_at(specializations,argument_start+i)).is_ok){return meter::fail(11,module_index,0,0);}i+=1;}
    let demands=switch(body_types::collect(v::at<syntax::Parsed>(sources,module_index),metadata,module_index,declaration,kind,id,ref arguments,graph)){case .ok(p):{p}case .err(p):{return p;}};
    if(!meter::charge(work,body_types::work(ref demands))){return meter::fail(10,module_index,0,0);}i=0;while(i<body_types::count(ref demands)){let r=body_types::at(ref demands,i);if(v::len<body_types::Request>(requests)>=16384){return meter::fail(10,r.module_index,r.start,r.end);}let u32 code=meter::mark(graph,pending,seen,work,r.type);if(code!=0){return meter::fail(code,r.module_index,r.start,r.end);}if(!v::push<body_types::Request>(requests,r).is_ok){return meter::fail(11,r.module_index,r.start,r.end);}i+=1;}return meter::fail(0,0,0,0);
}
fn Result<Interfaces,join::Problem> build_mode(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref v::Vec<binding::Coordinate> seeds,bool retain_imports){
    if(v::len<syntax::Parsed>(sources)!=signatures::module_count(metadata)){return .err(meter::fail(12,0,0,0));}var usize i=0;while(i<v::len<syntax::Parsed>(sources)){if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return .err(meter::fail(12,i,0,0));}i+=1;}
    let binding_valid=binding::validate(bound,sources,metadata);if(binding_valid.code!=0){return .err(binding_valid);}
    var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<Shape> shapes=v::empty<Shape>();var v::Vec<Field> fields=v::empty<Field>();var v::Vec<Callable> callables=v::empty<Callable>();var v::Vec<usize> parameters=v::empty<usize>();var bytes::Buffer names=bytes::empty();var v::Vec<usize> pending=v::empty<usize>();var bytes::Buffer seen=bytes::empty();var meter::Budget budget=meter::Budget{.work=0};
    i=1;while(i<=signatures::type_count(metadata)){if(!meter::charge(ref mut budget,1)){return .err(meter::fail(10,0,0,0));}if(!v::push<types::Type>(ref mut graph,signatures::type_at(metadata,i)).is_ok){return .err(meter::fail(11,0,0,0));}i+=1;}
    var usize selected=binding::first(bound);while(selected<binding::stop(bound)){let source=v::at<syntax::Parsed>(sources,selected);var usize d=0;while(d<syntax::declaration_count(source)){let checked=contracts::declaration(source,selected,d,signatures::sdk_vector_source(metadata,selected),ref mut budget.work);if(checked.code!=0){return .err(checked);}let item=syntax::declaration_at(source,d);if((item.kind==4 || item.kind==6 || item.kind==11) && item.generic_parameters==0){let r=signatures::declaration_type(metadata,selected,d+1,ref mut graph);if(r.problem.code!=0){return .err(meter::fail(r.problem.code,selected,r.problem.start,r.problem.end));}let code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,r.type);if(code!=0){return .err(meter::fail(code,selected,item.start,item.end));}}d+=1;}selected+=1;}
    let roots=switch(annotation_builder::annotations(sources,metadata,bound,ref mut graph,ref mut budget)){case .ok(t):{t}case .err(p):{return .err(p);}};
    i=0;while(i<v::len<Annotation>(ref roots)){let a=*v::at<Annotation>(ref roots,i);if(a.type!=0){let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,a.type);if(code!=0){return .err(meter::fail(code,binding::first(bound),0,0));}}i+=1;}
    var v::Vec<body_types::Request> demanded=v::empty<body_types::Request>();var usize source_index=binding::first(bound);while(source_index<binding::stop(bound)){let records=switch(body_types::ordinary_records(v::at<syntax::Parsed>(sources,source_index),metadata,source_index,ref mut graph)){case .ok(p):{p}case .err(p):{return .err(p);}};if(!meter::charge(ref mut budget,body_types::work(ref records))){return .err(meter::fail(10,source_index,0,0));}var usize r=0;while(r<body_types::count(ref records)){let item=body_types::at(ref records,r);if(v::len<body_types::Request>(ref demanded)>=16384){return .err(meter::fail(10,item.module_index,item.start,item.end));}let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,item.type);if(code!=0){return .err(meter::fail(code,item.module_index,item.start,item.end));}if(!v::push<body_types::Request>(ref mut demanded,item).is_ok){return .err(meter::fail(11,item.module_index,item.start,item.end));}r+=1;}source_index+=1;}
    var usize region_error=0;var usize region_arena=0;
    var generics::Plan specializations=switch(generics::build(sources,metadata,bound,ref mut graph)){case .ok(p):{p}case .err(p):{return .err(p);}};
    if(retain_imports){specializations=switch(generics::extend_imports(sources,metadata,bound,seeds,specializations,ref mut graph)){case .ok(p):{p}case .err(p):{return .err(p);}};}
    if(!meter::charge(ref mut budget,generics::work(ref specializations))){return .err(meter::fail(4503,binding::first(bound),0,0));}
    i=1;while(i<=binding::target_count(bound)){let c=binding::target_at(bound,i);if(c.kind==3 || c.kind==9){let p=declarations::callable(sources,metadata,c,ref mut graph,ref mut callables,ref mut parameters,ref mut names,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}}
        else if(c.kind==4 || c.kind==6 || c.kind==11){let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,c.module_index),c.declaration-1);if(d.generic_parameters==0){let r=signatures::declaration_type(metadata,c.module_index,c.declaration,ref mut graph);if(r.problem.code!=0){return .err(meter::fail(r.problem.code,c.module_index,r.problem.start,r.problem.end));}let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,r.type);if(code!=0){return .err(meter::fail(code,c.module_index,c.start,c.end));}}}i+=1;
    }
    i=0;while(i<v::len<binding::Coordinate>(seeds)){let c=*v::at<binding::Coordinate>(seeds,i);if(c.module_index>=v::len<syntax::Parsed>(sources) || c.declaration==0 || c.declaration>syntax::declaration_count(v::at<syntax::Parsed>(sources,c.module_index))){return .err(meter::fail(12,c.module_index,c.start,c.end));}let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,c.module_index),c.declaration-1);if(d.kind!=c.kind || d.start!=c.start || d.end!=c.end || (d.kind!=3 && d.kind!=9) || d.generic_parameters!=0){return .err(meter::fail(12,c.module_index,c.start,c.end));}let p=declarations::callable(sources,metadata,c,ref mut graph,ref mut callables,ref mut parameters,ref mut names,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}i+=1;}
    var usize vectors=0;var usize cursor=0;var usize instance_cursor=0;var usize dependency_cursor=0;var usize cleanup_cursor=0;var usize cleanup_body_cursor=0;var bool regions_scanned=false;var usize region_instances=0;var usize region_cleanups=0;
    while(true){
        if(!regions_scanned || region_instances!=generics::count(ref specializations) || region_cleanups!=generics::cleanup_count(ref specializations)){let requested=switch(regions::collect(sources,metadata,binding::first(bound),binding::stop(bound),ref specializations,ref mut graph,ref mut pending,ref mut seen,ref mut budget)){case .ok(value):{value}case .err(p):{return .err(p);}};if(requested.error_type!=0){region_error=requested.error_type;}if(requested.arena_type!=0){region_arena=requested.arena_type;}regions_scanned=true;region_instances=generics::count(ref specializations);region_cleanups=generics::cleanup_count(ref specializations);}
        while(instance_cursor<generics::count(ref specializations)){let s=generics::at(ref specializations,instance_cursor+1);let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,s.result);if(code!=0){return .err(meter::fail(code,s.module_index,0,0));}var usize j=0;while(j<s.parameter_count){let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,generics::parameter_at(ref specializations,s.parameter_start+j));if(code!=0){return .err(meter::fail(code,s.module_index,0,0));}j+=1;}
            let p=body_demands(sources,metadata,ref specializations,1,instance_cursor+1,ref mut graph,ref mut demanded,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}instance_cursor+=1;
        }
        while(cleanup_body_cursor<generics::cleanup_count(ref specializations)){let p=body_demands(sources,metadata,ref specializations,2,cleanup_body_cursor+1,ref mut graph,ref mut demanded,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}cleanup_body_cursor+=1;}
        while(dependency_cursor<generics::dependency_count(ref specializations)){let d=generics::dependency_at(ref specializations,dependency_cursor);let usize type=if(d.aggregate_type!=0){d.aggregate_type}else{d.intrinsic_type};if(type!=0){let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,type);if(code!=0){return .err(meter::fail(code,0,0,0));}}dependency_cursor+=1;}
        while(cleanup_cursor<generics::cleanup_dependency_count(ref specializations)){let d=generics::cleanup_dependency_at(ref specializations,cleanup_cursor);let usize type=if(d.aggregate_type!=0){d.aggregate_type}else{d.intrinsic_type};if(type!=0){let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,type);if(code!=0){return .err(meter::fail(code,0,0,0));}}cleanup_cursor+=1;}
        var v::Vec<usize> cleanup_seeds=v::empty<usize>();while(cursor<v::len<usize>(ref pending)){let usize id=*v::at<usize>(ref pending,cursor);let t=types::at(ref graph,id);if(!meter::charge(ref mut budget,1)){return .err(meter::fail(10,t.module_index,0,0));}if(t.kind==13){if(vectors>=vector_plan::limit()){return .err(meter::fail(4503,t.module_index,0,0));}vectors+=1;}
            if(t.kind==2 || t.kind==3 || t.kind==4 || t.kind==13){let p=declarations::shape(sources,metadata,id,ref mut graph,ref mut shapes,ref mut fields,ref mut names,ref mut pending,ref mut seen,binding::first(bound),binding::stop(bound),ref mut budget);if(p.code!=0){return .err(p);}}
            if(t.kind==3 || t.kind==13){if(!v::push<usize>(ref mut cleanup_seeds,id).is_ok){return .err(meter::fail(11,t.module_index,0,0));}}
            if(t.kind>=5 && t.kind<=8 || t.kind==13 || t.kind==16 || t.kind==17 || t.kind==18 || t.kind==12 && t.first!=0){let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,t.first);if(code!=0){return .err(meter::fail(code,0,0,0));}if(t.kind==6){let u32 code=meter::mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,t.second);if(code!=0){return .err(meter::fail(code,0,0,0));}}}cursor+=1;
        }
        if(v::len<usize>(ref cleanup_seeds)==0){break;}let usize old_work=generics::work(ref specializations);specializations=switch(generics::extend_cleanups(sources,metadata,specializations,ref mut graph,ref cleanup_seeds)){case .ok(p):{p}case .err(p):{return .err(p);}};if(!meter::charge(ref mut budget,generics::work(ref specializations)-old_work)){return .err(meter::fail(4503,binding::first(bound),0,0));}
    }
    let fields_valid=contracts::fields(sources,ref graph,ref shapes,ref fields,ref mut budget.work);if(fields_valid.code!=0){return .err(fields_valid);}
    let owned=switch(signatures::own_types(metadata,graph)){case .ok(g):{g}case .err(code):{return .err(meter::fail(code,0,0,0));}};
    return .ok(Interfaces{.region_error,.region_arena,.graph=owned,.shapes,.fields,.callables,.parameters,.annotations=roots,.specializations,.body_types=demanded,.names,.work=budget.work,.first=binding::first(bound),.stop=binding::stop(bound)});
}
pub fn usize type_count(ref Interfaces p){return types::graph_count(ref p.graph);}
pub fn types::Type type_at(ref Interfaces p,usize id){return types::graph_at(ref p.graph,id);}
pub fn usize shape_count(ref Interfaces p){return v::len<Shape>(ref p.shapes);}
pub fn Shape shape_at(ref Interfaces p,usize index){return *v::at<Shape>(ref p.shapes,index);}
pub fn usize field_count(ref Interfaces p){return v::len<Field>(ref p.fields);}
pub fn Field field_at(ref Interfaces p,usize index){return *v::at<Field>(ref p.fields,index);}
pub fn ref u8[] field_name(ref Interfaces p,usize index){let f=field_at(p,index);return bytes::view(ref p.names)[f.name_start..f.name_end];}
pub fn usize callable_count(ref Interfaces p){return v::len<Callable>(ref p.callables);}
pub fn Callable callable_at(ref Interfaces p,usize index){return *v::at<Callable>(ref p.callables,index);}
pub fn usize parameter_at(ref Interfaces p,usize index){return *v::at<usize>(ref p.parameters,index);}
pub fn ref u8[] linker_name(ref Interfaces p,usize index){let c=callable_at(p,index);return bytes::view(ref p.names)[c.link_start..c.link_end];}
pub fn usize work(ref Interfaces p){return p.work;}
pub fn usize annotation_count(ref Interfaces p){return v::len<Annotation>(ref p.annotations);}
pub fn Annotation annotation_at(ref Interfaces p,usize root){assert(root>0 && root<=annotation_count(p));return *v::at<Annotation>(ref p.annotations,root-1);}
pub fn ref generics::Plan specializations(ref Interfaces p){return ref p.specializations;}
pub fn Result<usize,u32> field_measured(ref Interfaces p,usize type,usize caller,ref u8[] name,ref mut usize probes){
    if(type==0 || type>type_count(p)){return .err(18);}var usize owner=type;let t=type_at(p,owner);if(t.kind==7 || t.kind==8){owner=t.first;}
    var usize i=0;while(i<shape_count(p)){*probes=*probes+1;let s=shape_at(p,i);if(s.type==owner){let inner=type_at(p,owner);if(inner.kind!=2 && inner.kind!=3 && inner.kind!=13){return .err(18);}if((inner.kind==3 || inner.kind==13) && caller!=s.module_index){return .err(6201);}var usize f=0;while(f<s.field_count){*probes=*probes+name.len+1;if(bytes::equal(field_name(p,s.field_start+f),name)){return .ok(s.field_start+f+1);}f+=1;}return .err(18);}i+=1;}return .err(18);
}
pub fn Result<usize,u32> field(ref Interfaces p,usize type,usize caller,ref u8[] name){var usize probes=0;return field_measured(p,type,caller,name,ref mut probes);}
pub fn usize body_type_count(ref Interfaces p){return v::len<body_types::Request>(ref p.body_types);}
pub fn body_types::Request body_type_at(ref Interfaces p,usize index){return *v::at<body_types::Request>(ref p.body_types,index);}

pub fn usize first(ref Interfaces p){return p.first;}
pub fn usize stop(ref Interfaces p){return p.stop;}

pub fn Result<Interfaces,join::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound){let seeds=v::empty<binding::Coordinate>();return build_seeded(sources,metadata,bound,ref seeds);}

pub fn Result<Interfaces,join::Problem> build_seeded(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref v::Vec<binding::Coordinate> seeds){return build_mode(sources,metadata,bound,seeds,true);}
// A project unit checks external signatures without copying their ordinary
// bodies into this unit. Project linking must validate and supply those bodies.
pub fn Result<Interfaces,join::Problem> build_unit_seeded(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref v::Vec<binding::Coordinate> seeds){return build_mode(sources,metadata,bound,seeds,false);}

pub fn usize region_error(ref Interfaces p){return p.region_error;}
pub fn usize region_arena(ref Interfaces p){return p.region_arena;}
