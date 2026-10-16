module mir_verify_shape;
import pointer_types as pointers;
import mir_reference as refs;
import c_abi_types as casts;
import mir_model as m;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;
import raw_view_intrinsics as raw;
import storage_intrinsics as storage;
import mir_verify_nominal as nominal;
import mir_verify_arrays as arrays;
import mir_verify_literals as literals;
import mir_verify_views as views;
import mir_verify_enums as enums;
import ffi_symbols as ffi;
import sdk_scopes as sdk;

pub fn m::Problem problem(u32 code, usize start, usize end) { return m::Problem { .code, .start, .end }; }
pub fn bool local(ref m::Program p, usize id, usize function, u32 type) {
    if (id == 0 || id > m::local_count(p)) { return false; }
    let m::Local value = m::local_at(p,id);
    return value.function == function && (type == 0 || value.type == type);
}
pub fn u32 mask(u32 operator, u32 type, bool unary) {
    if (operator == 0 && type == 6 && unary) { return 16; }
    if (unary) { return if (operator == 2) { 1 } else { 0 }; }
    if (operator == 1 || operator == 2 || operator == 5) { return 1; }
    if (operator == 6 || operator == 7) { return if (type == 1 || type == 2) { 6 } else { 2 }; }
    return if (operator == 11) { 9 } else if (operator == 12) { 8 } else { 0 };
}
fn bool unary(u32 operator, u32 type) {
    return if (operator == 3) { type == 6 } else if (operator == 2) { type == 1 || type == 2 }
        else { (operator == 1 || operator == 4) && type >= 1 && type <= 5 };
}
fn bool binary(u32 operator, u32 type) {
    return operator >= 1 && operator <= 18 && (type >= 1 && type <= 5 || (type == 6 || pointers::valid(type)) && (operator == 13 || operator == 14));
}
fn bool integer(ref u8[] text, u32 type) {
    if (text.len == 0) { return false; }
    let bool negative = text[0] == 45;
    let usize first = if (negative) { 1 } else { 0 };
    if (first == text.len || negative && type != 1 && type != 2) { return false; }
    let ref u8[] digits = text[first..];
    if (digits.len > 1 && digits[0] == 48 || negative && digits.len == 1 && digits[0] == 48) { return false; }
    let ref u8[] maximum = if (type == 1) { if (negative) { b"2147483648" } else { b"2147483647" } }
        else if (type == 2) { if (negative) { b"9223372036854775808" } else { b"9223372036854775807" } }
        else if (type == 3) { b"255" } else if (type == 4) { b"4294967295" } else { b"18446744073709551615" };
    if (digits.len > maximum.len) { return false; }
    var bool smaller = digits.len < maximum.len; var usize i = 0;
    while (i < digits.len) {
        if (digits[i] < 48 || digits[i] > 57) { return false; }
        if (!smaller) { if (digits[i] > maximum[i]) { return false; } smaller = digits[i] < maximum[i]; }
        i += 1;
    }
    return true;
}
fn bool operation(ref m::Program p, usize id, m::Operation o) {
    if(o.kind==39){let usize args=storage::arguments(o.operator);
        return shapes::valid_type(m::nominal(p),o.owner_type,false) && storage::result(o.operator,o.owner_type)==o.type && o.boolean && (if(args==3){o.argument_count==1 && local(p,m::edge_at(p,o.argument_start),o.function,5) && o.result!=m::edge_at(p,o.argument_start)}else{o.argument_start==0 && o.argument_count==0}) && o.payload_start==0 && o.payload_end==0
            && shapes::storage_rule(m::nominal(p),m::source_view(p),o.start,o.end,o.operator)
            && (if(o.type==7){o.result==0}else{local(p,o.result,o.function,o.type) && o.result!=o.left && o.result!=o.right})
            && (if(args==0){o.left==0}else if(o.operator>=6){o.left>0 && o.left<=m::local_count(p) && m::local_at(p,o.left).function==o.function && storage::owner(o.operator,m::local_at(p,o.left).type) && shapes::storage_owner(m::nominal(p),m::local_at(p,o.left).type)}else{local(p,o.left,o.function,storage::argument_type(o.operator,o.owner_type,0))}) && (if(args<2){o.right==0}else{local(p,o.right,o.function,storage::argument_type(o.operator,o.owner_type,1))});
    }
    if(o.kind==38){return (o.type==9 || o.type==10 || o.type==11 || o.type==12) && o.boolean && raw::owner(o.owner_type,o.type) && local(p,o.result,o.function,o.type) && local(p,o.left,o.function,o.owner_type)
        && o.right>0 && o.right<=m::local_count(p) && raw::data(m::local_at(p,o.right).type,o.type) && m::local_at(p,o.right).function==o.function && o.operator==0 && o.argument_count==1 && local(p,m::edge_at(p,o.argument_start),o.function,5) && o.payload_start==0 && o.payload_end==0;}
    if(o.kind==37){return pointers::cast_allowed(o.owner_type,o.type) && o.boolean && local(p,o.result,o.function,o.type) && local(p,o.left,o.function,o.owner_type) && o.result!=o.left && o.right==0 && o.operator==0 && o.argument_start==0 && o.argument_count==0 && o.payload_start==0 && o.payload_end==0;}
    if(o.kind==36){return pointers::valid(o.type) && o.owner_type==o.type && local(p,o.result,o.function,o.type) && o.left==0 && o.right==0 && o.operator==0 && o.argument_start==0 && o.argument_count==0 && o.payload_start==0 && o.payload_end==0 && !o.boolean;}
    if(o.kind==35){return casts::allowed(o.owner_type,o.type) && local(p,o.result,o.function,o.type) && local(p,o.left,o.function,o.owner_type) && o.result!=o.left && o.right==0 && o.operator==0 && o.argument_start==0 && o.argument_count==0 && o.payload_start==0 && o.payload_end==0 && !o.boolean;}
    if(o.kind>=28){return refs::operation(p,o);}
    if(o.kind==24 || o.kind==25 || o.kind==26 || o.kind==27){
        if(!shapes::needs_drop(o.type) || o.owner_type!=o.type || o.operator!=0 || o.right!=0 || o.argument_start!=0 || o.argument_count!=0 || o.payload_start!=0 || o.payload_end!=0 || o.boolean || !local(p,o.left,o.function,o.type)){return false;}
        if(o.kind==24){return o.result!=o.left && local(p,o.result,o.function,o.type);}
        if(o.kind==27){return o.result==0 && m::burnable(p,o.type);}
        if(!m::may_drop(p,o.type,0)){return false;}
        if(o.kind==26){return o.result==0 && shapes::valid_type(m::nominal(p),o.type,false);}
        let usize at=m::drop_index(p,o.type);return o.result==0 && at!=0 && !m::drop_at(p,at-1).nonempty;
    }
    if (o.kind >= 21) { return enums::operation(p,o); }
    if (o.kind >= 18) { return views::operation(p,o); }
    if (o.kind >= 14) { return literals::operation(p,id,o); }
    if (o.kind >= 9) { return arrays::operation(p,o); }
    if (o.kind >= 6) { return nominal::operation(p,o); }
    if (o.argument_start != 0 || o.argument_count != 0) { return false; }
    if (o.kind == 2) { return !shapes::needs_drop(o.type) && local(p,o.result,o.function,o.type) && local(p,o.left,o.function,o.type) && o.owner_type == o.type && o.operator == 0 && o.right == 0 && o.payload_start == 0 && o.payload_end == 0 && !o.boolean; }
    if(o.kind==4 && pointers::valid(o.owner_type)){return binary(o.operator,o.owner_type) && o.type==6 && local(p,o.result,o.function,6) && local(p,o.left,o.function,o.owner_type) && local(p,o.right,o.function,o.owner_type) && o.argument_start==0 && o.argument_count==0 && o.payload_start==0 && o.payload_end==0 && !o.boolean;}
    if (o.owner_type < 1 || o.owner_type > 6 || o.type < 1 || o.type > 6) { return false; }
    if (o.kind != 5 && !local(p,o.result,o.function,o.type)) { return false; }
    if (o.kind == 1) {
        if (o.owner_type != o.type || o.left != 0 || o.right != 0 || o.operator != 0) { return false; }
        if (o.type == 6) { return o.payload_start == 0 && o.payload_end == 0; }
        return !o.boolean && integer(m::literal_text(p,id),o.type);
    }
    if (o.payload_start != 0 || o.payload_end != 0 || o.boolean || !local(p,o.left,o.function,o.owner_type)) { return false; }
    if (o.kind == 3) { return o.type == o.owner_type && o.right == 0 && unary(o.operator,o.owner_type); }
    if (o.kind == 4) { return binary(o.operator,o.owner_type) && local(p,o.right,o.function,o.owner_type) && o.type == if (o.operator >= 13) { 6 } else { o.owner_type }; }
    return o.type == o.owner_type && (if (o.right == 0) { o.operator == 0 && o.owner_type == 6 || unary(o.operator,o.owner_type) }
        else { binary(o.operator,o.owner_type) && local(p,o.right,o.function,o.owner_type) }) && mask(o.operator,o.owner_type,o.right == 0) != 0;
}
fn bool empty_fields(m::Block b) { return b.left == 0 && b.right == 0 && b.operator == 0 && b.check == 0 && b.trap_mask == 0; }
fn bool no_call(m::Block b) { return b.callee == 0 && b.destination == 0 && b.argument_start == 0 && b.argument_count == 0; }
fn bool mark(ref mut bytes::Buffer flags, usize id) {
    let ref mut u8[] data = bytes::view_mut(flags);
    if (data[id] != 0) { return false; } data[id] = 1; return true;
}
fn bool filled(ref bytes::Buffer flags, usize first) {
    let ref u8[] data = bytes::view(flags); var usize i = first;
    while (i < data.len) { if (data[i] == 0) { return false; } i += 1; } return true;
}

// Precondition: assemble has checked all arena indexes/ranges. No executable
// record, call signature or checked edge is trusted merely because it links.
pub fn m::Problem check(ref m::Program p) {
    if(!shapes::abi_valid(m::nominal(p),m::source_view(p))){return problem(5102,0,m::source_view(p).len);}
    var bytes::Buffer params = bytes::empty(); var bytes::Buffer ops = bytes::empty();
    var bytes::Buffer edges = bytes::empty(); var bytes::Buffer signatures = bytes::empty();
    var bytes::Buffer false_constants = bytes::empty(); var v::Vec<usize> definitions = v::empty<usize>();
    var v::Vec<usize> bodies = v::empty<usize>(); var v::Vec<usize> guards = v::empty<usize>(); var v::Vec<usize> traps = v::empty<usize>();
    if (!bytes::resize_zeroed(ref mut params,m::local_count(p) + 1).is_ok || !bytes::resize_zeroed(ref mut ops,m::operation_count(p) + 1).is_ok
        || !bytes::resize_zeroed(ref mut edges,m::edge_count(p)).is_ok || !bytes::resize_zeroed(ref mut signatures,m::signature_count(p)).is_ok
        || !bytes::resize_zeroed(ref mut false_constants,m::local_count(p) + 1).is_ok) { return problem(3,0,0); }
    var usize i = 0;
    while (i <= m::local_count(p)) { if (!v::push<usize>(ref mut definitions,0).is_ok) { return problem(3,0,0); } i += 1; }
    i = 0;
    while (i <= m::target_count(p)) { if (!v::push<usize>(ref mut bodies,0).is_ok) { return problem(3,0,0); } i += 1; }
    i = 0; while (i <= m::block_count(p)) {
        if (!v::push<usize>(ref mut guards,0).is_ok || !v::push<usize>(ref mut traps,0).is_ok) { return problem(3,0,0); } i += 1;
    }
    i = 1; while (i <= m::function_count(p)) {
        let m::Function f = m::function_at(p,i); let m::Target t = m::target_at(p,f.target);
        if(m::drop_target(p,f.target) && (f.public || f.parameter_count!=1)){return problem(1001,f.start,f.end);}
        if (*v::at<usize>(ref bodies,f.target) != 0) { return problem(1001,f.start,f.end); }
        { let ref mut usize slot = v::at_mut<usize>(ref mut bodies,f.target); *slot = i; }
        var usize j = 0; while (j < f.parameter_count) {
            let usize at = f.parameter_start + j; let usize id = m::edge_at(p,at);
            if (!mark(ref mut edges,at) || !mark(ref mut params,id)) { return problem(1001,f.start,f.end); } j += 1;
        }
        i += 1;
    }
    i = 1; while (i <= m::target_count(p)) {
        let m::Target t = m::target_at(p,i);
        if (t.effects>4095 || t.external!=t.foreign || t.module_index != m::module_index(p) || (if(t.foreign){*v::at<usize>(ref bodies,i)!=0}else{*v::at<usize>(ref bodies,i)==0})) { return problem(1001,t.start,t.end); }
        if(t.foreign){if(!t.unsafe_call || !shapes::foreign_result(m::nominal(p),t.result)){return problem(5106,t.start,t.end);}
            let usize rule=shapes::sdk_rule(m::nominal(p),m::source_view(p),t.start,t.end,t.link_start,t.link_end);
            if(!ffi::valid(m::foreign_text(p,i)) && rule==0){return problem(5106,t.start,t.end);}
            if(rule!=0 && (!shapes::sdk_result(m::nominal(p),m::source_view(p),rule,t.result) || t.effects!=sdk::effects(rule) || t.signature_count!=sdk::parameter_count(rule))){return problem(5102,t.start,t.end);}
            var usize n=0;while(n<t.signature_count){if(!shapes::foreign_scalar(m::nominal(p),m::signature_at(p,t.signature_start+n),false) || rule!=0 && !shapes::sdk_parameter(m::nominal(p),m::source_view(p),rule,n,m::signature_at(p,t.signature_start+n))){return problem(5102,t.start,t.end);}n+=1;}
        }else if(t.link_start!=0 || t.link_end!=0){return problem(1001,t.start,t.end);}
        var usize j = 1; while (j < i) {
            let m::Target other = m::target_at(p,j);
            if(t.foreign && other.foreign && bytes::equal(m::foreign_text(p,i),m::foreign_text(p,j))){if(t.result!=other.result || t.signature_count!=other.signature_count){return problem(5106,t.start,t.end);}var usize n=0;while(n<t.signature_count){if(m::signature_at(p,t.signature_start+n)!=m::signature_at(p,other.signature_start+n)){return problem(5106,t.start,t.end);}n+=1;}}
            if (other.binding == t.binding || other.module_index == t.module_index && other.declaration == t.declaration) { return problem(1001,t.start,t.end); } j += 1;
        }
        j = 0; while (j < t.signature_count) { if (!mark(ref mut signatures,t.signature_start + j)) { return problem(1001,t.start,t.end); } j += 1; }
        i += 1;
    }
    i = 1; while (i <= m::local_count(p)) {
        let m::Local l = m::local_at(p,i);
        if (l.parameter != (bytes::view(ref params)[i] != 0)) { return problem(1001,l.start,l.end); } i += 1;
    }
    i = 1; while (i <= m::block_count(p)) {
        let m::Block b = m::block_at(p,i); let m::Function f = m::function_at(p,b.function);
        if(b.kind!=5 && b.unsafe_call){return problem(1001,b.start,b.end);}
        var usize op = b.first;
        while (op != 0) {
            let m::Operation o = m::operation_at(p,op);
            if (!mark(ref mut ops,op) || !operation(p,op,o) || o.kind == 5 && (op != b.last || b.kind != 4 || b.check != op)) { return problem(1001,o.start,o.end); }
            if(o.kind==26 || o.kind==25){let u32 required=m::required_drop_effects(p,o.type);if((m::target_at(p,f.target).effects&required)!=required){return problem(7102,o.start,o.end);}}
            if (o.kind == 6 || o.kind == 21 || o.kind == 9 || o.kind == 11 || (o.kind == 19 || o.kind == 20 || o.kind==38 || o.kind==39)) { var usize j = 0; while (j < o.argument_count) { if (!mark(ref mut edges,o.argument_start + j)) { return problem(1001,o.start,o.end); } j += 1; } }
            if (o.result != 0) { let ref mut usize count = v::at_mut<usize>(ref mut definitions,o.result); *count += 1; }
            if (o.kind == 1 && o.type == 6 && !o.boolean) { let ref mut u8[] flags = bytes::view_mut(ref mut false_constants); flags[o.result] = 1; }
            op = o.next;
        }
        if (b.kind == 1) {
            let u32 result = m::target_at(p,f.target).result;
            if (!empty_fields(b) || !no_call(b) || b.type != 0 || (if (result == 7) { b.value != 0 } else { !local(p,b.value,b.function,result) })) { return problem(1001,b.start,b.end); }
        } else if (b.kind == 2 || b.kind == 3) {
            if (!empty_fields(b) || !no_call(b) || (if (b.kind == 2) { b.value != 0 || b.type != 0 } else { !local(p,b.value,b.function,6) || b.type != 6 })) { return problem(1001,b.start,b.end); }
        } else if (b.kind == 4) {
            let m::Operation o = m::operation_at(p,b.check);
            if (!no_call(b) || b.value != 0 || o.left != b.left || o.right != b.right || o.operator != b.operator || o.owner_type != b.type
                || b.trap_mask != mask(o.operator,o.owner_type,o.right == 0) || b.target == f.entry || b.alternative == f.entry) { return problem(1501,b.start,b.end); }
            if (*v::at<usize>(ref guards,b.target) != 0 || *v::at<usize>(ref traps,b.alternative) != 0) { return problem(1501,b.start,b.end); }
            { let ref mut usize slot = v::at_mut<usize>(ref mut guards,b.target); *slot = i; }
            { let ref mut usize slot = v::at_mut<usize>(ref mut traps,b.alternative); *slot = i; }
            let m::Block failed = m::block_at(p,b.alternative);
            if (failed.first != 0) { return problem(1501,failed.start,failed.end); }
            if (o.operator != 0) {
                let m::Block normal = m::block_at(p,b.target);
                if (normal.first == 0) { return problem(1501,normal.start,normal.end); }
                let m::Operation actual = m::operation_at(p,normal.first);
                if (actual.kind != if (o.right == 0) { 3 } else { 4 } || actual.left != o.left || actual.right != o.right || actual.operator != o.operator || actual.owner_type != o.owner_type) { return problem(1501,actual.start,actual.end); }
            }
        } else if (b.kind == 5) {
            let m::Target t = m::target_at(p,b.callee);
            if(b.unsafe_call!=t.unsafe_call){return problem(5101,b.start,b.end);}
            if((m::target_at(p,f.target).effects&t.effects)!=t.effects){return problem(7102,b.start,b.end);}
            let usize drop=if(b.type==0){0}else{m::drop_index(p,b.type)};
            if (b.left!=0 || b.right!=0 || b.operator!=0 || b.check!=0 || b.trap_mask!=0 || b.value != 0 || b.alternative != 0 || (if(b.type==0){m::drop_target(p,b.callee)}else{drop==0 || m::drop_at(p,drop-1).target!=b.callee}) || (if (t.result == 7) { b.destination != 0 } else { !local(p,b.destination,b.function,t.result) })) { return problem(1001,b.start,b.end); }
            if (b.destination != 0) { let ref mut usize count = v::at_mut<usize>(ref mut definitions,b.destination); *count += 1; }
            var usize j = 0; while (j < b.argument_count) {
                let usize at = b.argument_start + j;
                if (!mark(ref mut edges,at) || !local(p,m::edge_at(p,at),b.function,m::signature_at(p,t.signature_start + j))) { return problem(1001,b.start,b.end); } j += 1;
            }
        } else {
            if (!no_call(b) || b.value != 0 || b.left != 0 || b.right != 0 || b.operator != 0 || b.type != 0 || b.check != 0 || b.kind == 7 && b.trap_mask != 0) { return problem(1001,b.start,b.end); }
        }
        i += 1;
    }
    // A guarded continuation cannot be entered through an unchecked bypass.
    i = 1; while (i <= m::block_count(p)) {
        let m::Block b = m::block_at(p,i);
        var usize edge = 0; while (edge < 2) {
            let usize to = if (edge == 0) { b.target } else { b.alternative };
            if (to != 0) {
                let usize guard = *v::at<usize>(ref guards,to); let usize trap = *v::at<usize>(ref traps,to);
                if (guard != 0 && (guard != i || edge != 0) || trap != 0 && (trap != i || edge != 1)) { return problem(1501,b.start,b.end); }
            }
            edge += 1;
        }
        if (b.kind == 6 && *v::at<usize>(ref traps,i) == 0) { return problem(1501,b.start,b.end); }
        if (b.kind == 7) {
            let usize from = *v::at<usize>(ref guards,i);
            if (from == 0 || b.first != 0) { return problem(1501,b.start,b.end); }
            let m::Operation checked = m::operation_at(p,m::block_at(p,from).check);
            if (checked.operator != 0 || checked.owner_type != 6 || m::local_at(p,checked.left).parameter || *v::at<usize>(ref definitions,checked.left) != 1
                || bytes::view(ref false_constants)[checked.left] == 0) { return problem(1501,b.start,b.end); }
        }
        var usize op = b.first; while (op != 0) {
            let m::Operation o = m::operation_at(p,op);
            if ((o.kind == 3 || o.kind == 4) && mask(o.operator,o.owner_type,o.kind == 3) != 0) {
                let usize from = *v::at<usize>(ref guards,i);
                if (op != b.first || from == 0) { return problem(1501,o.start,o.end); }
                let m::Operation checked = m::operation_at(p,m::block_at(p,from).check);
                if (checked.operator != o.operator || checked.left != o.left || checked.right != o.right || checked.owner_type != o.owner_type) { return problem(1501,o.start,o.end); }
            }
            op = o.next;
        }
        i += 1;
    }
    if (!filled(ref ops,1) || !filled(ref edges,0) || !filled(ref signatures,0)) { return problem(1001,0,0); }
    return problem(0,0,0);
}
