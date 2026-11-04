module project_mcp_stream;
import std::bytes as bytes;
import std::io as io;

pub struct Reader{u8[4096] chunk;usize at;usize size;bool ended;}
pub fn Result<Reader,u32> open(){return .ok(Reader{.chunk=[0;4096],.at=0,.size=0,.ended=false});}
fn Result<bool,u32> complete(ref mut bytes::Buffer line){let n=bytes::len(line);if(n>0 && bytes::view(line)[n-1]==13 && !bytes::truncate(line,n-1).is_ok){return .err(2);}return .ok(true);}
// One owned frame at a time. Match ScanLines' CRLF/final-EOF behavior and raw
// token allowance; no request waits for the client's stream to reach EOF.
@effects(console) pub fn Result<bool,u32> next(ref mut Reader reader,ref mut bytes::Buffer line){
 if(bytes::len(line)!=0){return .err(2);}
 while(true){
  if(reader.at==reader.size){if(reader.ended){return if(bytes::len(line)==0){.ok(false)}else{complete(line)};}let n=switch(io::read_stdin(ref mut reader.chunk)){case .ok(p):{p}case .err(_):{return .err(2);}};reader.at=0;reader.size=n;if(n==0){reader.ended=true;continue;}}
  let ref u8[] data=ref reader.chunk;var usize end=reader.at;while(end<reader.size && data[end]!=10){end+=1;}let count=end-reader.at;let used=bytes::len(line);if(used>65536 || count>65536-used){return .err(2);}if(!bytes::append(line,data[reader.at..end]).is_ok){return .err(2);}reader.at=end;
  if(end<reader.size){reader.at+=1;return complete(line);}
 }
 return .err(2);
}
@effects(console) pub fn i32 write(ref u8[] data){return if(switch(io::write_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}}){0}else{4};}
