module module_unit_cleanup_policy;
import module_unit_move_model as g;
import module_unit_moves as moves;
import module_unit_value_model as m;
import module_unit_values as values;
import module_unit_interfaces as interfaces;
import syntax as syntax;
import effect_declarations as effects;
import std::bytes as bytes;
import std::vec as v;

pub resource Plan {v::Vec<g::Diagnostic> diagnostics;usize work;cleanup{}}
struct Context {m::Frame frame;usize work;}
pub fn usize count(ref Plan p){return v::len<g::Diagnostic>(ref p.diagnostics);}
pub fn g::Diagnostic at(ref Plan p,usize index){return *v::at<g::Diagnostic>(ref p.diagnostics,index);}
pub fn usize work(ref Plan p){return p.work;}
fn m::Problem fail(ref Context c,u32 code,usize start,usize end){return m::fail(c.frame,code,start,end,0,0);}
fn Result<bool,m::Problem> mark(usize type,ref mut v::Vec<usize> seen,ref mut Context c){var usize i=0;while(i<v::len<usize>(seen)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}if(*v::at<usize>(seen,i)==type){return .ok(false);}i+=1;}if(v::len<usize>(seen)>=4096 || !m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}return if(v::push<usize>(seen,type).is_ok){.ok(true)}else{.err(fail(c,11,0,0))};}
fn Result<interfaces::Shape,m::Problem> shape(ref interfaces::Interfaces imported,usize type,ref mut Context c){var usize i=0;while(i<interfaces::shape_count(imported)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}let s=interfaces::shape_at(imported,i);if(s.type==type){return .ok(s);}i+=1;}return .err(fail(c,12,0,0));}
// Missing drop on an outer resource owns the rejection. Otherwise inspect its
// ordered fields so a droppable wrapper cannot destroy a no-drop inner value.
fn m::Problem blockers(ref values::Plan primary,ref interfaces::Interfaces imported,usize type,ref mut v::Vec<usize> denied,ref mut v::Vec<usize> seen,usize depth,ref mut Context c){
    if(type==0){return fail(c,0,0,0);}if(depth>=64 || !m::charge(ref mut c.work,1)){return fail(c,10,0,0);}let t=values::type_at(primary,type);
    if(t.kind==5 || t.kind==6){let first=blockers(primary,imported,t.first,denied,seen,depth+1,c);return if(first.code!=0 || t.kind==5){first}else{blockers(primary,imported,t.second,denied,seen,depth+1,c)};}
    if(t.kind!=3 && t.kind!=13){return fail(c,0,0,0);}let fresh=switch(mark(type,seen,c)){case .ok(b):{b}case .err(p):{return p;}};if(!fresh){return fail(c,0,0,0);}let s=switch(shape(imported,type,c)){case .ok(s):{s}case .err(p):{return p;}};
    if((s.abilities&1)==0){return if(v::push<usize>(denied,type).is_ok){fail(c,0,0,0)}else{fail(c,11,s.start,s.end)};}var usize i=0;while(i<s.field_count){if(!m::charge(ref mut c.work,1)){return fail(c,10,s.start,s.end);}let field=interfaces::field_at(imported,s.field_start+i);let p=blockers(primary,imported,field.type,denied,seen,depth+1,c);if(p.code!=0){return p;}i+=1;}return fail(c,0,0,0);
}
// Complete-root cleanup runs each nominal owner's declared body and its
// ordered resource fields. Tagged alternatives retain the same requirement.
fn m::Problem required(ref values::Plan primary,ref interfaces::Interfaces imported,usize type,ref mut v::Vec<usize> owners,ref mut v::Vec<usize> seen,usize depth,ref mut Context c){
    if(type==0){return fail(c,0,0,0);}if(depth>=64 || !m::charge(ref mut c.work,1)){return fail(c,10,0,0);}let t=values::type_at(primary,type);
    if(t.kind==5 || t.kind==6){let first=required(primary,imported,t.first,owners,seen,depth+1,c);return if(first.code!=0 || t.kind==5){first}else{required(primary,imported,t.second,owners,seen,depth+1,c)};}
    if(t.kind!=3 && t.kind!=13){return fail(c,0,0,0);}let fresh=switch(mark(type,seen,c)){case .ok(b):{b}case .err(p):{return p;}};if(!fresh){return fail(c,0,0,0);}if(!v::push<usize>(owners,type).is_ok){return fail(c,11,0,0);}let s=switch(shape(imported,type,c)){case .ok(s):{s}case .err(p):{return p;}};var usize i=0;while(i<s.field_count){if(!m::charge(ref mut c.work,1)){return fail(c,10,s.start,s.end);}let f=interfaces::field_at(imported,s.field_start+i);let p=required(primary,imported,f.type,owners,seen,depth+1,c);if(p.code!=0){return p;}i+=1;}return fail(c,0,0,0);
}

fn m::Problem emit(g::Fact request,u32 code,usize module_index,usize start,usize end,ref mut v::Vec<g::Diagnostic> diagnostics,ref mut Context c){if(start>=end){return fail(c,12,request.start,request.end);}var usize i=0;while(i<v::len<g::Diagnostic>(diagnostics)){if(!m::charge(ref mut c.work,1)){return fail(c,10,request.start,request.end);}let d=*v::at<g::Diagnostic>(diagnostics,i);if(d.code==code && d.module_index==c.frame.module_index && d.start==request.start && d.end==request.end && d.field_module==module_index && d.field_start==start && d.field_end==end){return fail(c,0,0,0);}i+=1;}if(v::len<g::Diagnostic>(diagnostics)>=8192 || !m::charge(ref mut c.work,1)){return fail(c,10,request.start,request.end);}let d=g::Diagnostic{.code,.frame=request.frame,.module_index=c.frame.module_index,.start=request.start,.end=request.end,.declaration_start=0,.declaration_end=0,.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module=module_index,.field_start=start,.field_end=end};return if(v::push<g::Diagnostic>(diagnostics,d).is_ok){fail(c,0,0,0)}else{fail(c,11,request.start,request.end)};}
fn Result<usize,m::Problem> anchor(ref syntax::Parsed source,usize declaration,ref u8[] name,ref mut Context c){let d=syntax::declaration_at(source,declaration-1);var usize i=0;while(i<d.attribute_count){let a=syntax::attribute_at(source,d.attribute_start+i);let text=syntax::source_view(source)[a.name_start..a.name_end];if(!m::charge(ref mut c.work,text.len+1)){return .err(fail(c,10,a.start,a.end));}if(bytes::equal(text,name)){return .ok(d.attribute_start+i+1);}i+=1;}return .ok(0);}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources,ref values::Plan primary,ref interfaces::Interfaces imported,ref moves::Plan moved){
    var v::Vec<g::Diagnostic> diagnostics=v::empty<g::Diagnostic>();var Context c=Context{.frame=m::Frame{.kind=0,.context=0,.module_index=0,.declaration=0,.result=0},.work=moves::work(moved)};var usize i=0;
    while(i<moves::cleanup_count(moved)){let request=moves::cleanup_at(moved,i);if(request.frame==0 || request.frame>values::count(primary) || request.kind!=5 || request.type==0 || request.type>values::type_count(primary)){return .err(fail(ref c,12,request.start,request.end));}c.frame=m::frame(values::at(primary,request.frame-1));let caller=v::at<syntax::Parsed>(sources,c.frame.module_index);let declaration=syntax::declaration_at(caller,c.frame.declaration-1);if(!m::charge(ref mut c.work,declaration.attribute_count+1)){return .err(fail(ref c,10,request.start,request.end));}let authority=effects::read(caller,c.frame.declaration-1);if(authority.code!=0){return .err(fail(ref c,authority.code,authority.start,authority.end));}
        var v::Vec<usize> denied=v::empty<usize>();var v::Vec<usize> seen=v::empty<usize>();let blocked=blockers(primary,imported,request.type,ref mut denied,ref mut seen,0,ref mut c);if(blocked.code!=0){return .err(blocked);}
        if(v::len<usize>(ref denied)!=0){var usize j=0;while(j<v::len<usize>(ref denied)){let s=switch(shape(imported,*v::at<usize>(ref denied,j),ref mut c)){case .ok(s):{s}case .err(p):{return .err(p);}};let source=v::at<syntax::Parsed>(sources,s.module_index);let at=switch(anchor(source,s.declaration,b"abilities",ref mut c)){case .ok(n):{n}case .err(p):{return .err(p);}};if(at==0){return .err(fail(ref c,12,s.start,s.end));}let a=syntax::attribute_at(source,at-1);let p=emit(request,1108,s.module_index,a.start,a.end,ref mut diagnostics,ref mut c);if(p.code!=0){return .err(p);}j+=1;}}
        else{var v::Vec<usize> owners=v::empty<usize>();discard v::truncate<usize>(ref mut seen,0);let needs=required(primary,imported,request.type,ref mut owners,ref mut seen,0,ref mut c);if(needs.code!=0){return .err(needs);}var usize j=0;while(j<v::len<usize>(ref owners)){let s=switch(shape(imported,*v::at<usize>(ref owners,j),ref mut c)){case .ok(s):{s}case .err(p):{return .err(p);}};if(!effects::includes(authority.bits,s.cleanup_effects)){let source=v::at<syntax::Parsed>(sources,s.module_index);let at=switch(anchor(source,s.declaration,b"effects",ref mut c)){case .ok(n):{n}case .err(p):{return .err(p);}};if(at==0){return .err(fail(ref c,12,s.start,s.end));}let a=syntax::attribute_at(source,at-1);let p=emit(request,7102,s.module_index,a.start,a.end,ref mut diagnostics,ref mut c);if(p.code!=0){return .err(p);}}j+=1;}}
        i+=1;
    }return .ok(Plan{.diagnostics,.work=c.work});
}
