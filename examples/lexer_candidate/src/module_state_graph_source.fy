module module_state_graph_source;
import module_state_graph_model as m;
import module_state_graph_program as verified;
import module_state_graph_source_context as context;
import module_state_graph_source_flow as flow;
import module_state_declarations as declarations;
import module_state_layout as state;
import module_catalog as catalog;
import project_cache_encoding as budget;
import syntax as syntax;
import scanner as scanner;
import module_registry as paths;
import std::bytes as bytes;
import std::vec as v;

// Source production only. No Complete, artifact restoration or execution token.
pub struct Span{usize start;usize end;}
pub struct Owner{usize module_index;usize package;usize declaration;usize identity;Span declaration_span;Span body_span;Span effects_span;}
pub resource Produced{v::Vec<syntax::Parsed> sources;verified::Verified proof;Owner owner;v::Vec<context::Span> spans;Span exit;cleanup{}}
pub fn ref u8[] source(ref Produced p){return syntax::source_view(v::at<syntax::Parsed>(ref p.sources,0));}
pub fn ref syntax::Parsed parsed(ref Produced p){return v::at<syntax::Parsed>(ref p.sources,0);}
pub fn Owner owner(ref Produced p){return p.owner;}
pub fn ref u8[] body(ref Produced p){return source(p)[p.owner.body_span.start..p.owner.body_span.end];}
pub fn Span operation_span(ref Produced p,usize index){let s=*v::at<context::Span>(ref p.spans,index);return Span{.start=s.start,.end=s.end};}
pub fn Span exit_span(ref Produced p){return p.exit;}
pub fn ref m::Graph graph(ref Produced p){return verified::root(ref p.proof);}
pub fn ref verified::Verified program(ref Produced p){return ref p.proof;}
pub fn ref state::Plan layouts(ref Produced p){return verified::layouts(ref p.proof);}
pub fn bool matches(ref Produced p,ref u8[] current){return bytes::equal(source(p),current);}
pub fn usize source_limit(){return 65536;}
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn Result<u32,u32> effects(ref syntax::Parsed p,syntax::Declaration d,ref mut Span span,ref mut usize work){
    if(d.attribute_count>1){return .err(1001);}if(d.attribute_count==0){return .ok(0);}let a=syntax::attribute_at(p,d.attribute_start);*span=Span{.start=a.start,.end=a.end};
    if(!bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"effects")){return .err(1001);}var u32 result=0;var usize i=0;
    while(i<a.argument_count){if(!budget::charge(work,1)){return .err(1002);}let argument=syntax::attribute_argument_at(p,a.argument_start+i);let name=syntax::attribute_argument_text(p,a.argument_start+i);
        let u32 bit=if(bytes::equal(name,b"storage_read")){1}else if(bytes::equal(name,b"storage_write")){2}else{0};if(argument.kind!=1 || bit==0 || (result&bit)!=0){return .err(1001);}result|=bit;i+=1;
    }return .ok(result);
}
fn u32 function(ref syntax::Parsed p,usize at,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}let d=syntax::declaration_at(p,at);
    if(d.public || d.unsafe_end!=0 || d.owner_end!=0 || d.generic_parameters!=0 || d.body==0 || d.result_type==0){return 1001;}
    var usize parameter=0;while(parameter<d.entry_count){if(!budget::charge(work,syntax::entry_count(p)+1)){return 1002;}let entry=syntax::entry_at(p,d.entry_start+parameter);if(entry.kind==1){return 1001;}parameter+=1;}
    if(syntax::type_at(p,d.result_type).kind!=1 || !bytes::equal(syntax::type_text(p,d.result_type),b"void")){return 1001;}
    var Span span=Span{.start=0,.end=0};return switch(effects(p,d,ref mut span,work)){case .ok(_):{0}case .err(code):{code}};
}
// Sort only the small helper namespace, retaining exact source declaration IDs.
fn Result<v::Vec<context::Function>,u32> functions(ref syntax::Parsed p,ref mut usize root,ref mut usize work){
    var v::Vec<context::Function> out=v::empty<context::Function>();var usize states=0;var usize i=1;
    while(i<syntax::declaration_count(p)){if(!budget::charge(work,1)){return .err(1002);}let d=syntax::declaration_at(p,i);
        if(d.kind==10){states+=1;}else if(d.kind==3){let code=function(p,i,work);if(code!=0){return .err(code);}let name=syntax::declaration_name(p,i+1);
            if(bytes::equal(name,b"transaction")){if(*root!=0){return .err(1001);};*root=i+1;}else{
                if(v::len<context::Function>(ref out)>=verified::helper_limit()){return .err(1002);}var Span ignored=Span{.start=0,.end=0};let mask=switch(effects(p,d,ref mut ignored,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
                let row=context::Function{.declaration=i,.index=0,.effects=mask,.name=context::Span{.start=d.name_start,.end=d.name_end}};if(!v::push<context::Function>(ref mut out,row).is_ok){return .err(1003);}var usize at=v::len<context::Function>(ref out)-1;
                while(at>0){if(!budget::charge(work,1)){return .err(1002);}let a=*v::at<context::Function>(ref out,at-1);let b=*v::at<context::Function>(ref out,at);let raw=syntax::source_view(p);
                    if(!budget::charge(work,a.name.end-a.name.start+b.name.end-b.name.start+1)){return .err(1002);}if(!budget::before(raw[b.name.start..b.name.end],raw[a.name.start..a.name.end])){break;}
                    {let previous=v::at_mut<context::Function>(ref mut out,at-1);*previous=b;}{let current=v::at_mut<context::Function>(ref mut out,at);*current=a;}at-=1;
                }
            }
        }else{return .err(1001);}i+=1;
    }
    if(states!=1 || *root==0){return .err(1001);}i=0;while(i<v::len<context::Function>(ref out)){let current=v::at_mut<context::Function>(ref mut out,i);current.index=i+1;i+=1;}return .ok(out);
}
fn u32 append(ref mut bytes::Buffer out,ref u8[] data,ref mut usize work){
    if(data.len>m::byte_limit() || bytes::len(out)>m::byte_limit()-data.len){return 1002;}let code=budget::raw(out,data,work);return if(code==0){0}else if(code==4){1003}else{1002};
}
pub fn Result<Produced,u32> produce(ref u8[] input,ref mut usize work){
    if(input.len>source_limit() || !budget::charge(work,input.len+1)){return .err(1002);}
    let owned=switch(bytes::copy(input)){case .ok(value):{value}case .err(_):{return .err(1003);}};
    let stream=switch(scanner::collect(owned,false,65536,65536,256,16777216)){case .ok(value):{value}case .err(code):{return .err(if(code==3){1003}else if(code==4 || code==5){1002}else{1001});}};
    let parsed=switch(syntax::parse(stream,512)){case .ok(value):{value}case .err(code):{return .err(if(code==3){1003}else if(code==4){1002}else{1001});}};
    if(syntax::problem_count(ref parsed)!=0 || syntax::declaration_count(ref parsed)<3 || syntax::module_count(ref parsed)!=1){return .err(1001);}
    if(syntax::declaration_count(ref parsed)>verified::helper_limit()+3){return .err(1002);}let module_decl=syntax::declaration_at(ref parsed,0);if(module_decl.kind!=1 || module_decl.attribute_count!=0 || module_decl.public){return .err(1001);}
    var usize root=0;let helpers=switch(functions(ref parsed,ref mut root,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let d=syntax::declaration_at(ref parsed,root-1);var Span effect_span=Span{.start=0,.end=0};let mask=switch(effects(ref parsed,d,ref mut effect_span,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();if(!v::push<syntax::Parsed>(ref mut sources,parsed).is_ok){return .err(1003);}
    let c=switch(catalog::build(ref sources)){case .ok(value):{value}case .err(problem):{return .err(mapped(problem.code));}};
    var usize measured=0;let plan=switch(declarations::build_graph_metered(ref sources,ref c,ref mut measured)){case .ok(value):{if(!budget::charge(work,measured)){return .err(1002);}value}case .err(problem):{if(!budget::charge(work,measured)){return .err(1002);}return .err(mapped(problem.code));}};
    if(state::count(ref plan)!=1){return .err(1001);}let p=v::at<syntax::Parsed>(ref sources,0);let identity=catalog::identity(ref c,0,root);if(identity==0){return .err(1001);}
    var v::Vec<context::Span> spans=v::empty<context::Span>();var context::Span exit=context::Span{.start=0,.end=0};
    let graph=switch(flow::build(p,root-1,ref plan,ref helpers,identity,m::operation_limit(),ref mut spans,ref mut exit,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    if(m::header(ref graph).effects!=mask){return .err(1001);}if(v::len<m::Operation>(m::operations(ref graph))>m::operation_limit()){return .err(1002);}var usize remaining=m::operation_limit()-v::len<m::Operation>(m::operations(ref graph));
    var bytes::Buffer names=bytes::empty();var v::Vec<verified::Name> positions=v::empty<verified::Name>();var v::Vec<m::Graph> graphs=v::empty<m::Graph>();var usize i=0;
    var bytes::Buffer module_name=bytes::empty();let named=paths::normalize(syntax::source_view(p)[module_decl.name_start..module_decl.name_end],ref mut module_name);if(named!=0){return .err(mapped(named));}
    while(i<v::len<context::Function>(ref helpers)){let row=*v::at<context::Function>(ref helpers,i);let start=bytes::len(ref names);let prefix=append(ref mut names,bytes::view(ref module_name),work);if(prefix!=0){return .err(prefix);}let separator=append(ref mut names,b"::",work);if(separator!=0){return .err(separator);}let suffix=append(ref mut names,syntax::source_view(p)[row.name.start..row.name.end],work);if(suffix!=0){return .err(suffix);}
        if(!v::push<verified::Name>(ref mut positions,verified::Name{.start,.end=bytes::len(ref names)}).is_ok){return .err(1003);}var v::Vec<context::Span> discarded=v::empty<context::Span>();var context::Span last=context::Span{.start=0,.end=0};
        let helper=switch(flow::build(p,row.declaration,ref plan,ref helpers,identity,remaining,ref mut discarded,ref mut last,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        if(m::header(ref helper).effects!=row.effects){return .err(1001);}if(v::len<m::Operation>(m::operations(ref helper))>remaining){return .err(1002);}remaining-=v::len<m::Operation>(m::operations(ref helper));if(!v::push<m::Graph>(ref mut graphs,helper).is_ok){return .err(1003);}i+=1;
    }
    let proof=switch(verified::verify(verified::assemble(graph,graphs,positions,names),ref plan,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let block=syntax::statement_at(p,d.body);let owner=Owner{.module_index=0,.package=catalog::module_owner(ref c,0),.declaration=root,.identity,.declaration_span=Span{.start=d.start,.end=d.end},.body_span=Span{.start=block.start,.end=block.end},.effects_span=effect_span};
    return .ok(Produced{.sources,.proof,.owner,.spans,.exit=Span{.start=exit.start,.end=exit.end}});
}
