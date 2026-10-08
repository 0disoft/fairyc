module tests::sdk_buffer_foreign;
import tests::sdk_vector_expand as support;
import sdk_scopes as sdk;
import foreign_declarations as foreign;
import pointer_types as pointers;
import type_state as scalar;
import syntax as syntax;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn usize first(ref u8[] source,ref u8[] needle){var usize i=0;while(i<=source.len && needle.len<=source.len-i){if(bytes::equal(source[i..i+needle.len],needle)){return i+1;}i+=1;}return 0;}

// Exercise Buffer extern authority independently of executable SDK bodies.
fn Result<m::Program,m::Problem> owned(ref u8[] input,u32 mutation){
    let p=support::text(input);var v::Vec<sdk::Scope> scopes=switch(sdk::collect(ref p)){case .ok(s):{s}case .err(_):{assert(false);return owned(input,mutation);}};
    assert(v::len<sdk::Scope>(ref scopes)==1 && v::at<sdk::Scope>(ref scopes,0).sdk_id==2);
    {let ref mut sdk::Scope s=v::at_mut<sdk::Scope>(ref mut scopes,0);s.start+=5;s.end+=5;if(mutation==4){s.sdk_id=6;}}
    if(mutation==1){discard v::clear<sdk::Scope>(ref mut scopes);}
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"main0").is_ok && bytes::append(ref mut source,input).is_ok);
    if(mutation==7){let ref mut u8[] raw=bytes::view_mut(ref mut source);raw[5]=77;}
    var v::Vec<m::Target> targets=v::empty<m::Target>();var v::Vec<m::Signature> signatures=v::empty<m::Signature>();
    assert(v::push<m::Target>(ref mut targets,m::Target{.foreign=false,.link_start=0,.link_end=0,.unsafe_call=false,.effects=0,.binding=1,.module_index=0,.declaration=1,.result=1,.signature_start=0,.signature_count=0,.start=0,.end=4,.external=false}).is_ok);
    var usize index=0;var usize count=0;
    while(index<syntax::declaration_count(ref p)){let d=syntax::declaration_at(ref p,index);if(d.kind==9){
        let info=foreign::read(ref p,index);assert(info.code==0 && sdk::parsed_foreign(ref p,index,info.link_start,info.link_end));
        let usize start=v::len<m::Signature>(ref signatures);var usize e=d.entry_start;
        while(e<d.entry_start+d.entry_count){let entry=syntax::entry_at(ref p,e);if(entry.kind==1){let n=syntax::type_at(ref p,entry.type_root);var u32 type=if(n.kind==1){scalar::primitive(ref p,entry.type_root)}else{pointers::resolve(ref p,entry.type_root)};
            if(mutation==3 && count==0 && e==d.entry_start){type=4;}assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type}).is_ok);
        }e+=1;}
        let n=syntax::type_at(ref p,d.result_type);var u32 result=if(n.kind==1){scalar::primitive(ref p,d.result_type)}else{pointers::resolve(ref p,d.result_type)};
        if(mutation==2 && count==0){result=5;}
        var usize name_start=5+d.name_start;
        if(mutation==5 && count==6){name_start=5+first(input,b"buffer_drop_raw")-1;assert(name_start<5+d.name_start);}
        assert(v::push<m::Target>(ref mut targets,m::Target{.foreign=true,.link_start=5+info.link_start,.link_end=5+info.link_end,.unsafe_call=true,.effects=if(mutation==6 && count==0){1}else{0},.binding=count+2,.module_index=0,.declaration=index+2,.result,.signature_start=start,.signature_count=v::len<m::Signature>(ref signatures)-start,.start=name_start,.end=name_start+d.name_end-d.name_start,.external=true}).is_ok);count+=1;
    }index+=1;}assert(count==7);
    var bytes::Buffer payload=bytes::empty();assert(bytes::append(ref mut payload,b"0").is_ok);
    var v::Vec<m::Function> functions=v::empty<m::Function>();assert(v::push<m::Function>(ref mut functions,m::Function{.target=1,.entry=1,.local_start=0,.local_count=1,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=4,.public=false}).is_ok);
    var v::Vec<m::Local> locals=v::empty<m::Local>();assert(v::push<m::Local>(ref mut locals,m::Local{.function=1,.binding=0,.type=1,.parameter=false,.mutable=false,.start=4,.end=5}).is_ok);
    var m::Operation zero=state::operation(1,1,1,1,0,0,0,4,5);zero.function=1;zero.block=1;zero.payload_end=1;
    var v::Vec<m::Operation> ops=v::empty<m::Operation>();assert(v::push<m::Operation>(ref mut ops,zero).is_ok);
    var m::Block done=state::block(1,0,5);done.kind=1;done.value=1;done.first=1;done.last=1;
    var v::Vec<m::Block> blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    return m::assemble_nominal(source,payload,targets,signatures,functions,locals,blocks,ops,v::empty<usize>(),0,shapes::assemble_sdk(v::empty<shapes::Shape>(),v::empty<shapes::Field>(),v::empty<shapes::Abi>(),scopes));
}
@effects(filesystem) @test fn void buffer_foreign_coordinates_and_signatures_survive_parser_owners(){
    let input=support::sdk_source();let p=switch(owned(bytes::view(ref input),0)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0 && m::target_count(ref p)==8);
    var bytes::Buffer output=bytes::empty();assert(c::render(ref p,ref mut output)==0);
    var usize i=2;while(i<=8){let target=m::target_at(ref p,i);let usize rule=shapes::sdk_rule(m::nominal(ref p),m::source_view(ref p),target.start,target.end,target.link_start,target.link_end);
        assert(rule==i+7 && first(bytes::view(ref output),m::foreign_text(ref p,i))!=0);i+=1;
    }
    var u32 mode=1;while(mode<=7){switch(owned(bytes::view(ref input),mode)){
        case .ok(bad):{assert(mode!=4 && mode!=7);assert(verify::check(ref bad).code==if(mode==1 || mode==5){5106}else{5102});var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);assert(c::render(ref bad,ref mut out)!=0 && bytes::equal(bytes::view(ref out),b"prefix"));}
        case .err(e):{assert((mode==4 || mode==7) && e.code==5102);}
    };mode+=1;}
    let fake=support::text(b"module user;@link_name(\"fy_std_buffer_initialize_v0\")unsafe extern C fn u32 buffer_initialize_raw(usize capacity,ptr mut void output);");assert(foreign::read(ref fake,1).code==5106);
}
