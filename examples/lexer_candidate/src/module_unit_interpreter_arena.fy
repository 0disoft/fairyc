module module_unit_interpreter_arena;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_sdk_sources as sources;
import module_unit_sdk_authority as authority;
import module_unit_value_model as budget;
import syntax as syntax;
import std::bytes as bytes;

// Semantic execution authenticates the retained SDK source independently of
// C generation. Names in user modules never grant host/intrinsic authority.
pub struct Plan {usize type;usize create;}
pub fn Result<Plan,u32> prepare(ref ir::Program p){
    var Plan plan=Plan{.type=0,.create=0};var usize work=0;
    let checked=sources::check(ir::sdk_units(p),ref mut work);if(checked.code!=0){return .err(if(checked.code==1002){2}else{4});}
    var usize u=0;while(u<sources::count(ir::sdk_units(p))){let unit=sources::at(ir::sdk_units(p),u);
        if(unit.sdk_id==1){
            let original=switch(authority::parsed(ir::sdk_units(p),u,ref mut work)){case .ok(p):{p}case .err(e):{return .err(if(e.code==1002){2}else{4});}};
            var usize i=1;while(i<=clean::type_count(ir::cleanup_ir(p))){if(!budget::charge(ref mut work,1)){return .err(2);}let t=clean::type_at(ir::cleanup_ir(p),i);
                if(t.kind==3 && t.module_index==unit.module_index){if(t.declaration==0 || t.declaration>syntax::declaration_count(ref original)){return .err(4);}let d=syntax::declaration_at(ref original,t.declaration-1);
                    if(bytes::equal(syntax::source_view(ref original)[d.name_start..d.name_end],b"Arena")){
                        if(plan.type!=0 || d.kind!=6 || d.field_count!=3){return .err(4);}let glue=clean::glue_at(ir::cleanup_ir(p),i);if(glue.kind!=1 || !glue.nonempty || glue.field_count!=3){return .err(4);}
                        var usize j=0;while(j<3){let field=syntax::field_at(ref original,d.field_start+j);let actual=clean::field_at(ir::cleanup_ir(p),glue.field_start+j);if(!authority::type_matches(p,ref original,unit.module_index,field.type_root,actual.type,0,ref mut work)){return .err(4);}j+=1;}plan.type=i;
                    }
                }i+=1;
            }
            i=1;while(i<=ir::count(p)){if(!budget::charge(ref mut work,1)){return .err(2);}let f=clean::frame_at(ir::cleanup_ir(p),i);
                if(f.module_index==unit.module_index && f.kind==3){if(f.declaration==0 || f.declaration>syntax::declaration_count(ref original)){return .err(4);}let d=syntax::declaration_at(ref original,f.declaration-1);
                    if(bytes::equal(syntax::source_view(ref original)[d.name_start..d.name_end],b"create")){
                        let result=clean::type_at(ir::cleanup_ir(p),f.result);if(plan.create!=0 || plan.type==0 || d.kind!=3 || result.kind!=6 || result.first!=plan.type){return .err(4);}
                        let error=clean::type_at(ir::cleanup_ir(p),result.second);if(error.kind!=2 || error.module_index!=unit.module_index || error.declaration==0 || error.declaration>syntax::declaration_count(ref original)){return .err(4);}let e=syntax::declaration_at(ref original,error.declaration-1);if(!bytes::equal(syntax::source_view(ref original)[e.name_start..e.name_end],b"AllocError")){return .err(4);}
                        let body=ir::at(p,i);var usize j=1;while(j<=ir::local_count(body)){if(ir::local_at(body,j).kind==1 || ir::local_at(body,j).kind==10){return .err(4);}j+=1;}plan.create=i;
                    }
                }i+=1;
            }
        }u+=1;
    }return .ok(plan);
}
