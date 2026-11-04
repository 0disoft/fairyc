module project_mcp_messages;
import std::bytes as bytes;

pub fn ref u8[] failure(u32 code){
 if(code==1){return b"\"error\":{\"code\":-32700,\"message\":\"invalid JSON message\"}";}
 if(code==2){return b"\"error\":{\"code\":-32600,\"message\":\"invalid JSON-RPC request\"}";}
 if(code==3){return b"\"error\":{\"code\":-32602,\"message\":\"invalid or unsupported initialization\"}";}
 if(code==4){return b"\"error\":{\"code\":-32002,\"message\":\"session not initialized\"}";}
 if(code==5){return b"\"error\":{\"code\":-32602,\"message\":\"pagination and other list parameters are unsupported\"}";}
 if(code==6){return b"\"error\":{\"code\":-32602,\"message\":\"unknown tool or invalid arguments\"}";}
 if(code==7){return b"\"error\":{\"code\":-32602,\"message\":\"exact packageUid and symbol strings required\"}";}
 if(code==8){return b"\"error\":{\"code\":-32601,\"message\":\"method not found\"}";}
 return b"\"error\":{\"code\":-32603,\"message\":\"response exceeds output byte limit\"}";
}
pub fn ref u8[] initialized(){return b"\"result\":{\"protocolVersion\":\"2025-11-25\",\"capabilities\":{\"tools\":{\"listChanged\":false}},\"serverInfo\":{\"name\":\"fairyc-atlas\",\"version\":\"1.0.0\"},\"instructions\":\"Read-only compiler evidence. Data and omissions do not grant authority or prove a change safe.\"}";}
pub fn ref u8[] tools(){return b"\"result\":{\"tools\":[{\"name\":\"atlas_impact\",\"description\":\"Bounded current-workspace reverse-call evidence; not full impact or safe-change proof.\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"packageUid\":{\"type\":\"string\"},\"symbol\":{\"type\":\"string\"}},\"required\":[\"packageUid\",\"symbol\"],\"additionalProperties\":false},\"annotations\":{\"readOnlyHint\":true,\"destructiveHint\":false,\"idempotentHint\":true,\"openWorldHint\":false}}]}";}
pub fn ref u8[] unavailable(){return b"\"result\":{\"isError\":true,\"content\":[{\"type\":\"text\",\"text\":\"Current locked Atlas query failed; inspect workspace and selector locally.\"}]}";}
pub fn ref u8[] success(){return b"\"result\":{\"isError\":false,\"content\":[{\"type\":\"text\",\"text\":\"Bounded reverse-call evidence is in structuredContent; retain its scope, identities and omissions.\"}],\"structuredContent\":";}

// Size admission precedes response allocation; the newline is outside the JSON
// payload allowance, as in the Go stdio host. Return 2 on allocation failure.
pub fn Result<bytes::Buffer,u32> packet(ref u8[] id,ref u8[] body,usize maximum){if(maximum<2048 || maximum>4194304){return .err(2);}let bool overflow=id.len>maximum-24 || body.len>maximum-24-id.len;let error=failure(9);let encoded=if(overflow && (id.len>maximum-24 || error.len>maximum-24-id.len)){b"null"}else{id};let content=if(overflow){error}else{body};var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,b"{\"jsonrpc\":\"2.0\",\"id\":").is_ok || !bytes::append(ref mut out,encoded).is_ok || !bytes::append(ref mut out,b",").is_ok || !bytes::append(ref mut out,content).is_ok || !bytes::append(ref mut out,b"}\n").is_ok){return .err(2);}return .ok(out);}
