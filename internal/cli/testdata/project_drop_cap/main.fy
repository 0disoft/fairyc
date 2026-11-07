module main;
import module_unit_cleanup_mir_model as clean;
import module_unit_cleanup_mir_verify as proof;
import module_cleanup_artifact_encode as encoder;
import module_cleanup_artifact_decode as decoder;
import candidate_artifact_wire as wire;
import module_project_cleanup as linked;
import module_project_types as graph;
import module_project_frames as frames;
import module_project_metadata as metadata;
import module_unit_body_ir_model as body;
import module_unit_interfaces as interfaces;
import module_unit_sdk_sources as sdk;
import module_state_layout as states;
import module_types as types;
import std::vec as v;
import std::bytes as bytes;
import std::io as io;

// Fresh detached cleanup owners only. Repeated references test structural table
// capacity, not executable duplicate cleanup, body proof, Complete or cache reuse.
fn v::Vec<types::Type> records(){var v::Vec<types::Type> out=v::empty<types::Type>();assert(v::push<types::Type>(ref mut out,types::Type{.kind=1,.scalar=7,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);assert(v::push<types::Type>(ref mut out,types::Type{.kind=3,.scalar=0,.module_index=0,.declaration=1,.first=0,.second=0}).is_ok);return out;}
fn clean::Frame frame(usize unit){return clean::Frame{.kind=3,.context=1,.module_index=unit,.declaration=2,.result=1,.effects=0,.receiver=0,.receiver_type=0};}
fn u32 reason(usize i){let n=i%9;return if(n==0){1}else if(n==1){2}else if(n==2){3}else if(n==3){4}else if(n==4){5}else if(n==5){6}else if(n==6){7}else if(n==7){8}else{9};}
fn Result<clean::Program,clean::Problem> fresh(bool project,bool aggregates,usize count,usize history,usize unit,usize operands,usize localid,bool invalid){
    var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=1,.frame=0,.nonempty=false,.effects=0,.abilities=1,.field_start=0,.field_count=0}).is_ok);
    var v::Vec<clean::Frame> fs=v::empty<clean::Frame>();assert(v::push<clean::Frame>(ref mut fs,frame(unit)).is_ok);
    var v::Vec<clean::Operand> os=v::empty<clean::Operand>();var usize i=0;while(i<operands){assert(v::push<clean::Operand>(ref mut os,clean::Operand{.frame=1,.kind=1,.id=localid,.type=2,.start=10,.end=20}).is_ok);i+=1;}
    var v::Vec<clean::Drop> ds=v::empty<clean::Drop>();i=0;while(i<count){assert(v::push<clean::Drop>(ref mut ds,clean::Drop{.operand=if(invalid && i+1==count){0}else{1},.exit=reason(i),.start=100+i*2,.end=101+i*2}).is_ok);i+=1;}
    return if(aggregates){if(project){clean::assemble_project_aggregates(records(),fs,glues,v::empty<clean::Field>(),os,ds,history)}else{clean::assemble_aggregates(records(),fs,glues,v::empty<clean::Field>(),os,ds,history)}}else{if(project){clean::assemble_project(records(),fs,glues,v::empty<clean::Field>(),os,ds,history)}else{clean::assemble(records(),fs,glues,v::empty<clean::Field>(),os,ds,history)}};
}
fn clean::Program accepted(bool project,bool aggregates,usize count,usize unit){return switch(fresh(project,aggregates,count,0,unit,1,16384,false)){case .ok(p):{p}case .err(_):{assert(false);accepted(project,aggregates,0,unit)}};}
fn u32 refused(bool project,usize count,usize history,usize operands,usize localid){return switch(fresh(project,false,count,history,0,operands,localid,false)){case .ok(_):{0}case .err(e):{e.code}};}
fn wire::Payload encoded(ref clean::Program p,ref mut usize work){return switch(encoder::encode(p,work)){case .ok(p):{p}case .err(_):{assert(false);encoded(p,work)}};}
fn Result<clean::Program,u32> decoded(ref u8[] raw,bool aggregates,ref mut usize work){return if(aggregates){decoder::decode_aggregates(raw,work)}else{decoder::decode(raw,work)};}
fn u32 decode_code(ref u8[] raw,bool aggregates){var usize work=0;return switch(decoded(raw,aggregates,ref mut work)){case .ok(_):{0}case .err(code):{code}};}
fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);copied(b"")}};}
// Independent big-endian scalar writer, not the artifact transport helper.
fn void put(ref mut bytes::Buffer raw,usize offset,usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";var usize rest=value;var usize i=0;let out=bytes::view_mut(raw);while(i<8){out[offset+7-i]=(digits[(rest/16)%16]<<4)|digits[rest%16];rest/=256;i+=1;}assert(rest==0);}
fn void number(ref mut bytes::Buffer out,usize n){let start=bytes::len(out);assert(bytes::resize_zeroed(out,start+8).is_ok);put(out,start,n);}
fn bool same_drop(clean::Drop a,clean::Drop b){return a.operand==b.operand && a.exit==b.exit && a.start==b.start && a.end==b.end;}
fn void transport(ref mut bytes::Buffer out,bool project,bool aggregates,usize n){
    let p=accepted(project,aggregates,n,0);var usize ew=0;let packet=encoded(ref p,ref mut ew);let raw=wire::content(ref packet);assert(raw.len==376+n*32);var usize dw=0;
    let restored=switch(decoded(raw,aggregates,ref mut dw)){case .ok(p):{p}case .err(_):{assert(false);accepted(project,aggregates,0,0)}};
    assert(clean::project(ref restored)==project && clean::aggregates(ref restored)==aggregates && clean::work(ref restored)==0 && clean::drop_count(ref restored)==n && proof::check(ref restored).code==0);
    var usize i=0;while(i<n){assert(same_drop(clean::drop_at(ref p,i),clean::drop_at(ref restored,i)));i+=1;}var usize again_work=0;let again=encoded(ref restored,ref mut again_work);assert(bytes::equal(raw,wire::content(ref again)));
    if(n==32768 && !aggregates){
        // The project flag is decoded before this count preflight. No huge
        // allocation or partially reconstructed Program escapes these refusals.
        var bytes::Buffer bad=copied(raw);put(ref mut bad,368,32769);assert(decode_code(bytes::view(ref bad),false)==1002);put(ref mut bad,368,32768);put(ref mut bad,8,0);assert(decode_code(bytes::view(ref bad),false)==1002);put(ref mut bad,8,2);assert(decode_code(bytes::view(ref bad),false)==1001);
        var usize cut=0;while(cut<10){let usize size=if(cut==0){0}else if(cut==1){7}else if(cut==2){15}else if(cut==3){23}else if(cut==4){231}else if(cut==5){303}else if(cut==6){367}else if(cut==7){375}else if(cut==8){raw.len-32}else{raw.len-1};assert(decode_code(raw[..size],false)==1001);cut+=1;}
        var bytes::Buffer tail=copied(raw);assert(bytes::append_byte(ref mut tail,0).is_ok && decode_code(bytes::view(ref tail),false)==1001);
        var usize measured=0;assert(proof::check_metered(ref p,ref mut measured).code==0);var usize exact=1048576-measured;assert(proof::check_metered(ref p,ref mut exact).code==0 && exact==1048576);exact=1048576-measured+1;assert(proof::check_metered(ref p,ref mut exact).code==1002 && exact<=1048576);
        var usize budget=16777216-ew;let exact_packet=encoded(ref p,ref mut budget);assert(budget==16777216 && bytes::equal(raw,wire::content(ref exact_packet)));budget=16777216-ew+1;assert(switch(encoder::encode(ref p,ref mut budget)){case .ok(_):{false}case .err(code):{code==1002}} && budget<=16777216);
        budget=16777216-dw;assert(switch(decoded(raw,false,ref mut budget)){case .ok(_):{true}case .err(_):{false}} && budget==16777216);budget=16777216-dw+1;assert(switch(decoded(raw,false,ref mut budget)){case .ok(_):{false}case .err(code):{code==1002}} && budget<=16777216);
        let invalid=switch(fresh(true,false,n,0,0,1,16384,true)){case .ok(_):{clean::Problem{.code=0,.module_index=0,.start=0,.end=0}}case .err(e):{e}};assert(invalid.code==1001 && invalid.start==100+(n-1)*2 && invalid.end==101+(n-1)*2);
    }
    if(aggregates){assert(decode_code(raw,false)==1001);}else{assert(decode_code(raw,true)==1001);}
    number(out,if(project){1}else{0});number(out,if(aggregates){1}else{0});number(out,n);number(out,ew);number(out,dw);number(out,raw.len);assert(bytes::append(out,raw).is_ok);
}
// Test-only fresh wrappers intentionally carry no body/Complete authority. Only
// project_cleanup::build consumes them, testing selected drop remapping directly.
fn body::Program wrapped(clean::Program cleanup_ir){return body::drop_cap_wrap(cleanup_ir);}
fn void merge(ref mut bytes::Buffer out,usize n,bool skip){
    let units=(n+16383)/16384;let count=units+if(skip){1}else{0};var v::Vec<body::Program> inputs=v::empty<body::Program>();var v::Vec<graph::Unit> type_units=v::empty<graph::Unit>();var v::Vec<frames::Range> ranges=v::empty<frames::Range>();var v::Vec<usize> maps=v::empty<usize>();var v::Vec<clean::Frame> fs=v::empty<clean::Frame>();var v::Vec<frames::Origin> origins=v::empty<frames::Origin>();var usize i=0;var usize rest=n;
    while(i<count){let usize take=if(i==units){16384}else if(rest>16384){16384}else{rest};assert(v::push<body::Program>(ref mut inputs,wrapped(accepted(false,false,take,if(i==units){0}else{i}))).is_ok);assert(v::push<graph::Unit>(ref mut type_units,graph::Unit{.start=0,.count=2}).is_ok);assert(v::push<frames::Range>(ref mut ranges,frames::Range{.start=i,.count=1}).is_ok);assert(v::push<usize>(ref mut maps,if(i==units){1}else{i+1}).is_ok);if(i<units){assert(v::push<clean::Frame>(ref mut fs,frame(i)).is_ok);assert(v::push<frames::Origin>(ref mut origins,frames::Origin{.unit=i,.frame=1}).is_ok);rest-=take;}i+=1;}
    let graph=graph::drop_cap_plan(types::drop_cap_graph(records()),type_units);let functions=frames::drop_cap_plan(fs,origins,ranges,maps);var v::Vec<metadata::Choice> choices=v::empty<metadata::Choice>();assert(v::push<metadata::Choice>(ref mut choices,metadata::Choice{.unit=0,.type=0,.layout=0}).is_ok);assert(v::push<metadata::Choice>(ref mut choices,metadata::Choice{.unit=0,.type=2,.layout=1}).is_ok);let layouts=metadata::drop_cap_plan(choices);
    var usize work=0;let u32 code=switch(linked::build(ref inputs,ref graph,ref functions,ref layouts,ref mut work)){case .ok(p):{assert(n<=32768 && clean::project(ref p) && clean::drop_count(ref p)==n && clean::operand_count(ref p)==units && proof::check(ref p).code==0);var usize d=0;while(d<n){let expected=clean::drop_at(body::cleanup_ir(v::at<body::Program>(ref inputs,d/16384)),d%16384);var rebased=expected;rebased.operand=d/16384+1;assert(same_drop(clean::drop_at(ref p,d),rebased));d+=1;}0}case .err(e):{e.code}};
    assert(code==if(n>32768){1002}else{0} && work<=16777216);i=0;while(i<count){let input=body::cleanup_ir(v::at<body::Program>(ref inputs,i));assert(!clean::project(input) && clean::operand_count(input)==1 && clean::operand_at(input,1).frame==1 && clean::operand_at(input,1).id==16384 && clean::drop_at(input,0).operand==1 && proof::check(input).code==0);i+=1;}
    number(out,n);number(out,if(skip){1}else{0});number(out,code as usize);number(out,work);
}
@effects(console) fn i32 main(){
    assert(clean::drop_limit(false)==16384 && clean::drop_limit(true)==32768);
    assert(refused(false,16385,0,1,16384)==1002 && refused(true,32769,0,1,16384)==1002);
    assert(refused(false,1,1048576,1,16384)==0 && refused(false,1,1048577,1,16384)==1002 && refused(true,1,16777216,1,16384)==0 && refused(true,1,16777217,1,16384)==1002);
    assert(refused(true,1,0,16385,1)==1002 && refused(true,1,0,1,16385)==1001);
    var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"FYDPCAP1").is_ok);number(ref mut out,6);
    transport(ref mut out,false,false,16384);transport(ref mut out,true,false,16384);transport(ref mut out,true,false,23100);transport(ref mut out,true,false,32767);transport(ref mut out,true,false,32768);transport(ref mut out,true,true,32768);
    number(ref mut out,6);merge(ref mut out,16384,false);merge(ref mut out,23100,false);merge(ref mut out,32767,false);merge(ref mut out,32768,false);merge(ref mut out,32769,false);merge(ref mut out,32768,true);
    assert(switch(io::write_bytes(bytes::view(ref out))){case .ok(n):{n==bytes::len(ref out)}case .err(_):{false}});return 0;
}
