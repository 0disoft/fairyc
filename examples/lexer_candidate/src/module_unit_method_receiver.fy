module module_unit_method_receiver;
import module_unit_name_model as names;
import module_unit_interfaces as interfaces;
import module_signatures as signatures;
import module_types as types;
import syntax as syntax;

// This pre-value pass selects a nominal method owner, never a checked value or
// borrow permission. The ordinary argument/type/Flow passes still verify every
// initializer, receiver, move and mutable borrow before any IR can escape.
fn bool charge(ref mut usize work){if(*work>=1048576){return false;};*work+=1;return true;}
fn Result<types::Type,u32> nominal(ref interfaces::Interfaces imported,usize id){
    if(id==0 || id>interfaces::type_count(imported)){return .err(9);}let t=interfaces::type_at(imported,id);
    if(t.kind==7 || t.kind==8){if(t.first==0 || t.first>=id){return .err(9);}let inner=interfaces::type_at(imported,t.first);if(inner.kind==2 || inner.kind==3){return .ok(inner);}return .err(9);}
    return if(t.kind==2 || t.kind==3){.ok(t)}else{.err(9)};
}
pub fn Result<types::Type,u32> resolve(ref syntax::Parsed source,ref signatures::Program metadata,ref interfaces::Interfaces imported,ref names::Resolved resolved,usize module_index,usize symbol,ref mut usize work){
    var usize selected=symbol;var usize depth=0;
    while(depth<64){
        depth+=1;if(!charge(work)){return .err(10);}if(selected==0 || selected>names::local_count(resolved)){return .err(9);}let local=names::symbol_at(resolved,selected);
        if(local.type!=0){return nominal(imported,local.type);}if(local.entry==0 || local.pattern!=0){return .err(9);}
        // Entry IDs belong to a lowered body AST. Bind its exact written span
        // back to retained source instead of treating it as a source entry ID.
        var usize expression=0;var usize i=0;
        while(i<syntax::entry_count(source)){if(!charge(work)){return .err(10);}let e=syntax::entry_at(source,i);i+=1;if(e.name_start==local.start && e.name_end==local.end){if(expression!=0){return .err(9);}expression=e.expression;}}
        if(expression==0){return .err(9);}var usize next=0;
        while(expression!=0 && depth<64){
            depth+=1;if(!charge(work)){return .err(10);}let e=syntax::expression_at(source,expression);
            if(e.kind==3 || e.kind==21 || e.kind==22){expression=e.left;continue;}
            if(e.kind==2){
                i=1;while(i<=names::reference_count(resolved)){if(!charge(work)){return .err(10);}let r=names::reference_at(resolved,i);i+=1;if(r.source_index==expression && r.kind==1){if(next!=0 || r.symbol>=selected){return .err(9);}next=r.symbol;}}
                break;
            }
            if(e.kind==10 && e.left!=0 && syntax::expression_at(source,e.left).kind==2){
                let found=signatures::declaration_lookup(metadata,module_index,syntax::expression_text(source,e.left));if(found.code!=0){return .err(found.code);}let d=signatures::declaration_at(metadata,found.item);if(d.parameters!=0 || d.kind!=4 && d.kind!=6){return .err(9);}
                return .ok(types::Type{.kind=if(d.kind==4){2}else{3},.scalar=0,.module_index=d.module_index,.declaration=d.declaration,.first=0,.second=0});
            }
            if(e.kind==6 && e.left!=0 && syntax::expression_at(source,e.left).kind==2){
                let found=signatures::lookup(metadata,module_index,syntax::expression_text(source,e.left));if(found.code!=0){return .err(found.code);}return nominal(imported,signatures::at(metadata,found.item).result);
            }
            return .err(9);
        }
        if(next==0){return .err(if(depth>=64){10}else{9});}selected=next;
    }return .err(10);
}
