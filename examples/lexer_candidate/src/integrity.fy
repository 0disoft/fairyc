module integrity;
import syntax as syntax;
import types as types;
import records as records;

fn bool span(ref syntax::Parsed p, usize start, usize end) {
    return start < end && end <= syntax::source_view(p).len;
}
fn bool value_kind(u32 kind) {
    return (kind >= 1 && kind <= 6) || (kind >= 8 && kind <= 10) || kind == 12 || kind == 13 || kind == 15 || (kind >= 20 && kind <= 24) || kind == 26 || kind == 28 || kind == 29 || kind == 30 || kind == 31;
}
fn bool expr(ref syntax::Parsed p, usize id, usize before, u32 kind) {
    if (id == 0 || id >= before || id > syntax::expression_count(p)) { return false; }
    let u32 actual = syntax::expression_at(p,id).kind;
    return if (kind == 0) { value_kind(actual) } else { actual == kind };
}
fn bool optional_expr(ref syntax::Parsed p, usize id, usize before, u32 kind) {
    return id == 0 || expr(p,id,before,kind);
}
fn bool type_ref(ref syntax::Parsed p, usize id, usize before, u32 kind) {
    if (id == 0 || id >= before || id > syntax::type_count(p)) { return false; }
    let u32 actual = syntax::type_at(p,id).kind;
    return if (kind == 0) { (actual >= 1 && actual <= 6) || actual == 8 || actual == 11 || actual == 12 } else { actual == kind };
}
fn bool stmt(ref syntax::Parsed p, usize id, usize before, u32 kind) {
    if (id == 0 || id >= before || id > syntax::statement_count(p)) { return false; }
    let u32 actual = syntax::statement_at(p,id).kind;
    return if (kind == 0) { actual >= 1 && actual <= 14 && actual != 7 } else { actual == kind };
}
pub fn bool valid_expression(ref syntax::Parsed p, syntax::Expression n, usize id) {
    if (id == 0 || id > syntax::expression_count(p) || !span(p,n.start,n.end) || n.token >= syntax::source_token_count(p)) { return false; }
    let usize statements = syntax::statement_count(p) + 1;
    let usize type_limit = syntax::type_count(p) + 1;
    if (n.kind == 1 || n.kind == 2 || n.kind == 17 || n.kind == 28) { return n.left == 0 && n.right == 0; }
    if (n.kind == 3 || n.kind == 4 || n.kind == 8 || n.kind == 21 || n.kind == 22 || n.kind == 30) { return expr(p,n.left,id,0) && n.right == 0; }
    if (n.kind == 5 || n.kind == 9) { return expr(p,n.left,id,0) && expr(p,n.right,id,0); }
    if (n.kind == 29) { return expr(p,n.left,id,0) && (expr(p,n.right,id,1) || expr(p,n.right,id,2)); }
    if (n.kind == 6) { return expr(p,n.left,id,0) && optional_expr(p,n.right,id,7); }
    if (n.kind == 7 || n.kind == 11 || n.kind == 25) { return expr(p,n.left,id,0) && optional_expr(p,n.right,id,n.kind); }
    if (n.kind == 10) { return (expr(p,n.left,id,2) || expr(p,n.left,id,20) || expr(p,n.left,id,28)) && optional_expr(p,n.right,id,11); }
    if (n.kind == 12 || n.kind == 31) { return stmt(p,n.left,statements,1) && n.right == 0; }
    if (n.kind == 13) { return expr(p,n.left,id,0) && expr(p,n.right,id,14); }
    if (n.kind == 14) { return stmt(p,n.left,statements,1) && (n.right == 0 || expr(p,n.right,id,12) || expr(p,n.right,id,13)); }
    if (n.kind == 15) { return expr(p,n.left,id,0) && optional_expr(p,n.right,id,19); }
    if (n.kind == 16) { return optional_expr(p,n.left,id,17) && n.right == 0; }
    if (n.kind == 18) { return expr(p,n.left,id,16) && stmt(p,n.right,statements,1); }
    if (n.kind == 19) { return expr(p,n.left,id,18) && optional_expr(p,n.right,id,19); }
    if (n.kind == 20) { return expr(p,n.left,id,2) && type_ref(p,n.right,type_limit,7); }
    if (n.kind == 23) { return expr(p,n.left,id,0) && type_ref(p,n.right,type_limit,0); }
    if (n.kind == 24) { return n.left == 0 && optional_expr(p,n.right,id,25); }
    if (n.kind == 26) { return expr(p,n.left,id,0) && expr(p,n.right,id,27); }
    if (n.kind == 27) { return optional_expr(p,n.left,id,0) && optional_expr(p,n.right,id,0); }
    return false;
}
pub fn bool valid_type(ref syntax::Parsed p, types::Node n, usize id) {
    if (id == 0 || id > syntax::type_count(p) || !span(p,n.start,n.end)) { return false; }
    if (n.kind == 1 || n.kind == 9 || n.kind == 10 || n.kind == 13) { return n.left == 0 && n.right == 0; }
    if (n.kind == 14) { return type_ref(p,n.left,id,13) && (n.right == 0 || type_ref(p,n.right,id,14)); }
    if ((n.kind >= 2 && n.kind <= 5) || n.kind == 11 || n.kind == 12) { return type_ref(p,n.left,id,0) && n.right == 0; }
    if (n.kind == 6) { return type_ref(p,n.left,id,0) && type_ref(p,n.right,id,7); }
    if (n.kind == 7) { return type_ref(p,n.left,id,0) && (n.right == 0 || type_ref(p,n.right,id,7)); }
    if (n.kind == 8) { return type_ref(p,n.left,id,0) && (type_ref(p,n.right,id,9) || type_ref(p,n.right,id,10)); }
    return false;
}
pub fn bool valid_statement(ref syntax::Parsed p, syntax::Statement n, usize id) {
    if (id == 0 || id > syntax::statement_count(p) || !span(p,n.start,n.end)) { return false; }
    let usize expressions = syntax::expression_count(p) + 1;
    if (n.kind == 1) {
        if (!optional_expr(p,n.expression,expressions,0) || n.target != 0 || n.entry != 0 || n.operator != 0) { return false; }
        var usize link = n.first; var usize count = 0; var usize before = id;
        while (link != 0) {
            if (!stmt(p,link,before,7)) { return false; }
            before = link; link = syntax::statement_at(p,link).second; count += 1;
        }
        return count == n.second;
    }
    if (n.kind == 7) { return stmt(p,n.first,id,0) && (n.second == 0 || stmt(p,n.second,id,7)) && n.expression == 0 && n.target == 0 && n.entry == 0 && n.operator == 0; }
    if (n.kind == 2 || n.kind == 3) { return n.entry > 0 && n.entry <= syntax::entry_count(p) && (if (n.kind == 2) { expr(p,n.expression,expressions,0) } else { optional_expr(p,n.expression,expressions,0) }) && n.target == 0 && n.first == 0 && n.second == 0 && n.operator == 0; }
    if (n.kind == 4) { return expr(p,n.target,expressions,0) && expr(p,n.expression,expressions,0) && n.operator < syntax::source_token_count(p) && n.first == 0 && n.second == 0 && n.entry == 0; }
    if (n.target != 0 || n.entry != 0 || n.operator != 0) { return false; }
    if (n.kind == 14) { return optional_expr(p,n.expression,expressions,0) && stmt(p,n.first,id,1) && stmt(p,n.second,id,15); }
    if (n.kind == 15) { return n.expression == 0 && (n.first == 0 || stmt(p,n.first,id,2) || stmt(p,n.first,id,4)) && (n.second == 0 || stmt(p,n.second,id,4)); }
    if (n.kind == 5 || n.kind == 8) { return expr(p,n.expression,expressions,0) && stmt(p,n.first,id,1) && (if (n.kind == 8) { n.second == 0 } else { n.second == 0 || stmt(p,n.second,id,1) || stmt(p,n.second,id,5) }); }
    if (n.first != 0 || n.second != 0) { return false; }
    if (n.kind == 6 || n.kind == 11 || n.kind == 12 || n.kind == 13) { return expr(p,n.expression,expressions,0); }
    return (n.kind == 9 || n.kind == 10) && n.expression == 0;
}
pub fn bool valid(ref syntax::Parsed p) {
    var usize i = 1;
    while (i <= syntax::expression_count(p)) { if (!valid_expression(p,syntax::expression_at(p,i),i)) { return false; } i += 1; }
    i = 1;
    while (i <= syntax::type_count(p)) { if (!valid_type(p,syntax::type_at(p,i),i)) { return false; } i += 1; }
    i = 1;
    while (i <= syntax::statement_count(p)) { if (!valid_statement(p,syntax::statement_at(p,i),i)) { return false; } i += 1; }
    i = 0;
    while (i < syntax::declaration_count(p)) {
        let syntax::Declaration d = syntax::declaration_at(p,i);
        if (!span(p,d.start,d.end)) { return false; }
        if (d.generic_parameters == 0) {
            if (d.generic_start != 0 || d.generic_end != 0) { return false; }
        } else {
            if ((d.kind != 3 && d.kind != 4 && d.kind != 6) || d.generic_start < d.name_end || d.generic_end <= d.generic_start || d.generic_end > d.end || !type_ref(p,d.generic_parameters,syntax::type_count(p) + 1,14)) { return false; }
            var usize link = d.generic_parameters;
            while (link != 0) {
                let types::Node name = syntax::type_at(p,syntax::type_at(p,link).left);
                if (name.start <= d.generic_start || name.end >= d.generic_end) { return false; }
                link = syntax::type_at(p,link).right;
            }
        }
        if (d.kind == 8 || d.kind == 10) {
            if (d.name_start != 0 || d.name_end != 0) { return false; }
        } else if (d.name_start < d.start || d.name_start >= d.name_end || d.name_end > d.end) { return false; }
        if (d.entry_start > syntax::entry_count(p) || d.entry_count > syntax::entry_count(p) - d.entry_start || !valid_declaration_fields(p,d)) { return false; }
        if (d.kind == 3) {
            if (!stmt(p,d.body,syntax::statement_count(p) + 1,1) || !type_ref(p,d.result_type,syntax::type_count(p) + 1,0)) { return false; }
        } else if (d.kind == 8) {
            if (d.body != 0 || d.result_type != 0 || !expr(p,d.expression,syntax::expression_count(p) + 1,0)) { return false; }
        } else if (d.kind == 9) {
            if (d.body != 0 || d.expression != 0 || !type_ref(p,d.result_type,syntax::type_count(p) + 1,0)) { return false; }
        } else if (d.kind == 7) {
            if (d.body != 0 || !type_ref(p,d.result_type,syntax::type_count(p) + 1,0) || !expr(p,d.expression,syntax::expression_count(p) + 1,0)) { return false; }
        } else if (d.body != 0 || d.result_type != 0 || (d.kind != 1 && d.kind != 2 && d.kind != 4 && d.kind != 6 && d.kind != 10 && d.kind != 11)) { return false; }
        if (d.cleanup_body != 0 && (d.kind != 6 || !stmt(p,d.cleanup_body,syntax::statement_count(p) + 1,1))) { return false; }
        if (!optional_expr(p,d.expression,syntax::expression_count(p) + 1,0)) { return false; }
        i += 1;
    }
    i = 0;
    while (i < syntax::field_count(p)) {
        if (!valid_field(p,syntax::field_at(p,i),i)) { return false; }
        i += 1;
    }
    return true;
}
pub fn bool valid_declaration_fields(ref syntax::Parsed p, syntax::Declaration d) {
    if (!span(p,d.start,d.end) || d.field_start > syntax::field_count(p) || d.field_count > syntax::field_count(p) - d.field_start) { return false; }
    var usize field = d.field_start;
    var usize pending = field;
    while (field < d.field_start + d.field_count) {
        let u32 kind = syntax::field_at(p,field).kind;
        if (if (d.kind == 11) { kind != 3 && kind != 4 } else if (d.kind == 10) { kind != 1 && kind != 2 } else { (d.kind != 4 && d.kind != 6) || kind != 0 }) { return false; }
        if (d.kind == 11 && kind == 3) {
            if (syntax::field_at(p,field).payload_count != field - pending) { return false; }
            if (syntax::field_at(p,field).payload_open != 0) {
                if (syntax::field_at(p,field).payload_start != pending) { return false; }
            } else if (pending != field) { return false; }
            pending = field + 1;
        }
        if (syntax::field_at(p,field).start < d.start || syntax::field_at(p,field).end > d.end) { return false; }
        field += 1;
    }
    return d.kind != 11 || pending == field;
}
pub fn bool valid_field(ref syntax::Parsed p, records::Field n, usize index) {
    if (index >= syntax::field_count(p)) { return false; }
    let usize start = n.start;
    let usize end = n.end;
    let usize name_start = n.name_start;
    let usize name_end = n.name_end;
    let u32 kind = n.kind;
    let usize key = n.key_type;
    return span(p,start,end) && name_start >= start && name_start < name_end && name_end <= end && kind <= 4
        && (kind == 0 || !n.public)
        && (kind != 4 || (n.public_start == 0 && n.public_end == 0))
        && (if (kind == 3) { n.type_root == 0 && n.public_start == 0 && n.public_end == 0 } else { type_ref(p,n.type_root,syntax::type_count(p) + 1,0) })
        && (if (kind == 2) { type_ref(p,key,syntax::type_count(p) + 1,0) } else { key == 0 }) && valid_payload(p,n,index);
}
fn bool valid_payload(ref syntax::Parsed p, records::Field n, usize index) {
    let usize open = n.payload_open;
    let usize end = n.payload_end;
    let usize first = n.payload_start;
    let usize count = n.payload_count;
    if (open == 0) { return end == 0 && first == 0 && count == 0; }
    if (n.kind != 3 || open < n.name_end || end <= open || end >= n.end || first > index || count != index - first) { return false; }
    var usize previous = open;
    var usize i = first;
    while (i < index) {
        if (syntax::field_at(p,i).kind != 4 || syntax::field_at(p,i).start <= previous || syntax::field_at(p,i).end >= end) { return false; }
        previous = syntax::field_at(p,i).end - 1;
        i += 1;
    }
    return true;
}
