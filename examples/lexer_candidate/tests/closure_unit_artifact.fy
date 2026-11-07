module tests::closure_unit_artifact;
import module_unit_body_ir_model as ir;
import tests::module_body_artifact_sample as scalar_sample;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_verify as proof;
import module_unit_body_ir_closures as closures;
import module_unit_cleanup_mir_model as clean;
import module_unit_checked_bodies as checked;
import module_unit_interfaces as interfaces;
import module_unit_sdk_sources as sdk;
import module_state_layout as state;
import module_signatures as signatures;
import module_types as types;
import module_cleanup_artifact_encode as ce;
import module_cleanup_artifact_decode as cd;
import module_body_artifact_encode as be;
import module_body_artifact_decode as bd;
import candidate_artifact_wire as wire;
import copied_closure_tables as private;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

pub fn Result<ir::Program,u32> from_source(ref u8[] text){
    let data=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{return .err(11);}};
    let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{return .err(1001);}};
    let source=switch(syntax::parse(stream,512)){case .ok(s):{s}case .err(_):{return .err(1001);}};
    if(syntax::problem_count(ref source)!=0){return .err(syntax::problem_at(ref source,0).code);}var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,source).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let owned=switch(checked::build(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    return switch(lower::lower_checked(ref sources,ref owned)){case .ok(p):{.ok(p)}case .err(e):{.err(e.code)}};
}

fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{assert(false);copied(raw)}};}
fn u8 octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn void word(ref mut bytes::Buffer raw,usize at,usize value){var usize n=value;var usize i=0;assert(at+8<=bytes::len(raw));while(i<8){bytes::view_mut(raw)[at+7-i]=octet(n%256);n/=256;i+=1;}assert(n==0);}
fn usize local_offset(ref ir::Body b){return 64+ir::node_count(b)*344+ir::statement_count(b)*96;}
// Change serialized records, never the immutable original Program. Mode 6 also
// covers returned-only factories/copies: it does not depend on an invocation.
fn bytes::Buffer forged(ref u8[] raw,ref ir::Program p,usize frame,u32 mode){
    var out=copied(raw);let b=ir::at(p,frame);var usize i=0;
    while(i<ir::node_count(b)){let n=ir::node_at(b,i);let at=56+i*344;
        if(n.value.kind==36 && (mode==1 || mode==2)){word(ref mut out,at+72,if(mode==1){0}else{n.value.target+1});}
        if(mode==3 && n.syntax.token>private::snapshot_flag()){word(ref mut out,at+280,private::snapshot_flag());}i+=1;
    }
    i=0;while(i<ir::call_count(b)){let c=ir::call_at(b,i);let at=local_offset(b)+8+ir::local_count(b)*144+8+i*88;
        if(c.kind==4 && mode==4){word(ref mut out,at+80,0);}if(c.kind==4 && mode==5){word(ref mut out,at+32,1);}i+=1;
    }
    i=1;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(mode==6 && l.kind==2 && closures::leaf(ir::cleanup_ir(p),l.type)){word(ref mut out,local_offset(b)+8+(i-1)*144+32,0);}i+=1;}return out;
}
fn Result<ir::Program,u32> restored(ref ir::Program p,u32 mode,bool old){
    let closure=clean::closures(ir::cleanup_ir(p));var usize work=0;
    let packet=switch(if(closure){ce::encode_closures(ir::cleanup_ir(p),ref mut work)}else{ce::encode(ir::cleanup_ir(p),ref mut work)}){case .ok(b):{b}case .err(e):{return .err(e);}};work=0;
    let cleaned=switch(if(closure){cd::decode_closures(wire::content(ref packet),ref mut work)}else{cd::decode(wire::content(ref packet),ref mut work)}){case .ok(c):{c}case .err(e):{return .err(e);}};
    assert(clean::work(ref cleaned)==0);var bodies=v::empty<ir::Body>();var usize i=1;
    while(i<=ir::count(p)){work=0;let data=switch(if(closure){be::encode_closures(p,i,ref mut work)}else{be::encode(p,i,ref mut work)}){case .ok(b):{b}case .err(e):{return .err(e);}};
        let bad=forged(wire::content(ref data),p,i,mode);if(closure && !bytes::equal(bytes::view(ref bad),wire::content(ref data))){assert(body_rejected(bytes::view(ref bad),p,i)==1001);}work=0;let body=switch(if(closure){bd::decode_closures_component(bytes::view(ref bad),ref cleaned,i,ref mut work)}else{bd::decode(bytes::view(ref bad),ref cleaned,i,ref mut work)}){case .ok(b):{b}case .err(e):{return .err(e);}};
        assert(v::push<ir::Body>(ref mut bodies,body).is_ok);i+=1;
    }
    var layouts=v::empty<ir::Layout>();var fields=v::empty<interfaces::Field>();var callables=v::empty<interfaces::Callable>();var parameters=v::empty<usize>();i=0;
    while(i<ir::layout_count(p)){assert(v::push<ir::Layout>(ref mut layouts,ir::layout_at(p,i)).is_ok);i+=1;}i=0;
    while(i<ir::field_count(p)){assert(v::push<interfaces::Field>(ref mut fields,ir::field_at(p,i)).is_ok);i+=1;}i=0;
    while(i<ir::callable_count(p)){var c=ir::callable_at(p,i);if(mode==7){c.unsafe_call=true;}assert(v::push<interfaces::Callable>(ref mut callables,c).is_ok);i+=1;}i=0;
    while(i<ir::parameter_count(p)){assert(v::push<usize>(ref mut parameters,ir::parameter_at(p,i)).is_ok);i+=1;}
    // These original-source unit controls contain no state or SDK owners.
    assert(state::count(ir::state_layouts(p))==0 && sdk::count(ir::sdk_units(p))==0);let names=copied(ir::names_view(p));work=0;
    let result=if(!old && (closure || mode==8)){ir::assemble_artifact_closures(state::empty(),sdk::empty(),cleaned,bodies,layouts,fields,callables,parameters,names,ref mut work)}else{ir::assemble_artifact(state::empty(),sdk::empty(),cleaned,bodies,layouts,fields,callables,parameters,names,ref mut work)};
    return switch(result){case .ok(p):{assert(ir::work(ref p)==0 && work!=0);.ok(p)}case .err(e):{.err(e.code)}};
}
fn void equal_components(ref ir::Program a,ref ir::Program b){
    let closure=clean::closures(ir::cleanup_ir(a));var usize work=0;let x=switch(if(closure){ce::encode_closures(ir::cleanup_ir(a),ref mut work)}else{ce::encode(ir::cleanup_ir(a),ref mut work)}){case .ok(x):{x}case .err(_):{assert(false);return;}};work=0;let y=switch(if(closure){ce::encode_closures(ir::cleanup_ir(b),ref mut work)}else{ce::encode(ir::cleanup_ir(b),ref mut work)}){case .ok(y):{y}case .err(_):{assert(false);return;}};assert(bytes::equal(wire::content(ref x),wire::content(ref y)));var usize i=1;
    while(i<=ir::count(a)){work=0;let x=switch(if(closure){be::encode_closures(a,i,ref mut work)}else{be::encode(a,i,ref mut work)}){case .ok(x):{x}case .err(_):{assert(false);return;}};work=0;let y=switch(if(closure){be::encode_closures(b,i,ref mut work)}else{be::encode(b,i,ref mut work)}){case .ok(y):{y}case .err(_):{assert(false);return;}};assert(bytes::equal(wire::content(ref x),wire::content(ref y)));i+=1;}
}
@effects(console) pub fn void verify(ref u8[] name,ref u8[] text,usize constructs,usize invocations,usize captures){
    let p=switch(from_source(text)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(name);discard io::write_stderr_u32(e);assert(false);return;}};
    let fresh=switch(restored(ref p,0,false)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(name);discard io::write_stderr_u32(e);assert(false);return;}};assert(proof::check(ref fresh).code==0);equal_components(ref p,ref fresh);
    let closure=clean::closures(ir::cleanup_ir(ref p));var usize work=0;
    if(closure){assert(switch(ce::encode(ir::cleanup_ir(ref p),ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});assert(switch(be::encode(ref p,1,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});assert(switch(restored(ref p,0,true)){case .ok(_):{false}case .err(e):{e==1001}});
        var bool initialized=false;var usize i=1;while(i<=ir::count(ref p)){let b=ir::at(ref p,i);var usize j=1;while(j<=ir::local_count(b)){let l=ir::local_at(b,j);initialized=initialized || l.kind==2 && closures::leaf(ir::cleanup_ir(ref p),l.type);j+=1;}i+=1;}
        var u32 mode=1;while(mode<=7){let applicable=if(mode<=2 || mode==7){constructs!=0}else if(mode==3){captures!=0}else if(mode<=5){invocations!=0}else{initialized};if(applicable){assert(switch(restored(ref p,mode,false)){case .ok(_):{false}case .err(e):{e==1001}});}mode+=1;}
        controls(ref p);
    }else{assert(switch(restored(ref p,8,false)){case .ok(_):{false}case .err(e):{e==1001}});assert(switch(ce::encode_closures(ir::cleanup_ir(ref p),ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});assert(switch(be::encode_closures(ref p,1,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});}
}

fn u32 clean_rejected(ref u8[] raw){var usize work=0;return switch(cd::decode_closures(raw,ref mut work)){case .ok(_):{0}case .err(e):{e}};}
fn u32 body_rejected(ref u8[] raw,ref ir::Program p,usize frame){var usize work=0;return switch(bd::decode_closures(raw,p,frame,ref mut work)){case .ok(_):{0}case .err(e):{e}};}
fn void controls(ref ir::Program p){
    var usize cost=0;let packet=switch(ce::encode_closures(ir::cleanup_ir(p),ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};let raw=wire::content(ref packet);assert(bytes::equal(raw[..8],b"FYCLEAN2"));var usize work=16777216-cost;
    assert(switch(ce::encode_closures(ir::cleanup_ir(p),ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==16777216);work=16777216-cost+1;assert(switch(ce::encode_closures(ir::cleanup_ir(p),ref mut work)){case .ok(_):{false}case .err(e):{e==1002}});
    cost=0;assert(switch(cd::decode_closures(raw,ref mut cost)){case .ok(_):{true}case .err(_):{false}});work=16777216-cost;assert(switch(cd::decode_closures(raw,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==16777216);work=16777216-cost+1;assert(switch(cd::decode_closures(raw,ref mut work)){case .ok(_):{false}case .err(e):{e==1002}});
    assert(clean_rejected(raw[..raw.len-1])!=0);var bad=copied(raw);assert(bytes::append_byte(ref mut bad,0).is_ok && clean_rejected(bytes::view(ref bad))==1001);
    var u32 mode=0;while(mode<8){bad=copied(raw);if(mode<=1){bytes::view_mut(ref mut bad)[7]=if(mode==0){48}else{49};}else if(mode==2){word(ref mut bad,8,2);}else if(mode==3){word(ref mut bad,16,2);}else if(mode==4){word(ref mut bad,24,4097);}else if(mode==5){bytes::view_mut(ref mut bad)[32]=1;}else{var usize i=1;while(i<=clean::type_count(ir::cleanup_ir(p))){if(clean::type_at(ir::cleanup_ir(p),i).kind==23){word(ref mut bad,32+(i-1)*48+if(mode==6){8}else{0},if(mode==6){1}else{19});}i+=1;}}assert(clean_rejected(bytes::view(ref bad))==if(mode==4){1002}else{1001});mode+=1;}
    work=0;assert(switch(cd::decode(raw,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});work=0;assert(switch(cd::decode_aggregates(raw,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});
    var usize frame=1;while(frame<=ir::count(p)){cost=0;let packet=switch(be::encode_closures(p,frame,ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};let raw=wire::content(ref packet);assert(bytes::equal(raw[..8],b"FYBODY01"));work=33554432-cost;assert(switch(be::encode_closures(p,frame,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==33554432);work=33554432-cost+1;assert(switch(be::encode_closures(p,frame,ref mut work)){case .ok(_):{false}case .err(e):{e==1002}});
        cost=0;assert(switch(bd::decode_closures(raw,p,frame,ref mut cost)){case .ok(_):{true}case .err(_):{false}});work=33554432-cost;assert(switch(bd::decode_closures(raw,p,frame,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==33554432);work=33554432-cost+1;assert(switch(bd::decode_closures(raw,p,frame,ref mut work)){case .ok(_):{false}case .err(e):{e==1002}});
        // Rejection must remain an error, even where flag+untrusted ID would
        // overflow usize. Test the last safe sum, first overflow and maximum.
        let usize maximum=18446744073709551615;let b=ir::at(p,frame);var usize n=0;
        while(n<ir::node_count(b)){let row=ir::node_at(b,n);var usize edge=0;while(edge<3){
            if(row.value.kind==36){bad=copied(raw);word(ref mut bad,56+n*344+72,if(edge==2){maximum}else{maximum-private::helper_flag()+edge});assert(body_rejected(bytes::view(ref bad),p,frame)==1001);}
            if(row.value.kind==2 && row.syntax.token>private::snapshot_flag()){assert(row.value.binding!=0);bad=copied(raw);word(ref mut bad,local_offset(b)+8+(row.value.binding-1)*144+24,if(edge==2){maximum}else{maximum-private::snapshot_flag()+edge});assert(body_rejected(bytes::view(ref bad),p,frame)==1001);}
            edge+=1;}n+=1;}
        assert(body_rejected(raw[..raw.len-1],p,frame)!=0);bad=copied(raw);assert(bytes::append_byte(ref mut bad,0).is_ok && body_rejected(bytes::view(ref bad),p,frame)==1001);bad=copied(raw);bytes::view_mut(ref mut bad)[7]=48;assert(body_rejected(bytes::view(ref bad),p,frame)==1001);bad=copied(raw);word(ref mut bad,8,frame+1);assert(body_rejected(bytes::view(ref bad),p,frame)==1001);bad=copied(raw);word(ref mut bad,48,16385);assert(body_rejected(bytes::view(ref bad),p,frame)==1002);work=0;assert(switch(bd::decode(raw,ir::cleanup_ir(p),frame,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});frame+=1;
    }
}
fn Result<clean::Program,clean::Problem> minimal(bool closure,bool project,bool aggregates,bool leaf){
    var graph=v::empty<types::Type>();var glues=v::empty<clean::Glue>();assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=1,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);if(leaf){assert(v::push<types::Type>(ref mut graph,types::Type{.kind=23,.scalar=0,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);}var usize i=0;while(i<v::len<types::Type>(ref graph)){assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);i+=1;}var usize work=0;
    return if(closure){clean::assemble_closure_artifact(graph,v::empty<clean::Frame>(),glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),project,aggregates,ref mut work)}else if(aggregates){clean::assemble_aggregate_artifact(graph,v::empty<clean::Frame>(),glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),project,ref mut work)}else{clean::assemble_artifact(graph,v::empty<clean::Frame>(),glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),project,ref mut work)};
}
// Independent byte oracle: one scalar, optional canonical leaf, zero glue and
// count words. No serializer is used to build the expected record payload.
pub fn void families(){var u32 mode=0;while(mode<8){let closure=(mode&4)!=0;let project=(mode&2)!=0;let aggregates=(mode&1)!=0;let p=switch(minimal(closure,project,aggregates,closure)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;let packet=switch(if(closure){ce::encode_closures(ref p,ref mut work)}else{ce::encode(ref p,ref mut work)}){case .ok(p):{p}case .err(_):{assert(false);return;}};var expected=bytes::empty();let usize size=if(closure){272}else{160};assert(bytes::resize_zeroed(ref mut expected,size).is_ok);let marker=if(closure){b"FYCLEAN2"}else if(aggregates){b"FYCLEAN1"}else{b"FYCLEAN0"};var usize i=0;while(i<8){bytes::view_mut(ref mut expected)[i]=marker[i];i+=1;}word(ref mut expected,8,if(project){1}else{0});let usize count_at=if(closure){24}else{16};if(closure){word(ref mut expected,16,if(aggregates){1}else{0});}word(ref mut expected,count_at,if(closure){2}else{1});word(ref mut expected,count_at+8,1);word(ref mut expected,count_at+16,1);if(closure){word(ref mut expected,count_at+56,23);}let raw=wire::content(ref packet);assert(bytes::equal(raw,bytes::view(ref expected)));work=0;
        let fresh=switch(if(closure){cd::decode_closures(raw,ref mut work)}else if(aggregates){cd::decode_aggregates(raw,ref mut work)}else{cd::decode(raw,ref mut work)}){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(clean::project(ref fresh)==project && clean::aggregates(ref fresh)==aggregates && clean::closures(ref fresh)==closure && clean::work(ref fresh)==0);
        if(!closure){work=0;assert(switch(cd::decode_closures(raw,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});work=0;assert(switch(ce::encode_closures(ref p,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});}mode+=1;
    }
    assert(switch(minimal(true,false,false,false)){case .ok(_):{false}case .err(e):{e.code==1001}});assert(switch(minimal(false,false,false,true)){case .ok(_):{false}case .err(e):{e.code==1001}});
}

pub fn void scalar_golden(ref u8[] expected){let p=scalar_sample::build();var usize work=0;let packet=switch(be::encode(ref p,1,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let raw=wire::content(ref packet);assert(bytes::equal(raw,expected));work=0;assert(switch(bd::decode(raw,ir::cleanup_ir(ref p),1,ref mut work)){case .ok(_):{true}case .err(_):{false}});work=0;assert(switch(bd::decode_closures(raw,ref p,1,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});let fresh=switch(restored(ref p,0,false)){case .ok(p):{p}case .err(_):{assert(false);return;}};equal_components(ref p,ref fresh);}
