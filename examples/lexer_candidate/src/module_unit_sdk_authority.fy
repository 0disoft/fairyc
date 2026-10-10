module module_unit_sdk_authority;
import module_unit_sdk_sources as sources;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_interfaces as interfaces;
import module_unit_value_model as budget;
import sdk_scopes as sdk;
import foreign_declarations as foreign;
import effect_declarations as effects;
import pointer_types as pointers;
import type_state as scalar;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
// Reconstruct only from retained authenticated bytes. Declaration IDs remain
// local to the original SDK file; no joined source or declaration remapping.
pub fn Result<syntax::Parsed,clean::Problem> parsed(ref sources::Sources owned,usize index,ref mut usize work){
    if(index>=sources::count(owned)){return .err(fail(1001,0,0,0));}let unit=sources::at(owned,index);let raw=sources::source_at(owned,index);
    if(!budget::charge(work,raw.len+1)){return .err(fail(1002,unit.module_index,0,raw.len));}
    let input=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{return .err(fail(1003,unit.module_index,0,raw.len));}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(e):{return .err(fail(if(e==3){1003}else{1001},unit.module_index,0,raw.len));}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(e):{return .err(fail(if(e==3){1003}else{1001},unit.module_index,0,raw.len));}};
    if(syntax::problem_count(ref p)!=0 || syntax::module_count(ref p)!=1){return .err(fail(1001,unit.module_index,0,raw.len));}return .ok(p);
}
pub fn bool type_matches(ref ir::Program p,ref syntax::Parsed original,usize module_index,usize root,usize id,usize depth,ref mut usize work){
    if(depth>=64 || !budget::charge(work,1)){return false;}if(root==0 || root>syntax::type_count(original) || id==0 || id>clean::type_count(ir::cleanup_ir(p))){return false;}
    let n=syntax::type_at(original,root);let t=clean::type_at(ir::cleanup_ir(p),id);
    if(n.kind==5){return type_matches(p,original,module_index,n.left,id,depth+1,work);}
    if(n.kind==1){let name=syntax::type_text(original,root);let primitive=scalar::primitive(original,root);
        if(primitive!=7 || bytes::equal(name,b"void")){return t.kind==1 && t.scalar==primitive;}
        if(t.kind!=2 || t.module_index!=module_index || t.declaration==0 || t.declaration>syntax::declaration_count(original)){return false;}
        let d=syntax::declaration_at(original,t.declaration-1);return d.kind==4 && bytes::equal(name,syntax::source_view(original)[d.name_start..d.name_end]);
    }
    if(n.kind==11 || n.kind==12){if(t.kind!=12){return false;}if(t.first==0){let code=pointers::resolve(original,root);return code!=0 && t.scalar==code;}return t.scalar==if(n.kind==12){1}else{0} && type_matches(p,original,module_index,n.left,t.first,depth+1,work);}
    if(n.kind==2 || n.kind==3){return t.kind==if(n.kind==2){7}else{8} && type_matches(p,original,module_index,n.left,t.first,depth+1,work);}
    return false;
}
// Validate the supplied callable, not an asserted linker-name/authority flag.
// The authentic source declaration fixes its symbol, effects and full signature.
pub fn Result<usize,clean::Problem> check_foreign(ref ir::Program p,ref syntax::Parsed original,usize module_index,usize sdk_id,interfaces::Callable c,ref u8[] link,ref mut usize work){
    if(!budget::charge(work,1)){return .err(fail(1002,module_index,0,0));}
    if(c.module_index!=module_index || c.declaration==0 || c.declaration>syntax::declaration_count(original) || c.parameter_start>ir::parameter_count(p) || c.parameter_count>ir::parameter_count(p)-c.parameter_start){return .err(fail(1001,module_index,0,0));}
    let d=syntax::declaration_at(original,c.declaration-1);let authority=effects::read(original,c.declaration-1);
    if(d.kind!=9 || !c.foreign || c.template || !c.unsafe_call || d.unsafe_end==0 || d.body!=0 || d.generic_parameters!=0 || d.owner_end!=0 || authority.code!=0 || c.effects!=authority.bits || c.must_use!=authority.must_use){return .err(fail(1001,module_index,d.start,d.end));}
    let declared=foreign::read_signature(original,c.declaration-1,true,true);
    if(declared.code!=0 || !bytes::equal(link,syntax::source_view(original)[declared.link_start..declared.link_end])){return .err(fail(1001,module_index,d.start,d.end));}
    let scope=sdk::Scope{.sdk_id,.start=0,.end=syntax::source_view(original).len};let rule=sdk::foreign_rule(syntax::source_view(original),scope,d.name_start,d.name_end,declared.link_start,declared.link_end);
    if(rule==0 || !type_matches(p,original,module_index,d.result_type,c.result,0,work)){return .err(fail(if(*work>=1048576){1002}else{1001},module_index,d.start,d.end));}
    var usize count=0;var usize i=d.entry_start;
    while(i<d.entry_start+d.entry_count){if(!budget::charge(work,1)){return .err(fail(1002,module_index,d.start,d.end));}let e=syntax::entry_at(original,i);
        if(e.kind==1){if(count>=c.parameter_count || !type_matches(p,original,module_index,e.type_root,ir::parameter_at(p,c.parameter_start+count),0,work)){return .err(fail(if(*work>=1048576){1002}else{1001},module_index,e.start,e.end));}count+=1;}i+=1;
    }
    if(count!=c.parameter_count || count!=sdk::parameter_count(rule)){return .err(fail(1001,module_index,d.start,d.end));}return .ok(rule);
}
pub fn clean::Problem check(ref ir::Program p,ref mut usize work){
    var usize unit_index=0;while(unit_index<sources::count(ir::sdk_units(p))){let unit=sources::at(ir::sdk_units(p),unit_index);var bool needed=false;var usize i=0;
        while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return fail(1002,unit.module_index,0,0);}if(ir::callable_at(p,i).module_index==unit.module_index){needed=true;}i+=1;}
        if(needed){let original=switch(parsed(ir::sdk_units(p),unit_index,work)){case .ok(p):{p}case .err(e):{return e;}};i=0;
            while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return fail(1002,unit.module_index,0,0);}let c=ir::callable_at(p,i);
                if(c.module_index==unit.module_index){if(c.declaration==0 || c.declaration>syntax::declaration_count(ref original)){return fail(1001,unit.module_index,0,0);}let d=syntax::declaration_at(ref original,c.declaration-1);
                    if(c.foreign || d.kind==9){switch(check_foreign(p,ref original,unit.module_index,unit.sdk_id,c,ir::linker_name(p,i),work)){case .ok(_):{}case .err(e):{return e;}}}
                }i+=1;
            }
        }unit_index+=1;
    }return fail(0,0,0,0);
}
pub fn Result<usize,clean::Problem> rule(ref ir::Program p,usize index,ref mut usize work){
    if(index>=ir::callable_count(p)){return .err(fail(1001,0,0,0));}let c=ir::callable_at(p,index);let found=sources::find(ir::sdk_units(p),c.module_index);if(found==0){return .ok(0);}
    let unit=sources::at(ir::sdk_units(p),found-1);let original=switch(parsed(ir::sdk_units(p),found-1,work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    return check_foreign(p,ref original,unit.module_index,unit.sdk_id,c,ir::linker_name(p,index),work);
}
