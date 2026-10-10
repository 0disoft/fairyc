module module_unit_loan_iteration;
import module_unit_origin_inputs as inputs;
import module_unit_origin_sets as sets;
import module_unit_origin_flow_model as g;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_loan_model as loan;
import module_unit_loan_checks as loans;
import module_unit_loan_facts as facts;
import module_unit_loan_generations as generations;
import module_unit_loan_reference as references;
import std::vec as v;

pub struct Seed {bool shared_hold;usize symbol;g::Value held;g::Value item;bool borrowed;bool mutable;}
pub fn Result<usize,m::Problem> symbol(ref m::Values body,usize target,ref mut g::Context c){var usize i=1;while(i<=m::local_count(body)){if(!m::charge(ref mut c.work,1)){return .err(g::fail(c,10,0,0));}let local=m::local_at(body,i);if(local.kind==5 && local.source_index==target){return .ok(i);}i+=1;}return .err(g::fail(c,12,0,0));}
pub fn Result<Seed,m::Problem> prepare(ref inputs::Inputs input,ref values::Plan primary,ref m::Values body,ref v::Vec<g::Value> env,usize target,m::Expression view,g::Value value,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut v::Vec<loan::Held> held,ref mut v::Vec<loan::Conflict> conflicts,ref mut g::Context c){
    let item=switch(symbol(body,target,c)){case .ok(n):{n}case .err(p):{return .err(p);}};let local=m::local_at(body,item);if(view.type==0 || local.type==0){return .err(g::fail(c,12,view.start,view.end));}
    let source_type=inputs::type_at(input,primary,c.frame,view.type);let item_type=inputs::type_at(input,primary,c.frame,local.type);let borrowed=item_type.kind==7 || item_type.kind==8;let mutable=item_type.kind==8;var selected=value;var bool shared_hold=borrowed && !mutable && source_type.kind!=16;
    if(source_type.kind==16){let root=switch(references::root(body,view.source_index,c)){case .ok(e):{e}case .err(p):{return .err(p);}};let root_type=inputs::type_at(input,primary,c.frame,root.type);
        if(root_type.kind==7 || root_type.kind==8){if(root.binding==0 || root.binding>v::len<g::Value>(env)){return .err(g::fail(c,12,view.start,view.end));}selected=*v::at<g::Value>(env,root.binding-1);let parent=switch(inputs::binding_owner(input,c.frame,root)){case .ok(n):{n}case .err(p):{return .err(p);}};
            if(mutable){selected=switch(generations::child(selected,parent,view.start,view.end,entries,members,records,c)){case .ok(v):{v}case .err(p):{return .err(p);}};}else if(root_type.kind==8){shared_hold=true;selected=switch(facts::seed(selected,false,parent,view.start,view.end,entries,members,records,c)){case .ok(v):{v}case .err(p):{return .err(p);}};}
        }else{let owner=switch(loans::root(input,body,view.source_index,c)){case .ok(n):{n}case .err(p):{return .err(p);}};if(owner==0){return .err(g::fail(c,12,view.start,view.end));}selected.origin=switch(sets::owner(owner)){case .ok(n):{n}case .err(code):{return .err(g::fail(c,code,view.start,view.end));}};selected.loan=switch(loans::create(owner,mutable,view.start,view.end,records,held,conflicts,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
            if(mutable){let root=v::at_mut<loan::Loan>(records,selected.loan-3);root.subview=true;}}
    }
    // A written slice or an implicit array descriptor is retained independently
    // of the per-iteration item. Binding 0 names this hidden descriptor.
    if(source_type.kind==16 || view.kind==26){let kept=facts::retain(selected,0,entries,members,records,c,view.start,view.end);if(kept.code!=0){return .err(kept);}let used=facts::used(selected,entries,members,records,c,view.start,view.end);if(used.code!=0){return .err(used);}}
    var element=g::Value{.subview=false,.loan=0,.origin=0,.allocation=0};if(borrowed){element=if(mutable){switch(generations::child(selected,0,view.start,view.end,entries,members,records,c)){case .ok(v):{v}case .err(p):{return .err(p);}}}else{switch(facts::seed(selected,false,0,view.start,view.end,entries,members,records,c)){case .ok(v):{v}case .err(p):{return .err(p);}}};let kept=facts::retain(element,inputs::frame(input,c.frame).local_start+item,entries,members,records,c,view.start,view.end);if(kept.code!=0){return .err(kept);}}
    return .ok(Seed{.shared_hold,.symbol=item,.held=selected,.item=element,.borrowed,.mutable});
}
