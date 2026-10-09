module module_unit_value_collect;
import module_unit_value_model as m;
import module_unit_value_walk as walk;
import module_unit_body_ast as ast;
import module_unit_name_model as names;
import module_unit_calls as calls;
import module_unit_bodies as bodies;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_types as types;
import syntax as syntax;
import type_state as scalars;
import std::bytes as bytes;
import std::vec as v;

// seed table uses body-local callee IDs; a result type does not validate arguments.
pub fn Result<v::Vec<usize>,m::Problem> seeds(ref ast::Ast a,m::Frame frame){
    var v::Vec<usize> result=v::empty<usize>();var usize i=0;while(i<ast::expression_count(a)){if(!v::push<usize>(ref mut result,0).is_ok){return .err(m::fail(frame,11,0,0,0,0));}i+=1;}return .ok(result);
}
pub fn m::Problem concrete_seeds(ref bodies::Plan owned,ref interfaces::Interfaces imported,m::Frame frame,ref mut v::Vec<usize> table,ref mut usize work){
    let usize count=if(frame.kind==1){bodies::edge_count(bodies::at(owned,frame.context))}else{bodies::cleanup_edge_count(bodies::cleanup_at(owned,frame.context))};var usize i=0;
    while(i<count){if(!m::charge(work,1)){return m::fail(frame,10,0,0,0,0);}let edge=if(frame.kind==1){bodies::edge_at(bodies::at(owned,frame.context),i)}else{bodies::cleanup_edge_at(bodies::cleanup_at(owned,frame.context),i)};
        if(edge.target.instance!=0){let slot=v::at_mut<usize>(table,edge.expression-1);*slot=generics::at(interfaces::specializations(imported),edge.target.instance).result;}i+=1;
    }return m::fail(frame,0,0,0,0,0);
}
pub fn m::Problem ordinary_seeds(ref ast::Ast a,ref interfaces::Interfaces imported,m::Frame frame,usize offset,ref mut v::Vec<usize> table,ref mut usize work){
    var usize i=1;while(i<=ast::expression_count(a)){if(!m::charge(work,1)){return m::fail(frame,10,0,0,0,0);}if(ast::expression_at(a,i).kind==20){let c=generics::call_at(interfaces::specializations(imported),offset+ast::expression_origin(a,i));if(c.instance!=0){let slot=v::at_mut<usize>(table,i-1);*slot=generics::at(interfaces::specializations(imported),c.instance).result;}}i+=1;}return m::fail(frame,0,0,0,0,0);
}
pub fn Result<m::Values,m::Problem> project(ref syntax::Parsed source,ref ast::Ast a,ref names::Resolved resolved,m::Frame frame,ref calls::Plan declared,ref mut usize cursor,ref v::Vec<usize> seeds,ref v::Vec<types::Type> graph,ref m::Atoms atoms){
    var usize work=0;var v::Vec<m::Expression> expressions=v::empty<m::Expression>();var v::Vec<m::Local> locals=v::empty<m::Local>();var v::Vec<m::Return> returns=v::empty<m::Return>();var usize i=1;
    if(names::use_count(resolved)!=ast::expression_count(a) || v::len<usize>(seeds)!=ast::expression_count(a)){return .err(m::fail(frame,12,0,0,0,0));}
    while(i<=ast::expression_count(a)){let original=ast::expression_origin(a,i);let e=syntax::expression_at(source,original);let n=ast::expression_at(a,i);var usize type=0;var u32 state=0;
        if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,e.start,e.end,0,0));}
        if(e.kind!=n.kind || e.start!=n.start || e.end!=n.end){return .err(m::fail(frame,12,e.start,e.end,0,0));}
        if(e.kind==2){let usize use=names::use_at(resolved,i);if(use!=0){let r=names::reference_at(resolved,use);if(r.source_index!=original || r.start!=e.start || r.end!=e.end){return .err(m::fail(frame,12,e.start,e.end,0,0));}if(r.kind==1){type=names::symbol_at(resolved,r.symbol).type;}else if(r.kind==3 || r.declaration_kind==3 || r.declaration_kind==9 || r.declaration_kind==4 || r.declaration_kind==6){state=2;}}}
        // Lists and callee/type syntax have no value of their own.
        else if(e.kind==7 || e.kind==11 || e.kind==14 || e.kind==16 || e.kind==17 || e.kind==18 || e.kind==19 || e.kind==20 || e.kind==25 || e.kind==32 || e.kind==33){state=2;}
        if(e.kind==6){if(n.left!=0){type=*v::at<usize>(seeds,n.left-1);}if(*cursor<calls::count(declared)){let c=calls::at(declared,*cursor);if(c.context_kind==frame.kind && c.context==frame.context && c.expression==i){if(c.module_index!=frame.module_index || c.declaration!=frame.declaration || c.source_index!=original || c.start!=e.start || c.end!=e.end || type!=0 && type!=c.result){return .err(m::fail(frame,12,e.start,e.end,0,0));}type=c.result;*cursor=*cursor+1;}}}
        if(type>types::count(graph)){return .err(m::fail(frame,12,e.start,e.end,0,type));}if(type!=0){state=1;}
        let usize left=m::index(ref expressions,e.left);let usize right=m::index(ref expressions,e.right);let op=syntax::expression_text(source,original);
        let bool flexible=if(e.kind==1){syntax::literal_kind(source,original)==5}else if(e.kind==3){left!=0 && v::at<m::Expression>(ref expressions,left-1).flexible}else if(e.kind==4){left!=0 && (bytes::equal(op,b"+") || bytes::equal(op,b"-") || bytes::equal(op,b"~")) && v::at<m::Expression>(ref expressions,left-1).flexible}else if(e.kind==5){scalars::operation(op) && left!=0 && right!=0 && v::at<m::Expression>(ref expressions,left-1).flexible && v::at<m::Expression>(ref expressions,right-1).flexible}else{false};
        if(!v::push<m::Expression>(ref mut expressions,m::Expression{.source_index=original,.kind=e.kind,.start=e.start,.end=e.end,.left=e.left,.right=e.right,.expected=0,.type,.state,.flexible,.minimum=false}).is_ok){return .err(m::fail(frame,11,e.start,e.end,0,0));}i+=1;
    }
    i=1;while(i<=names::local_count(resolved)){let s=names::symbol_at(resolved,i);var usize source_index=0;var usize initializer=0;if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,s.start,s.end,0,0));}
        if(s.entry!=0){source_index=ast::entry_origin(a,s.entry);let e=syntax::entry_at(source,source_index-1);if(e.name_start!=s.start || e.name_end!=s.end){return .err(m::fail(frame,12,s.start,s.end,0,0));}initializer=e.expression;}else if(s.pattern!=0){source_index=ast::expression_origin(a,s.pattern);}
        if(!v::push<m::Local>(ref mut locals,m::Local{.kind=s.kind,.source_index,.initializer,.expected=s.type,.type=s.type,.start=s.start,.end=s.end}).is_ok){return .err(m::fail(frame,11,s.start,s.end,0,0));}i+=1;
    }
    i=1;while(i<=ast::statement_count(a)){let s=ast::statement_at(a,i);if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,s.start,s.end,0,0));}if(s.kind==3){let usize expression=if(s.expression==0){0}else{ast::expression_origin(a,s.expression)};if(!v::push<m::Return>(ref mut returns,m::Return{.expression,.expected=frame.result,.actual=0,.checked=false,.start=s.start,.end=s.end}).is_ok){return .err(m::fail(frame,11,s.start,s.end,0,0));}}i+=1;}
    return walk::resolve(source,resolved,frame,expressions,locals,returns,graph,atoms,work);
}
