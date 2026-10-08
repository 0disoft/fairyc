module tests::sdk_foreign;
import sdk_scopes as sdk;
import scanner as scanner;
import syntax as syntax;
import module_program as programs;
import hir_shapes as shapes;
import mir_model as m;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;
import std::fs as fs;
import std::process as process;
import std::text as text;

@effects(filesystem) fn bytes::Buffer source(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize length=switch(process::arg(2,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};
    let filename=switch(text::validate_utf8(bytes::view(ref path)[..length])){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};
    var fs::File f=switch(fs::open_read_utf8(filename)){case .ok(f):{f}case .err(_):{assert(false);return bytes::empty();}};
    var bytes::Buffer b=bytes::empty();assert(bytes::resize_zeroed(ref mut b,65537).is_ok);var usize used=0;
    while(true){let usize n=switch(fs::read(ref mut f,bytes::view_mut(ref mut b)[used..])){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};used+=n;assert(used<=65536);if(n==0){break;}}
    assert(fs::close(f).is_ok && bytes::resize_zeroed(ref mut b,used).is_ok);return b;
}
fn syntax::Parsed parsed(bytes::Buffer b){let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}
fn syntax::Parsed literal(ref u8[] s){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,s).is_ok);return parsed(b);}
fn m::Program owned(bytes::Buffer source){
    var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut files,literal(b"/* caller trivia */ module main;import std::process as p;fn i32 main(){assert(p::arg_count()>0);return 7;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut files,parsed(source)).is_ok);
    assert(v::push<syntax::Parsed>(ref mut files,literal(b"/* following file trivia */ module helper;fn i32 value(){return 7;}")).is_ok);
    return switch(programs::compile(ref files)){case .ok(p):{p}case .err(_):{assert(false);return owned(bytes::empty());}};
}
fn Result<m::Program,m::Problem> changed(ref m::Program p,u32 mode){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,m::source_view(p)).is_ok);
    let scope=shapes::sdk_at(m::nominal(p),0);
    if(mode==3){let ref mut u8[] raw=bytes::view_mut(ref mut source);raw[scope.start]=77;}
    var bytes::Buffer payload=bytes::empty();var v::Vec<m::Operation> ops=v::empty<m::Operation>();var usize i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.payload_end>o.payload_start){assert(bytes::append(ref mut payload,m::literal_text(p,i)).is_ok);}assert(v::push<m::Operation>(ref mut ops,o).is_ok);i+=1;}
    var v::Vec<m::Target> targets=v::empty<m::Target>();i=1;var bool altered=false;
    while(i<=m::target_count(p)){var t=m::target_at(p,i);if(mode==2 && t.foreign && !altered){t.result=4;altered=true;}assert(v::push<m::Target>(ref mut targets,t).is_ok);i+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();i=0;while(i<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,i)}).is_ok);i+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();i=1;while(i<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();i=1;while(i<=m::local_count(p)){assert(v::push<m::Local>(ref mut locals,m::local_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();i=1;while(i<=m::block_count(p)){assert(v::push<m::Block>(ref mut blocks,m::block_at(p,i)).is_ok);i+=1;}
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<m::edge_count(p)){assert(v::push<usize>(ref mut edges,m::edge_at(p,i)).is_ok);i+=1;}
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();i=0;while(i<shapes::count(m::nominal(p))){assert(v::push<shapes::Shape>(ref mut records,shapes::at(m::nominal(p),i)).is_ok);i+=1;}
    var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();i=1;while(i<=shapes::field_count(m::nominal(p))){var f=shapes::field_at(m::nominal(p),i);if(mode==4 && f.type==5 && !altered){f.type=4;altered=true;}assert(v::push<shapes::Field>(ref mut fields,f).is_ok);i+=1;}
    var v::Vec<shapes::Abi> abi=v::empty<shapes::Abi>();i=0;while(i<shapes::abi_count(m::nominal(p))){assert(v::push<shapes::Abi>(ref mut abi,shapes::abi_at(m::nominal(p),i)).is_ok);i+=1;}
    var v::Vec<sdk::Scope> scopes=v::empty<sdk::Scope>();if(mode!=1){assert(v::push<sdk::Scope>(ref mut scopes,scope).is_ok);}
    return m::assemble_nominal(source,payload,targets,signatures,functions,locals,blocks,ops,edges,0,shapes::assemble_sdk(records,fields,abi,scopes));
}
@effects(filesystem) @test fn void sdk_source_and_foreign_signature_survive_owner_removal(){
    let p=owned(source());assert(shapes::sdk_count(m::nominal(ref p))==1 && shapes::sdk_at(m::nominal(ref p),0).sdk_id==6);
    assert(verify::check(ref p).code==0);var bytes::Buffer good=bytes::empty();assert(c::render(ref p,ref mut good)==0);
    var u32 mode=1;while(mode<=4){switch(changed(ref p,mode)){
        case .ok(bad):{assert(mode!=3);assert(verify::check(ref bad).code==if(mode==1){5106}else{5102});var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));}
        case .err(e):{assert(mode==3 && e.code==5102);}
    };mode+=1;}
}
