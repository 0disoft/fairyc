module module_unit_bodies;
import module_unit_body_ast as ast;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_signatures as signatures;
import module_types as types;
import module_ast_join as join;
import module_units as units;
import syntax as syntax;
import scanner as scanner;
import std::vec as v;
import storage_intrinsics as storage;

pub struct Edge {usize expression;generics::Dependency target;}
resource Source {usize module_index;scanner::Stream stream;cleanup{}}
pub resource Body {generics::Instance signature;usize source;ast::Ast ast;v::Vec<usize> arguments;v::Vec<usize> parameters;v::Vec<usize> annotations;v::Vec<Edge> edges;cleanup{}}
pub resource Cleaning {generics::Cleanup info;usize source;ast::Ast ast;v::Vec<usize> arguments;v::Vec<usize> annotations;v::Vec<Edge> edges;cleanup{}}
pub resource Plan {v::Vec<Source> sources;v::Vec<Body> bodies;v::Vec<Cleaning> cleanups;types::Graph graph;units::Usage usage;usize cooked_bytes;usize work;cleanup{}}
fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
fn void request(ref mut v::Vec<usize> table,usize root){if(root!=0){let slot=v::at_mut<usize>(table,root-1);*slot=1;}}
fn Result<v::Vec<usize>,join::Problem> annotations(ref ast::Ast p,ref syntax::Parsed original,generics::Instance instance,ref signatures::Program metadata,ref v::Vec<usize> args,ref mut v::Vec<types::Type> graph,ref mut usize work){
    if(!ast::charge(work,1)){return .err(fail(10,instance.module_index,0,0));}let bool vector=signatures::sdk_vector_source(metadata,instance.module_index);var v::Vec<usize> table=v::empty<usize>();var usize i=0;while(i<ast::type_count(p)){if(!ast::charge(work,1)){return .err(fail(10,instance.module_index,0,0));}if(!v::push<usize>(ref mut table,0).is_ok){return .err(fail(11,instance.module_index,0,0));}i+=1;}
    request(ref mut table,ast::declaration(p).result_type);i=0;while(i<ast::entry_count(p)){request(ref mut table,ast::entry_at(p,i).type_root);i+=1;}
    i=1;while(i<=ast::expression_count(p)){if(!ast::charge(work,1)){return .err(fail(10,instance.module_index,0,0));}let e=ast::expression_at(p,i);if(e.kind==23){request(ref mut table,e.right);}else if(e.kind==20){var usize root=e.right;while(root!=0){let n=ast::type_at(p,root);let usize original_root=ast::type_origin(p,n.left);let usize selected=storage::annotation_root(original,ast::expression_origin(p,i),original_root,vector);request(ref mut table,if(selected!=original_root){ast::type_at(p,n.left).left}else{n.left});root=n.right;}}i+=1;}
    i=1;while(i<=ast::statement_count(p)){let s=ast::statement_at(p,i);if(s.kind==18){request(ref mut table,s.second);}i+=1;}
    var bool limited=false;i=1;while(i<=ast::type_count(p)){if(*v::at<usize>(ref table,i-1)!=0){if(!ast::charge(work,1)){while(v::len<usize>(ref table)>=i){discard v::pop<usize>(ref mut table);}limited=true;break;}let slot=v::at_mut<usize>(ref mut table,i-1);*slot=ast::type_origin(p,i);}i+=1;}
    // Every slot now contains its provider root or zero. Validate that frozen
    // source once; a charged prefix still resolves before a later work error.
    let resolved=switch(if(v::len<usize>(args)==0){signatures::resolve_types(metadata,original,instance.module_index,ref table,graph)}else{signatures::specialize_types(metadata,original,instance.module_index,instance.declaration,ref table,args,graph)}){case .ok(p):{p}case .err(e):{return .err(fail(e.problem.code,instance.module_index,e.problem.start,e.problem.end));}};
    if(limited){return .err(fail(10,instance.module_index,0,0));}return .ok(resolved);
}
fn Result<v::Vec<Edge>,join::Problem> edges(ref generics::Plan instances,ref ast::Ast body,usize owner,bool cleaning,usize module_index,usize start,usize end,ref mut usize work){
    var v::Vec<Edge> result=v::empty<Edge>();let usize count=if(cleaning){generics::cleanup_dependency_count(instances)}else{generics::dependency_count(instances)};var usize i=0;while(i<count){if(!ast::charge(work,1)){return .err(fail(10,module_index,start,end));}let target=if(cleaning){generics::cleanup_dependency_at(instances,i)}else{generics::dependency_at(instances,i)};if(target.owner==owner){var usize expression=1;while(expression<=ast::expression_count(body) && ast::expression_origin(body,expression)!=target.source_index){if(!ast::charge(work,1)){return .err(fail(10,module_index,start,end));}expression+=1;}if(expression>ast::expression_count(body) || ast::expression_at(body,expression).kind!=20){return .err(fail(9,module_index,start,end));}if(!v::push<Edge>(ref mut result,Edge{.expression,.target}).is_ok){return .err(fail(11,module_index,start,end));}}i+=1;}
    i=1;while(i<=ast::expression_count(body)){if(ast::expression_at(body,i).kind==20){var usize e=0;var usize found=0;while(e<v::len<Edge>(ref result)){if(!ast::charge(work,1)){return .err(fail(10,module_index,start,end));}if(v::at<Edge>(ref result,e).expression==i){found+=1;}e+=1;}if(found!=1){let n=ast::expression_at(body,i);return .err(fail(9,module_index,n.start,n.end));}}i+=1;}return .ok(result);
}
fn Result<usize,u32> source(ref mut v::Vec<Source> owned,ref syntax::Parsed original,usize module_index,ref mut units::Usage usage,ref mut usize cooked,ref mut usize work){
    var usize i=0;while(i<v::len<Source>(owned)){if(!ast::charge(work,1)){return .err(10);}if(v::at<Source>(owned,i).module_index==module_index){return .ok(i);}i+=1;}
    let usize bytes=syntax::source_view(original).len;if(bytes>65536 || usage.bytes>1048576 || bytes>1048576-usage.bytes || !ast::charge(work,bytes)){return .err(10);}usage.bytes+=bytes;
    let stream=switch(syntax::copy_source(original,work)){case .ok(s):{s}case .err(c):{return .err(c);}};
    let usize stored=scanner::storage(ref stream).cooked_bytes;if(*cooked>65536 || stored>65536-*cooked || !ast::charge(work,stored)){return .err(10);};*cooked=*cooked+stored;
    if(!v::push<Source>(owned,Source{.module_index,.stream}).is_ok){return .err(11);}return .ok(v::len<Source>(owned)-1);
}
fn bool measure(ref ast::Ast p,ref mut units::Usage u){u.declarations+=1;u.expressions+=ast::expression_count(p);u.statements+=ast::statement_count(p);u.entries+=ast::entry_count(p);u.types+=ast::type_count(p);u.attributes+=ast::attribute_count(p);u.arguments+=ast::argument_count(p);
    return u.declarations<=512 && u.expressions<=16384 && u.statements<=8192 && u.entries<=1024 && u.types<=4096 && u.attributes<=4096 && u.arguments<=4096;
}
fn Result<Plan,join::Problem> collect(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref generics::Plan instances,v::Vec<types::Type> graph_input){
    var v::Vec<types::Type> graph=graph_input;
    let valid=generics::validate(instances,sources,metadata,ref graph);if(valid.code!=0){return .err(valid);}var v::Vec<Source> owned=v::empty<Source>();var v::Vec<Body> bodies=v::empty<Body>();var usize work=0;var usize cooked=0;var units::Usage usage=units::Usage{.bytes=0,.declarations=0,.expressions=0,.statements=0,.entries=0,.types=0,.fields=0,.attributes=0,.arguments=0};var usize id=1;
    while(id<=generics::count(instances)){var instance=generics::at(instances,id);let original=v::at<syntax::Parsed>(sources,instance.module_index);let d=syntax::declaration_at(original,instance.declaration-1);
        let source_index=switch(source(ref mut owned,original,instance.module_index,ref mut usage,ref mut cooked,ref mut work)){case .ok(i):{i}case .err(c):{return .err(fail(c,instance.module_index,d.start,d.end));}};
        let body=switch(ast::build(original,instance.declaration,ref mut work)){case .ok(p):{p}case .err(c):{return .err(fail(c,instance.module_index,d.start,d.end));}};
        if(!measure(ref body,ref mut usage)){return .err(fail(10,instance.module_index,d.start,d.end));}
        var v::Vec<usize> arguments=v::empty<usize>();var v::Vec<usize> parameters=v::empty<usize>();var usize i=0;
        while(i<instance.argument_count){if(!v::push<usize>(ref mut arguments,generics::argument_at(instances,instance.argument_start+i)).is_ok){return .err(fail(11,instance.module_index,d.start,d.end));}i+=1;}
        while(v::len<usize>(ref parameters)<instance.parameter_count){let usize index=v::len<usize>(ref parameters);if(!v::push<usize>(ref mut parameters,generics::parameter_at(instances,instance.parameter_start+index)).is_ok){return .err(fail(11,instance.module_index,d.start,d.end));}}
        let roots=switch(annotations(ref body,original,instance,metadata,ref arguments,ref mut graph,ref mut work)){case .ok(p):{p}case .err(p):{return .err(p);}};
        if(*v::at<usize>(ref roots,ast::declaration(ref body).result_type-1)!=instance.result){return .err(fail(9,instance.module_index,d.start,d.end));}
        var usize parameter=0;i=0;while(i<ast::entry_count(ref body)){let e=ast::entry_at(ref body,i);if(e.kind==1){if(parameter>=instance.parameter_count || *v::at<usize>(ref roots,e.type_root-1)!=*v::at<usize>(ref parameters,parameter)){return .err(fail(9,instance.module_index,e.start,e.end));}parameter+=1;}i+=1;}if(parameter!=instance.parameter_count){return .err(fail(9,instance.module_index,d.start,d.end));}
        let requests=switch(edges(instances,ref body,id,false,instance.module_index,d.start,d.end,ref mut work)){case .ok(p):{p}case .err(p):{return .err(p);}};
        instance.argument_start=0;instance.parameter_start=0;if(!v::push<Body>(ref mut bodies,Body{.signature=instance,.source=source_index,.ast=body,.arguments,.parameters,.annotations=roots,.edges=requests}).is_ok){return .err(fail(11,instance.module_index,d.start,d.end));}id+=1;
    }
    var v::Vec<Cleaning> cleanups=v::empty<Cleaning>();id=1;while(id<=generics::cleanup_count(instances)){var info=generics::cleanup_at(instances,id);let original=v::at<syntax::Parsed>(sources,info.module_index);let source_index=switch(source(ref mut owned,original,info.module_index,ref mut usage,ref mut cooked,ref mut work)){case .ok(i):{i}case .err(c):{return .err(fail(c,info.module_index,info.start,info.end));}};
        let body=switch(ast::build_cleanup(original,info.declaration,ref mut work)){case .ok(p):{p}case .err(c):{return .err(fail(c,info.module_index,info.start,info.end));}};if(!measure(ref body,ref mut usage)){return .err(fail(10,info.module_index,info.start,info.end));}
        var v::Vec<usize> arguments=v::empty<usize>();var usize i=0;while(i<info.argument_count){if(!v::push<usize>(ref mut arguments,generics::argument_at(instances,info.argument_start+i)).is_ok){return .err(fail(11,info.module_index,info.start,info.end));}i+=1;}
        let context=generics::Instance{.module_index=info.module_index,.declaration=info.declaration,.argument_start=0,.argument_count=info.argument_count,.result=0,.parameter_start=0,.parameter_count=0,.effects=info.effects,.must_use=false,.unsafe_call=false};
        let roots=switch(annotations(ref body,original,context,metadata,ref arguments,ref mut graph,ref mut work)){case .ok(p):{p}case .err(p):{return .err(p);}};let requests=switch(edges(instances,ref body,id,true,info.module_index,info.start,info.end,ref mut work)){case .ok(p):{p}case .err(p):{return .err(p);}};info.argument_start=0;
        if(!v::push<Cleaning>(ref mut cleanups,Cleaning{.info,.source=source_index,.ast=body,.arguments,.annotations=roots,.edges=requests}).is_ok){return .err(fail(11,info.module_index,info.start,info.end));}id+=1;
    }
    let retained=switch(signatures::own_types(metadata,graph)){case .ok(g):{g}case .err(c):{return .err(fail(c,0,0,0));}};return .ok(Plan{.sources=owned,.bodies,.cleanups,.graph=retained,.usage,.cooked_bytes=cooked,.work});
}
pub fn Result<Plan,join::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref interfaces::Interfaces imported){
    var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;while(i<=interfaces::type_count(imported)){if(!v::push<types::Type>(ref mut graph,interfaces::type_at(imported,i)).is_ok){return .err(fail(11,0,0,0));}i+=1;}return collect(sources,metadata,interfaces::specializations(imported),graph);
}
pub fn usize count(ref Plan p){return v::len<Body>(ref p.bodies);}
pub fn ref Body at(ref Plan p,usize id){return v::at<Body>(ref p.bodies,id-1);}
pub fn generics::Instance signature(ref Body p){return p.signature;}
pub fn ref ast::Ast body_ast(ref Body p){return ref p.ast;}
pub fn usize argument_at(ref Body p,usize index){return *v::at<usize>(ref p.arguments,index);}
pub fn usize parameter_at(ref Body p,usize index){return *v::at<usize>(ref p.parameters,index);}
pub fn usize annotation_at(ref Body p,usize root){return *v::at<usize>(ref p.annotations,root-1);}
pub fn usize edge_count(ref Body p){return v::len<Edge>(ref p.edges);}
pub fn Edge edge_at(ref Body p,usize index){return *v::at<Edge>(ref p.edges,index);}
pub fn ref scanner::Stream source_at(ref Plan p,usize id){let body=at(p,id);return ref v::at<Source>(ref p.sources,body.source).stream;}
pub fn usize source_count(ref Plan p){return v::len<Source>(ref p.sources);}
pub fn usize type_count(ref Plan p){return types::graph_count(ref p.graph);}
pub fn types::Type type_at(ref Plan p,usize id){return types::graph_at(ref p.graph,id);}
pub fn units::Usage usage(ref Plan p){return p.usage;}
pub fn usize work(ref Plan p){return p.work;}
pub fn usize cooked_bytes(ref Plan p){return p.cooked_bytes;}
pub fn usize cleanup_count(ref Plan p){return v::len<Cleaning>(ref p.cleanups);}
pub fn ref Cleaning cleanup_at(ref Plan p,usize id){return v::at<Cleaning>(ref p.cleanups,id-1);}
pub fn generics::Cleanup cleanup_info(ref Cleaning p){return p.info;}
pub fn ref ast::Ast cleanup_ast(ref Cleaning p){return ref p.ast;}
pub fn usize cleanup_argument_at(ref Cleaning p,usize index){return *v::at<usize>(ref p.arguments,index);}
pub fn usize cleanup_annotation_at(ref Cleaning p,usize root){return *v::at<usize>(ref p.annotations,root-1);}
pub fn usize cleanup_edge_count(ref Cleaning p){return v::len<Edge>(ref p.edges);}
pub fn Edge cleanup_edge_at(ref Cleaning p,usize index){return *v::at<Edge>(ref p.edges,index);}
pub fn ref scanner::Stream cleanup_source_at(ref Plan p,usize id){let c=cleanup_at(p,id);return ref v::at<Source>(ref p.sources,c.source).stream;}
fn bool room(usize current,usize added,usize cap){return current<=cap && added<=cap-current;}
pub fn bool fits(ref Plan p,units::Usage base){let u=p.usage;return room(base.bytes,u.bytes,1048576) && room(base.declarations,u.declarations,512) && room(base.expressions,u.expressions,16384) && room(base.statements,u.statements,8192) && room(base.entries,u.entries,1024) && room(base.types,u.types,4096) && room(base.attributes,u.attributes,4096) && room(base.arguments,u.arguments,4096);}
