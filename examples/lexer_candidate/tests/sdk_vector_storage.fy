module tests::sdk_vector_storage;
import tests::sdk_vector_expand as support;
import module_program as program;
import syntax as syntax;
import sdk_scopes as sdk;
import mir_model as m;
import mir_verify as verify;
import hir_shapes as shapes;
import c_emit as c;
import std::vec as v;
import std::bytes as bytes;

fn m::Program retained(ref u8[] raw){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(b"module main;import std::vec as v;struct Before{i32 n;}fn i32 main(){var v::Vec<i32> x=v::empty<i32>();assert(v::push<i32>(ref mut x,17).is_ok);let Option<i32> value=v::pop<i32>(ref mut x);assert(switch(value){case .some(n):{n==17}case .none:{false}});return 7;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(raw)).is_ok);
    return switch(program::compile(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained(raw);}};
}
fn Result<m::Program,m::Problem> changed(ref m::Program p,u32 mode){
    var bytes::Buffer source=bytes::empty();var bytes::Buffer payload=bytes::empty();assert(bytes::append(ref mut source,m::source_view(p)).is_ok);
    var v::Vec<m::Target> targets=v::empty<m::Target>();var usize i=1;while(i<=m::target_count(p)){assert(v::push<m::Target>(ref mut targets,m::target_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Signature> signatures=v::empty<m::Signature>();i=0;while(i<m::signature_count(p)){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=m::signature_at(p,i)}).is_ok);i+=1;}
    var v::Vec<m::Function> functions=v::empty<m::Function>();i=1;while(i<=m::function_count(p)){assert(v::push<m::Function>(ref mut functions,m::function_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Local> locals=v::empty<m::Local>();i=1;while(i<=m::local_count(p)){assert(v::push<m::Local>(ref mut locals,m::local_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Block> blocks=v::empty<m::Block>();i=1;while(i<=m::block_count(p)){assert(v::push<m::Block>(ref mut blocks,m::block_at(p,i)).is_ok);i+=1;}
    var v::Vec<m::Operation> ops=v::empty<m::Operation>();var bool altered=false;i=1;while(i<=m::operation_count(p)){var o=m::operation_at(p,i);if(o.payload_end>bytes::len(ref payload)){assert(bytes::resize_zeroed(ref mut payload,o.payload_end).is_ok);}if(o.payload_end>o.payload_start){let text=m::literal_text(p,i);let ref mut u8[] data=bytes::view_mut(ref mut payload);var usize j=0;while(j<text.len){data[o.payload_start+j]=text[j];j+=1;}}
        if(o.kind==39 && !altered && (mode<8 || mode<15 && o.operator==6 || mode>=15 && o.operator==8)){if(mode==1){o.operator=5;}else if(mode==2){o.owner_type=98304;}else if(mode==3){o.type=7;}else if(mode==4 || mode==8 || mode==20){o.boolean=false;}else if(mode==5 || mode==13 || mode==21){o.start=0;o.end=1;}else if(mode==6){o.argument_count=1;}else if(mode==9){o.operator=7;}else if(mode==10){o.left=o.right;}else if(mode==11){o.owner_type=3;}else if(mode==12 || mode==18){o.right=o.left;}else if(mode==14){o.left=0;}else if(mode==15){o.operator=6;}else if(mode==16){o.type=11;}else if(mode==17){o.argument_count=0;}else if(mode==19){o.owner_type=2;}else if(mode==22){o.argument_start=m::edge_count(p);}altered=true;}
        assert(v::push<m::Operation>(ref mut ops,o).is_ok);i+=1;
    }assert(altered);
    var v::Vec<usize> edges=v::empty<usize>();i=0;while(i<m::edge_count(p)){assert(v::push<usize>(ref mut edges,m::edge_at(p,i)).is_ok);i+=1;}
    var v::Vec<shapes::Shape> records=v::empty<shapes::Shape>();i=0;while(i<shapes::count(m::nominal(p))){assert(v::push<shapes::Shape>(ref mut records,shapes::at(m::nominal(p),i)).is_ok);i+=1;}
    var v::Vec<shapes::Field> fields=v::empty<shapes::Field>();i=1;while(i<=shapes::field_count(m::nominal(p))){assert(v::push<shapes::Field>(ref mut fields,shapes::field_at(m::nominal(p),i)).is_ok);i+=1;}
    var v::Vec<shapes::Abi> abi=v::empty<shapes::Abi>();i=0;while(i<shapes::abi_count(m::nominal(p))){assert(v::push<shapes::Abi>(ref mut abi,shapes::abi_at(m::nominal(p),i)).is_ok);i+=1;}
    var v::Vec<sdk::Scope> scopes=v::empty<sdk::Scope>();i=0;while(i<shapes::sdk_count(m::nominal(p))){let scope=shapes::sdk_at(m::nominal(p),i);assert(v::push<sdk::Scope>(ref mut scopes,scope).is_ok);if(mode==7){let ref mut u8[] raw=bytes::view_mut(ref mut source);raw[scope.start+1]=77;}i+=1;}
    var v::Vec<m::Drop> drops=v::empty<m::Drop>();i=0;while(i<m::drop_count(p)){assert(v::push<m::Drop>(ref mut drops,m::drop_at(p,i)).is_ok);i+=1;}
    return m::assemble_resources(source,payload,targets,signatures,functions,locals,blocks,ops,edges,m::module_index(p),shapes::assemble_sdk(records,fields,abi,scopes),drops);
}
fn m::Program borrowed(ref u8[] raw){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(b"module main;import std::vec as v;fn i32 main(){var v::Vec<i32> x=v::empty<i32>();assert(v::push<i32>(ref mut x,17).is_ok);assert(*v::at<i32>(ref x,0)==17);let ref mut i32 a=v::at_mut<i32>(ref mut x,0);*a=23;assert(*a==23);var v::Vec<i64> y=v::empty<i64>();assert(v::push<i64>(ref mut y,41).is_ok);assert(*v::at<i64>(ref y,0)==41);return 7;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(raw)).is_ok);
    return switch(program::compile(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return borrowed(raw);}};
}
fn m::Program viewed(ref u8[] raw){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(b"module main;import std::vec as v;fn i32 main(){var v::Vec<u8> x=v::empty<u8>();assert(v::push<u8>(ref mut x,7).is_ok);let ref u8[] a=v::view<u8>(ref x);assert(a[..][0]==7);let ref mut u8[] b=v::view_mut_u8(ref mut x);b[0]=11;assert(b[0]==11);return 7;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,support::text(raw)).is_ok);
    return switch(program::compile(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return viewed(raw);}};
}
@effects(filesystem)
@test fn void byte_views_retain_owned_authority_and_count(){
    let source=support::sdk_source();let p=viewed(bytes::view(ref source));assert(verify::check(ref p).code==0);
    var bool[10] seen=[false;10];var usize i=1;while(i<=m::operation_count(ref p)){let o=m::operation_at(ref p,i);if(o.kind==39){seen[o.operator as usize]=true;}i+=1;}
    assert(seen[8] && seen[9]);var bytes::Buffer good=bytes::empty();assert(c::render(ref p,ref mut good)==0);
    var u32 mode=15;while(mode<=22){switch(changed(ref p,mode)){
        case .ok(bad):{assert(verify::check(ref bad).code!=0);var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));}
        case .err(e):{assert(e.code!=0);}
    };mode+=1;}
}
@effects(filesystem)
@test fn void element_borrows_retain_owned_authority_and_origin(){
    let source=support::sdk_source();let p=borrowed(bytes::view(ref source));assert(verify::check(ref p).code==0);
    var bool[8] seen=[false;8];var usize i=1;while(i<=m::operation_count(ref p)){let o=m::operation_at(ref p,i);if(o.kind==39){seen[o.operator as usize]=true;}i+=1;}
    assert(seen[5] && seen[6] && seen[7]);var bytes::Buffer good=bytes::empty();assert(c::render(ref p,ref mut good)==0);
    var u32 mode=8;while(mode<=14){switch(changed(ref p,mode)){
        case .ok(bad):{assert(verify::check(ref bad).code!=0);var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));}
        case .err(e):{assert(e.code!=0);}
    };mode+=1;}
}
@effects(filesystem)
@test fn void storage_authority_and_scalar_shapes_survive_owner_removal(){
    let source=support::sdk_source();let p=retained(bytes::view(ref source));assert(verify::check(ref p).code==0);
    var bool[5] seen=[false;5];var usize i=1;while(i<=m::operation_count(ref p)){let o=m::operation_at(ref p,i);if(o.kind==39){assert(o.owner_type==1 && o.operator>=1 && o.operator<=4);seen[o.operator as usize]=true;}i+=1;}
    assert(seen[1] && seen[2] && seen[3] && seen[4]);var bytes::Buffer good=bytes::empty();assert(c::render(ref p,ref mut good)==0);
    var u32 mode=1;while(mode<=7){switch(changed(ref p,mode)){
        case .ok(bad):{assert(verify::check(ref bad).code!=0);var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));}
        case .err(e):{assert(e.code!=0);}
    };mode+=1;}
}
