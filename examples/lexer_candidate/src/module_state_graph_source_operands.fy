module module_state_graph_source_operands;
import module_state_graph_model as m;
import module_state_graph_source_context as context;
import module_state_graph_values as values;
import module_state_layout as state;
import module_registry as paths;
import project_cache_encoding as budget;
import literal_types as literals;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Raw facts for the source producer's owned, parser-accepted arena. These
// helpers publish no graph proof, Complete, cache permission or execution token.
// 1001 closed/malformed shape, 1002 size/work/ID bound, 1003 allocation.
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn u8 octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn u32 append(ref mut bytes::Buffer out,ref u8[] raw,ref mut usize work){let code=budget::raw(out,raw,work);return if(code==0){0}else if(code==4){1003}else{1002};}
fn bool expression(ref syntax::Parsed p,usize id){return id!=0 && id<=syntax::expression_count(p);}
fn Result<bytes::Buffer,u32> normalize(ref u8[] raw,ref mut usize work){
    if(raw.len>=16777216 || !budget::charge(work,raw.len+1)){return .err(1002);}
    var bytes::Buffer out=bytes::empty();let code=paths::normalize(raw,ref mut out);
    if(code!=0){return .err(mapped(code));}return .ok(out);
}
pub fn Result<bytes::Buffer,u32> string(ref syntax::Parsed p,usize expr,u32 literalKind,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}
    if(!expression(p,expr) || syntax::expression_at(p,expr).kind!=1 || syntax::literal_kind(p,expr)!=literalKind){return .err(1001);}
    let cooked=syntax::literal_cooked(p,expr);if(cooked.len>=16777216 || !budget::charge(work,cooked.len+1)){return .err(1002);}
    var bytes::Buffer out=bytes::empty();let code=literals::decode(ref mut out,cooked);
    if(code!=0){return .err(if(code==3){1003}else{1001});}return .ok(out);
}
pub fn u32 codec(ref syntax::Parsed p,usize typeRoot){
    if(typeRoot==0 || typeRoot>syntax::type_count(p) || syntax::type_at(p,typeRoot).kind!=1){return 0;}
    let name=syntax::type_text(p,typeRoot);
    return if(bytes::equal(name,b"bool")){1}else if(bytes::equal(name,b"u32")){2}else if(bytes::equal(name,b"u64")){3}else if(bytes::equal(name,b"bytes20")){4}else if(bytes::equal(name,b"bytes32")){5}else if(bytes::equal(name,b"bytes")){6}else if(bytes::equal(name,b"text")){7}else{0};
}
// Multiply in canonical bytes, retaining the full u64 domain independently of
// host usize width. Carry beyond the selected width rejects before publication.
fn u32 integer(ref u8[] raw,usize width,ref mut bytes::Buffer out,ref mut usize work){
    if(raw.len==0){return 1001;}if(raw.len>16777215/width || !budget::charge(work,raw.len*width+1)){return 1002;}
    var u8[8] encoded=[0;8];var usize i=0;
    while(i<raw.len){if(raw[i]<48 || raw[i]>57){return 1001;}var usize carry=(raw[i]-48) as usize;var usize j=width;
        while(j>0){j-=1;let digit=(encoded[j] as usize)*10+carry;encoded[j]=octet(digit%256);carry=digit/256;}
        if(carry!=0){return 1001;}i+=1;
    }let ref u8[] view=ref encoded;return append(out,view[..width],work);
}
pub fn Result<usize,u32> u32_literal(ref syntax::Parsed p,usize expr,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}
    if(!expression(p,expr) || syntax::expression_at(p,expr).kind!=1 || syntax::literal_kind(p,expr)!=5){return .err(1001);}
    var bytes::Buffer encoded=bytes::empty();let code=integer(syntax::expression_text(p,expr),4,ref mut encoded,work);if(code!=0){return .err(code);}
    let raw=bytes::view(ref encoded);var usize number=0;var usize i=0;while(i<4){number=number*256+(raw[i] as usize);i+=1;}return .ok(number);
}
pub fn Result<usize,u32> ungroup(ref syntax::Parsed p,usize expr,ref mut usize work){
    var usize id=expr;while(true){if(!budget::charge(work,1)){return .err(1002);}if(!expression(p,id)){return .err(1001);}
        let node=syntax::expression_at(p,id);if(node.kind!=3){return .ok(id);}if(node.left==0 || node.left>=id || node.right!=0){return .err(1001);}id=node.left;
    }return .err(1001);
}
// First matching binding wins, as in the Go environment. No caller rows change.
// Parsed identifier spans are raw source offsets; requested paths are normalized.
pub fn Result<usize,u32> lookup(ref syntax::Parsed p,ref v::Vec<context::Binding> env,ref u8[] name,ref mut usize work){
    let canonical=switch(normalize(name,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let source=syntax::source_view(p);var usize i=0;
    while(i<v::len<context::Binding>(env)){if(!budget::charge(work,1)){return .err(1002);}let row=*v::at<context::Binding>(env,i);
        if(row.start>row.end || row.end>source.len){return .err(1001);}let raw=source[row.start..row.end];
        if(raw.len>16777215-bytes::len(ref canonical) || !budget::charge(work,raw.len+bytes::len(ref canonical)+1)){return .err(1002);}
        if(bytes::equal(raw,bytes::view(ref canonical))){return .ok(i+1);}i+=1;
    }return .ok(0);
}
// Check the raw call/callee before normalization. Specializations, grouped
// callees and labeled arguments stay closed. One traversal accepts any argc;
// callers own the operation-specific count and operand checks.
pub fn Result<bytes::Buffer,u32> call_name(ref syntax::Parsed p,usize expr,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(!expression(p,expr)){return .err(1001);}let node=syntax::expression_at(p,expr);
    if(node.kind!=6 || !expression(p,node.left) || syntax::expression_at(p,node.left).kind!=2){return .err(1001);}
    var usize link=node.right;while(link!=0){if(!budget::charge(work,1)){return .err(1002);}
        if(!expression(p,link)){return .err(1001);}let arg=syntax::expression_at(p,link);
        if(arg.kind!=7 || arg.right>=link || syntax::call_argument_labeled(p,link)){return .err(1001);}link=arg.right;
    }return normalize(syntax::expression_text(p,node.left),work);
}
pub fn Result<m::Value,u32> reference(ref syntax::Parsed p,usize expr,ref v::Vec<context::Binding> env,ref mut usize work){
    let id=switch(ungroup(p,expr,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let node=syntax::expression_at(p,id);if(node.kind==2){let found=switch(lookup(p,env,syntax::expression_text(p,id),work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        if(found!=0){let row=*v::at<context::Binding>(env,found-1);if(row.value!=0){return .ok(m::Value{.id=row.value,.codec=row.codec});}}return .err(1001);
    }
    let name=switch(call_name(p,id,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    if(!bytes::equal(bytes::view(ref name),b"storage::found") || syntax::call_argument_count(p,id)!=1){return .err(1001);}
    let arg=syntax::call_argument(p,id,0);if(!expression(p,arg) || syntax::expression_at(p,arg).kind!=2){return .err(1001);}
    let found=switch(lookup(p,env,syntax::expression_text(p,arg),work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    if(found==0){return .err(1001);}let row=*v::at<context::Binding>(env,found-1);if(row.found==0){return .err(1001);}return .ok(m::Value{.id=row.found,.codec=1});
}
pub fn Result<m::Operand,u32> operand(ref syntax::Parsed p,usize expr,u32 codec,ref v::Vec<context::Binding> env,ref mut bytes::Buffer data,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(!expression(p,expr) || codec<1 || codec>7){return .err(1001);}
    if(syntax::expression_at(p,expr).kind!=1){let value=switch(reference(p,expr,env,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        if(value.codec!=codec){return .err(1001);}return .ok(m::input(value.id));
    }
    var bytes::Buffer encoded=bytes::empty();
    if(codec==1){let raw=syntax::expression_text(p,expr);if(!bytes::equal(raw,b"true") && !bytes::equal(raw,b"false")){return .err(1001);}let ref u8[] bit=if(bytes::equal(raw,b"true")){b"\x01"}else{b"\x00"};let code=append(ref mut encoded,bit,work);if(code!=0){return .err(code);}}
    else if(codec==2 || codec==3){if(syntax::literal_kind(p,expr)!=5){return .err(1001);}let code=integer(syntax::expression_text(p,expr),if(codec==2){4}else{8},ref mut encoded,work);if(code!=0){return .err(code);}}
    else{encoded=switch(string(p,expr,if(codec==7){7}else{8},work)){case .ok(value):{value}case .err(code):{return .err(code);}};}
    let code=values::canonical(codec,bytes::view(ref encoded),work);if(code!=0){return .err(code);}let start=bytes::len(data);
    if(start>m::byte_limit() || bytes::len(ref encoded)>m::byte_limit()-start){return .err(1002);}
    let copied=append(data,bytes::view(ref encoded),work);if(copied!=0){return .err(copied);}return .ok(m::Operand{.input=0,.codec,.start,.end=bytes::len(data)});
}
pub fn Result<usize,u32> field(ref syntax::Parsed p,usize expr,ref state::Plan layouts,ref mut usize work){
    let name=switch(string(p,expr,7,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    if(state::count(layouts)==0){return .err(1001);}let layout=state::at(layouts,0);var usize i=0;
    while(i<layout.field_count){let current=state::name(layouts,layout.field_start+i);
        if(current.len>16777215-bytes::len(ref name) || !budget::charge(work,current.len+bytes::len(ref name)+1)){return .err(1002);}
        if(bytes::equal(current,bytes::view(ref name))){return .ok(i);}i+=1;
    }return .err(1001);
}
pub fn Result<m::Operation,u32> access(ref syntax::Parsed p,usize expr,ref state::Plan layouts,ref v::Vec<context::Binding> env,ref mut context::Counters ids,ref mut bytes::Buffer data,ref mut usize work){
    let name=switch(call_name(p,expr,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let u32 kind=if(bytes::equal(bytes::view(ref name),b"storage::read")){1}else if(bytes::equal(bytes::view(ref name),b"storage::write")){2}else if(bytes::equal(bytes::view(ref name),b"storage::delete")){3}else{0};
    let argc=syntax::call_argument_count(p,expr);if(kind==0 || argc==0){return .err(1001);}
    let selected=switch(field(p,syntax::call_argument(p,expr,0),layouts,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let f=state::field_at(layouts,state::at(layouts,0).field_start+selected);let mapped=f.kind==2;
    if(argc!=1+if(mapped){1}else{0}+if(kind==2){1}else{0}){return .err(1001);}
    var m::Operand key=m::absent();var m::Operand value=m::absent();
    if(mapped){key=switch(operand(p,syntax::call_argument(p,expr,1),f.key,env,data,work)){case .ok(value):{value}case .err(code):{return .err(code);}};}
    if(kind==2){value=switch(operand(p,syntax::call_argument(p,expr,argc-1),f.value,env,data,work)){case .ok(value):{value}case .err(code):{return .err(code);}};}
    var usize result=0;var usize found=0;
    if(kind==1){result=switch(context::value(ids,work)){case .ok(value):{value}case .err(code):{return .err(code);}};found=switch(context::value(ids,work)){case .ok(value):{value}case .err(code):{return .err(code);}};}
    return .ok(m::Operation{.kind=kind+if(mapped){3}else{0},.capability=ids.capability,.field=selected,.key,.value,.result,.found,.cost=1,.callee=0,.borrow_input=0,.borrow_result=0,.view_offset=0,.view_length=0});
}
