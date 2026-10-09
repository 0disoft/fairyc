module module_unit_interfaces;
import module_unit_bind as binding;
import module_signatures as signatures;
import module_types as types;
import module_unit_generics as generics;
import module_unit_body_types as body_types;
import syntax as syntax;
import module_ast_join as join;
import resource_abilities as abilities;
import repr_c_declarations as repr;
import effect_declarations as effects;
import std::bytes as bytes;
import std::vec as v;

pub struct Shape {usize type;usize module_index;usize declaration;usize field_start;usize field_count;u32 abilities;bool repr_c;usize start;usize end;bool cleanup_nonempty;u32 cleanup_effects;}
pub struct Field {u32 kind;usize type;usize module_index;usize declaration;usize source_index;usize start;usize end;usize name_start;usize name_end;usize payload_start;usize payload_count;}
pub struct Callable {usize module_index;usize declaration;bool template;usize result;usize parameter_start;usize parameter_count;u32 effects;bool must_use;bool unsafe_call;bool foreign;usize link_start;usize link_end;}
pub struct Annotation {usize type;bool required;bool deferred;}
pub resource Interfaces {types::Graph graph;v::Vec<Shape> shapes;v::Vec<Field> fields;v::Vec<Callable> callables;v::Vec<usize> parameters;v::Vec<Annotation> annotations;generics::Plan specializations;v::Vec<body_types::Request> body_types;bytes::Buffer names;usize work;usize first;usize stop;cleanup{}}
struct Budget {usize work;}
fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut Budget work,usize amount){if(work.work>1048576 || amount>1048576-work.work){return false;}work.work+=amount;return true;}
fn u32 mark(ref mut v::Vec<types::Type> graph,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,ref mut Budget work,usize id){
    if(id==0 || id>types::count(graph)){return 9;}if(!charge(work,1)){return 10;}
    if(bytes::len(seen)<=types::count(graph) && !bytes::resize_zeroed(seen,types::count(graph)+1).is_ok){return 11;}
    if(bytes::view(seen)[id]!=0){return 0;}bytes::view_mut(seen)[id]=1;return if(v::push<usize>(pending,id).is_ok){0}else{11};
}
fn Result<usize,u32> append(ref mut bytes::Buffer names,ref mut Budget work,ref u8[] text){if(!charge(work,text.len)){return .err(10);}let usize start=bytes::len(names);if(text.len>65536-start){return .err(10);}return if(bytes::append(names,text).is_ok){.ok(start)}else{.err(11)};}
fn void request(ref mut v::Vec<Annotation> table,usize root){if(root!=0){let ref mut Annotation slot=v::at_mut<Annotation>(table,root-1);slot.required=true;}}
fn Result<v::Vec<Annotation>,join::Problem> annotations(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref mut v::Vec<types::Type> graph,ref mut Budget work){
    let p=binding::parsed(bound);var v::Vec<Annotation> table=v::empty<Annotation>();var usize i=0;
    while(i<syntax::type_count(p)){if(!charge(work,1)){return .err(fail(10,binding::first(bound),0,0));}if(!v::push<Annotation>(ref mut table,Annotation{.type=0,.required=false,.deferred=false}).is_ok){return .err(fail(11,binding::first(bound),0,0));}i+=1;}
    i=0;while(i<syntax::declaration_count(p)){request(ref mut table,syntax::declaration_at(p,i).result_type);i+=1;}
    i=0;while(i<syntax::entry_count(p)){request(ref mut table,syntax::entry_at(p,i).type_root);i+=1;}
    i=0;while(i<syntax::field_count(p)){request(ref mut table,syntax::field_at(p,i).type_root);i+=1;}
    i=1;while(i<=syntax::expression_count(p)){if(!charge(work,1)){return .err(fail(10,binding::first(bound),0,0));}let e=syntax::expression_at(p,i);if(e.kind==23){request(ref mut table,e.right);}else if(e.kind==20){var usize link=e.right;while(link!=0){let n=syntax::type_at(p,link);request(ref mut table,n.left);link=n.right;}}i+=1;}
    i=1;while(i<=syntax::statement_count(p)){let s=syntax::statement_at(p,i);if(s.kind==18){request(ref mut table,s.second);}i+=1;}
    var usize base=0;var usize module_index=binding::first(bound);
    while(module_index<binding::stop(bound)){let original=v::at<syntax::Parsed>(sources,module_index);var usize owner=0;i=1;
        while(i<=syntax::type_count(original)){let n=syntax::type_at(original,i);while(owner+1<syntax::declaration_count(original) && n.start>=syntax::declaration_at(original,owner).end){owner+=1;}
            if(v::at<Annotation>(ref table,base+i-1).required){let d=syntax::declaration_at(original,owner);if(d.generic_parameters!=0){let ref mut Annotation slot=v::at_mut<Annotation>(ref mut table,base+i-1);slot.deferred=true;}
                else{let r=signatures::resolve_type(metadata,original,module_index,i,graph);if(r.problem.code!=0){return .err(fail(r.problem.code,module_index,r.problem.start,r.problem.end));}let ref mut Annotation slot=v::at_mut<Annotation>(ref mut table,base+i-1);slot.type=r.type;}}
            i+=1;
        }base+=syntax::type_count(original);module_index+=1;
    }return .ok(table);
}
fn join::Problem callable(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,binding::Coordinate c,ref mut v::Vec<types::Type> graph,ref mut v::Vec<Callable> callables,ref mut v::Vec<usize> parameters,ref mut bytes::Buffer names,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,ref mut Budget work){
    if(!charge(work,1)){return fail(10,c.module_index,c.start,c.end);}var usize i=0;while(i<v::len<Callable>(callables)){if(!charge(work,1)){return fail(10,c.module_index,c.start,c.end);}let old=*v::at<Callable>(callables,i);if(old.module_index==c.module_index && old.declaration==c.declaration){return fail(0,0,0,0);}i+=1;}
    i=0;
    let source=v::at<syntax::Parsed>(sources,c.module_index);let d=syntax::declaration_at(source,c.declaration-1);let usize id=signatures::identity(metadata,c.module_index,c.declaration);
    let usize start=v::len<usize>(parameters);var usize result=0;var u32 required=0;var bool must_use=false;var bool unsafe_call=false;var bool foreign=false;var usize link_start=0;var usize link_end=0;
    if(d.generic_parameters!=0){let contract=effects::read(source,c.declaration-1);if(contract.code!=0){return fail(contract.code,c.module_index,contract.start,contract.end);}required=contract.bits;must_use=contract.must_use;unsafe_call=d.unsafe_end!=0;}
    else{if(id==0){return fail(9,c.module_index,c.start,c.end);}let s=signatures::at(metadata,id);result=s.result;required=s.effects;must_use=s.must_use;unsafe_call=s.unsafe_call;foreign=s.foreign;
        let u32 code=mark(graph,pending,seen,work,result);if(code!=0){return fail(code,c.module_index,c.start,c.end);}i=0;while(i<s.parameter_count){if(!charge(work,1)){return fail(10,c.module_index,c.start,c.end);}let usize type=signatures::parameter_at(metadata,s.parameter_start+i);let u32 code=mark(graph,pending,seen,work,type);if(code!=0){return fail(code,c.module_index,c.start,c.end);}if(v::len<usize>(parameters)>=32768 || !v::push<usize>(parameters,type).is_ok){return fail(if(v::len<usize>(parameters)>=32768){10}else{11},c.module_index,c.start,c.end);}i+=1;}
        if(foreign){link_start=switch(append(names,work,signatures::linker_name(metadata,id))){case .ok(n):{n}case .err(code):{return fail(code,c.module_index,c.start,c.end);}};link_end=bytes::len(names);}
    }
    if(v::len<Callable>(callables)>=512){return fail(10,c.module_index,c.start,c.end);}if(!v::push<Callable>(callables,Callable{.module_index=c.module_index,.declaration=c.declaration,.template=d.generic_parameters!=0,.result,.parameter_start=start,.parameter_count=v::len<usize>(parameters)-start,.effects=required,.must_use,.unsafe_call,.foreign,.link_start,.link_end}).is_ok){return fail(11,c.module_index,c.start,c.end);}return fail(0,0,0,0);
}
fn join::Problem shape(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize id,ref mut v::Vec<types::Type> graph,ref mut v::Vec<Shape> shapes,ref mut v::Vec<Field> fields,ref mut bytes::Buffer names,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,ref mut Budget work){
    let t=types::at(graph,id);let source=v::at<syntax::Parsed>(sources,t.module_index);let d=syntax::declaration_at(source,t.declaration-1);let usize field_start=v::len<Field>(fields);
    if(v::len<Shape>(shapes)>=512 || d.field_count>1024-field_start){return fail(10,t.module_index,d.start,d.end);}var usize i=0;
    while(i<d.field_count){if(!charge(work,1)){return fail(10,t.module_index,d.start,d.end);}let f=syntax::field_at(source,d.field_start+i);var usize type=0;
        if(f.type_root!=0){let resolved=signatures::resolve_type(metadata,source,t.module_index,f.type_root,graph);if(resolved.problem.code!=0){return fail(resolved.problem.code,t.module_index,resolved.problem.start,resolved.problem.end);}type=resolved.type;let u32 code=mark(graph,pending,seen,work,type);if(code!=0){return fail(code,t.module_index,f.start,f.end);}}
        let usize name_start=switch(append(names,work,syntax::source_view(source)[f.name_start..f.name_end])){case .ok(n):{n}case .err(code):{return fail(code,t.module_index,f.start,f.end);}};
        if(!v::push<Field>(fields,Field{.kind=f.kind,.type,.module_index=t.module_index,.declaration=t.declaration,.source_index=d.field_start+i,.start=f.start,.end=f.end,.name_start,.name_end=bytes::len(names),.payload_start=if(f.payload_count==0){0}else{field_start+f.payload_start-d.field_start},.payload_count=f.payload_count}).is_ok){return fail(11,t.module_index,f.start,f.end);}i+=1;
    }
    let representation=repr::read(source,t.declaration-1);if(representation.code!=0){return fail(representation.code,t.module_index,representation.start,representation.end);}var u32 cleanup_effects=0;if(d.kind==6){let contract=effects::read(source,t.declaration-1);if(contract.code!=0){return fail(contract.code,t.module_index,contract.start,contract.end);}cleanup_effects=contract.bits;}
    let bool nonempty=d.kind==6 && (syntax::block_statement_count(source,d.cleanup_body)!=0 || syntax::statement_at(source,d.cleanup_body).expression!=0);
    if(!v::push<Shape>(shapes,Shape{.type=id,.module_index=t.module_index,.declaration=t.declaration,.field_start,.field_count=d.field_count,.abilities=if(d.kind==6){abilities::mask(source,t.declaration-1)}else{0},.repr_c=representation.repr_c,.start=d.start,.end=d.end,.cleanup_nonempty=nonempty,.cleanup_effects}).is_ok){return fail(11,t.module_index,d.start,d.end);}return fail(0,0,0,0);
}
fn join::Problem body_demands(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref generics::Plan specializations,u32 kind,usize id,ref mut v::Vec<types::Type> graph,ref mut v::Vec<body_types::Request> requests,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,ref mut Budget work){
    var usize module_index=0;var usize declaration=0;var usize argument_start=0;var usize argument_count=0;
    if(kind==1){let c=generics::at(specializations,id);module_index=c.module_index;declaration=c.declaration;argument_start=c.argument_start;argument_count=c.argument_count;}
    else{let c=generics::cleanup_at(specializations,id);module_index=c.module_index;declaration=c.declaration;argument_start=c.argument_start;argument_count=c.argument_count;}
    var v::Vec<usize> arguments=v::empty<usize>();var usize i=0;while(i<argument_count){if(!v::push<usize>(ref mut arguments,generics::argument_at(specializations,argument_start+i)).is_ok){return fail(11,module_index,0,0);}i+=1;}
    let demands=switch(body_types::collect(v::at<syntax::Parsed>(sources,module_index),metadata,module_index,declaration,kind,id,ref arguments,graph)){case .ok(p):{p}case .err(p):{return p;}};
    if(!charge(work,body_types::work(ref demands))){return fail(10,module_index,0,0);}i=0;while(i<body_types::count(ref demands)){let r=body_types::at(ref demands,i);if(v::len<body_types::Request>(requests)>=16384){return fail(10,r.module_index,r.start,r.end);}let u32 code=mark(graph,pending,seen,work,r.type);if(code!=0){return fail(code,r.module_index,r.start,r.end);}if(!v::push<body_types::Request>(requests,r).is_ok){return fail(11,r.module_index,r.start,r.end);}i+=1;}return fail(0,0,0,0);
}
pub fn Result<Interfaces,join::Problem> build_seeded(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref v::Vec<binding::Coordinate> seeds){
    if(v::len<syntax::Parsed>(sources)!=signatures::module_count(metadata)){return .err(fail(12,0,0,0));}var usize i=0;while(i<v::len<syntax::Parsed>(sources)){if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return .err(fail(12,i,0,0));}i+=1;}
    let binding_valid=binding::validate(bound,sources,metadata);if(binding_valid.code!=0){return .err(binding_valid);}
    var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<Shape> shapes=v::empty<Shape>();var v::Vec<Field> fields=v::empty<Field>();var v::Vec<Callable> callables=v::empty<Callable>();var v::Vec<usize> parameters=v::empty<usize>();var bytes::Buffer names=bytes::empty();var v::Vec<usize> pending=v::empty<usize>();var bytes::Buffer seen=bytes::empty();var Budget budget=Budget{.work=0};
    i=1;while(i<=signatures::type_count(metadata)){if(!charge(ref mut budget,1)){return .err(fail(10,0,0,0));}if(!v::push<types::Type>(ref mut graph,signatures::type_at(metadata,i)).is_ok){return .err(fail(11,0,0,0));}i+=1;}
    let roots=switch(annotations(sources,metadata,bound,ref mut graph,ref mut budget)){case .ok(t):{t}case .err(p):{return .err(p);}};
    i=0;while(i<v::len<Annotation>(ref roots)){let a=*v::at<Annotation>(ref roots,i);if(a.type!=0){let u32 code=mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,a.type);if(code!=0){return .err(fail(code,binding::first(bound),0,0));}}i+=1;}
    var generics::Plan specializations=switch(generics::build(sources,metadata,bound,ref mut graph)){case .ok(p):{p}case .err(p):{return .err(p);}};
    if(!charge(ref mut budget,generics::work(ref specializations))){return .err(fail(4503,binding::first(bound),0,0));}
    i=1;while(i<=binding::target_count(bound)){let c=binding::target_at(bound,i);if(c.kind==3 || c.kind==9){let p=callable(sources,metadata,c,ref mut graph,ref mut callables,ref mut parameters,ref mut names,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}}
        else if(c.kind==4 || c.kind==6 || c.kind==11){let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,c.module_index),c.declaration-1);if(d.generic_parameters==0){let r=signatures::declaration_type(metadata,c.module_index,c.declaration,ref mut graph);if(r.problem.code!=0){return .err(fail(r.problem.code,c.module_index,r.problem.start,r.problem.end));}let u32 code=mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,r.type);if(code!=0){return .err(fail(code,c.module_index,c.start,c.end));}}}i+=1;
    }
    i=0;while(i<v::len<binding::Coordinate>(seeds)){let c=*v::at<binding::Coordinate>(seeds,i);if(c.module_index>=v::len<syntax::Parsed>(sources) || c.declaration==0 || c.declaration>syntax::declaration_count(v::at<syntax::Parsed>(sources,c.module_index))){return .err(fail(12,c.module_index,c.start,c.end));}let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,c.module_index),c.declaration-1);if(d.kind!=c.kind || d.start!=c.start || d.end!=c.end || (d.kind!=3 && d.kind!=9) || d.generic_parameters!=0){return .err(fail(12,c.module_index,c.start,c.end));}let p=callable(sources,metadata,c,ref mut graph,ref mut callables,ref mut parameters,ref mut names,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}i+=1;}
    var v::Vec<body_types::Request> demanded=v::empty<body_types::Request>();var usize vectors=0;var usize cursor=0;var usize instance_cursor=0;var usize dependency_cursor=0;var usize cleanup_cursor=0;var usize cleanup_body_cursor=0;
    while(true){
        while(instance_cursor<generics::count(ref specializations)){let s=generics::at(ref specializations,instance_cursor+1);let u32 code=mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,s.result);if(code!=0){return .err(fail(code,s.module_index,0,0));}var usize j=0;while(j<s.parameter_count){let u32 code=mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,generics::parameter_at(ref specializations,s.parameter_start+j));if(code!=0){return .err(fail(code,s.module_index,0,0));}j+=1;}
            let p=body_demands(sources,metadata,ref specializations,1,instance_cursor+1,ref mut graph,ref mut demanded,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}instance_cursor+=1;
        }
        while(cleanup_body_cursor<generics::cleanup_count(ref specializations)){let p=body_demands(sources,metadata,ref specializations,2,cleanup_body_cursor+1,ref mut graph,ref mut demanded,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}cleanup_body_cursor+=1;}
        while(dependency_cursor<generics::dependency_count(ref specializations)){let d=generics::dependency_at(ref specializations,dependency_cursor);let usize type=if(d.aggregate_type!=0){d.aggregate_type}else{d.intrinsic_type};if(type!=0){let u32 code=mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,type);if(code!=0){return .err(fail(code,0,0,0));}}dependency_cursor+=1;}
        while(cleanup_cursor<generics::cleanup_dependency_count(ref specializations)){let d=generics::cleanup_dependency_at(ref specializations,cleanup_cursor);let usize type=if(d.aggregate_type!=0){d.aggregate_type}else{d.intrinsic_type};if(type!=0){let u32 code=mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,type);if(code!=0){return .err(fail(code,0,0,0));}}cleanup_cursor+=1;}
        var v::Vec<usize> cleanup_seeds=v::empty<usize>();while(cursor<v::len<usize>(ref pending)){let usize id=*v::at<usize>(ref pending,cursor);let t=types::at(ref graph,id);if(!charge(ref mut budget,1)){return .err(fail(10,t.module_index,0,0));}if(t.kind==13){if(vectors>=64){return .err(fail(4503,t.module_index,0,0));}vectors+=1;}
            if(t.kind==2 || t.kind==3 || t.kind==4 || t.kind==13){let p=shape(sources,metadata,id,ref mut graph,ref mut shapes,ref mut fields,ref mut names,ref mut pending,ref mut seen,ref mut budget);if(p.code!=0){return .err(p);}}
            if(t.kind==3 || t.kind==13){if(!v::push<usize>(ref mut cleanup_seeds,id).is_ok){return .err(fail(11,t.module_index,0,0));}}
            if(t.kind>=5 && t.kind<=8 || t.kind==13 || t.kind==16 || t.kind==12 && t.first!=0){let u32 code=mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,t.first);if(code!=0){return .err(fail(code,0,0,0));}if(t.kind==6){let u32 code=mark(ref mut graph,ref mut pending,ref mut seen,ref mut budget,t.second);if(code!=0){return .err(fail(code,0,0,0));}}}cursor+=1;
        }
        if(v::len<usize>(ref cleanup_seeds)==0){break;}let usize old_work=generics::work(ref specializations);specializations=switch(generics::extend_cleanups(sources,metadata,specializations,ref mut graph,ref cleanup_seeds)){case .ok(p):{p}case .err(p):{return .err(p);}};if(!charge(ref mut budget,generics::work(ref specializations)-old_work)){return .err(fail(4503,binding::first(bound),0,0));}
    }
    let owned=switch(signatures::own_types(metadata,graph)){case .ok(g):{g}case .err(code):{return .err(fail(code,0,0,0));}};
    return .ok(Interfaces{.graph=owned,.shapes,.fields,.callables,.parameters,.annotations=roots,.specializations,.body_types=demanded,.names,.work=budget.work,.first=binding::first(bound),.stop=binding::stop(bound)});
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
