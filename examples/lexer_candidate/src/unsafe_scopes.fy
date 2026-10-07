module unsafe_scopes;
import syntax as syntax;

// Source checking only; the owned HIR rechecks lexical ancestry independently.
pub fn bool block(ref syntax::Parsed p,usize body){
    var usize i=1;while(i<=syntax::expression_count(p)){let e=syntax::expression_at(p,i);if(e.kind==31 && e.left==body){return true;}i+=1;}return false;
}
pub fn bool allowed(ref syntax::Parsed p,usize start,usize end){
    var usize i=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);
        if(d.kind==3 && d.unsafe_end!=0 && d.body!=0){let b=syntax::statement_at(p,d.body);if(b.start<=start && end<=b.end){return true;}}i+=1;}
    i=1;while(i<=syntax::expression_count(p)){let e=syntax::expression_at(p,i);if(e.kind==31){let b=syntax::statement_at(p,e.left);if(b.start<=start && end<=b.end){return true;}}i+=1;}return false;
}
