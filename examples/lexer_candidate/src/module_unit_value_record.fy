module module_unit_value_record;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_value_field as fields;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import std::vec as v;

pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);let named=syntax::expression_at(source,e.left);
    // Contextual enum records await the variant consumer, preserving their IDs.
    if(named.kind==28){return m::fail(frame,0,0,0,0,0);}if(e.target==0){return m::fail(frame,2,e.start,e.end,0,0);}let t=types::at(graph,e.target);if(t.kind!=2 && t.kind!=3 && t.kind!=13){return m::fail(frame,2,e.start,e.end,0,e.target);}if((t.kind==3 || t.kind==13) && t.module_index!=frame.module_index){return m::fail(frame,103,named.start,named.end,0,e.target);}
    var usize selected=0;var usize i=0;while(i<interfaces::shape_count(imported)){if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}if(interfaces::shape_at(imported,i).type==e.target){selected=i+1;break;}i+=1;}if(selected==0){return m::fail(frame,12,e.start,e.end,0,e.target);}let shape=interfaces::shape_at(imported,selected-1);
    var v::Vec<usize> seen=v::empty<usize>();i=0;while(i<shape.field_count){if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}if(!v::push<usize>(ref mut seen,0).is_ok){return m::fail(frame,11,e.start,e.end,0,0);}i+=1;}
    // Reverse the field list once; source order is independent of declaration order.
    var v::Vec<usize> order=v::empty<usize>();var usize link=e.right;while(link!=0){if(!m::charge(work,1) || v::len<usize>(ref order)>=1024){return m::fail(frame,10,e.start,e.end,0,0);}let row=syntax::expression_at(source,link);if(row.kind!=11){return m::fail(frame,12,row.start,row.end,0,0);}if(!v::push<usize>(ref mut order,link).is_ok){return m::fail(frame,11,row.start,row.end,0,0);}link=row.right;}
    var bool complete=true;i=v::len<usize>(ref order);while(i>0){i-=1;let usize original=*v::at<usize>(ref order,i);let row=syntax::expression_at(source,original);let usize wrapper=m::index(expressions,original);if(wrapper==0){return m::fail(frame,12,row.start,row.end,0,0);}let field=switch(fields::resolve(imported,graph,atoms,frame,*v::at<m::Expression>(expressions,wrapper-1),e.target,syntax::expression_text(source,original),work)){case .ok(f):{f}case .err(p):{return p;}};
        if(field.field.id==0 || field.field.id<=shape.field_start || field.field.id>shape.field_start+shape.field_count){return m::fail(frame,12,row.start,row.end,0,0);}let usize position=field.field.id-1-shape.field_start;if(*v::at<usize>(ref seen,position)!=0){return m::fail(frame,2,row.start,row.end,0,0);}let slot=v::at_mut<usize>(ref mut seen,position);*slot=original;
        let usize child=m::index(expressions,row.left);if(child==0){return m::fail(frame,12,row.start,row.end,0,0);}let p=scalar::evaluate(source,frame,child,field.type,imported,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let actual=*v::at<m::Expression>(expressions,child-1);if(actual.state==2 || actual.type!=0 && actual.type!=field.type){return m::fail(frame,1006,actual.start,actual.end,field.type,actual.type);}if(actual.type==0){complete=false;}let annotation=v::at_mut<m::Expression>(expressions,wrapper-1);annotation.field=field.field;annotation.expected=field.type;
    }
    if(v::len<usize>(ref order)!=shape.field_count){return m::fail(frame,2,e.start,e.end,0,e.target);}if(complete){let slot=v::at_mut<m::Expression>(expressions,id-1);slot.type=e.target;slot.expected=expected;slot.state=1;slot.place=m::no_place();}return m::fail(frame,0,0,0,0,0);
}
