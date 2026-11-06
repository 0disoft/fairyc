module module_state_graph_native_operations;
import module_state_graph_program as program;
import module_state_graph_model as m;
import module_state_graph_native_plan as plan;
import module_state_graph_native_text as w;
import module_state_layout as state;
import project_cache_encoding as budget;
import std::bytes as bytes;

// Layout/graph codecs preserve declaration order. The C runtime uses its own
// bytes,text,bytes20,bytes32 order; this is the Go overlayCodec mapping.
fn usize codec(u32 id){return if(id<=3){id as usize}else if(id==4){6}else if(id==5){7}else if(id==6){4}else{5};}
fn bool literal(ref m::Graph graph,m::Operand operand,ref u8[] name,ref mut bytes::Buffer out,ref mut w::State s){
    if(operand.start>operand.end || operand.end>m::data(graph).len || operand.codec<1 || operand.codec>7){s.code=1001;return false;}
    let raw=m::data(graph)[operand.start..operand.end];
    return w::text(out,s,b"    static const uint8_t ") && w::text(out,s,name) && w::text(out,s,b"_bytes[") &&
        w::number(out,s,if(raw.len==0){1}else{raw.len}) && w::text(out,s,b"] = {") &&
        w::octets(out,s,if(raw.len==0){b"\0"}else{raw}) && w::text(out,s,b"};\n    const fycs_graph_value_v1 ") &&
        w::text(out,s,name) && w::text(out,s,b" = {") && w::text(out,s,name) && w::text(out,s,b"_bytes,") &&
        w::number(out,s,raw.len) && w::text(out,s,b"u,") && w::number(out,s,codec(operand.codec)) && w::text(out,s,b"u};\n");
}
fn bool operand(ref plan::Plan p,usize body,m::Operand value,ref u8[] name,ref mut bytes::Buffer out,ref mut w::State s){
    if(value.input!=0){return w::value(out,s,p,body,value.input);}return w::text(out,s,name);
}
fn bool read_value(ref plan::Plan p,usize body,m::Operation op,ref mut bytes::Buffer out,ref mut w::State s){
    if(op.kind==7 || op.kind==8){return w::borrow(out,s,p,body,op.borrow_result) && w::text(out,s,b".value");}
    return w::value(out,s,p,body,op.result);
}
fn bool read_length(ref plan::Plan p,usize body,m::Operation op,ref mut bytes::Buffer out,ref mut w::State s){
    if(op.kind!=7 && op.kind!=8 && !w::text(out,s,b"found != 0u ? ")){return false;}
    if(!read_value(p,body,op,out,s) || !w::text(out,s,b".length")){return false;}
    return op.kind==7 || op.kind==8 || w::text(out,s,b" : 0u");
}
fn bool read(ref plan::Plan p,usize body,m::Operation op,usize value_codec,ref mut bytes::Buffer out,ref mut w::State s){
    let site=s.site;
    if(!w::text(out,s,b"    uint8_t found = 0u;\n") || !w::checked(out,s)){return false;}
    if(op.kind==7 || op.kind==8){
        if(!w::text(out,s,b"fycs_graph_borrow_v1_begin(context,key,") || !w::number(out,s,value_codec) || !w::text(out,s,b"u,&") ||
            !w::borrow(out,s,p,body,op.borrow_result) || !w::text(out,s,b",&found)") || !w::end(out,s)){return false;}
    }else{
        if(!w::text(out,s,b"fycs_graph_read_v1(context,key,") || !w::number(out,s,value_codec) || !w::text(out,s,b"u,&") ||
            !read_value(p,body,op,out,s) || !w::text(out,s,b",&found)") || !w::end(out,s)){return false;}
    }
    if(s.audit){
        if(!w::text(out,s,b"    audit_value = ") || !read_value(p,body,op,out,s) || !w::text(out,s,b"; audit_value.length = ") ||
            !read_length(p,body,op,out,s) || !w::text(out,s,b"; audit_found = found;\n")){return false;}
    }else{
        if(!w::checked(out,s) || !w::text(out,s,b"observe(user,context->operation,") || !w::number(out,s,site) || !w::text(out,s,b"u,key,") ||
            !w::number(out,s,value_codec) || !w::text(out,s,b"u,") || !read_value(p,body,op,out,s) || !w::text(out,s,b".bytes,") ||
            !read_length(p,body,op,out,s) || !w::text(out,s,b",found)") || !w::end(out,s)){return false;}
    }
    return w::text(out,s,b"    ") && w::value(out,s,p,body,op.found) && w::text(out,s,b".bytes = fygr_bool_v1+found; ") &&
        w::value(out,s,p,body,op.found) && w::text(out,s,b".length = 1u; ") && w::value(out,s,p,body,op.found) &&
        w::text(out,s,b".codec = FYCS_CODEC_BOOL;\n");
}
fn bool storage(ref program::Verified proof,ref plan::Plan p,usize body,ref m::Graph graph,m::Operation op,ref mut bytes::Buffer out,ref mut w::State s){
    let layouts=program::layouts(proof);let header=m::header(graph);
    if(header.layout>=state::count(layouts)){s.code=1001;return false;}let layout=state::at(layouts,header.layout);
    if(op.field>=layout.field_count){s.code=1001;return false;}let field=state::field_at(layouts,layout.field_start+op.field);
    if(field.kind==1){
        if(!w::text(out,s,b"    static const uint8_t key[32] = {") || !w::digest(out,s,field.prefix) || !w::text(out,s,b"};\n")){return false;}
    }else{
        if(!w::text(out,s,b"    static const uint8_t prefix[32] = {") || !w::digest(out,s,field.prefix) || !w::text(out,s,b"};\n    uint8_t key[32];\n")){return false;}
        if(op.key.input==0 && !literal(graph,op.key,b"literal_key",out,s)){return false;}
        if(!w::checked(out,s) || !w::text(out,s,b"fycs_graph_map_key_v1(prefix,&") || !operand(p,body,op.key,b"literal_key",out,s) ||
            !w::text(out,s,b",max_key,key)") || !w::end(out,s)){return false;}
        if(s.audit && (!w::text(out,s,b"    audit_key = ") || !operand(p,body,op.key,b"literal_key",out,s) || !w::text(out,s,b";\n"))){return false;}
    }
    if(op.kind==1 || op.kind==4 || op.kind==7 || op.kind==8){return read(p,body,op,codec(field.value),out,s);}
    if(op.kind==2 || op.kind==5){
        if(op.value.input==0 && !literal(graph,op.value,b"literal_value",out,s)){return false;}
        if(!w::checked(out,s) || !w::text(out,s,b"fycs_graph_write_v1(context,key,&") || !operand(p,body,op.value,b"literal_value",out,s) ||
            !w::text(out,s,b",0u)") || !w::end(out,s)){return false;}
        return !s.audit || (w::text(out,s,b"    audit_value = ") && operand(p,body,op.value,b"literal_value",out,s) && w::text(out,s,b"; audit_found = 1u;\n"));
    }
    return w::checked(out,s) && w::text(out,s,b"fycs_graph_write_v1(context,key,NULL,1u)") && w::end(out,s);
}

// Emit graph=true operations only. The parent owns the complete artifact;
// sticky writer failures never authorize publishing this partial buffer.
pub fn bool operation(ref program::Verified proof,ref plan::Plan p,usize body,ref m::Graph graph,m::Operation op,ref mut bytes::Buffer out,ref mut w::State s){
    if(s.code!=0){return false;}if(!budget::charge(ref mut s.work,1)){s.code=1002;return false;}
    if(op.kind<1 || op.kind>11 || body>=plan::body_count(p)){s.code=1001;return false;}s.site+=1;let site=s.site;
    if(op.kind==11){
        if(!w::checked(out,s) || !w::text(out,s,b"fycs_graph_step_v1(context,1u,1u)") || !w::end(out,s)){return false;}
        if(s.audit && (!w::checked(out,s) || !w::text(out,s,b"observe(user,context->operation,") || !w::number(out,s,site) ||
            !w::text(out,s,b"u,NULL,NULL,0u)") || !w::end(out,s))){return false;}
        return w::checked(out,s) && w::text(out,s,b"fygr_body_") && w::number(out,s,op.callee) &&
            w::text(out,s,b"(context,max_key,observe,user,exit_kind,trap_kind)") && w::end(out,s) &&
            w::text(out,s,b"    if (*exit_kind != 3u) { goto finished; }\n");
    }
    if(!w::text(out,s,b"    {\n")){return false;}
    if(s.audit && !w::text(out,s,b"    fycs_graph_value_v1 audit_key = {0}, audit_value = {0}; uint8_t audit_found = 0u;\n")){return false;}
    if(op.kind==9){
        if(!w::checked(out,s) || !w::text(out,s,b"fycs_graph_borrow_view_v1(context,&") || !w::borrow(out,s,p,body,op.borrow_input) ||
            !w::text(out,s,b",") || !w::number(out,s,op.view_offset) || !w::text(out,s,b"u,") || !w::number(out,s,op.view_length) ||
            !w::text(out,s,b"u,&") || !w::borrow(out,s,p,body,op.borrow_result) || !w::text(out,s,b")") || !w::end(out,s)){return false;}
    }else if(op.kind==10){
        if(!w::checked(out,s) || !w::text(out,s,b"fycs_graph_borrow_copy_v1(context,&") || !w::borrow(out,s,p,body,op.borrow_input) ||
            !w::text(out,s,b",&") || !w::value(out,s,p,body,op.result) || !w::text(out,s,b")") || !w::end(out,s)){return false;}
    }else if(!storage(proof,p,body,graph,op,out,s)){return false;}
    if(!w::checked(out,s) || !w::text(out,s,b"fycs_graph_step_v1(context,1u,1u)") || !w::end(out,s)){return false;}
    if(s.audit && (!w::checked(out,s) || !w::text(out,s,b"observe(user,context->operation,") || !w::number(out,s,site) ||
        !w::text(out,s,b"u,&audit_key,&audit_value,audit_found)") || !w::end(out,s))){return false;}
    return w::text(out,s,b"    }\n");
}
