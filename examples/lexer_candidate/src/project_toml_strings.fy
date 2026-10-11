module project_toml_strings;
import std::bytes as bytes;
import std::text as text;
import std::num as num;

// String-token decoding for the pinned TOML 1.1 decoder. The document owner
// checks UTF-8 once for the complete input and checks the surrounding grammar.
// This leaf validates decoded bytes too and leaves prior output intact on error.
pub struct Problem{u32 code;usize start;usize end;}
fn Problem fail(u32 code,usize start,usize end){return Problem{.code,.start,.end};}
fn u32 put(ref mut bytes::Buffer output,u8 value){
    if(bytes::len(output)>=1048576){return 4;}
    return if(bytes::append_byte(output,value).is_ok){0}else{3};
}
fn u32 encoded_byte(ref mut bytes::Buffer output,u32 value){
    return switch(num::u8_from_i64(value as i64)){case .ok(c):{put(output,c)}case .err(_):{3}};
}
fn u32 point(ref mut bytes::Buffer output,u32 value){
    if(value<=127){return encoded_byte(output,value);}
    if(value<=2047){
        let u32 first=encoded_byte(output,192+value/64);if(first!=0){return first;}
        return encoded_byte(output,128+value%64);
    }
    if(value<=65535){
        let u32 first=encoded_byte(output,224+value/4096);if(first!=0){return first;}
        let u32 second=encoded_byte(output,128+(value/64)%64);if(second!=0){return second;}
        return encoded_byte(output,128+value%64);
    }
    let u32 first=encoded_byte(output,240+value/262144);if(first!=0){return first;}
    let u32 second=encoded_byte(output,128+(value/4096)%64);if(second!=0){return second;}
    let u32 third=encoded_byte(output,128+(value/64)%64);if(third!=0){return third;}
    return encoded_byte(output,128+value%64);
}
fn u32 hex(u8 value){
    if(value>=48 && value<=57){return (value as u32)-48;}
    if(value>=65 && value<=70){return (value as u32)-55;}
    if(value>=97 && value<=102){return (value as u32)-87;}return 16;
}
fn bool whitespace(u8 value){return value==32 || value==9 || value==10 || value==13;}

fn Result<usize,Problem> decode(ref u8[] source,usize start,bool allow_multiline,ref mut bytes::Buffer output){
    if(start>=source.len){return .err(fail(1,start,start));}
    if(source[start]!=34 && source[start]!=39){return .err(fail(1,start,start+1));}
    let u8 quote=source[start];
    let bool multiline=source.len-start>=3 && source[start+1]==quote && source[start+2]==quote;
    if(multiline && !allow_multiline){return .err(fail(1,start,start+3));}
    var usize i=start+if(multiline){3}else{1};
    if(multiline && i<source.len){
        if(source[i]==10){i+=1;}
        else if(source[i]==13 && source.len-i>=2 && source[i+1]==10){i+=2;}
    }
    while(i<source.len){
        let u8 c=source[i];
        if(c==quote){
            if(!multiline){return .ok(i+1);}
            var usize end=i;while(end<source.len && source[end]==quote){end+=1;}
            let usize count=end-i;
            if(count>=3){
                if(count>5){return .err(fail(1,i,end));}
                var usize extra=count-3;while(extra>0){let u32 code=put(output,quote);if(code!=0){return .err(fail(code,i,end));}extra-=1;}
                return .ok(end);
            }
            while(i<end){let u32 code=put(output,quote);if(code!=0){return .err(fail(code,i,i+1));}i+=1;}continue;
        }
        if(c==92 && quote==34){
            let usize escape=i;i+=1;if(i==source.len){return .err(fail(1,escape,i));}
            if(multiline && whitespace(source[i])){
                var bool newline=false;
                while(i<source.len && whitespace(source[i])){
                    if(source[i]==13 && (source.len-i<2 || source[i+1]!=10)){return .err(fail(1,i,i+1));}
                    if(source[i]==10){newline=true;}i+=1;
                }
                if(!newline){return .err(fail(1,escape,i));}continue;
            }
            let u8 kind=source[i];i+=1;
            if(kind==120 || kind==117 || kind==85){
                let usize digits=if(kind==120){2}else if(kind==117){4}else{8};
                if(source.len-i<digits){return .err(fail(1,escape,source.len));}
                var u32 value=0;var usize n=0;
                while(n<digits){let u32 part=hex(source[i]);if(part==16){return .err(fail(1,i,i+1));}value=value*16+part;i+=1;n+=1;}
                if(value>1114111 || (value>=55296 && value<=57343)){return .err(fail(1,escape,i));}
                let u32 code=point(output,value);if(code!=0){return .err(fail(code,escape,i));}continue;
            }
            let u8 value=if(kind==98){8}else if(kind==116){9}else if(kind==110){10}else if(kind==102){12}else if(kind==114){13}else if(kind==101){27}else{kind};
            if(kind!=98 && kind!=116 && kind!=110 && kind!=102 && kind!=114 && kind!=101 && kind!=34 && kind!=92){return .err(fail(1,escape,i));}
            let u32 code=put(output,value);if(code!=0){return .err(fail(code,escape,i));}continue;
        }
        if(c==13){
            if(!multiline || source.len-i<2 || source[i+1]!=10){return .err(fail(1,i,i+1));}
        }else if((c<32 && c!=9 && !(multiline && c==10)) || c==127){return .err(fail(1,i,i+1));}
        let u32 code=put(output,c);if(code!=0){return .err(fail(code,i,i+1));}i+=1;
    }
    return .err(fail(1,start,source.len));
}

// Success returns the exact byte after the delimiter. Quoted-key callers pass
// false for allow_multiline; value callers pass true. No suffix is consumed.
pub fn Result<usize,Problem> append(ref u8[] source,usize start,bool allow_multiline,ref mut bytes::Buffer output){
    let usize initial=bytes::len(output);
    let usize at=if(start<=source.len){start}else{source.len};
    if(source.len>1048576 || initial>1048576){return .err(fail(4,at,at));}
    if(start>source.len){return .err(fail(1,at,at));}
    let usize end=switch(decode(source,start,allow_multiline,output)){
        case .ok(position):{position}
        case .err(problem):{discard bytes::truncate(output,initial);return .err(problem);}
    };
    if(!text::validate_utf8(bytes::view(output)[initial..]).is_ok){discard bytes::truncate(output,initial);return .err(fail(2,start,end));}
    return .ok(end);
}
