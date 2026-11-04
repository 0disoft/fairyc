module project_mcp_protocol;
import project_mcp_json as fields;
import project_mcp_messages as messages;
import project_json_tree as json;
import project_json_string as quoted;
import std::bytes as bytes;

pub struct Session{bool initialized;bool ready;}
pub resource Reply{bytes::Buffer id;bytes::Buffer packet;bytes::Buffer uid;bytes::Buffer symbol;bool pending;cleanup{}}
pub fn Session empty(){return Session{.initialized=false,.ready=false};}
pub fn bool pending(ref Reply p){return p.pending;}
pub fn ref u8[] packet(ref Reply p){return bytes::view(ref p.packet);}
pub fn ref u8[] uid(ref Reply p){return bytes::view(ref p.uid);}
pub fn ref u8[] symbol(ref Reply p){return bytes::view(ref p.symbol);}
fn Result<bytes::Buffer,u32> copy(ref u8[] data){var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,data).is_ok){return .err(2);}return .ok(out);}
fn Result<Reply,u32> response(ref u8[] id,ref u8[] body,usize maximum){let output=messages::packet(id,body,maximum)?;return .ok(Reply{.id=bytes::empty(),.packet=output,.uid=bytes::empty(),.symbol=bytes::empty(),.pending=false});}
fn Result<Reply,u32> failure(ref u8[] id,u32 code,usize maximum){return response(id,messages::failure(code),maximum);}
fn Result<bytes::Buffer,u32> identity(ref json::Tree t,usize at){if(fields::kind(t,at)==7){var usize meter=0;return switch(quoted::quote(json::value(t,at),ref mut meter)){case .ok(p):{.ok(p)}case .err(_):{.err(2)}};}return copy(fields::raw(t,at));}
fn bool initialize_valid(ref json::Tree t,usize params){let capabilities=fields::field(t,params,b"capabilities",true);let info=fields::field(t,params,b"clientInfo",true);return fields::kind(t,params)==1 && fields::string(t,fields::field(t,params,b"protocolVersion",true)).len>0 && fields::kind(t,capabilities)==1 && fields::kind(t,info)==1 && fields::string(t,fields::field(t,info,b"name",true)).len>0 && fields::string(t,fields::field(t,info,b"version",true)).len>0;}
fn bool call_shape(ref json::Tree t,usize params){if(fields::kind(t,params)!=1){return false;}var usize i=params+1;while(json::token(t,i).kind!=2){let key=json::value(t,i);if(!fields::equal(key,b"name",true) && !fields::equal(key,b"arguments",true) && !fields::equal(key,b"_meta",true)){return false;}i=json::next(t,i+2);if(json::token(t,i).kind==6){i+=1;}}return bytes::equal(fields::string(t,fields::field(t,params,b"name",true)),b"atlas_impact");}

// Pure session transition: it neither opens a workspace nor executes a callback.
// The host must finish a selected query before accepting another frame.
pub fn Result<Reply,u32> step(ref mut Session state,bytes::Buffer input,usize maximum){
 if(maximum<2048 || maximum>4194304 || bytes::len(ref input)>65536){return .err(2);}
 let t=switch(fields::parse(input)){case .ok(p):{p}case .err(_):{return failure(b"null",1,maximum);}};
 let id_at=fields::field(ref t,0,b"id",true);let method_at=fields::field(ref t,0,b"method",true);let version=fields::field(ref t,0,b"jsonrpc",true);let params=fields::field(ref t,0,b"params",true);
 if(fields::kind(ref t,0)!=1 || !bytes::equal(fields::string(ref t,version),b"2.0") || fields::string(ref t,method_at).len==0 || !fields::valid_id(ref t,id_at)){return failure(b"null",2,maximum);}
 let method=fields::string(ref t,method_at);
 if(id_at==json::count(ref t)){if(bytes::equal(method,b"notifications/initialized") && state.initialized && fields::empty_params(ref t,params)){state.ready=true;}return .ok(Reply{.id=bytes::empty(),.packet=bytes::empty(),.uid=bytes::empty(),.symbol=bytes::empty(),.pending=false});}
 let id=identity(ref t,id_at)?;let encoded=bytes::view(ref id);
 if(bytes::equal(method,b"ping")){return response(encoded,b"\"result\":{}",maximum);}
 if(bytes::equal(method,b"initialize")){if(state.initialized || !initialize_valid(ref t,params)){return failure(encoded,3,maximum);}state.initialized=true;return response(encoded,messages::initialized(),maximum);}
 if(bytes::equal(method,b"tools/list")){if(!state.ready){return failure(encoded,4,maximum);}if(!fields::empty_params(ref t,params)){return failure(encoded,5,maximum);}return response(encoded,messages::tools(),maximum);}
 if(!bytes::equal(method,b"tools/call")){return failure(encoded,8,maximum);}if(!state.ready){return failure(encoded,4,maximum);}let arguments=fields::field(ref t,params,b"arguments",true);
 if(!call_shape(ref t,params) || fields::kind(ref t,arguments)!=1 || fields::fields(ref t,arguments)!=2){return failure(encoded,6,maximum);}
 let package_uid=fields::string(ref t,fields::field(ref t,arguments,b"packageUid",false));let target=fields::string(ref t,fields::field(ref t,arguments,b"symbol",false));
 if(package_uid.len==0 || target.len==0){return failure(encoded,7,maximum);}let uid=copy(package_uid)?;let symbol=copy(target)?;
 return .ok(Reply{.id,.packet=bytes::empty(),.uid,.symbol,.pending=true});
}

pub fn Result<bytes::Buffer,u32> finish(ref Reply reply,ref u8[] data,bool available,usize maximum){if(!reply.pending){return .err(2);}if(!available){return messages::packet(bytes::view(ref reply.id),messages::unavailable(),maximum);}if(!fields::valid_output(data)){return .err(2);}let prefix=messages::success();if(data.len>maximum || prefix.len+data.len+1>maximum){return messages::packet(bytes::view(ref reply.id),messages::failure(9),maximum);}var bytes::Buffer body=bytes::empty();if(!bytes::append(ref mut body,prefix).is_ok || !bytes::append(ref mut body,data).is_ok || !bytes::append(ref mut body,b"}").is_ok){return .err(2);}return messages::packet(bytes::view(ref reply.id),bytes::view(ref body),maximum);}
