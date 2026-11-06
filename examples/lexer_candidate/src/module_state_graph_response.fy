module module_state_graph_response;
import module_state_graph_response_common as c;
import module_state_graph_response_wire as wire;
import module_state_graph_response_path as path;
import module_state_graph_response_evidence as observations;
import module_state_graph_transport as transport;
import module_state_graph_transport_request as input;
import module_state_graph_native as native;
import project_compiler_core as core;
import project_compiler_state_graph as complete;
import std::bytes as bytes;
import std::vec as v;

// Only the complete admission pipeline may publish this owner. Neither a
// parsed frame nor caller-provided path/storage facts can construct it.
pub resource Admitted{c::Header header;usize version;observations::Evidence evidence;bytes::Buffer category;cleanup{}}
pub fn c::Header header(ref Admitted result){return result.header;}
pub fn usize version(ref Admitted result){return result.version;}
pub fn ref u8[] trap_category(ref Admitted result){return bytes::view(ref result.category);}
pub fn usize read_count(ref Admitted result){return observations::read_count(ref result.evidence);}
pub fn c::Read read_at(ref Admitted result,usize index){return observations::read_at(ref result.evidence,index);}
pub fn ref u8[] read_value(ref Admitted result,usize index){return observations::read_value(ref result.evidence,index);}
pub fn usize write_count(ref Admitted result){return if(result.header.status==0 && result.header.exit==0){observations::write_count(ref result.evidence)}else{0};}
pub fn c::Write write_at(ref Admitted result,usize index){assert(index<write_count(result));return observations::write_at(ref result.evidence,index);}
pub fn ref u8[] write_value(ref Admitted result,usize index){assert(index<write_count(result));return observations::write_value(ref result.evidence,index);}

pub fn Result<Admitted,u32> decode(ref transport::Prepared owner,ref input::Input request,ref u8[] output,ref mut c::Work work){
    if(!c::charge(work,1) || !c::byte_charge(work,0)){return .err(1002);}
    let raw=input::content(request);let entry=transport::entry(owner);
    if(raw.len<152 || !c::byte_charge(work,96)){return .err(if(raw.len<152){1001}else{1002});}
    if(!bytes::equal(raw[8..40],native::layout_hash(entry)) || !bytes::equal(raw[40..72],native::body_hash(entry)) || !bytes::equal(raw[72..104],transport::program_hash(owner))){return .err(1001);}
    let h=switch(wire::header(output,raw,input::digest(request),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let checked=transport::checked(owner);let proof=complete::program(core::checked_state_graph(checked));
    let plan=switch(path::build(proof,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var route=path::start(ref plan);
    var evidence=switch(observations::begin(raw,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var usize cursor=186;var usize i=0;
    while(i<h.events){
        let event=switch(wire::event(output,ref mut cursor,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        let step=switch(path::advance(ref plan,ref mut route,event.site,event.operation,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        let code=observations::admit(ref mut evidence,proof,step,raw,output[event.key_start..event.key_end],output[event.value_start..event.value_end],event.found,event.operation,work);
        if(code!=0){return .err(code);}i+=1;
    }
    let matched=switch(path::counters(ref plan,ref route,h.status,h.cost,h.operations,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    if(!matched || h.status==0 && (path::exit(ref route)!=h.exit+1 || path::trap(ref route)!=h.trap)){return .err(1001);}
    var category=bytes::empty();
    if(h.status==0 && h.exit==2){
        let raw_category=path::trap_category(ref plan,h.trap);
        if(h.trap==0 || raw_category.len==0){return .err(1001);}
        if(!c::byte_charge(work,raw_category.len)){return .err(1002);}
        category=switch(bytes::copy(raw_category)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    }
    let committed=h.status==0 && h.exit==0;
    if(committed && (h.writes!=observations::write_count(ref evidence) || h.write_bytes!=observations::write_bytes(ref evidence))){return .err(1001);}
    // A forged final-write count must not size allocations independently of
    // the event-derived overlay. Reserve its verified length only once.
    if(!c::charge(work,h.writes)){return .err(1002);}
    var writes=v::empty<c::Write>();if(!v::reserve<c::Write>(ref mut writes,h.writes).is_ok){return .err(1003);}i=0;
    while(i<h.writes){
        let row=switch(wire::write(output,ref mut cursor,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(!c::charge(work,1)){return .err(1002);}if(!v::push<c::Write>(ref mut writes,row).is_ok){return .err(1003);}i+=1;
    }
    if(cursor!=output.len){return .err(1001);}
    let code=observations::finish(ref evidence,output,ref writes,h.write_bytes,committed,work);
    if(code!=0){return .err(code);}
    return .ok(Admitted{.header=h,.version=wire::number(raw,104,8),.evidence,.category});
}
