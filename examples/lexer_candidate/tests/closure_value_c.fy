module tests::closure_value_c;
import c_emit as emit;
import c_closure_values as closures;
import mir_model as m;
import mir_lower as mir;
import mir_verify as proof;
import hir_lower as hir;
import hir_shapes as shapes;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn void phase(ref u8[] name,ref u8[] raw,ref u8[] stage,u32 code,usize start,usize end){
    discard io::write_stderr("closure-C case=");discard io::write_stderr_bytes(name);discard io::write_stderr(" phase=");discard io::write_stderr_bytes(stage);
    discard io::write_stderr(" code=");discard io::write_stderr_u32(code);discard io::write_stderr(" span=");discard io::write_stderr_usize(start);discard io::write_stderr("..");discard io::write_stderr_usize(end);
    discard io::write_stderr("\nsource: ");discard io::write_stderr_bytes(raw);discard io::write_stderr("\n");assert(false);
}
fn u32 positive_code(i32 value){assert(value>=0 && value<=65536);var i32 left=value;var u32 code=0;while(left!=0){left-=1;code+=1;}return code;}
@effects(console) fn syntax::Parsed parsed(ref u8[] raw,ref u8[] name){
    let input=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{phase(name,raw,b"source-copy",3,0,raw.len);return parsed(raw,name);}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(code):{phase(name,raw,b"scanner",positive_code(code),0,raw.len);return parsed(raw,name);}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(code):{phase(name,raw,b"preparation",positive_code(code),0,raw.len);return parsed(raw,name);}};
    if(syntax::problem_count(ref p)!=0){let e=syntax::problem_at(ref p,0);phase(name,raw,b"syntax",e.code,e.start,e.end);}return p;
}
@effects(console) fn m::Program program(ref u8[] raw,ref u8[] name){
    let resolved=names::resolve(parsed(raw,name));let n=names::problem(ref resolved);if(n.code!=0){phase(name,raw,b"names",n.code,n.start,n.end);}
    let checked=typing::check(resolved);let t=typing::problem(ref checked);if(t.code!=0){phase(name,raw,b"typing",t.code,t.start,t.end);}
    let h=switch(hir::lower(checked,0)){case .ok(p):{p}case .err(e):{phase(name,raw,b"HIR",e.code,e.start,e.end);return program(raw,name);}};
    return switch(mir::lower(h)){case .ok(p):{p}case .err(e):{phase(name,raw,b"MIR",e.code,e.start,e.end);program(raw,name)}};
}
fn usize position(ref u8[] raw,ref u8[] part){var usize i=0;while(i<=raw.len){if(part.len<=raw.len-i && bytes::equal(raw[i..i+part.len],part)){return i;}i+=1;}return raw.len;}
@effects(console) pub fn void member(ref u8[] raw,ref u8[] name){
    let p=program(raw,name);let checked=proof::check(ref p);if(checked.code!=0){phase(name,raw,b"MIR-verify",checked.code,checked.start,checked.end);}
    if(!closures::admitted(ref p)){phase(name,raw,b"C-admission",4,0,raw.len);}
    var bytes::Buffer out=bytes::empty();let code=emit::render(ref p,ref mut out);if(code!=0){phase(name,raw,b"C-render",code,0,raw.len);}
    var bytes::Buffer again=bytes::empty();assert(emit::render(ref p,ref mut again)==0 && bytes::equal(bytes::view(ref out),bytes::view(ref again)));
    let text=bytes::view(ref out);
    if(closures::present(ref p)){
        assert(position(text,b"struct fy_closure_i32{")<text.len);
        assert(position(text,b"struct fy_closure_i32{")<position(text,b"fy_fn_"));
        assert(position(text,b"sizeof(struct fy_closure_i32)")==text.len);
    }else{
        assert(position(text,b"fy_closure_")==text.len && position(text,b"fy_callable")==text.len);
    }
    assert(io::write_usize(text.len).is_ok && io::write("\n").is_ok && io::write_bytes(text).is_ok);
}

// Copy owned, already-verified MIR through its explicit assembler. Mutations
// must fail there or before render writes a single byte to the output buffer.
fn bool refused(ref m::Program p,u32 mutation){
    let source=switch(bytes::copy(m::source_view(p))){case .ok(b):{b}case .err(_):{assert(false);return false;}};
    var bytes::Buffer payload=bytes::empty();var v::Vec<m::Target> targets=v::empty<m::Target>();var v::Vec<m::Signature> signatures=v::empty<m::Signature>();
    var v::Vec<m::Function> functions=v::empty<m::Function>();var v::Vec<m::Local> locals=v::empty<m::Local>();var v::Vec<m::Block> blocks=v::empty<m::Block>();
    var v::Vec<m::Operation> operations=v::empty<m::Operation>();var v::Vec<usize> edges=v::empty<usize>();var v::Vec<m::ClosureTarget> metadata=v::empty<m::ClosureTarget>();
    assert(m::closure_count(p)==1 && m::drop_count(p)==0);let c=m::closure_at(p,0);let helper=m::target_at(p,c.target);
    var usize creation=0;var usize invocation=0;var usize i=1;
    while(i<=m::operation_count(p)){if(m::operation_at(p,i).kind==40){creation=i;}i+=1;}
    i=1;while(i<=m::block_count(p)){if(m::block_at(p,i).kind==8){invocation=i;}i+=1;}assert(creation!=0 && invocation!=0);
    let made=m::operation_at(p,creation);let called=m::block_at(p,invocation);
    i=1;while(i<=m::target_count(p)){var t=m::target_at(p,i);if(i==c.target){if(mutation==12){t.effects=1;}if(mutation==13){t.unsafe_call=true;}if(mutation==14){t.result=13;}}assert(v::push<m::Target>(ref mut targets,t).is_ok);i+=1;}
    i=0;while(i<m::signature_count(p)){var u32 type=m::signature_at(p,i);if(mutation==15 && i==helper.signature_start){type=13;}assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type}).is_ok);i+=1;}
    i=1;while(i<=m::function_count(p)){var f=m::function_at(p,i);if(mutation==25 && f.target==c.target){f.public=true;}assert(v::push<m::Function>(ref mut functions,f).is_ok);i+=1;}
    i=1;while(i<=m::local_count(p)){assert(v::push<m::Local>(ref mut locals,m::local_at(p,i)).is_ok);i+=1;}
    i=1;while(i<=m::block_count(p)){var b=m::block_at(p,i);
        if(i==invocation){
            if(mutation==5){b.value=b.left;}if(mutation==6){b.left=b.value;}if(mutation==7){b.destination=b.value;}
            if(mutation==8){b.alternative=b.target;}if(mutation==9){b.callee=c.target;}if(mutation==10){b.type=6;}if(mutation==11){b.target=0;}
            if(mutation==21){b.unsafe_call=true;}if(mutation==22){b.argument_count=1;}
        }assert(v::push<m::Block>(ref mut blocks,b).is_ok);i+=1;
    }
    i=1;while(i<=m::operation_count(p)){var o=m::operation_at(p,i);
        if(o.payload_end>o.payload_start){let raw=m::literal_text(p,i);o.payload_start=bytes::len(ref payload);assert(bytes::append(ref mut payload,raw).is_ok);o.payload_end=bytes::len(ref payload);}
        if(i==creation){
            if(mutation==1){o.operator=0;}if(mutation==4){o.left=o.result;}if(mutation==16){o.owner_type=1;}if(mutation==17){o.boolean=true;}
            if(mutation==19){o.kind=2;o.operator=0;o.left=o.result;o.argument_start=0;o.argument_count=0;}
            if(mutation==20){o.argument_count=0;}
        }
        if(mutation==18 && m::function_at(p,o.function).target==c.target && o.kind==4){o.operator=8;}
        assert(v::push<m::Operation>(ref mut operations,o).is_ok);i+=1;
    }
    i=0;while(i<m::edge_count(p)){var usize value=m::edge_at(p,i);if(i==made.argument_start){if(mutation==27){value=made.result;}if(mutation==28){value=called.destination;}}assert(v::push<usize>(ref mut edges,value).is_ok);i+=1;}
    if(mutation!=2){var item=c;if(mutation==3){item.captures=5;}if(mutation==26){item.target=0;}assert(v::push<m::ClosureTarget>(ref mut metadata,item).is_ok);if(mutation==24){assert(v::push<m::ClosureTarget>(ref mut metadata,item).is_ok);}}
    let nominal=switch(shapes::copy(m::nominal(p))){case .ok(s):{s}case .err(_):{assert(false);return false;}};
    if(mutation==23){let old=m::assemble_resources(source,payload,targets,signatures,functions,locals,blocks,operations,edges,m::module_index(p),nominal,v::empty<m::Drop>());return switch(old){case .err(_):{true}case .ok(_):{false}};}
    let rebuilt=m::assemble_closures(source,payload,targets,signatures,functions,locals,blocks,operations,edges,m::module_index(p),nominal,v::empty<m::Drop>(),metadata);
    return switch(rebuilt){case .err(_):{true}case .ok(forged):{var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"prefix").is_ok);let code=emit::render(ref forged,ref mut out);code!=0 && bytes::equal(bytes::view(ref out),b"prefix")}};
}
@test @effects(console) fn void closure_c_refuses_forged_mir_before_output(){
    let p=program(b"module main;fn fn i32(i32) make(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}fn i32 main(){let fn i32(i32) action=make(40);return action(2);}",b"forged-MIR-baseline");
    assert(proof::check(ref p).code==0);var u32 mutation=1;while(mutation<=28){if(!refused(ref p,mutation)){discard io::write_stderr("closure-C accepted forged mutation=");discard io::write_stderr_u32(mutation);discard io::write_stderr("\n");assert(false);}mutation+=1;}
}
@effects(console) pub fn void reject_source(ref u8[] raw){
    let input=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return;}};
    let parsed_input=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{return;}};
    let resolved=names::resolve(parsed_input);if(names::problem(ref resolved).code!=0){return;}
    let typed=typing::check(resolved);if(typing::problem(ref typed).code!=0){return;}
    let h=switch(hir::lower(typed,0)){case .ok(p):{p}case .err(_):{return;}};
    let p=switch(mir::lower(h)){case .ok(p):{p}case .err(_):{return;}};
    var bytes::Buffer out=bytes::empty();if(emit::render(ref p,ref mut out)==0 || bytes::len(ref out)!=0){phase(b"source-refusal",raw,b"unexpected-C-admission",0,0,raw.len);}
}
