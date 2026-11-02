module copied_closure_reach;
import syntax as syntax;
import std::vec as v;

// A lifted helper overlaps its owner's source span. Project reachable arenas,
// not all rows inside that span; original closure AST rows remain source facts.
struct Task{u32 kind;usize id;}
pub resource Mask{v::Vec<usize> expressions;v::Vec<usize> statements;v::Vec<usize> entries;bool enabled;cleanup{}}
fn Result<v::Vec<usize>,u32> zeros(usize count){var v::Vec<usize> out=v::empty<usize>();var usize i=0;while(i<=count){if(!v::push<usize>(ref mut out,0).is_ok){return .err(11);}i+=1;}return .ok(out);}
fn bool push(ref mut v::Vec<Task> pending,u32 kind,usize id){return id==0 || v::len<Task>(pending)<32768 && v::push<Task>(pending,Task{.kind,.id}).is_ok;}
pub fn bool selected(ref Mask m,u32 kind,usize id){if(!m.enabled || kind==4){return true;}if(kind==1){return id<v::len<usize>(ref m.expressions) && *v::at<usize>(ref m.expressions,id)!=0;}if(kind==2){return id<v::len<usize>(ref m.statements) && *v::at<usize>(ref m.statements,id)!=0;}return id<v::len<usize>(ref m.entries) && *v::at<usize>(ref m.entries,id)!=0;}
pub fn Result<Mask,u32> build(ref syntax::Parsed file,syntax::Declaration d,ref mut usize work){
    let enabled=syntax::closure_prepared(file);var expressions=switch(zeros(if(enabled){syntax::expression_count(file)}else{0})){case .ok(p):{p}case .err(code):{return .err(code);}};var statements=switch(zeros(if(enabled){syntax::statement_count(file)}else{0})){case .ok(p):{p}case .err(code):{return .err(code);}};var entries=switch(zeros(if(enabled){syntax::entry_count(file)}else{0})){case .ok(p):{p}case .err(code):{return .err(code);}};
    if(enabled){var usize i=d.entry_start;while(i<d.entry_start+d.entry_count){if(syntax::entry_at(file,i).kind==1){let slot=v::at_mut<usize>(ref mut entries,i+1);*slot=1;}i+=1;}var v::Vec<Task> pending=v::empty<Task>();if(!push(ref mut pending,2,d.body) || !push(ref mut pending,2,d.cleanup_body) || !push(ref mut pending,1,d.expression)){return .err(11);}
        while(v::len<Task>(ref pending)!=0){if(*work>=1048576){return .err(10);};*work=*work+1;let task=*v::at<Task>(ref pending,v::len<Task>(ref pending)-1);discard v::pop<Task>(ref mut pending);
            if(task.kind==1){if(task.id>=v::len<usize>(ref expressions)){return .err(9);}if(*v::at<usize>(ref expressions,task.id)!=0){continue;}{let slot=v::at_mut<usize>(ref mut expressions,task.id);*slot=1;}let n=syntax::expression_at(file,task.id);if(n.kind==34 || n.kind==35){return .err(3);}let u32 left=if(n.kind==12 || n.kind==14 || n.kind==31){2}else{1};if(!push(ref mut pending,left,n.left)){return .err(11);}if(n.kind!=20 && n.kind!=23 && !push(ref mut pending,if(n.kind==18){2}else{1},n.right)){return .err(11);}}
            else{if(task.id>=v::len<usize>(ref statements)){return .err(9);}if(*v::at<usize>(ref statements,task.id)!=0){continue;}{let slot=v::at_mut<usize>(ref mut statements,task.id);*slot=1;}let s=syntax::statement_at(file,task.id);if(s.entry!=0){if(s.entry>=v::len<usize>(ref entries)){return .err(9);}let slot=v::at_mut<usize>(ref mut entries,s.entry);*slot=1;}if(!push(ref mut pending,1,s.expression) || !push(ref mut pending,1,s.target) || !push(ref mut pending,2,s.first)){return .err(11);}if((s.kind==7 || s.kind==14 || s.kind==15) && !push(ref mut pending,2,s.second)){return .err(11);}}
        }
    }return .ok(Mask{.expressions,.statements,.entries,.enabled});
}
