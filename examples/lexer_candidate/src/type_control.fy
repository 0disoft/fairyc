module type_control;
import syntax as syntax;
import std::bytes as bytes;

// ADR 0213: header trees have no calls, borrows, indexing or block effects.
pub fn bool plain_header(ref syntax::Parsed parsed, usize id, usize depth) {
    if (id == 0 || depth >= 64) { return false; }
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (node.kind == 1 || node.kind == 2) { return true; }
    if (node.kind == 3 || node.kind == 4) { return plain_header(parsed,node.left,depth + 1); }
    if (node.kind == 5) { return plain_header(parsed,node.left,depth + 1) && plain_header(parsed,node.right,depth + 1); }
    return false;
}

// Match the reference's return gate over admitted syntax. Loops are not proof
// of a return, and plain standalone blocks do not promote nested returns.
pub fn bool expression_terminates(ref syntax::Parsed parsed, usize id, usize depth) {
    if (id == 0 || depth >= 64) { return false; }
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (node.kind == 15) {
        if(node.right==0){return false;}var usize link=node.right;while(link!=0){let arm=syntax::expression_at(parsed,syntax::expression_at(parsed,link).left);if(!block_terminates(parsed,arm.right,depth+1)){return false;}link=syntax::expression_at(parsed,link).right;}return true;
    }
    if (node.kind != 13) { return false; }
    let syntax::Expression branches = syntax::expression_at(parsed,node.right);
    if (branches.right == 0 || !block_terminates(parsed,branches.left,depth + 1)) { return false; }
    let syntax::Expression alternative = syntax::expression_at(parsed,branches.right);
    return if (alternative.kind == 13) { expression_terminates(parsed,branches.right,depth + 1) }
        else { block_terminates(parsed,alternative.left,depth + 1) };
}
pub fn bool block_terminates(ref syntax::Parsed parsed, usize id, usize depth) {
    if (depth >= 64) { return false; }
    let syntax::Statement body = syntax::statement_at(parsed,id);
    var usize index = 0;
    while (index < syntax::block_statement_count(parsed,id)) {
        let syntax::Statement item = syntax::statement_at(parsed,syntax::block_statement(parsed,id,index));
        if (item.kind == 3) { return true; }
        if (item.kind == 11 && item.expression != 0) {
            let syntax::Expression assertion = syntax::expression_at(parsed,item.expression);
            if (assertion.kind == 1 && bytes::equal(syntax::expression_text(parsed,item.expression),b"false")) { return true; }
        }
        if (expression_terminates(parsed,item.expression,depth + 1)) { return true; }
        index += 1;
    }
    return body.expression != 0;
}
