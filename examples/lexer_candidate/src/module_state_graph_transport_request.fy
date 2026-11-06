module module_state_graph_transport_request;
import module_state_graph_transport as transport;
import module_state_graph_native as native;
import module_state_graph_request as request;
import std::bytes as bytes;

// The structural packet encoder cannot forge this compiled-owner input.
pub resource Input{request::Packet packet;cleanup{}}
pub fn ref u8[] content(ref Input p){return request::content(ref p.packet);}
pub fn ref u8[] digest(ref Input p){return request::digest(ref p.packet);}
pub fn Result<Input,u32> encode(ref transport::Prepared owner,ref request::Snapshot snapshot,request::Limits limits,ref mut request::Work work){
    let entry=transport::entry(owner);
    return switch(request::encode(transport::plan(owner),native::layout_hash(entry),native::body_hash(entry),transport::program_hash(owner),snapshot,limits,work)){case .ok(packet):{.ok(Input{.packet})}case .err(code):{.err(code)}};
}
