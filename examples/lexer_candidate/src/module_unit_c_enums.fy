module module_unit_c_enums;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_enums as proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_aggregate as aggregate;
import module_unit_c_scalar as scalar;
import c_text as w;
import std::bytes as bytes;

pub fn bool variant(ref mut bytes::Buffer out,ref mut w::State s,usize id){return w::text(out,s,b"fy_variant_") && w::number(out,s,id);}
pub fn bool definition(ref ir::Program p,usize type,usize index,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let shape=ir::layout_at(p,index-1).value;if(shape.type!=type || clean::type_at(ir::cleanup_ir(p),type).kind!=4){s.code=1001;return false;}
    if(!aggregate::type(p,type,out,s) || !w::text(out,s,b"{int32_t fy_tag;union{uint8_t fy_empty;")){return false;}
    var usize i=0;while(i<shape.field_count){if(!budget::charge(work,1)){s.code=1002;return false;}let id=shape.field_start+i+1;let f=ir::field_at(p,id-1);
        if(f.kind==3){if(!w::text(out,s,b"struct{")){return false;}var usize j=0;while(j<f.payload_count){if(!budget::charge(work,1)){s.code=1002;return false;}let child=f.payload_start+j+1;let field=ir::field_at(p,child-1);if(field.kind!=4 || !aggregate::type(p,field.type,out,s) || !w::text(out,s,b" ") || !aggregate::field(out,s,child) || !w::text(out,s,b";")){return false;}j+=1;}
            if(f.payload_count==0 && !w::text(out,s,b"uint8_t fy_empty;")){return false;}if(!w::text(out,s,b"} ") || !variant(out,s,id) || !w::text(out,s,b";")){return false;}
        }i+=1;
    }return w::text(out,s,b"}fy_payload;};\n");
}
pub fn bool expression(ref ir::Program p,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let e=ir::node_at(b,id-1).value;let index=proof::layout(p,e.type,work,ref mut s.code);if(index==0 || (e.kind!=28 && e.kind!=10) || !proof::member(p,e.field,e.type,3)){if(s.code==0){s.code=1001;}return false;}
    let shape=ir::layout_at(p,index-1).value;var usize tag=0;var usize i=shape.field_start;while(i+1<e.field.id){if(!budget::charge(work,1)){s.code=1002;return false;}if(ir::field_at(p,i).kind==3){tag+=1;}i+=1;}
    if(!scalar::value(out,s,id) || !w::text(out,s,b"=(") || !aggregate::type(p,e.type,out,s) || !w::text(out,s,b"){.fy_tag=") || !w::number(out,s,tag) || !w::text(out,s,b"};\n")){return false;}
    var usize link=e.right;while(link!=0){if(!budget::charge(work,1)){s.code=1002;return false;}let pair=lookup::node(b,link);if(pair==0){s.code=1001;return false;}let row=ir::node_at(b,pair-1).value;let child=lookup::node(b,row.left);if(child==0 || !proof::member(p,row.field,e.type,4)){s.code=1001;return false;}
        if(!scalar::value(out,s,id) || !w::text(out,s,b".fy_payload.") || !variant(out,s,e.field.id) || !w::text(out,s,b".") || !aggregate::field(out,s,row.field.id) || !w::text(out,s,b"=") || !scalar::value(out,s,child) || !w::text(out,s,b";\n")){return false;}link=row.right;
    }return true;
}
