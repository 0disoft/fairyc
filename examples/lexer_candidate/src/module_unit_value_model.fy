module module_unit_value_model;
import module_types as types;
import module_unit_body_ast as ast;
import std::vec as v;

pub struct Frame {u32 kind;usize context;usize module_index;usize declaration;usize result;}
// state 0 unresolved value; 1 known value type; 2 non-value syntax/callee.
pub struct Expression {usize source_index;u32 kind;usize start;usize end;usize left;usize right;usize target;usize expected;usize type;u32 state;bool flexible;bool minimum;}
pub struct Atoms {usize[8] scalar;usize text;usize bytes;usize cstr;}
pub struct Seed {usize result;usize instance;}
pub struct Invocation {u32 kind;usize expression;usize source_index;usize target_module;usize target_declaration;usize instance;usize argument_start;usize argument_count;usize start;usize end;bool checked;}
pub struct Argument {usize invocation;usize position;usize expression;usize source_index;usize expected;usize actual;usize start;usize end;usize label_start;usize label_end;bool checked;bool shared;}
pub struct Scan {Problem problem;bool changed;}
pub struct Local {u32 kind;usize source_index;usize initializer;usize expected;usize type;usize start;usize end;}
pub struct Return {usize expression;usize expected;usize actual;bool checked;usize start;usize end;}
pub struct Problem {u32 code;usize module_index;usize start;usize end;usize expected;usize actual;}
pub resource Values {Frame frame;v::Vec<Expression> expressions;v::Vec<Local> locals;v::Vec<Return> returns;v::Vec<Invocation> calls;v::Vec<Argument> arguments;usize unresolved;usize inferred;usize checked_returns;usize work;cleanup{}}
pub fn Values assemble(Frame frame,v::Vec<Expression> expressions,v::Vec<Local> locals,v::Vec<Return> returns,v::Vec<Invocation> calls,v::Vec<Argument> arguments,usize unresolved,usize inferred,usize checked_returns,usize work){return Values{.frame,.expressions,.locals,.returns,.calls,.arguments,.unresolved,.inferred,.checked_returns,.work};}
pub fn Frame frame(ref Values p){return p.frame;}
pub fn usize work(ref Values p){return p.work;}
pub fn usize unresolved(ref Values p){return p.unresolved;}
pub fn usize inferred(ref Values p){return p.inferred;}
pub fn usize checked_returns(ref Values p){return p.checked_returns;}
pub fn Problem fail(Frame f,u32 code,usize start,usize end,usize expected,usize actual){return Problem{.code,.module_index=f.module_index,.start,.end,.expected,.actual};}
pub fn bool charge(ref mut usize work,usize amount){return ast::charge(work,amount);}
pub fn usize index(ref v::Vec<Expression> expressions,usize source_index){if(source_index==0){return 0;}var usize low=0;var usize high=v::len<Expression>(expressions);while(low<high){let usize mid=low+(high-low)/2;if(v::at<Expression>(expressions,mid).source_index<source_index){low=mid+1;}else{high=mid;}}return if(low<v::len<Expression>(expressions) && v::at<Expression>(expressions,low).source_index==source_index){low+1}else{0};}
pub fn usize find(ref v::Vec<types::Type> graph,u32 kind,u32 scalar){var usize i=1;while(i<=types::count(graph)){let t=types::at(graph,i);if(t.kind==kind && t.scalar==scalar && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0){return i;}i+=1;}return 0;}
pub fn Atoms atoms(ref v::Vec<types::Type> graph){var Atoms p=Atoms{.scalar=[0;8],.text=0,.bytes=0,.cstr=0};var usize i=1;while(i<=types::count(graph)){let t=types::at(graph,i);if(t.kind==1 && t.scalar<=7){let usize slot=t.scalar as usize;p.scalar[slot]=i;}else if(t.kind==9){p.text=i;}else if(t.kind==10){p.bytes=i;}else if(t.kind==14){p.cstr=i;}i+=1;}return p;}
pub fn usize atom(ref Atoms p,u32 scalar){return p.scalar[scalar as usize];}
pub fn usize local_count(ref Values p){return v::len<Local>(ref p.locals);}
pub fn Local local_at(ref Values p,usize id){return *v::at<Local>(ref p.locals,id-1);}
pub fn usize expression_count(ref Values p){return v::len<Expression>(ref p.expressions);}
pub fn Expression expression_at(ref Values p,usize id){return *v::at<Expression>(ref p.expressions,id-1);}
pub fn usize return_count(ref Values p){return v::len<Return>(ref p.returns);}
pub fn Return return_at(ref Values p,usize index){return *v::at<Return>(ref p.returns,index);}
pub fn usize call_count(ref Values p){return v::len<Invocation>(ref p.calls);}
pub fn Invocation call_at(ref Values p,usize index){return *v::at<Invocation>(ref p.calls,index);}
pub fn usize argument_count(ref Values p){return v::len<Argument>(ref p.arguments);}
pub fn Argument argument_at(ref Values p,usize index){return *v::at<Argument>(ref p.arguments,index);}
pub fn usize checked_calls(ref Values p){var usize count=0;var usize i=0;while(i<call_count(p)){if(call_at(p,i).checked){count+=1;}i+=1;}return count;}
pub fn usize checked_arguments(ref Values p){var usize count=0;var usize i=0;while(i<argument_count(p)){if(argument_at(p,i).checked){count+=1;}i+=1;}return count;}
pub fn usize shared_arguments(ref Values p){var usize count=0;var usize i=0;while(i<argument_count(p)){if(argument_at(p,i).shared){count+=1;}i+=1;}return count;}
pub fn usize cast_count(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){if(expression_at(p,i).kind==23){count+=1;}i+=1;}return count;}
pub fn usize checked_casts(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){let e=expression_at(p,i);if(e.kind==23 && e.state==1 && e.type==e.target){count+=1;}i+=1;}return count;}
