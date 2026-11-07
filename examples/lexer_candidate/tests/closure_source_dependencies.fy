module tests::closure_source_dependencies;
import generic_source_requests as requests;
import generic_source_catalog as templates;
import generic_source_dependency_model as model;
import generic_package_bindings as bindings;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import types as types;
import body_state as body;
import bodies as entries;
import records as records;
import attributes as attributes;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] raw){let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn coordinates::Coordinates coords(){let text=switch(bytes::copy(b"fypc1-sha256-1111111111111111111111111111111111111111111111111111111111111111")){case .ok(p):{p}case .err(_):{assert(false);return coords();}};var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=1,.start=0,.end=bytes::len(ref text)}).is_ok);return coordinates::assemble(text,rows);}
fn templates::Catalog collected(ref v::Vec<syntax::Parsed> sources){let table=coords();var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,1).is_ok);let imports=v::empty<catalog::Binding>();var usize work=0;let bound=switch(bindings::build(sources,ref owners,ref imports,ref table,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return collected(sources);}};return switch(templates::build_project(bound,sources,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);collected(sources)}};}
fn v::Vec<syntax::Parsed> source(ref u8[] raw){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);return sources;}
fn void admitted(ref u8[] raw,bool prepared){let sources=source(raw);let p=v::at<syntax::Parsed>(ref sources,0);assert(syntax::closure_prepared(p)==prepared);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(signatures::module_count(ref metadata)==1);let catalog=collected(ref sources);var usize work=0;let result=switch(requests::build(catalog,ref sources,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(model::edge_count(requests::data(ref result))==0 && model::range_count(requests::data(ref result))==1 && work>0);}
@test fn void closure_dependencies_admit_original_sources(){
    admitted(b"module main;fn fn i32(i32) keep(fn i32(i32) action){return action;}",false);
    admitted(b"module main;fn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset+x;};return action;}",true);
    admitted(b"module main;struct Holder{fn i32(i32) action;}fn Holder keep(Holder value){return value;}",false);
    admitted(b"module main;fn i32 keep(i32 value){return value;}",false);
}
fn void signature_refused(ref u8[] raw){let sources=source(raw);assert(switch(signatures::build(ref sources)){case .ok(_):{false}case .err(p):{p.code==9}});}
fn void child_refused(ref u8[] raw){let sources=source(raw);let catalog=collected(ref sources);var usize work=0;switch(requests::build(catalog,ref sources,ref mut work)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4501 && p.module_index==0 && p.start<p.end);}};}
@test fn void closure_dependencies_keep_source_signature_and_child_refusals(){
    signature_refused(b"module main;fn fn i64(i32) keep(fn i64(i32) action){return action;}");
    signature_refused(b"module main;fn fn i32() keep(fn i32() action){return action;}");
    signature_refused(b"module main;fn fn i32(i32,i32) keep(fn i32(i32,i32) action){return action;}");
    signature_refused(b"module main;fn void use(Option<fn i32(i32)> action){}");
    child_refused(b"module main;fn fn _fy_generic_result(i32) keep(){return 0;}");
    child_refused(b"module main;fn fn i32(_fy_generic_parameter) keep(){return 0;}");
}
// Only refusal controls use reconstructed arenas. Accepted cases above always
// use original scanner/parser owners, including the written-copy literal path.
fn syntax::Parsed forged(ref syntax::Parsed p,u32 mode){
    var usize work=0;let stream=switch(syntax::copy_source(p,ref mut work)){case .ok(s):{s}case .err(_):{assert(false);return forged(p,mode);}};
    var v::Vec<syntax::Declaration> declarations=v::empty<syntax::Declaration>();var v::Vec<types::Node> nodes=v::empty<types::Node>();var v::Vec<body::Node> expressions=v::empty<body::Node>();var v::Vec<body::Statement> statements=v::empty<body::Statement>();var v::Vec<entries::Entry> locals=v::empty<entries::Entry>();
    let root=syntax::declaration_at(p,1).result_type;let closure=syntax::type_at(p,root);var usize i=0;while(i<syntax::declaration_count(p)){assert(v::push<syntax::Declaration>(ref mut declarations,syntax::declaration_at(p,i)).is_ok);i+=1;}
    i=1;while(i<=syntax::type_count(p)){var n=syntax::type_at(p,i);if(i==root){if(mode==0){n.left=0;}else if(mode==1){n.left=root;}else if(mode==2){n.left=syntax::type_count(p)+1;}else if(mode==3){n.right=root;}}if(i==closure.right){if(mode==4){n.kind=7;}else if(mode==5){n.left=0;}else if(mode==6){n.left=i;}else if(mode==7){n.right=i;}}if(i==closure.left && mode==8){n.kind=14;}if(i==syntax::type_at(p,closure.right).left && mode==9){n.kind=14;}assert(v::push<types::Node>(ref mut nodes,n).is_ok);i+=1;}
    i=1;while(i<=syntax::expression_count(p)){let n=syntax::expression_at(p,i);assert(v::push<body::Node>(ref mut expressions,body::Node{.kind=n.kind,.start=n.start,.end=n.end,.token=n.token,.left=n.left,.right=n.right}).is_ok);i+=1;}
    i=1;while(i<=syntax::statement_count(p)){let n=syntax::statement_at(p,i);assert(v::push<body::Statement>(ref mut statements,body::Statement{.kind=n.kind,.start=n.start,.end=n.end,.expression=n.expression,.target=n.target,.first=n.first,.second=n.second,.entry=n.entry,.operator=n.operator}).is_ok);i+=1;}
    i=0;while(i<syntax::entry_count(p)){let e=syntax::entry_at(p,i);assert(v::push<entries::Entry>(ref mut locals,entries::Entry{.kind=e.kind,.start=e.start,.end=e.end,.name_start=e.name_start,.name_end=e.name_end,.type_start=e.type_start,.type_end=e.type_end,.type_root=e.type_root,.expression=e.expression,.id=i+1}).is_ok);i+=1;}
    return syntax::assemble(stream,declarations,v::empty<syntax::Problem>(),expressions,locals,statements,nodes,v::empty<records::Field>(),v::empty<attributes::Attribute>(),v::empty<attributes::Argument>());
}
@test fn void closure_dependencies_reject_malformed_owned_links(){
    let original=parsed(b"module main;fn fn i32(i32) keep(){return 0;}");var u32 mode=0;while(mode<10){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,forged(ref original,mode)).is_ok);let catalog=collected(ref sources);var usize work=0;switch(requests::build(catalog,ref sources,ref mut work)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==9 && p.module_index==0 && p.start<p.end);}};mode+=1;}
}
@test fn void closure_dependencies_keep_shared_work_boundary(){
    let sources=source(b"module main;fn fn i32(i32) keep(fn i32(i32) action){return action;}");var usize cost=0;let baseline=switch(requests::build(collected(ref sources),ref sources,ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(cost>0 && cost<16777216 && model::edge_count(requests::data(ref baseline))==0);var usize work=16777216-cost;let fit=switch(requests::build(collected(ref sources),ref sources,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(work==16777216 && model::range_count(requests::data(ref fit))==1);work=16777216-cost+1;assert(switch(requests::build(collected(ref sources),ref sources,ref mut work)){case .ok(_):{false}case .err(p):{p.code==10}} && work<=16777216);
}
