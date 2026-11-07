module tests::closure_project_artifact;
import project_compiler_core as core;
import module_project_artifact_encode as encode;
import module_project_artifact_decode as decode;
import module_project_artifact_model as artifact;
import module_program_artifact_wire as budget;
import candidate_artifact_wire as wire;
import module_project_identified_ir as identified;
import module_project_interfaces as interfaces;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_body_ir_closures as closures;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_names as names;
import module_unit_c_emit as emit;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import sha256_stream as sha;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

// This producer enters the existing original-source Complete factory. It never
// manufactures a detached project, changes mode flags or supplies a Go graph.
pub struct Failure {u32 phase;u32 stage;u32 code;usize module_index;usize start;usize end;}
resource Input {v::Vec<syntax::Parsed> sources;v::Vec<usize> owners;v::Vec<catalog::Binding> imports;coordinates::Coordinates table;cleanup{}}
resource Owned {core::Complete complete;usize entry;cleanup{}}
fn Failure failed(u32 phase,u32 code,usize module_index,usize start,usize end){return Failure{.phase,.stage=0,.code,.module_index,.start,.end};}
fn Result<syntax::Parsed,Failure> parsed(ref u8[] text,usize module_index){
    let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{return .err(failed(1,11,module_index,0,0));}};
    let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return .err(failed(1,9,module_index,0,text.len));}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{return .err(failed(1,9,module_index,0,text.len));}};
    if(syntax::problem_count(ref p)!=0){let e=syntax::problem_at(ref p,0);return .err(failed(1,e.code,module_index,e.start,e.end));}return .ok(p);
}
fn Result<Input,Failure> original(ref u8[] root,ref u8[] first,ref u8[] second){
    var sources=v::empty<syntax::Parsed>();var owners=v::empty<usize>();var imports=v::empty<catalog::Binding>();var usize m=0;
    while(m<3){let text=if(m==0){root}else if(m==1){first}else{second};if(text.len!=0){let p=switch(parsed(text,m)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!v::push<syntax::Parsed>(ref mut sources,p).is_ok || !v::push<usize>(ref mut owners,0).is_ok){return .err(failed(1,11,m,0,0));}}m+=1;}
    let declarations=switch(catalog::build(ref sources)){case .ok(p):{p}case .err(e):{return .err(failed(1,e.code,e.module_index,e.start,e.end));}};
    m=0;while(m<v::len<syntax::Parsed>(ref sources)){let p=v::at<syntax::Parsed>(ref sources,m);var usize d=1;while(d<syntax::declaration_count(p)){let row=syntax::declaration_at(p,d);if(row.kind==2){var usize probes=0;let alias=syntax::source_view(p)[row.name_start..row.name_end];let resolved=catalog::import_target_measured(ref declarations,m,alias,ref mut probes);if(resolved.code!=0 || resolved.item==0){return .err(failed(1,9,m,row.start,row.end));}if(!v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=m,.declaration=d+1,.target=resolved.item-1}).is_ok){return .err(failed(1,11,m,row.start,row.end));}}d+=1;}m+=1;}
    // A fixed private package identity, independent of source content. Source
    // stamps and interface hashes are derived by production owners separately.
    var digest=sha::empty();var text=bytes::empty();if(!sha::update(ref mut digest,b"closure-project-c-original-source-fixture/v1") || !bytes::append(ref mut text,b"fypc1-sha256-").is_ok || !sha::append_hex(ref digest,ref mut text)){return .err(failed(1,11,0,0,0));}
    var rows=v::empty<coordinates::Coordinate>();if(!v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=bytes::len(ref text)}).is_ok){return .err(failed(1,11,0,0,0));}return .ok(Input{.sources,.owners,.imports,.table=coordinates::assemble(text,rows)});
}
fn wire::Payload encoded(ref core::Complete p){var usize work=0;return switch(encode::encode_closures(core::checked_program(p),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);encoded(p)}};}
fn Result<core::Complete,Failure> restored(ref u8[] raw,ref Input input){
    var usize source=0;var usize artifact_work=0;var usize link=0;
    let saved=switch(decode::decode_closures(raw,ref mut artifact_work)){case .ok(p):{p}case .err(e):{return .err(failed(30,e,0,0,0));}};
    return switch(core::restore_complete_closure_sources(saved,ref input.sources,ref input.owners,ref input.imports,ref input.table,false,ref mut source,ref mut artifact_work,ref mut link)){case .ok(p):{assert(source!=0 && artifact_work!=0 && link!=0);.ok(p)}case .err(e):{var error=failed(40+e.phase,e.code,e.module_index,e.start,e.end);error.stage=e.stage;return .err(error);}};
}
fn void envelope(ref u8[] raw){
    assert(bytes::equal(raw[..8],b"FYPRJ002"));var usize work=0;assert(decode::preflight_closures(raw,ref mut work)==0);work=0;assert(decode::preflight(raw,ref mut work)==1001);work=0;assert(decode::preflight_aggregates(raw,ref mut work)==1001);
    var usize cost=0;let owner=switch(decode::decode_closures(raw,ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(proof::check(artifact::program(ref owner)).code==0 && clean::project(ir::cleanup_ir(artifact::program(ref owner))));
    work=0;let again=switch(encode::restored_closures(ref owner,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(raw,wire::content(ref again)));work=0;assert(!encode::restored(ref owner,ref mut work).is_ok);
    work=budget::work_limit()-cost;assert(decode::decode_closures(raw,ref mut work).is_ok && work==budget::work_limit());work=budget::work_limit()-cost+1;assert(switch(decode::decode_closures(raw,ref mut work)){case .ok(_):{false}case .err(e):{e==1002}});
    work=0;assert(!decode::decode_closures(raw[..raw.len-1],ref mut work).is_ok);var bad=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::append_byte(ref mut bad,0).is_ok);work=0;assert(switch(decode::decode_closures(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});
    bytes::view_mut(ref mut bad)[7]=49;work=0;assert(!decode::decode_closures(bytes::view(ref bad),ref mut work).is_ok);bytes::view_mut(ref mut bad)[7]=50;bytes::view_mut(ref mut bad)[23]=49;work=0;assert(!decode::decode_closures(bytes::view(ref bad),ref mut work).is_ok);
}
pub fn void framing(){
    // Independent outer framing oracle, not produced by either codec.
    let scalar=b"FYPRJ000\x00\x00\x00\x00\x00\x00\x00\x08FYPROG00\x00\x00\x00\x00\x00\x00\x00\x08FYPROV00\x00\x00\x00\x00\x00\x00\x00\x08FYIDS000";
    let closed=b"FYPRJ002\x00\x00\x00\x00\x00\x00\x00\x08FYPROG02\x00\x00\x00\x00\x00\x00\x00\x08FYPROV00\x00\x00\x00\x00\x00\x00\x00\x08FYIDS000";
    var usize work=0;assert(scalar.len==56 && closed.len==56 && decode::preflight(scalar,ref mut work)==0);work=0;assert(decode::preflight_closures(closed,ref mut work)==0);var usize i=0;while(i<closed.len){work=0;assert(decode::preflight_closures(closed[..i],ref mut work)!=0);i+=1;}
    work=budget::work_limit()+1;assert(switch(decode::decode_closures(closed,ref mut work)){case .ok(_):{false}case .err(e):{e==1002}} && work==budget::work_limit()+1);
    let input=switch(original(b"module main;fn i32 main(){return 42;}",b"",b"")){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize source=0;var usize link=0;let complete=switch(core::build_complete_sources(ref input.sources,ref input.owners,ref input.imports,ref input.table,ref mut source,ref mut link)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    work=0;let packet=switch(encode::encode(core::checked_program(ref complete),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let raw=wire::content(ref packet);assert(bytes::equal(raw[..8],b"FYPRJ000"));work=0;let saved=switch(decode::decode(raw,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let again=switch(encode::restored(ref saved,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(raw,wire::content(ref again)));work=0;assert(!decode::decode_closures(raw,ref mut work).is_ok);work=0;assert(!encode::encode_closures(core::checked_program(ref complete),ref mut work).is_ok);
}
pub fn void forged(){
    let a=switch(original(b"module main;fn i32 main(){let seed=40;let fn i32(i32) action=fn [i32 seed] i32(i32 x){return seed+x;};return action(2);}",b"",b"")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let b=switch(original(b"module main;fn i32 main(){let seed=40;let fn i32(i32) action=fn [i32 seed] i32(i32 x){return seed-x;};return action(2);}",b"",b"")){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize source=0;var usize link=0;
    let first=switch(core::build_complete_sources(ref a.sources,ref a.owners,ref a.imports,ref a.table,ref mut source,ref mut link)){case .ok(p):{p}case .err(_):{assert(false);return;}};source=0;link=0;let second=switch(core::build_complete_sources(ref b.sources,ref b.owners,ref b.imports,ref b.table,ref mut source,ref mut link)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let old=encoded(ref first);let changed=encoded(ref second);let raw=wire::content(ref old);let other=wire::content(ref changed);var usize work=0;var c=switch(budget::cursor_marker(raw,b"FYPRJ002",ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let old_program=budget::member_from(raw,ref mut c,budget::byte_limit(),b"FYPROG02",ref mut work);
    var d=switch(budget::cursor_marker(other,b"FYPRJ002",ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let new_program=budget::member_from(other,ref mut d,budget::byte_limit(),b"FYPROG02",ref mut work);var budget::Status status=budget::Status{.code=0};var packet=budget::writer_marker(b"FYPRJ002",ref mut status,ref mut work);budget::member(ref mut packet,other[new_program.start..new_program.end],ref mut status,ref mut work);assert(status.code==0 && bytes::append(ref mut packet,raw[old_program.end..]).is_ok);
    // The replacement graph is independently valid and the saved exact source
    // stamp is still A. Only source-to-helper recipes can reject this splice.
    work=0;assert(decode::decode_closures(bytes::view(ref packet),ref mut work).is_ok);assert(!restored(bytes::view(ref packet),ref a).is_ok);
}
fn Result<Owned,Failure> retained(ref Input input,usize constructs,bool same_declaration){
    var usize source_work=0;var usize link_work=0;
    let cold=switch(core::build_complete_sources(ref input.sources,ref input.owners,ref input.imports,ref input.table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{var error=failed(2,e.code,e.module_index,e.start,e.end);error.stage=e.stage;error.phase=10+e.phase;return .err(error);}};
    let data=encoded(ref cold);let raw=wire::content(ref data);envelope(raw);
    var usize old_work=0;assert(!encode::encode(core::checked_program(ref cold),ref mut old_work).is_ok);
    let complete=switch(restored(raw,input)){case .ok(p):{p}case .err(e):{return .err(e);}};let again=encoded(ref complete);assert(bytes::equal(raw,wire::content(ref again)));
    assert(!core::is_state_graph(ref complete) && source_work!=0 && link_work!=0 && interfaces::count(core::public_interfaces(ref complete))==1);
    let project=core::checked_program(ref complete);let p=identified::ir_program(project);let cleaned=ir::cleanup_ir(p);
    assert(clean::project(cleaned) && clean::closures(cleaned) && !clean::aggregates(cleaned) && proof::check(p).code==0 && identified::frame_count(project)==ir::count(p));
    var usize entry=0;var usize made=0;var usize first_declaration=0;var usize first_module=0;var bool shared_number=false;var usize f=1;
    while(f<=ir::count(p)){let owner=clean::frame_at(cleaned,f);let source=v::at<syntax::Parsed>(ref input.sources,owner.module_index);if(owner.module_index==0 && owner.kind==3 && bytes::equal(syntax::declaration_name(source,owner.declaration),b"main")){assert(entry==0);entry=f;}
        let b=ir::at(p,f);var usize n=0;while(n<ir::node_count(b)){let node=ir::node_at(b,n);if(node.value.kind==36){var usize captures=0;var usize tail=node.value.right;while(tail!=0){captures+=1;let at=ir::node(b,tail);assert(at!=0);tail=ir::node_at(b,at-1).value.right;}var usize work=0;let helper=closures::helper(p,f,node.value.target,captures,ref mut work);assert(helper!=0 && clean::frame_at(cleaned,helper).module_index==owner.module_index && clean::frame_at(cleaned,helper).declaration==node.value.target);if(made==0){first_declaration=node.value.target;first_module=owner.module_index;}else{shared_number=shared_number || first_declaration==node.value.target && first_module!=owner.module_index;}made+=1;}n+=1;}f+=1;
    }assert(entry!=0 && made==constructs && (!same_declaration || shared_number));return .ok(Owned{.complete,.entry});
}
fn bool contains(ref u8[] text,ref u8[] part){var usize i=0;while(part.len<=text.len && i<=text.len-part.len){if(bytes::equal(text[i..i+part.len],part)){return true;}i+=1;}return false;}
fn bool member(ref mut bytes::Buffer packet,ref u8[] data){return bytes::append_usize(packet,data.len).is_ok && bytes::append(packet,b"\n").is_ok && bytes::append(packet,data).is_ok;}
@effects(console) fn void failure(ref u8[] name,Failure e){discard io::write_stderr("closure-project-c case=");discard io::write_stderr_bytes(name);discard io::write_stderr(" phase=");discard io::write_stderr_u32(e.phase);discard io::write_stderr(" stage=");discard io::write_stderr_u32(e.stage);discard io::write_stderr(" code=");discard io::write_stderr_u32(e.code);discard io::write_stderr(" module=");discard io::write_stderr_usize(e.module_index);discard io::write_stderr(" span=");discard io::write_stderr_usize(e.start);discard io::write_stderr("..");discard io::write_stderr_usize(e.end);discard io::write_stderr("\n");}
@effects(console) pub fn void produce(ref u8[] name,ref u8[] root,ref u8[] first,ref u8[] second,usize constructs,bool same_declaration){
    let input=switch(original(root,first,second)){case .ok(p):{p}case .err(e):{failure(name,e);assert(false);return;}};
    var usize probe_source=0;var usize probe_link=0;
    let cold=switch(core::build_complete_sources(ref input.sources,ref input.owners,ref input.imports,ref input.table,ref mut probe_source,ref mut probe_link)){case .ok(p):{p}case .err(_):{assert(false);return;}};let saved=encoded(ref cold);let raw_saved=wire::content(ref saved);
    var changed=switch(bytes::copy(root)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::append(ref mut changed,b"\n// source-stamp drift\n").is_ok);let stale=switch(original(bytes::view(ref changed),first,second)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(!restored(raw_saved,ref stale).is_ok);
    if(first.len!=0){let reordered=switch(original(first,root,second)){case .ok(p):{p}case .err(_):{assert(false);return;}};let moved=switch(restored(raw_saved,ref reordered)){case .ok(p):{p}case .err(e):{failure(name,e);assert(false);return;}};let p=identified::ir_program(core::checked_program(ref moved));assert(proof::check(p).code==0);var usize f=1;while(f<=ir::count(p)){let b=ir::at(p,f);var usize c=0;while(c<ir::call_count(b)){let call=ir::call_at(b,c);if(call.kind==4){assert(call.target_module==0 && call.target_declaration==0 && call.instance==0);}c+=1;}f+=1;}}
    let owned=switch(retained(ref input,constructs,same_declaration)){case .ok(p):{p}case .err(e):{failure(name,e);assert(false);return;}};
    let project=core::checked_program(ref owned.complete);let raw=identified::ir_program(project);
    // Keep Complete alive through both emissions. The single buffer is another
    // view of this same checked project, never a separately lowered unit.
    var single=bytes::empty();let rendered=emit::render(raw,owned.entry,ref mut single);if(rendered.code!=0){failure(name,failed(3,rendered.code,rendered.module_index,rendered.start,rendered.end));assert(false);return;}
    var usize naming_work=0;let named=switch(names::build_project(project,ref mut naming_work)){case .ok(p):{p}case .err(e):{failure(name,failed(4,e.code,e.module_index,e.start,e.end));assert(false);return;}};
    let split=switch(emit::emit_project_closures(raw,named,owned.entry,1,true)){case .ok(p):{p}case .err(e):{failure(name,failed(5,e.code,e.module_index,e.start,e.end));assert(false);return;}};
    assert(contains(bytes::view(ref single),b"fy_captures[4]") && contains(emit::emission_header(ref split),b"fy_captures[4]") && contains(bytes::view(ref single),b"fy_invoke"));
    var usize adapters=0;var usize i=0;while(i<emit::emission_unit_count(ref split)){if(contains(emit::emission_source(ref split,i),b"(int32_t const *fy_env,int32_t fy_arg)")){adapters+=1;}i+=1;}assert(adapters!=0);
    var packet=bytes::empty();assert(bytes::append(ref mut packet,b"FYCPARTS1\n").is_ok && member(ref mut packet,emit::emission_header(ref split)) && member(ref mut packet,emit::emission_literals(ref split)) && bytes::append_usize(ref mut packet,emit::emission_unit_count(ref split)).is_ok && bytes::append(ref mut packet,b"\n").is_ok);i=0;
    while(i<emit::emission_unit_count(ref split)){assert(member(ref mut packet,emit::emission_source(ref split,i)));i+=1;}
    // Publish a whole case only after Complete and both C outputs succeed.
    var framed=bytes::empty();assert(member(ref mut framed,name) && member(ref mut framed,bytes::view(ref single)) && member(ref mut framed,bytes::view(ref packet)));assert(io::write_bytes(bytes::view(ref framed)).is_ok);
}
