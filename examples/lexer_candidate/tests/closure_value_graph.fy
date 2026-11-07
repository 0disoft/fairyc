module tests::closure_value_graph;
import module_types as types;
import module_catalog as catalog;
import closure_value_type_source as closure;
import syntax as syntax;
import syntax_parse_driver as parser;
import types as nodes;
import scanner as scanner;
import module_unit_cleanup_mir_model as executable;
import file_source as file;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;

fn scanner::Stream scanned(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{assert(false);return scanned(raw);}};return switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);scanned(raw)}};}
fn syntax::Parsed parsed(ref u8[] raw){let p=switch(parser::parse_diagnostics(scanned(raw),512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};assert(syntax::problem_count(ref p)==0);return p;}
fn catalog::Catalog catalog_for(ref u8[] raw){var files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(raw)).is_ok);return switch(catalog::build(ref files)){case .ok(c):{c}case .err(_):{assert(false);catalog_for(raw)}};}
fn types::Type node(u32 kind,u32 scalar,usize module_index,usize declaration,usize first,usize second){return types::Type{.kind,.scalar,.module_index,.declaration,.first,.second};}
fn void leaf(types::Type t){assert(t.kind==23 && t.scalar==0 && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0);}
fn usize result(ref syntax::Parsed p,usize ordinal){var usize i=0;var usize at=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==3 || d.kind==9){if(at==ordinal){return d.result_type;}at+=1;}i+=1;}assert(false);return 0;}
fn void sentinel(ref v::Vec<types::Type> graph){assert(types::count(graph)==1);let t=types::at(graph,1);assert(t.kind==1 && t.scalar==1 && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0);}
fn void rejected(ref u8[] raw){
    let p=parsed(raw);let c=catalog_for(raw);let root=result(ref p,0);let n=syntax::type_at(ref p,root);var graph=v::empty<types::Type>();assert(types::canonical(ref mut graph,node(1,1,0,0,0,0))==1);
    let ordinary=types::resolve(ref c,ref p,0,root,ref mut graph);assert(ordinary.type==0 && ordinary.problem.code!=0 && ordinary.problem.module_index==0 && ordinary.problem.start>=n.start && ordinary.problem.end<=n.end && ordinary.problem.end>ordinary.problem.start);sentinel(ref graph);
    let empty=v::empty<usize>();let aggregate=types::aggregate_resolve(ref c,ref p,0,0,root,ref empty,ref mut graph);assert(aggregate.type==0 && aggregate.problem.code!=0);sentinel(ref graph);
}
@effects(filesystem) fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(file::manifest(view[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] wire,ref mut usize at){assert(*at<=wire.len && wire.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(wire[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] wire,ref mut usize at){let n=number(wire,at);assert(*at<=wire.len && n<=wire.len-*at);let start=*at;*at+=n;return wire[start..*at];}

@test @effects(filesystem) fn void closure_graph_matches_go_identity_and_source_spans(){
    let input=fixture();let wire=bytes::view(ref input);assert(wire.len>=16 && bytes::equal(wire[..8],b"FYCVGR01"));var usize at=8;let cases=number(wire,ref mut at);assert(cases>=16);var usize i=0;var usize admitted=0;
    while(i<cases){let raw=field(wire,ref mut at);let start=number(wire,ref mut at);let end=number(wire,ref mut at);let kind=number(wire,ref mut at);let accept=number(wire,ref mut at);let identity=field(wire,ref mut at);let p=parsed(raw);let root=result(ref p,0);let n=syntax::type_at(ref p,root);assert(n.start==start && n.end==end && (n.kind as usize)==kind);
        if(kind==17){let r=syntax::type_at(ref p,n.left);assert(r.start==number(wire,ref mut at) && r.end==number(wire,ref mut at));let parameters=number(wire,ref mut at);var usize link=n.right;var usize count=0;while(link!=0){count+=1;link=syntax::type_at(ref p,link).right;}assert(count==parameters);var usize j=0;while(j<count){link=n.right;var usize rest=count-1;while(rest>j){link=syntax::type_at(ref p,link).right;rest-=1;}let parameter=syntax::type_at(ref p,syntax::type_at(ref p,link).left);assert(parameter.start==number(wire,ref mut at) && parameter.end==number(wire,ref mut at));j+=1;}}
        if(accept==1){assert(kind==17 && closure::signature(ref p,root) && bytes::equal(identity,b"fn i32(i32)"));let c=catalog_for(raw);var graph=v::empty<types::Type>();let resolved=types::resolve(ref c,ref p,0,root,ref mut graph);assert(resolved.problem.code==0 && resolved.type==1 && types::count(ref graph)==1 && types::validate(ref c,ref graph)==0);leaf(types::at(ref graph,resolved.type));let empty=v::empty<usize>();let repeated=types::aggregate_resolve(ref c,ref p,0,0,root,ref empty,ref mut graph);assert(repeated.problem.code==0 && repeated.type==resolved.type && types::count(ref graph)==1);let owned=switch(types::assemble(ref c,graph)){case .ok(g):{g}case .err(_):{assert(false);return;}};assert(!types::aggregate_graph(ref owned) && types::validate_graph(ref c,ref owned)==0);leaf(types::graph_at(ref owned,1));admitted+=1;}
        else{assert(identity.len==0);rejected(raw);}i+=1;
    }
    assert(admitted>=3);let malformed=number(wire,ref mut at);i=0;while(i<malformed){let raw=field(wire,ref mut at);let expected=number(wire,ref mut at);let stream=scanned(raw);var type_nodes=v::empty<nodes::Node>();let bad=nodes::parse(ref stream,0,0,ref mut type_nodes);assert(bad.root==0 && bad.status==0);if(expected!=0){assert(scanner::token_info(ref stream,bad.next).start==expected);}i+=1;}assert(at==wire.len);
}

@test fn void closure_graph_preserves_roots_batches_and_source_authority(){
    let raw=b"module sample;struct Job{fn i32(i32) action;}fn fn i32(i32) identity(fn i32(i32) action){let fn i32(i32) copy=action;return copy;}fn fn u32(i32) bad(fn u32(i32) action){return action;}";
    let p=parsed(raw);let c=catalog_for(raw);var graph=v::empty<types::Type>();assert(types::canonical(ref mut graph,node(1,1,0,0,0,0))==1);let empty=v::empty<usize>();let root=result(ref p,0);
    var roots=v::empty<usize>();assert(v::push<usize>(ref mut roots,root).is_ok && v::push<usize>(ref mut roots,0).is_ok && v::push<usize>(ref mut roots,result(ref p,1)).is_ok);
    switch(types::resolve_many(ref c,ref p,0,ref roots,ref mut graph)){case .ok(_):{assert(false);}case .err(e):{assert(e.request==3 && e.problem.start==syntax::type_at(ref p,result(ref p,1)).start);}};sentinel(ref graph);
    switch(types::aggregate_many(ref c,ref p,0,0,ref roots,ref empty,ref mut graph)){case .ok(_):{assert(false);}case .err(e):{assert(e.request==3);}};sentinel(ref graph);
    discard v::pop<usize>(ref mut roots);let ids=switch(types::resolve_many(ref c,ref p,0,ref roots,ref mut graph)){case .ok(ids):{ids}case .err(_):{assert(false);return;}};assert(*v::at<usize>(ref ids,0)==2 && *v::at<usize>(ref ids,1)==0 && types::count(ref graph)==2);
    let f=syntax::field_at(ref p,0);assert(types::resolve(ref c,ref p,0,f.type_root,ref mut graph).type==2);let d=syntax::declaration_at(ref p,2);var usize i=0;var usize references=0;while(i<d.entry_count){let e=syntax::entry_at(ref p,d.entry_start+i);if(e.type_root!=0 && syntax::type_at(ref p,e.type_root).kind==17){assert(types::resolve(ref c,ref p,0,e.type_root,ref mut graph).type==2);references+=1;}i+=1;}assert(references==2 && types::count(ref graph)==2);
    let argument=types::resolve_argument(ref c,ref p,0,0,root,ref empty,ref mut graph);assert(argument.type==0 && argument.problem.code==4502 && types::count(ref graph)==2);
    let changed=parsed(b"module sample;fn fn i32(i32) other(fn i32(i32) action){return action;}");assert(types::resolve(ref c,ref changed,0,result(ref changed,0),ref mut graph).problem.code==12 && types::count(ref graph)==2);
    let grouped=parsed(b"module sample;fn (fn i32(i32)) keep((fn i32(i32)) action){return action;}");let grouped_c=catalog_for(syntax::source_view(ref grouped));let child=syntax::type_at(ref grouped,result(ref grouped,0)).left;assert(types::resolve(ref grouped_c,ref grouped,0,child,ref mut graph).problem.code==9 && types::count(ref graph)==2);
    let owned=switch(types::assemble_aggregates(ref c,graph)){case .ok(g):{g}case .err(_):{assert(false);return;}};assert(types::aggregate_graph(ref owned) && types::validate_graph(ref c,ref owned)==0);leaf(types::graph_at(ref owned,2));
}

@test fn void closure_graph_keeps_nominal_storage_and_generic_boundaries(){
    let raw=b"module sample;struct Job{fn i32(i32) action;}struct Outer{Job job;}struct Plain{i32 value;}struct Generic<T>{T value;}fn Outer keep(Outer value){return value;}fn Option<Job> option(){}fn Result<i32,Outer> result(){}fn ptr Job pointer(){}fn ref Outer borrowed(){}fn Job[2] array(){}fn Generic<fn i32(i32)> generic(){}";
    let p=parsed(raw);let c=catalog_for(raw);let empty=v::empty<usize>();var graph=v::empty<types::Type>();let direct=types::resolve(ref c,ref p,0,result(ref p,0),ref mut graph);assert(direct.problem.code==0 && types::at(ref graph,direct.type).kind==2 && types::validate(ref c,ref graph)==0);let before=types::count(ref graph);var usize i=1;
    while(i<7){assert(types::resolve(ref c,ref p,0,result(ref p,i),ref mut graph).problem.code!=0 && types::count(ref graph)==before);assert(types::aggregate_resolve(ref c,ref p,0,0,result(ref p,i),ref empty,ref mut graph).problem.code!=0 && types::count(ref graph)==before);i+=1;}
    assert(types::resolve_argument(ref c,ref p,0,0,result(ref p,0),ref empty,ref mut graph).problem.code==4502 && types::count(ref graph)==before);
    let resource_source=b"module sample;resource Holder{fn i32(i32) action;cleanup{}}fn Holder keep(Holder value){return value;}";rejected(resource_source);
    rejected(b"module sample;@repr(C) struct Holder{fn i32(i32) action;}fn Holder keep(Holder value){return value;}");
    rejected(b"module sample;struct Holder{Option<fn i32(i32)> action;}fn Holder keep(Holder value){return value;}");
    rejected(b"module sample;struct Holder{Option<Holder> next;fn i32(i32) action;}fn Holder keep(Holder value){return value;}");
    rejected(b"module sample;fn fn i32(i32) keep<T>(fn i32(i32) value){return value;}");
    rejected(b"module sample;unsafe extern C fn fn i32(i32) foreign(fn i32(i32) value);");
    let resource_p=parsed(resource_source);let resource_c=catalog_for(resource_source);let field=syntax::field_at(ref resource_p,0);assert(types::resolve(ref resource_c,ref resource_p,0,field.type_root,ref mut graph).problem.code==9 && types::count(ref graph)==before);
    let recursive=b"module sample;struct Recursive{ptr Recursive next;}fn Option<Recursive> keep(Option<Recursive> value){return value;}";let recursive_p=parsed(recursive);let recursive_c=catalog_for(recursive);var legacy=v::empty<types::Type>();assert(types::resolve(ref recursive_c,ref recursive_p,0,result(ref recursive_p,0),ref mut legacy).problem.code==0 && types::validate(ref recursive_c,ref legacy)==0);
    // A supplied closure ID never becomes a generic substitution or tuple.
    let source=b"module sample;struct Generic<T>{T value;}fn T keep<T>(T value){return value;}fn fn i32(i32) direct(fn i32(i32) value){return value;}";let q=parsed(source);let qc=catalog_for(source);var values=v::empty<types::Type>();let admitted=types::resolve(ref qc,ref q,0,result(ref q,1),ref mut values);assert(admitted.problem.code==0);var args=v::empty<usize>();assert(v::push<usize>(ref mut args,admitted.type).is_ok);assert(types::specialize(ref qc,ref q,0,3,result(ref q,0),ref args,ref mut values).problem.code==4502 && types::count(ref values)==1);assert(types::aggregate(ref qc,0,2,ref args,ref mut values).problem.code==4502 && types::count(ref values)==1);
}

fn u32 executable_code(u32 kind,u32 scalar){
    var records=v::empty<types::Type>();assert(v::push<types::Type>(ref mut records,node(kind,scalar,0,0,0,0)).is_ok);var glues=v::empty<executable::Glue>();assert(v::push<executable::Glue>(ref mut glues,executable::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
    return switch(executable::assemble(records,v::empty<executable::Frame>(),glues,v::empty<executable::Field>(),v::empty<executable::Operand>(),v::empty<executable::Drop>(),0)){case .ok(_):{0}case .err(p):{p.code}};
}

@test fn void closure_graph_validates_payloads_capacity_and_execution_separation(){
    let c=catalog_for(b"module sample;");var graph=v::empty<types::Type>();assert(types::canonical(ref mut graph,node(23,0,0,0,0,0))==1 && types::validate(ref c,ref graph)==0);assert(types::canonical(ref mut graph,node(23,0,0,0,0,0))==1);
    var usize i=0;while(i<5){var t=node(23,0,0,0,0,0);if(i==0){t.scalar=1;}else if(i==1){t.module_index=1;}else if(i==2){t.declaration=1;}else if(i==3){t.first=1;}else{t.second=1;}assert(v::push<types::Type>(ref mut graph,t).is_ok);assert(types::validate(ref c,ref graph)==9);discard v::pop<types::Type>(ref mut graph);i+=1;}
    assert(v::push<types::Type>(ref mut graph,node(23,0,0,0,0,0)).is_ok && types::validate(ref c,ref graph)==9);discard v::pop<types::Type>(ref mut graph);
    var u32 wrapper=5;while(wrapper<=18){if(wrapper<=8 || wrapper==12 || wrapper==16 || wrapper==17 || wrapper==18){assert(v::push<types::Type>(ref mut graph,node(wrapper,0,0,0,1,if(wrapper==6 || wrapper==16){1}else{0})).is_ok);assert(types::validate(ref c,ref graph)==9);discard v::pop<types::Type>(ref mut graph);}wrapper+=1;}
    assert(v::push<types::Type>(ref mut graph,node(22,0,0,0,1,0)).is_ok);assert(!types::assemble_aggregates(ref c,graph).is_ok);
    assert(executable_code(1,1)==0 && executable_code(23,0)==1001);
    let literal=b"module sample;fn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset+x;};return action;}";let retained=parsed(literal);assert(!syntax::closure_prepared(ref retained));assert(switch(syntax::parse(scanned(literal),512)){case .ok(_):{false}case .err(code):{code==4}});
    // The synthetic prefix exercises intern capacity only; it is not published.
    let source=b"module sample;fn fn i32(i32) keep(fn i32(i32) value){return value;}";let p=parsed(source);let source_c=catalog_for(source);var full=v::empty<types::Type>();i=0;while(i<4096){assert(v::push<types::Type>(ref mut full,node(1,1,i,0,0,0)).is_ok);i+=1;}let overflow=types::resolve(ref source_c,ref p,0,result(ref p,0),ref mut full);assert(overflow.type==0 && overflow.problem.code==10 && types::count(ref full)==4096);assert(types::at(ref full,4096).module_index==4095);discard v::pop<types::Type>(ref mut full);let exact=types::resolve(ref source_c,ref p,0,result(ref p,0),ref mut full);assert(exact.problem.code==0 && exact.type==4096 && types::count(ref full)==4096);leaf(types::at(ref full,4096));assert(types::resolve(ref source_c,ref p,0,result(ref p,0),ref mut full).type==4096 && types::count(ref full)==4096);assert(v::push<types::Type>(ref mut full,node(1,2,0,0,0,0)).is_ok && types::validate(ref source_c,ref full)==10);
}
