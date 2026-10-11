module module_unit_c_bytes;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import syntax as syntax;
import c_text as w;
import std::bytes as bytes;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
// Called only for the retained, authenticated std::bytes source. Keep the
// ordinary frame and ABI; replace only this exact SDK function's implementation.
pub fn Result<usize,clean::Problem> find(ref ir::Program p,ref syntax::Parsed original,usize module_index,ref mut usize work){
    var usize declaration=0;var usize i=0;while(i<syntax::declaration_count(original)){if(!budget::charge(work,1)){return .err(fail(1002,module_index,0,0));}let d=syntax::declaration_at(original,i);if(d.kind==3 && bytes::equal(syntax::source_view(original)[d.name_start..d.name_end],b"equal")){declaration=i+1;break;}i+=1;}
    if(declaration==0){return .err(fail(1001,module_index,0,0));}let d=syntax::declaration_at(original,declaration-1);var usize found=0;i=1;
    while(i<=ir::count(p)){if(!budget::charge(work,1)){return .err(fail(1002,module_index,d.start,d.end));}let f=clean::frame_at(ir::cleanup_ir(p),i);if((f.kind==1 || f.kind==3) && f.module_index==module_index && f.declaration==declaration){let b=ir::at(p,i);let result=clean::type_at(ir::cleanup_ir(p),f.result);
        if(found!=0 || result.kind!=1 || result.scalar!=6 || f.effects!=0 || ir::start(b)!=d.start || ir::end(b)!=d.end || ir::source_bytes(b)!=syntax::source_view(original).len){return .err(fail(1001,module_index,d.start,d.end));}
        var usize parameters=0;var usize local=1;while(local<=ir::local_count(b)){if(!budget::charge(work,1)){return .err(fail(1002,module_index,d.start,d.end));}let l=ir::local_at(b,local);if(l.kind==1){if(clean::type_at(ir::cleanup_ir(p),l.type).kind!=10){return .err(fail(1001,module_index,d.start,d.end));}parameters+=1;}local+=1;}if(parameters!=2){return .err(fail(1001,module_index,d.start,d.end));}found=i;
    }i+=1;}return .ok(found);
}
pub fn bool body(ref ir::Body b,ref mut bytes::Buffer out,ref mut w::State s){
    var usize left=0;var usize right=0;var usize i=1;while(i<=ir::local_count(b)){if(ir::local_at(b,i).kind==1){if(left==0){left=i;}else{right=i;}}i+=1;}
    if(left==0 || right==0){s.code=1001;return false;}
    // Never call memcmp with empty, possibly null views. Both logical extents
    // must match before any read, including embedded NUL and non-UTF8 bytes.
    return w::text(out,s,b"return ") && w::local(out,s,left) && w::text(out,s,b".fy_length==") && w::local(out,s,right) && w::text(out,s,b".fy_length && (") && w::local(out,s,left) && w::text(out,s,b".fy_length==0 || memcmp(") && w::local(out,s,left) && w::text(out,s,b".fy_data,") && w::local(out,s,right) && w::text(out,s,b".fy_data,") && w::local(out,s,left) && w::text(out,s,b".fy_length)==0);\n}\n");
}
