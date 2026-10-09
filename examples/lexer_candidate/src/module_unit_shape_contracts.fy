module module_unit_shape_contracts;
import module_unit_interfaces as interfaces;
import module_types as types;
import module_ast_join as join;
import syntax as syntax;
import resource_abilities as abilities;
import std::bytes as bytes;
import std::vec as v;

fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut usize work,usize amount){if(*work>1048576 || amount>1048576-*work){return false;};*work=*work+amount;return true;}
pub fn join::Problem declaration(ref syntax::Parsed source,usize module_index,usize declaration,bool vector,ref mut usize work){
    let d=syntax::declaration_at(source,declaration);if(d.kind!=4 && d.kind!=6 && d.kind!=11){return fail(0,0,0,0);}if(!charge(work,1)){return fail(10,module_index,d.start,d.end);}
    if(d.generic_parameters!=0 && !(d.kind==6 && vector)){return fail(2,module_index,d.start,d.end);}if(d.kind==6){let a=abilities::read(source,declaration);if(!charge(work,a.work)){return fail(10,module_index,d.start,d.end);}if(a.code!=0){return fail(a.code,module_index,a.start,a.end);}}
    var usize i=0;while(i<d.field_count){let f=syntax::field_at(source,d.field_start+i);let name=syntax::source_view(source)[f.name_start..f.name_end];if(!charge(work,name.len+1)){return fail(10,module_index,f.start,f.end);}
        if(f.kind==0 || f.kind==3){var usize j=0;while(j<i){let old=syntax::field_at(source,d.field_start+j);if(!charge(work,name.len+1)){return fail(10,module_index,f.start,f.end);}if(old.kind==f.kind && bytes::equal(name,syntax::source_view(source)[old.name_start..old.name_end])){return fail(102,module_index,f.name_start,f.name_end);}j+=1;}}
        if(f.kind==3){var usize j=0;while(j<f.payload_count){let child=syntax::field_at(source,f.payload_start+j);let spelling=syntax::source_view(source)[child.name_start..child.name_end];var usize k=0;while(k<j){if(!charge(work,spelling.len+1)){return fail(10,module_index,child.start,child.end);}let old=syntax::field_at(source,f.payload_start+k);if(bytes::equal(spelling,syntax::source_view(source)[old.name_start..old.name_end])){return fail(102,module_index,child.name_start,child.name_end);}k+=1;}let t=syntax::type_at(source,child.type_root);if(t.kind==8){return fail(2,module_index,t.start,t.end);}j+=1;}}
        if(f.kind==0 && f.type_root!=0){let t=syntax::type_at(source,f.type_root);if(t.kind==2 || t.kind==3){return fail(1003,module_index,t.start,t.end);}if(t.kind==8 && (d.kind==6 || syntax::type_at(source,t.left).kind==8)){return fail(2,module_index,t.start,t.end);}}
        i+=1;
    }return fail(0,0,0,0);
}
fn Result<bool,join::Problem> owns_resource(ref v::Vec<types::Type> graph,usize id,usize module_index,usize start,usize end,ref mut usize work,usize depth){
    if(depth>=64 || !charge(work,1)){return .err(fail(10,module_index,start,end));}let t=types::at(graph,id);if(t.kind==3 || t.kind==13){return .ok(true);}if(t.kind==5 || t.kind==6){let first=switch(owns_resource(graph,t.first,module_index,start,end,work,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(first || t.kind==5){return .ok(first);}return owns_resource(graph,t.second,module_index,start,end,work,depth+1);}return .ok(false);
}
fn Result<bool,join::Problem> inert(ref v::Vec<types::Type> graph,ref v::Vec<interfaces::Shape> shapes,ref v::Vec<interfaces::Field> fields,usize id,usize module_index,usize start,usize end,ref mut usize work,ref mut v::Vec<usize> path,usize depth){
    if(depth>=64 || !charge(work,1)){return .err(fail(10,module_index,start,end));}let t=types::at(graph,id);if(t.kind==1){return .ok(t.scalar!=7);}if(t.kind==12){return .ok(true);}if(t.kind==5 || t.kind==6){let wrapper=v::at_mut<usize>(path,depth);*wrapper=id;let first=switch(inert(graph,shapes,fields,t.first,module_index,start,end,work,path,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!first || t.kind==5){return .ok(first);}return inert(graph,shapes,fields,t.second,module_index,start,end,work,path,depth+1);}if(t.kind!=2 && t.kind!=4){return .ok(false);}
    var usize i=0;while(i<depth){if(!charge(work,1)){return .err(fail(10,module_index,start,end));}if(*v::at<usize>(path,i)==id){return .ok(false);}i+=1;}let slot=v::at_mut<usize>(path,depth);*slot=id;i=0;
    while(i<v::len<interfaces::Shape>(shapes)){if(!charge(work,1)){return .err(fail(10,module_index,start,end));}let s=*v::at<interfaces::Shape>(shapes,i);if(s.type==id){var usize j=0;while(j<s.field_count){let f=*v::at<interfaces::Field>(fields,s.field_start+j);if(f.kind==0 || f.kind==4){let valid=switch(inert(graph,shapes,fields,f.type,module_index,start,end,work,path,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!valid){return .ok(false);}}j+=1;}return .ok(true);}i+=1;}return .err(fail(12,module_index,start,end));
}
fn Result<bool,join::Problem> scalar_leaf(ref v::Vec<types::Type> graph,ref v::Vec<interfaces::Shape> shapes,ref v::Vec<interfaces::Field> fields,usize id,usize module_index,usize start,usize end,ref mut usize work,ref mut v::Vec<usize> path,usize depth){
    if(depth>=64 || !charge(work,1)){return .err(fail(10,module_index,start,end));}let t=types::at(graph,id);if(t.kind==1){return .ok(t.scalar>=1 && t.scalar<=6);}if(t.kind!=2){return .ok(false);}var usize i=0;while(i<depth){if(!charge(work,1)){return .err(fail(10,module_index,start,end));}if(*v::at<usize>(path,i)==id){return .ok(false);}i+=1;}let slot=v::at_mut<usize>(path,depth);*slot=id;i=0;
    while(i<v::len<interfaces::Shape>(shapes)){if(!charge(work,1)){return .err(fail(10,module_index,start,end));}let s=*v::at<interfaces::Shape>(shapes,i);if(s.type==id){if(s.field_count==0){return .ok(false);}var usize j=0;while(j<s.field_count){let f=*v::at<interfaces::Field>(fields,s.field_start+j);let child=switch(scalar_leaf(graph,shapes,fields,f.type,module_index,start,end,work,path,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!child){return .ok(false);}j+=1;}return .ok(true);}i+=1;}return .err(fail(12,module_index,start,end));
}
pub fn join::Problem fields(ref v::Vec<syntax::Parsed> sources,ref v::Vec<types::Type> graph,ref v::Vec<interfaces::Shape> shapes,ref v::Vec<interfaces::Field> field_table,ref mut usize work){
    var v::Vec<usize> path=v::empty<usize>();var usize i=0;while(i<64){if(!charge(work,1)){return fail(10,0,0,0);}if(!v::push<usize>(ref mut path,0).is_ok){return fail(11,0,0,0);}i+=1;}i=0;
    while(i<v::len<interfaces::Shape>(shapes)){let s=*v::at<interfaces::Shape>(shapes,i);let owner=types::at(graph,s.type);if(owner.kind==2 || owner.kind==3 || owner.kind==13){var usize j=0;while(j<s.field_count){let f=*v::at<interfaces::Field>(field_table,s.field_start+j);if(!charge(work,1)){return fail(10,f.module_index,f.start,f.end);}let original=syntax::field_at(v::at<syntax::Parsed>(sources,f.module_index),f.source_index);let at=syntax::type_at(v::at<syntax::Parsed>(sources,f.module_index),original.type_root);let t=types::at(graph,f.type);
            if(t.kind>=7 && t.kind<=11 || t.kind==14 || t.kind==15 || t.kind==17 || t.kind==18){return fail(1003,f.module_index,at.start,at.end);}if(owner.kind==2){let owns=switch(owns_resource(graph,f.type,f.module_index,at.start,at.end,work,0)){case .ok(b):{b}case .err(p):{return p;}};if(owns){return fail(1002,f.module_index,at.start,at.end);}}
            if(t.kind==16){if(owner.kind!=2 || s.repr_c){return fail(2,f.module_index,at.start,at.end);}let valid=switch(scalar_leaf(graph,shapes,field_table,t.first,f.module_index,at.start,at.end,work,ref mut path,0)){case .ok(b):{b}case .err(p):{return p;}};if(!valid){return fail(2,f.module_index,at.start,at.end);}}
            j+=1;
        }}i+=1;
    }i=0;while(i<v::len<interfaces::Field>(field_table)){let f=*v::at<interfaces::Field>(field_table,i);if(f.kind==4){if(!charge(work,1)){return fail(10,f.module_index,f.start,f.end);}let original=syntax::field_at(v::at<syntax::Parsed>(sources,f.module_index),f.source_index);let at=syntax::type_at(v::at<syntax::Parsed>(sources,f.module_index),original.type_root);let valid=switch(inert(graph,shapes,field_table,f.type,f.module_index,at.start,at.end,work,ref mut path,0)){case .ok(b):{b}case .err(p):{return p;}};if(!valid){return fail(1004,f.module_index,at.start,at.end);}}i+=1;}return fail(0,0,0,0);
}
