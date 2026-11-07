module tests::closure_value_public_text;
import module_public_type_text as text;
import module_public_type_ids as identities;
import module_public_interface_model as model;
import module_public_interface_extract as extract;
import module_public_interface as public_interface;
import module_signatures as signatures;
import module_catalog as catalog;
import module_types as types;
import module_unit_interfaces as typed;
import module_unit_bind as bind;
import syntax as syntax;
import scanner as scanner;
import file_source as file;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;

resource Context{v::Vec<syntax::Parsed> sources;signatures::Program metadata;typed::Interfaces graph;cleanup{}}
fn syntax::Parsed parsed(ref u8[] raw){
    let b=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{assert(false);return parsed(raw);}};
    let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(raw);}};
    return switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};
}
fn Context context(ref u8[] raw){
    var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return context(raw);}};
    let bound=switch(bind::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return context(raw);}};
    let graph=switch(typed::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return context(raw);}};
    return Context{.sources,.metadata,.graph};
}
fn text::Coordinates coordinates(){return text::assemble(bytes::empty(),v::empty<text::Coordinate>());}
fn text::Coordinates owner_coordinate(ref u8[] value){var storage=bytes::empty();assert(bytes::append(ref mut storage,value).is_ok);var rows=v::empty<text::Coordinate>();assert(v::push<text::Coordinate>(ref mut rows,text::Coordinate{.owner=0,.start=0,.end=value.len}).is_ok);return text::assemble(storage,rows);}
fn identities::Table names(ref Context p){
    var rows=v::empty<identities::Row>();var usize i=1;while(i<=typed::type_count(ref p.graph)){
        assert(v::push<identities::Row>(ref mut rows,identities::Row{.type=typed::type_at(ref p.graph,i),.owner=0,.name=model::Text{.start=0,.end=0}}).is_ok);i+=1;
    }
    var stamps=v::empty<identities::Stamp>();assert(v::push<identities::Stamp>(ref mut stamps,identities::Stamp{.owner=signatures::module_owner(ref p.metadata,0),.digest=signatures::source_digest(ref p.metadata,0)}).is_ok);
    return identities::assemble(bytes::empty(),rows,stamps);
}
fn Result<bytes::Buffer,u32> render(ref Context p,ref identities::Table named,ref text::Coordinates coordinates,usize id,u32 mode,ref mut usize work){
    if(mode==0){return text::render_records_prepared(named,ref p.sources,ref p.metadata,typed::type_records(ref p.graph),coordinates,0,id,false,work);}
    if(mode==1){return text::render(ref p.sources,ref p.metadata,ref p.graph,coordinates,0,id,work);}
    if(mode==2){return text::render_prepared(named,ref p.sources,ref p.metadata,ref p.graph,coordinates,0,id,work);}
    if(mode==3){return text::render_identified(named,ref p.sources,ref p.metadata,ref p.graph,coordinates,0,id,work);}
    return text::render_records_prepared(named,ref p.sources,ref p.metadata,typed::type_records(ref p.graph),coordinates,0,id,true,work);
}
fn void equal(Result<bytes::Buffer,u32> result,ref u8[] expected){switch(result){case .ok(b):{assert(bytes::equal(bytes::view(ref b),expected));}case .err(_):{assert(false);}};}
fn void error(Result<bytes::Buffer,u32> result,u32 expected){switch(result){case .ok(_):{assert(false);}case .err(code):{assert(code==expected);}};}
fn Result<extract::Projection,u32> project(ref Context p,ref identities::Table named,ref text::Coordinates coordinates,u32 mode,ref mut usize work){
    if(mode==0){return extract::extract(ref p.sources,ref p.metadata,ref p.graph,coordinates,0,work);}
    if(mode==1){return extract::extract_identified(named,ref p.sources,ref p.metadata,ref p.graph,coordinates,0,work);}
    return extract::extract_records_prepared(ref p.sources,ref p.metadata,typed::type_records(ref p.graph),typed::shape_records(ref p.graph),typed::field_records(ref p.graph),typed::name_records(ref p.graph),coordinates,named,0,mode==3,work);
}
fn void descriptor_matches_go(ref Context p,ref identities::Table named,ref text::Coordinates coordinates,ref u8[] expected_bytes,ref u8[] expected_hash){
    var u32 mode=0;while(mode<4){var usize cost=0;let projection=switch(project(p,named,coordinates,mode,ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576 && !extract::has_templates(ref projection));
        let descriptor=extract::descriptor(ref projection);var usize work=0;equal(public_interface::encode(signatures::state_layouts(ref p.metadata),0,descriptor,ref mut work),expected_bytes);work=0;equal(signatures::public_interface_hash(ref p.metadata,0,descriptor,ref mut work),expected_hash);
        work=1048576-cost;switch(project(p,named,coordinates,mode,ref mut work)){case .ok(_):{assert(work==1048576);}case .err(_):{assert(false);}};
        work=1048576-cost+1;switch(project(p,named,coordinates,mode,ref mut work)){case .ok(_):{assert(false);}case .err(code):{assert(code==10 && work<=1048576);}};mode+=1;
    }
    var raw=bytes::empty();assert(bytes::append(ref mut raw,syntax::source_view(v::at<syntax::Parsed>(ref p.sources,0))).is_ok && bytes::append(ref mut raw,b"\n// stale descriptor source").is_ok);var changed=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut changed,parsed(bytes::view(ref raw))).is_ok);var usize work=0;
    switch(extract::extract(ref changed,ref p.metadata,ref p.graph,coordinates,0,ref mut work)){case .ok(_):{assert(false);}case .err(code):{assert(code==12);}};work=0;
    switch(extract::extract_identified(named,ref changed,ref p.metadata,ref p.graph,coordinates,0,ref mut work)){case .ok(_):{assert(false);}case .err(code):{assert(code==12);}};
}
fn usize result(ref Context p){return signatures::at(ref p.metadata,1).result;}
fn usize parameter(ref Context p){let s=signatures::at(ref p.metadata,1);assert(s.parameter_count==1);return signatures::parameter_at(ref p.metadata,s.parameter_start);}
fn types::Type node(u32 kind,u32 scalar,usize module_index,usize declaration,usize first,usize second){return types::Type{.kind,.scalar,.module_index,.declaration,.first,.second};}
fn v::Vec<types::Type> copy_graph(ref Context p){
    var graph=v::empty<types::Type>();var usize i=1;while(i<=typed::type_count(ref p.graph)){assert(v::push<types::Type>(ref mut graph,typed::type_at(ref p.graph,i)).is_ok);i+=1;}return graph;
}
fn Result<bytes::Buffer,u32> raw(ref Context p,ref v::Vec<types::Type> graph,usize id,bool current,ref mut usize work){
    let named=identities::empty();let coordinates=coordinates();return text::render_records_prepared(ref named,ref p.sources,ref p.metadata,graph,ref coordinates,0,id,current,work);
}
@effects(filesystem) fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(file::manifest(view[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] wire,ref mut usize at){assert(*at<=wire.len && wire.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(wire[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] wire,ref mut usize at){let n=number(wire,at);assert(*at<=wire.len && n<=wire.len-*at);let start=*at;*at+=n;return wire[start..*at];}

// All positive records come from real authored sources via signatures/interfaces.
// The Go frontend and moduleinterface extractor supply the expected spellings.
@test @effects(filesystem) fn void closure_public_text_matches_go_and_preserves_legacy_bytes(){
    let input=fixture();let wire=bytes::view(ref input);assert(wire.len>=16 && bytes::equal(wire[..8],b"FYCVPT01"));var usize at=8;let count=number(wire,ref mut at);assert(count>=8);let coordinate=field(wire,ref mut at);let coords=owner_coordinate(coordinate);var usize i=0;
    while(i<count){let source=field(wire,ref mut at);let expected_result=field(wire,ref mut at);let expected_parameter=field(wire,ref mut at);let expected_field=field(wire,ref mut at);let expected_bytes=field(wire,ref mut at);let expected_hash=field(wire,ref mut at);let p=context(source);let named=names(ref p);let empty=identities::empty();let coordinates=coordinates();var usize field_id=0;
        if(expected_field.len!=0){var usize j=0;while(j<typed::field_count(ref p.graph)){let f=typed::field_at(ref p.graph,j);if(bytes::equal(typed::field_name(ref p.graph,j),b"action")){field_id=f.type;}j+=1;}assert(field_id!=0 && field_id==result(ref p));}
        var u32 mode=0;while(mode<5){var usize work=0;equal(render(ref p,ref named,ref coordinates,result(ref p),mode,ref mut work),expected_result);assert(work>0 && work<=1048576);work=0;equal(render(ref p,ref named,ref coordinates,parameter(ref p),mode,ref mut work),expected_parameter);if(field_id!=0){work=0;equal(render(ref p,ref named,ref coordinates,field_id,mode,ref mut work),expected_field);}mode+=1;}
        if(bytes::equal(expected_result,b"fn i32(i32)")){let t=typed::type_at(ref p.graph,result(ref p));assert(t.kind==23 && t.scalar==0 && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0);var usize work=0;equal(render(ref p,ref empty,ref coordinates,result(ref p),0,ref mut work),expected_result);assert(work==13);}
        descriptor_matches_go(ref p,ref named,ref coords,expected_bytes,expected_hash);
        i+=1;
    }assert(at==wire.len);
}

@test fn void closure_public_text_exact_work_and_errors(){
    let p=context(b"module sample;pub fn fn i32(i32) keep(fn i32(i32) action){return action;}");let named=names(ref p);let coordinates=coordinates();let id=result(ref p);var u32 mode=0;
    while(mode<5){var usize cost=0;equal(render(ref p,ref named,ref coordinates,id,mode,ref mut cost),b"fn i32(i32)");if(mode!=3){assert(cost==13);}var usize work=1048576-cost;equal(render(ref p,ref named,ref coordinates,id,mode,ref mut work),b"fn i32(i32)");assert(work==1048576);
        work=1048576-cost+1;error(render(ref p,ref named,ref coordinates,id,mode,ref mut work),10);assert(work==1048565);
        work=1048576;error(render(ref p,ref named,ref coordinates,id,mode,ref mut work),10);assert(work==1048576);
        work=1048577;error(render(ref p,ref named,ref coordinates,id,mode,ref mut work),10);assert(work==1048577);mode+=1;
    }
    var usize work=0;error(raw(ref p,typed::type_records(ref p.graph),0,false,ref mut work),9);assert(work==0);error(raw(ref p,typed::type_records(ref p.graph),typed::type_count(ref p.graph)+1,false,ref mut work),9);assert(work==0);
    let empty=identities::empty();error(text::render_records_prepared(ref empty,ref p.sources,ref p.metadata,typed::type_records(ref p.graph),ref coordinates,512,id,false,ref mut work),9);assert(work==0);
    let missing=v::empty<syntax::Parsed>();error(text::render_records_prepared(ref empty,ref missing,ref p.metadata,typed::type_records(ref p.graph),ref coordinates,0,id,false,ref mut work),9);assert(work==0);
    var too_many=copy_graph(ref p);while(types::count(ref too_many)<=4096){assert(v::push<types::Type>(ref mut too_many,node(1,1,0,0,0,0)).is_ok);}error(raw(ref p,ref too_many,id,false,ref mut work),9);assert(work==0);
    // Existing depth/error precedence: the 64th edge fails before a leaf visit.
    var deep=copy_graph(ref p);var usize tail=id;var usize depth=0;while(depth<63){assert(v::push<types::Type>(ref mut deep,node(5,0,0,0,tail,0)).is_ok);tail=types::count(ref deep);depth+=1;}work=0;error(raw(ref p,ref deep,tail,false,ref mut work),9);assert(work==568);
    assert(v::push<types::Type>(ref mut deep,node(5,0,0,0,tail,0)).is_ok);work=0;error(raw(ref p,ref deep,types::count(ref deep),false,ref mut work),10);assert(work==576);
}

@test fn void closure_public_text_rejects_forged_payloads_and_wrappers(){
    let p=context(b"module sample;pub fn fn i32(i32) keep(fn i32(i32) action){return action;}");let id=result(ref p);var usize i=0;
    while(i<5){var graph=copy_graph(ref p);let t=v::at_mut<types::Type>(ref mut graph,id-1);if(i==0){t.scalar=1;}else if(i==1){t.module_index=1;}else if(i==2){t.declaration=1;}else if(i==3){t.first=id;}else{t.second=id;}var usize work=0;error(raw(ref p,ref graph,id,false,ref mut work),9);assert(work==1);work=1048575;error(raw(ref p,ref graph,id,true,ref mut work),9);assert(work==1048576);work=1048576;error(raw(ref p,ref graph,id,true,ref mut work),10);assert(work==1048576);i+=1;}
    var graph=copy_graph(ref p);let scalar=types::canonical(ref mut graph,node(1,1,0,0,0,0));var u32 kind=5;
    while(kind<=22){if(kind<=8 || kind==12 || kind==13 || kind==16 || kind==17 || kind==18 || kind==22){let usize second=if(kind==6){scalar}else if(kind==16){2}else{0};assert(v::push<types::Type>(ref mut graph,node(kind,0,0,0,id,second)).is_ok);let wrapper=types::count(ref graph);var usize work=0;error(raw(ref p,ref graph,wrapper,false,ref mut work),if(kind==13){4503}else{9});assert(work>0 && work<64);discard v::pop<types::Type>(ref mut graph);
        if(kind==6){assert(v::push<types::Type>(ref mut graph,node(6,0,0,0,scalar,id)).is_ok);work=0;error(raw(ref p,ref graph,types::count(ref graph),true,ref mut work),9);discard v::pop<types::Type>(ref mut graph);}
    }kind+=1;}
    // An outer wrapper must not turn a nested closure or forged leaf into text.
    assert(v::push<types::Type>(ref mut graph,node(5,0,0,0,id,0)).is_ok);let inner=types::count(ref graph);assert(v::push<types::Type>(ref mut graph,node(6,0,0,0,scalar,inner)).is_ok);var usize work=0;error(raw(ref p,ref graph,types::count(ref graph),false,ref mut work),9);
}

@test fn void closure_public_text_keeps_source_and_name_authentication(){
    let p=context(b"module sample;pub struct Leaf{i32 value;}pub fn fn i32(i32) keep(fn i32(i32) action){return action;}");let named=names(ref p);let coordinates=coordinates();let id=result(ref p);var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module sample;pub struct Leaf{i32 value;}pub fn fn i32(i32) keep(fn i32(i32) action){return action;} // changed")).is_ok);
    var usize work=0;error(text::render_identified(ref named,ref sources,ref p.metadata,ref p.graph,ref coordinates,0,id,ref mut work),12);
    var usize nominal=1;while(nominal<=typed::type_count(ref p.graph) && typed::type_at(ref p.graph,nominal).kind!=2){nominal+=1;}assert(nominal<=typed::type_count(ref p.graph));work=0;error(text::render(ref sources,ref p.metadata,ref p.graph,ref coordinates,0,nominal,ref mut work),12);
    // Raw/prepared helpers retain their documented text-only authority. Builtins
    // need no source spelling; callers still authenticate before publication.
    let empty=identities::empty();work=0;equal(text::render_records_prepared(ref empty,ref sources,ref p.metadata,typed::type_records(ref p.graph),ref coordinates,0,id,false,ref mut work),b"fn i32(i32)");assert(work==13);
    work=0;error(text::render_prepared(ref empty,ref p.sources,ref p.metadata,ref p.graph,ref coordinates,0,id,ref mut work),9);assert(work==0);
    var rows=v::empty<identities::Row>();var usize i=1;while(i<=typed::type_count(ref p.graph)){var t=typed::type_at(ref p.graph,i);if(i==id){t.scalar=1;}assert(v::push<identities::Row>(ref mut rows,identities::Row{.type=t,.owner=0,.name=model::Text{.start=0,.end=0}}).is_ok);i+=1;}
    var stamps=v::empty<identities::Stamp>();assert(v::push<identities::Stamp>(ref mut stamps,identities::Stamp{.owner=0,.digest=signatures::source_digest(ref p.metadata,0)}).is_ok);let forged=identities::assemble(bytes::empty(),rows,stamps);work=0;error(text::render_identified(ref forged,ref p.sources,ref p.metadata,ref p.graph,ref coordinates,0,id,ref mut work),9);
}
