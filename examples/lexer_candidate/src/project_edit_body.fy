module project_edit_body;
import syntax as syntax;
import scanner as scanner;
import project_path as work;
import sha256_stream as sha;
import std::bytes as bytes;

// This is a syntax-only plan. The workspace host must verify the complete edited
// overlay before reporting verified=true or publishing anything.
pub resource Change {
    bytes::Buffer before;
    syntax::Parsed after;
    bytes::Buffer before_hash;
    bytes::Buffer after_hash;
    usize start;
    usize end;
    cleanup {}
}
pub fn usize maximum_body(){return 65536;}
pub fn ref u8[] before(ref Change p){return bytes::view(ref p.before);}
pub fn ref u8[] after(ref Change p){return syntax::source_view(ref p.after);}
pub fn ref syntax::Parsed parsed(ref Change p){return ref p.after;}
pub fn ref u8[] before_hash(ref Change p){return bytes::view(ref p.before_hash);}
pub fn ref u8[] after_hash(ref Change p){return bytes::view(ref p.after_hash);}
pub fn usize start(ref Change p){return p.start;}
pub fn usize end(ref Change p){return p.end;}

pub fn Result<syntax::Parsed,u32> parse(bytes::Buffer source){
    let tokens=switch(scanner::collect(source,false,8388608,1048576,256,16777216)){
        case .ok(p):{p} case .err(_):{return .err(5);}
    };
    let parsed=switch(syntax::parse(tokens,512)){
        case .ok(p):{p} case .err(_):{return .err(5);}
    };
    if(syntax::problem_count(ref parsed)!=0){return .err(5);}
    return .ok(parsed);
}
// Unicode White_Space, matching bytes.TrimSpace in the reference implementation.
// BOM and zero-width space are deliberately not whitespace.
fn usize space(ref u8[] s,usize at){
    let c=s[at];
    if(c==32 || c>=9 && c<=13){return 1;}
    if(s.len-at>=2 && c==194 && (s[at+1]==133 || s[at+1]==160)){return 2;}
    if(s.len-at<3){return 0;}
    if(c==225 && s[at+1]==154 && s[at+2]==128){return 3;}
    if(c==226 && s[at+1]==128 && (s[at+2]>=128 && s[at+2]<=138 || s[at+2]==168 || s[at+2]==169 || s[at+2]==175)){return 3;}
    if(c==226 && s[at+1]==129 && s[at+2]==159 || c==227 && s[at+1]==128 && s[at+2]==128){return 3;}
    return 0;
}
fn Result<bytes::Buffer,u32> digest(ref u8[] source){
    var state=sha::empty();var out=bytes::empty();
    if(!sha::update(ref mut state,source) || !sha::append_hex(ref state,ref mut out)){return .err(2);}
    return .ok(out);
}
// 1 subject/body policy; 2 allocation; 3 shared work/size budget; 4 stale hash;
// 5 syntax failure; 6 replacement escaped the selected AST body.
pub fn Result<Change,u32> replace(ref syntax::Parsed source,usize declaration,ref u8[] expected,ref u8[] body,ref mut usize meter){
    let raw=syntax::source_view(source);
    if(body.len>maximum_body()){return .err(1);}
    if(raw.len>8388608 || !work::charge(meter,raw.len+body.len+65)){return .err(3);}
    if(declaration>=syntax::declaration_count(source) || syntax::problem_count(source)!=0 || scanner::diagnostic_count(syntax::source_stream(source))!=0){return .err(1);}
    let d=syntax::declaration_at(source,declaration);
    if(d.kind!=3 || d.public || d.unsafe_end!=d.unsafe_start || d.owner_end!=d.owner_start || d.attribute_count!=0 || d.body==0 || d.body>syntax::statement_count(source) || bytes::equal(syntax::declaration_name(source,declaration+1),b"main")){return .err(1);}
    let span=syntax::statement_at(source,d.body);
    if(span.kind!=1 || span.start>span.end || span.end>raw.len){return .err(1);}
    let before_hash=digest(raw)?;
    if(expected.len!=64 || !bytes::equal(bytes::view(ref before_hash),expected)){return .err(4);}
    var usize first=0;
    while(first<body.len){let n=space(body,first);if(n==0){break;}first+=n;}
    var usize at=first;var usize last=first;
    while(at<body.len){let n=space(body,at);if(n==0){at+=1;last=at;}else{at+=n;}}
    if(last-first<2 || body[first]!=123 || body[last-1]!=125){return .err(1);}
    let replacement=body[first..last];
    if(raw.len-(span.end-span.start)>8388608-replacement.len){return .err(3);}
    var output=bytes::empty();
    if(!bytes::append(ref mut output,raw[..span.start]).is_ok || !bytes::append(ref mut output,replacement).is_ok || !bytes::append(ref mut output,raw[span.end..]).is_ok){return .err(2);}
    if(!work::charge(meter,bytes::len(ref output)+1)){return .err(3);}
    let after_hash=digest(bytes::view(ref output))?;
    let edited=parse(output)?;var usize matches=0;var usize i=0;
    while(i<syntax::declaration_count(ref edited)){
        if(!work::charge(meter,1)){return .err(3);}
        let candidate=syntax::declaration_at(ref edited,i);
        if(candidate.kind==3 && candidate.name_start==d.name_start && bytes::equal(syntax::declaration_name(ref edited,i+1),syntax::declaration_name(source,declaration+1))){
            if(candidate.body==0 || candidate.body>syntax::statement_count(ref edited)){return .err(6);}
            let b=syntax::statement_at(ref edited,candidate.body);
            if(b.start!=span.start || b.end!=span.start+replacement.len){return .err(6);}
            matches+=1;
        }
        i+=1;
    }
    if(matches!=1){return .err(6);}
    let before=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return .err(2);}};
    return .ok(Change{.before,.after=edited,.before_hash,.after_hash,.start=span.start,.end=span.end});
}
