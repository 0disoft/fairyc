module module_state_graph_source;
import module_state_graph_model as m;
import module_state_graph_verify as core;
import module_state_graph_values as values;
import module_state_declarations as declarations;
import module_state_layout as state;
import module_catalog as catalog;
import project_cache_encoding as budget;
import literal_types as literals;
import syntax as syntax;
import scanner as scanner;
import module_registry as paths;
import std::bytes as bytes;
import std::vec as v;

// Source production only. No Complete, artifact restoration or execution token.
pub struct Span{usize start;usize end;}
pub struct Owner{usize module_index;usize package;usize declaration;usize identity;Span declaration_span;Span body_span;Span effects_span;}
pub resource Produced{v::Vec<syntax::Parsed> sources;core::Verified proof;Owner owner;v::Vec<Span> spans;Span exit;cleanup{}}
pub fn ref u8[] source(ref Produced p){return syntax::source_view(v::at<syntax::Parsed>(ref p.sources,0));}
pub fn Owner owner(ref Produced p){return p.owner;}
pub fn ref u8[] body(ref Produced p){return source(p)[p.owner.body_span.start..p.owner.body_span.end];}
pub fn Span operation_span(ref Produced p,usize index){return *v::at<Span>(ref p.spans,index);}
pub fn Span exit_span(ref Produced p){return p.exit;}
pub fn ref m::Graph graph(ref Produced p){return core::graph(ref p.proof);}
pub fn ref state::Plan layouts(ref Produced p){return core::layouts(ref p.proof);}
pub fn bool matches(ref Produced p,ref u8[] current){return bytes::equal(source(p),current);}
pub fn usize source_limit(){return 65536;}
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn u8 octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn u32 append(ref mut bytes::Buffer out,ref u8[] raw,ref mut usize work){let code=budget::raw(out,raw,work);return if(code==0){0}else if(code==4){1003}else{1002};}
fn Result<bytes::Buffer,u32> string(ref syntax::Parsed p,usize id,u32 kind,ref mut usize work){
    if(id==0 || syntax::expression_at(p,id).kind!=1 || syntax::literal_kind(p,id)!=kind){return .err(1001);}
    let cooked=syntax::literal_cooked(p,id);if(!budget::charge(work,cooked.len+1)){return .err(1002);}
    var bytes::Buffer out=bytes::empty();let code=literals::decode(ref mut out,cooked);
    if(code!=0){return .err(if(code==3){1003}else{1001});}return .ok(out);
}
// Decimal multiplication in canonical bytes accepts all u64 values without
// relying on host usize width or silently wrapping at the u32/u64 boundary.
fn u32 integer(ref u8[] raw,usize width,ref mut bytes::Buffer out,ref mut usize work){
    if(raw.len==0 || !budget::charge(work,raw.len*width+1)){return if(raw.len==0){1001}else{1002};}
    var u8[8] encoded=[0;8];var usize i=0;while(i<raw.len){if(raw[i]<48 || raw[i]>57){return 1001;}
        var usize carry=(raw[i]-48) as usize;var usize j=width;while(j>0){j-=1;let digit=(encoded[j] as usize)*10+carry;encoded[j]=octet(digit%256);carry=digit/256;}
        if(carry!=0){return 1001;}i+=1;
    }let ref u8[] view=ref encoded;return append(out,view[..width],work);
}
fn Result<m::Operand,u32> operand(ref syntax::Parsed p,usize id,u32 codec,ref mut bytes::Buffer data,ref mut usize work){
    if(id==0 || syntax::expression_at(p,id).kind!=1){return .err(1001);}let start=bytes::len(data);var bytes::Buffer encoded=bytes::empty();
    if(codec==1){let raw=syntax::expression_text(p,id);if(!bytes::equal(raw,b"true") && !bytes::equal(raw,b"false")){return .err(1001);}let ref u8[] octet=if(bytes::equal(raw,b"true")){b"\x01"}else{b"\x00"};let code=append(ref mut encoded,octet,work);if(code!=0){return .err(code);}}
    else if(codec==2 || codec==3){if(syntax::literal_kind(p,id)!=5){return .err(1001);}let code=integer(syntax::expression_text(p,id),if(codec==2){4}else{8},ref mut encoded,work);if(code!=0){return .err(code);}}
    else if(codec>=4 && codec<=7){encoded=switch(string(p,id,if(codec==7){7}else{8},work)){case .ok(value):{value}case .err(code):{return .err(code);}};}
    else{return .err(1001);}
    let code=values::canonical(codec,bytes::view(ref encoded),work);if(code!=0){return .err(code);}let copied=append(data,bytes::view(ref encoded),work);if(copied!=0){return .err(copied);}
    return .ok(m::Operand{.input=0,.codec,.start,.end=bytes::len(data)});
}
fn Result<usize,u32> field(ref syntax::Parsed p,usize id,ref state::Plan plan,ref mut usize work){
    let name=switch(string(p,id,7,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let layout=state::at(plan,0);var usize i=0;while(i<layout.field_count){let current=state::name(plan,layout.field_start+i);if(!budget::charge(work,current.len+bytes::len(ref name)+1)){return .err(1002);}
        if(bytes::equal(current,bytes::view(ref name))){return .ok(i);}i+=1;
    }return .err(1001);
}
fn Result<u32,u32> effects(ref syntax::Parsed p,syntax::Declaration d,ref mut Span span,ref mut usize work){
    if(d.attribute_count>1){return .err(1001);}if(d.attribute_count==0){return .ok(0);}let a=syntax::attribute_at(p,d.attribute_start);*span=Span{.start=a.start,.end=a.end};
    if(!bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"effects")){return .err(1001);}var u32 result=0;var usize i=0;
    while(i<a.argument_count){if(!budget::charge(work,1)){return .err(1002);}let argument=syntax::attribute_argument_at(p,a.argument_start+i);let name=syntax::attribute_argument_text(p,a.argument_start+i);
        let u32 bit=if(bytes::equal(name,b"storage_read")){1}else if(bytes::equal(name,b"storage_write")){2}else{0};if(argument.kind!=1 || bit==0 || (result&bit)!=0){return .err(1001);}result=result|bit;i+=1;
    }return .ok(result);
}
fn u32 lower_body(ref syntax::Parsed p,usize body,ref state::Plan plan,usize capability,ref mut bytes::Buffer data,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<Span> spans,ref mut m::Terminator term,ref mut Span exit,ref mut usize work){
    let block=syntax::statement_at(p,body);if(block.kind!=1 || block.expression!=0 || block.second==0){return 1001;}if(block.second>m::operation_limit()+1){return 1002;}
    // Parsed block links are reversed. Materialize their IDs once, preserving
    // source order in O(N) instead of repeatedly walking the linked tail.
    var usize[1025] statements=[0;1025];var usize link=block.first;var usize i=block.second;
    while(i>0){if(!budget::charge(work,1)){return 1002;}i-=1;let row=syntax::statement_at(p,link);statements[i]=row.first;link=row.second;}
    var usize next=1;i=0;while(i<block.second){if(!budget::charge(work,1)){return 1002;}let s=syntax::statement_at(p,statements[i]);if(s.kind!=6 || s.expression==0 || term.kind!=0){return 1001;}
        let e=syntax::expression_at(p,s.expression);if(e.kind!=6 || e.left==0 || syntax::expression_at(p,e.left).kind!=2){return 1001;}
        let raw_name=syntax::expression_text(p,e.left);if(!budget::charge(work,raw_name.len+1)){return 1002;}var bytes::Buffer canonical=bytes::empty();let named=paths::normalize(raw_name,ref mut canonical);if(named!=0){return mapped(named);}let name=bytes::view(ref canonical);let argc=syntax::call_argument_count(p,s.expression);
        if(argc>3){return 1001;}var usize arg=0;while(arg<argc){if(syntax::call_argument_labeled(p,syntax::call_argument_entry(p,s.expression,arg))){return 1001;}arg+=1;}
        if(bytes::equal(name,b"storage::commit") || bytes::equal(name,b"storage::reject") || bytes::equal(name,b"storage::trap")){
            let trap=bytes::equal(name,b"storage::trap");if(argc!=if(trap){1}else{0}){return 1001;};*term=m::exit(if(trap){5}else if(bytes::equal(name,b"storage::commit")){3}else{4});*exit=Span{.start=s.start,.end=s.end};
            if(trap){let category=switch(string(p,syntax::call_argument(p,s.expression,0),7,work)){case .ok(value):{value}case .err(code):{return code;}};let code=values::category(bytes::view(ref category),work);if(code!=0){return code;}
                term.category_start=bytes::len(data);let copied=append(data,bytes::view(ref category),work);if(copied!=0){return copied;}term.category_end=bytes::len(data);}
        }else{
            let u32 kind=if(bytes::equal(name,b"storage::read")){1}else if(bytes::equal(name,b"storage::write")){2}else if(bytes::equal(name,b"storage::delete")){3}else{0};if(kind==0 || argc==0){return 1001;}
            if(v::len<m::Operation>(ops)>=m::operation_limit()){return 1002;}let selected=switch(field(p,syntax::call_argument(p,s.expression,0),plan,work)){case .ok(value):{value}case .err(code):{return code;}};
            let f=state::field_at(plan,state::at(plan,0).field_start+selected);let usize count=1+if(f.kind==2){1}else{0}+if(kind==2){1}else{0};if(argc!=count){return 1001;}
            var m::Operand key=m::absent();var m::Operand value=m::absent();if(f.kind==2){key=switch(operand(p,syntax::call_argument(p,s.expression,1),f.key,data,work)){case .ok(value):{value}case .err(code):{return code;}};}
            if(kind==2){value=switch(operand(p,syntax::call_argument(p,s.expression,argc-1),f.value,data,work)){case .ok(value):{value}case .err(code):{return code;}};}
            let read=kind==1;let op=m::Operation{.kind=kind+if(f.kind==2){3}else{0},.capability,.field=selected,.key,.value,.result=if(read){next}else{0},.found=if(read){next+1}else{0},.cost=1,.callee=0,.borrow_input=0,.borrow_result=0,.view_offset=0,.view_length=0};
            if(!v::push<m::Operation>(ops,op).is_ok || !v::push<Span>(spans,Span{.start=e.start,.end=e.end}).is_ok){return 1003;}if(read){next+=2;}
        }i+=1;
    }return if(term.kind!=0){0}else{1001};
}
// Input is copied before parsing. No caller-provided Parsed/layout/IR is trusted.
// Errors: 1001 malformed/closed source shape, 1002 bounded work/size, 1003 allocation.
pub fn Result<Produced,u32> produce(ref u8[] input,ref mut usize work){
    if(input.len>source_limit() || !budget::charge(work,input.len+1)){return .err(1002);}
    let owned=switch(bytes::copy(input)){case .ok(value):{value}case .err(_):{return .err(1003);}};
    let stream=switch(scanner::collect(owned,false,65536,65536,256,16777216)){case .ok(value):{value}case .err(code):{return .err(if(code==3){1003}else if(code==4 || code==5){1002}else{1001});}};
    let parsed=switch(syntax::parse(stream,512)){case .ok(value):{value}case .err(code):{return .err(if(code==3){1003}else if(code==4){1002}else{1001});}};
    if(syntax::problem_count(ref parsed)!=0 || syntax::declaration_count(ref parsed)!=3 || syntax::module_count(ref parsed)!=1){return .err(1001);}
    let module_decl=syntax::declaration_at(ref parsed,0);if(module_decl.kind!=1 || module_decl.attribute_count!=0 || module_decl.public){return .err(1001);}var usize root=0;var usize states=0;var usize i=1;
    while(i<3){let d=syntax::declaration_at(ref parsed,i);if(d.kind==10){states+=1;}else if(d.kind==3 && root==0){root=i;}else{return .err(1001);}i+=1;}
    if(states!=1 || root==0){return .err(1001);}let d=syntax::declaration_at(ref parsed,root);
    if(d.public || d.unsafe_end!=0 || d.owner_end!=0 || d.generic_parameters!=0 || d.entry_count!=0 || d.body==0 || d.result_type==0 || !bytes::equal(syntax::declaration_name(ref parsed,root+1),b"transaction")){return .err(1001);}
    if(syntax::type_at(ref parsed,d.result_type).kind!=1 || !bytes::equal(syntax::type_text(ref parsed,d.result_type),b"void")){return .err(1001);}
    var Span effect_span=Span{.start=0,.end=0};let mask=switch(effects(ref parsed,d,ref mut effect_span,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();if(!v::push<syntax::Parsed>(ref mut sources,parsed).is_ok){return .err(1003);}
    let c=switch(catalog::build(ref sources)){case .ok(value):{value}case .err(problem):{return .err(mapped(problem.code));}};
    var usize measured=0;let plan=switch(declarations::build_graph_metered(ref sources,ref c,ref mut measured)){case .ok(value):{if(!budget::charge(work,measured)){return .err(1002);}value}case .err(problem):{if(!budget::charge(work,measured)){return .err(1002);}return .err(mapped(problem.code));}};
    if(state::count(ref plan)!=1){return .err(1001);}let p=v::at<syntax::Parsed>(ref sources,0);let identity=catalog::identity(ref c,0,root+1);if(identity==0){return .err(1001);}
    var bytes::Buffer data=bytes::empty();var v::Vec<m::Operation> ops=v::empty<m::Operation>();var v::Vec<Span> spans=v::empty<Span>();var m::Terminator term=m::exit(0);var Span exit=Span{.start=0,.end=0};
    let code=lower_body(p,d.body,ref plan,identity,ref mut data,ref mut ops,ref mut spans,ref mut term,ref mut exit,work);if(code!=0){return .err(code);}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();if(!v::push<m::Block>(ref mut blocks,m::Block{.id=0,.parameter_start=0,.parameter_count=0,.operation_start=0,.operation_count=v::len<m::Operation>(ref ops),.term}).is_ok){return .err(1003);}
    let graph=m::assemble(m::Header{.layout=0,.capability=identity,.effects=mask,.entry=0},data,v::empty<m::Value>(),v::empty<usize>(),ops,blocks);
    let proof=switch(core::verify(graph,ref plan,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let block=syntax::statement_at(p,d.body);let owner=Owner{.module_index=0,.package=catalog::module_owner(ref c,0),.declaration=root+1,.identity,.declaration_span=Span{.start=d.start,.end=d.end},.body_span=Span{.start=block.start,.end=block.end},.effects_span=effect_span};
    return .ok(Produced{.sources,.proof,.owner,.spans,.exit});
}
