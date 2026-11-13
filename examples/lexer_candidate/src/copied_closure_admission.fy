module copied_closure_admission;
import syntax as syntax;
import std::bytes as bytes;

pub fn bool reserved(ref u8[] name){return name.len>=12 && bytes::equal(name[..12],b"_fy_closure_");}
pub fn bool needed(ref syntax::Parsed p){var usize i=1;while(i<=syntax::expression_count(p)){if(syntax::expression_at(p,i).kind==34){return true;}i+=1;}return false;}
pub fn bool function(ref syntax::Parsed p,syntax::Declaration d){var usize i=1;while(i<=syntax::expression_count(p)){let n=syntax::expression_at(p,i);if(n.kind==34 && n.start>=d.start && n.end<=d.end){return true;}i+=1;}return false;}
fn bool i32_type(ref syntax::Parsed p,usize root){return root!=0 && syntax::type_at(p,root).kind==1 && bytes::equal(syntax::type_text(p,root),b"i32");}
fn bool copied_scalar(ref syntax::Parsed p,usize root){return i32_type(p,root) || root!=0 && syntax::type_at(p,root).kind==1 && bytes::equal(syntax::type_text(p,root),b"u32");}
// The first-class descriptor remains the original all-i32 family.
pub fn bool value_signature(ref syntax::Parsed p,usize root){let n=syntax::expression_at(p,root);if(n.kind!=34 || n.right==0){return false;}let h=syntax::expression_at(p,n.right);if(h.kind!=35 || h.right%129!=1 || h.right/129>4 || !i32_type(p,h.token)){return false;}var usize i=0;while(i<h.right/129+1){if(!i32_type(p,syntax::entry_at(p,h.left+i-1).type_root)){return false;}i+=1;}i=0;while(i<syntax::block_statement_count(p,n.left)){let s=syntax::statement_at(p,syntax::block_statement(p,n.left,i));if(s.kind==2 && s.entry!=0){let e=syntax::entry_at(p,s.entry-1);if(e.type_root!=0 && !i32_type(p,e.type_root)){return false;}}i+=1;}return true;}
fn bool bound(ref syntax::Parsed p,syntax::Expression h,usize body,usize visible,ref u8[] name){
    var usize i=0;while(i<h.right/129+h.right%129){let e=syntax::entry_at(p,h.left+i-1);if(bytes::equal(syntax::source_view(p)[e.name_start..e.name_end],name)){return true;}i+=1;}
    i=0;while(i<visible){let s=syntax::statement_at(p,syntax::block_statement(p,body,i));let e=syntax::entry_at(p,s.entry-1);if(bytes::equal(syntax::source_view(p)[e.name_start..e.name_end],name)){return true;}i+=1;}return false;
}
fn bool pure(ref syntax::Parsed p,usize root,syntax::Expression h,usize body,usize visible,bool initializing,usize depth,ref mut usize nodes){if(root==0 || root>syntax::expression_count(p) || depth>=64 || *nodes>=64){return false;};*nodes=*nodes+1;let n=syntax::expression_at(p,root);if(n.kind==1){return !initializing || syntax::literal_kind(p,root)==5;}if(n.kind==2){return bound(p,h,body,visible,syntax::expression_text(p,root));}
    if(n.kind==3){return pure(p,n.left,h,body,visible,initializing,depth+1,nodes);}let op=syntax::expression_text(p,root);if(n.kind==4){return (bytes::equal(op,b"+") || bytes::equal(op,b"-")) && pure(p,n.left,h,body,visible,initializing,depth+1,nodes);}if(n.kind==5){return (bytes::equal(op,b"+") || bytes::equal(op,b"-") || bytes::equal(op,b"*") || bytes::equal(op,b"/") || bytes::equal(op,b"%")) && pure(p,n.left,h,body,visible,initializing,depth+1,nodes) && pure(p,n.right,h,body,visible,initializing,depth+1,nodes);}return false;
}
pub fn u32 validate(ref syntax::Parsed p,usize root){let n=syntax::expression_at(p,root);if(n.kind!=34 || n.right==0 || n.left==0){return 3;}let h=syntax::expression_at(p,n.right);let captures=h.right/129;let parameters=h.right%129;if(h.kind!=35 || captures>4 || parameters>4 || !copied_scalar(p,h.token)){return 3;}var usize i=0;
    while(i<captures+parameters){let e=syntax::entry_at(p,h.left+i-1);let name=syntax::source_view(p)[e.name_start..e.name_end];if(!copied_scalar(p,e.type_root) || reserved(name)){return 3;}var usize j=0;while(j<i){let old=syntax::entry_at(p,h.left+j-1);if(bytes::equal(name,syntax::source_view(p)[old.name_start..old.name_end])){return 3;}j+=1;}i+=1;}
    let b=syntax::statement_at(p,n.left);if(b.kind!=1){return 3;}var usize value=b.expression;var usize locals=syntax::block_statement_count(p,n.left);
    if(value==0){if(locals==0){return 3;}let s=syntax::statement_at(p,syntax::block_statement(p,n.left,locals-1));if(s.kind!=3 || s.expression==0){return 3;}value=s.expression;locals-=1;}if(locals>16){return 3;}
    var usize nodes=0;i=0;while(i<locals){let s=syntax::statement_at(p,syntax::block_statement(p,n.left,i));if(s.kind!=2 || s.entry==0 || s.expression==0){return 3;}let e=syntax::entry_at(p,s.entry-1);let name=syntax::source_view(p)[e.name_start..e.name_end];
        if(e.kind!=2 || e.type_root!=0 && !copied_scalar(p,e.type_root) || name.len==0 || reserved(name) || bound(p,h,n.left,i,name) || !pure(p,s.expression,h,n.left,i,true,0,ref mut nodes)){return 3;}i+=1;
    }return if(pure(p,value,h,n.left,locals,false,0,ref mut nodes)){0}else{3};
}
