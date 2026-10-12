module windows_command;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Private CreateProcessW transport: at most 32767 UTF-16 words including NUL.
// Quotes follow the Go Windows host's EscapeArg rule; executable quotes reject.
// The complete builder owns its mutable buffer and publishes no partial line.
pub struct Problem{u32 kind;usize offset;}
fn Problem fail(u32 kind,usize offset){return Problem{.kind,.offset};}
fn u8 octet(u32 value){var u32 remaining=value;var u8 out=0;var u8 mask=128;var u32 bit=128;while(bit!=0){if(remaining>=bit){out+=mask;remaining-=bit;}bit/=2;mask/=2;}return out;}
fn bool word(ref mut bytes::Buffer out,u32 value){return bytes::len(out)<=65530 && bytes::append_byte(out,octet(value%256)).is_ok && bytes::append_byte(out,octet(value/256)).is_ok;}
fn bool slashes(ref mut bytes::Buffer out,usize count){var usize i=0;while(i<count){if(!word(out,92)){return false;}i+=1;}return true;}
fn Result<usize,Problem> valid(ref u8[] input){if(input.len>131064){return .err(fail(2,0));}if(!text::validate_utf8(input).is_ok){return .err(fail(1,0));}var usize i=0;while(i<input.len){if(input[i]==0){return .err(fail(1,i));}i+=1;}return .ok(input.len);}
fn bool character(ref mut bytes::Buffer out,ref u8[] input,ref mut usize at){let first=input[*at] as u32;var u32 code=first;var usize width=1;
    if(first>=240){width=4;code=(first-240)*262144+((input[*at+1] as u32)-128)*4096+((input[*at+2] as u32)-128)*64+(input[*at+3] as u32)-128;}
    else if(first>=224){width=3;code=(first-224)*4096+((input[*at+1] as u32)-128)*64+(input[*at+2] as u32)-128;}
    else if(first>=192){width=2;code=(first-192)*64+(input[*at+1] as u32)-128;};
    *at=*at+width;return if(code<=65535){word(out,code)}else{word(out,55296+(code-65536)/1024) && word(out,56320+(code-65536)%1024)};
}
fn Problem output_failure(ref bytes::Buffer out,usize at){return fail(if(bytes::len(out)>=65532){2}else{3},at);}
fn bool quoted(ref u8[] input){if(input.len==0){return true;}var usize i=0;while(i<input.len){if(input[i]==32 || input[i]==9){return true;}i+=1;}return false;}
fn Result<bytes::Buffer,Problem> executable(ref u8[] input){if(input.len==0){return .err(fail(1,0));}switch(valid(input)){case .ok(_):{}case .err(e):{return .err(e);}};
    let quote=quoted(input);var bytes::Buffer wide=bytes::empty();if(quote && !word(ref mut wide,34)){return .err(fail(3,0));}var usize at=0;
    while(at<input.len){if(input[at]==34){return .err(fail(1,at));}if(!character(ref mut wide,input,ref mut at)){return .err(output_failure(ref wide,at));}}
    if(quote){var usize trailing=0;while(trailing<input.len && input[input.len-trailing-1]==92){trailing+=1;}if(!slashes(ref mut wide,trailing) || !word(ref mut wide,34)){return .err(output_failure(ref wide,at));}}return .ok(wide);
}
fn Problem argument(ref mut bytes::Buffer out,ref u8[] input){switch(valid(input)){case .ok(_):{}case .err(e):{return e;}};
    let quote=quoted(input);if(!word(out,32) || quote && !word(out,34)){return output_failure(out,0);}var usize at=0;
    while(at<input.len){var usize count=0;while(at<input.len && input[at]==92){count+=1;at+=1;}
        if(!slashes(out,if(at<input.len && input[at]==34 || at==input.len && quote){count*2}else{count})){return output_failure(out,at);}
        if(at==input.len){break;}if(input[at]==34 && !word(out,92)){return output_failure(out,at);}if(!character(out,input,ref mut at)){return output_failure(out,at);}
    }return if(!quote || word(out,34)){fail(0,0)}else{output_failure(out,at)};
}
pub fn Result<bytes::Buffer,Problem> path(ref u8[] input){if(input.len==0){return .err(fail(1,0));}switch(valid(input)){case .ok(_):{}case .err(e):{return .err(e);}};var bytes::Buffer out=bytes::empty();var usize at=0;
    while(at<input.len){if(!character(ref mut out,input,ref mut at)){return .err(output_failure(ref out,at));}}
    if(!bytes::append_byte(ref mut out,0).is_ok || !bytes::append_byte(ref mut out,0).is_ok){return .err(fail(3,at));}return .ok(out);
}
// A nonempty executable consumes at least one word. Every argument consumes at
// least a separator and one word, so 16383 arguments cannot fit, even before
// quoting. All smaller counts are admitted only by their actual encoded length.
pub fn Result<bytes::Buffer,Problem> build(ref u8[] program,ref v::Vec<bytes::Buffer> arguments){if(v::len<bytes::Buffer>(arguments)>16382){return .err(fail(2,0));}
    var bytes::Buffer out=switch(executable(program)){case .ok(p):{p}case .err(e):{return .err(e);}};var usize i=0;
    while(i<v::len<bytes::Buffer>(arguments)){let e=argument(ref mut out,bytes::view(v::at<bytes::Buffer>(arguments,i)));if(e.kind!=0){return .err(e);}i+=1;}
    if(!bytes::append_byte(ref mut out,0).is_ok || !bytes::append_byte(ref mut out,0).is_ok){return .err(fail(3,0));}return .ok(out);
}
