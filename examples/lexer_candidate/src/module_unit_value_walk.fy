module module_unit_value_walk;
import module_unit_value_model as m;
import module_unit_name_model as names;
import module_types as types;
import module_unit_value_scalar as scalar;
import module_unit_argument_types as args;
import syntax as syntax;
import std::vec as v;

pub fn Result<m::Values,m::Problem> resolve(ref syntax::Parsed source,ref names::Resolved resolved,m::Frame frame,v::Vec<m::Expression> expressions_input,v::Vec<m::Local> locals_input,v::Vec<m::Return> returns_input,v::Vec<m::Invocation> calls_input,v::Vec<m::Argument> arguments_input,ref v::Vec<types::Type> graph,ref m::Atoms atoms,usize initial_work){
    var v::Vec<m::Expression> expressions=expressions_input;var v::Vec<m::Local> locals=locals_input;var v::Vec<m::Return> returns=returns_input;var v::Vec<m::Invocation> calls=calls_input;var v::Vec<m::Argument> arguments=arguments_input;var usize work=initial_work;var bool changed=true;var usize rounds=0;
    while(changed && rounds<64){changed=false;rounds+=1;var usize i=0;
        while(i<v::len<m::Local>(ref locals)){let local=*v::at<m::Local>(ref locals,i);if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,local.start,local.end,0,0));}let usize id=m::index(ref expressions,local.initializer);if(id!=0){let p=scalar::evaluate(source,frame,id,local.expected,graph,atoms,ref mut expressions,ref mut work,0);if(p.code!=0){return .err(p);}let e=*v::at<m::Expression>(ref expressions,id-1);if(e.state==2){return .err(m::fail(frame,1006,e.start,e.end,local.expected,0));}if(e.type!=0){if(types::at(graph,e.type).kind==1 && types::at(graph,e.type).scalar==7 || local.expected!=0 && local.expected!=e.type){return .err(m::fail(frame,1006,e.start,e.end,local.expected,e.type));}if(local.type==0){let slot=v::at_mut<m::Local>(ref mut locals,i);slot.type=e.type;changed=true;}}}i+=1;}
        i=1;while(i<=v::len<m::Expression>(ref expressions)){let e=*v::at<m::Expression>(ref expressions,i-1);if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,e.start,e.end,0,0));}var usize type=0;
            if(e.kind==2 && e.state==0){let usize use=names::use_at(resolved,i);if(use!=0){let r=names::reference_at(resolved,use);if(r.kind==1){type=v::at<m::Local>(ref locals,r.symbol-1).type;}}}
            else if(e.kind==3 && e.state==0){let usize child=m::index(ref expressions,e.left);if(child!=0){type=v::at<m::Expression>(ref expressions,child-1).type;}}
            if(e.type==0 && type!=0){let slot=v::at_mut<m::Expression>(ref mut expressions,i-1);slot.type=type;slot.state=1;changed=true;}i+=1;
        }
        let checked=args::resolve(source,frame,graph,atoms,ref mut expressions,ref mut calls,ref mut arguments,ref mut work);if(checked.problem.code!=0){return .err(checked.problem);}changed=changed || checked.changed;
        i=0;while(i<v::len<m::Return>(ref returns)){let r=*v::at<m::Return>(ref returns,i);if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,r.start,r.end,0,0));}let usize id=m::index(ref expressions,r.expression);var usize actual=0;
            if(r.expression==0){actual=m::atom(atoms,7);}else if(id!=0){let p=scalar::evaluate(source,frame,id,r.expected,graph,atoms,ref mut expressions,ref mut work,0);if(p.code!=0){return .err(p);}let e=*v::at<m::Expression>(ref expressions,id-1);if(e.state==2){return .err(m::fail(frame,1006,e.start,e.end,r.expected,0));}actual=e.type;}
            if(actual!=0){if(actual!=r.expected){let usize start=if(id==0){r.start}else{v::at<m::Expression>(ref expressions,id-1).start};let usize end=if(id==0){r.end}else{v::at<m::Expression>(ref expressions,id-1).end};return .err(m::fail(frame,1006,start,end,r.expected,actual));}let slot=v::at_mut<m::Return>(ref mut returns,i);slot.actual=actual;slot.checked=true;}i+=1;
        }
    }if(changed){return .err(m::fail(frame,10,0,0,0,0));}var usize unresolved=0;var usize inferred=0;var usize checked_returns=0;var usize i=0;while(i<v::len<m::Expression>(ref expressions)){if(v::at<m::Expression>(ref expressions,i).state==0){unresolved+=1;}i+=1;}i=0;while(i<v::len<m::Local>(ref locals)){let s=*v::at<m::Local>(ref locals,i);if(s.expected==0 && s.type!=0){inferred+=1;}i+=1;}i=0;while(i<v::len<m::Return>(ref returns)){if(v::at<m::Return>(ref returns,i).checked){checked_returns+=1;}i+=1;}
    return .ok(m::assemble(frame,expressions,locals,returns,calls,arguments,unresolved,inferred,checked_returns,work));
}
